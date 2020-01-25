package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/asishrs/smartthings-ringalarmv2/cmd"
	"github.com/asishrs/smartthings-ringalarmv2/httputil"
	"github.com/asishrs/smartthings-ringalarmv2/public"
	"github.com/asishrs/smartthings-ringalarmv2/wsutil"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func clientError(status int) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       http.StatusText(status),
	}, nil
}

func getAccessToken(apiRequest public.Request) (string, error) {
	if apiRequest.RefreshToken != "" {
		log.Println("Using Refresh Token to Authenticate Ring API")
		oauthResponse, _ := httputil.AuthRequestWithRefreshToken("https://oauth.ring.com/oauth/token", httputil.OAuthRequestWithRefreshToken{"ring_official_ios", "refresh_token", apiRequest.RefreshToken})
		return oauthResponse.AccessToken, nil
	} else {
		log.Println("Using User Name & Password to Authenticate Ring API")
		oauthResponse, _ := httputil.AuthRequest("https://oauth.ring.com/oauth/token", httputil.OAuthRequest{"ring_official_ios", "password", apiRequest.Password, "client", apiRequest.User}, "")
		return oauthResponse.AccessToken, nil
	}
}

func getLocationId(apiRequest public.Request, accessToken string) string {
	locationID := apiRequest.LocationID
	if len(locationID) == 0 {
		locationID = httputil.LocationRequest("https://app.ring.com/rhq/v1/devices/v1/locations", accessToken)
	}

	return locationID
}

func getZID(apiRequest public.Request) (string, error) {
	zID := apiRequest.ZID
	if len(zID) == 0 {
		accessToken, _ := getAccessToken(apiRequest)
		locationID := getLocationId(apiRequest, accessToken)
		ringDeviceInfo, err := getDevices(locationID, accessToken)
		if err != nil {
			log.Println(err)
			return "", err
		}
		for i := range ringDeviceInfo.Body {
			if ringDeviceInfo.Body[i].General.V2.DeviceType == "access-code" {
				zID = ringDeviceInfo.Body[i].General.V2.AdapterZID
			}
		}
	}
	return zID, nil
}

func getDevices(locationID string, accessToken string) (*wsutil.RingDeviceInfo, error) {
	connection := httputil.ConnectionRequest("https://app.ring.com/api/v1/rs/connections", locationID, accessToken)
	return wsutil.ActiveDevices(connection)
}

func getStatus(apiRequest public.Request) (events.APIGatewayProxyResponse, error) {
	//log.Printf("Request: %v", apiRequest)
	accessToken, _ := getAccessToken(apiRequest)
	locationID := getLocationId(apiRequest, accessToken)
	log.Printf("LocationID %v", locationID)

	var ringEvents []public.RingDeviceEvent
	history := httputil.HistoryRequest("https://app.ring.com/api/v1/rs/history", accessToken, locationID, strconv.Itoa(apiRequest.HistoryLimit))

	for i := range history {
		//result, _ := json.Marshal(history[i])
		//log.Printf("Histoy - %v", string(result))
		//log.Printf("Device: %s, Time : %v, Type: %s\n", history[i].Context.AffectedEntityName, time.Unix(0, history[i].Context.EventOccurredTsMs*int64(time.Millisecond)), history[i].Body[0].Impulse.ImpulseTypes[0].ImpulseType)
		val := history[i].Body[0].General.V2.AdapterType
		if history[i].Body[0].Impulse.ImpulseTypes != nil {
			val = history[i].Body[0].Impulse.ImpulseTypes[0].ImpulseType
		}
		ringEvents = append(ringEvents, public.RingDeviceEvent{history[i].Context.AffectedEntityName, history[i].Context.EventOccurredTsMs, val})
	}

	var deviceStatus []public.RingDeviceStatus
	ringDeviceInfo, _ := getDevices(locationID, accessToken)
	for i := range ringDeviceInfo.Body {
		// log.Printf("RDName: %s, Type: %s, Fault: %v, Mode: %s\n", ringDeviceInfo.Body[i].General.V2.Name, ringDeviceInfo.Body[i].General.V2.DeviceType, ringDeviceInfo.Body[i].Device.V1.Faulted, ringDeviceInfo.Body[i].Device.V1.Mode)
		deviceStatus = append(deviceStatus, public.RingDeviceStatus{ringDeviceInfo.Body[i].General.V2.ZID, ringDeviceInfo.Body[i].General.V2.Name, ringDeviceInfo.Body[i].General.V2.DeviceType, ringDeviceInfo.Body[i].Device.V1.Faulted, ringDeviceInfo.Body[i].Device.V1.Mode})
	}

	// for i := range deviceStatus {
	// 	log.Printf("DName: %s, Type: %s, Fault: %v, Mode: %s\n", deviceStatus[i].Name, deviceStatus[i].Type, deviceStatus[i].Faulted, deviceStatus[i].Mode)
	// }

	result, err := json.Marshal(public.Response{deviceStatus, ringEvents})
	if err != nil {
		log.Println(err)
		return clientError(http.StatusInternalServerError)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(result),
	}, nil
}

func setStatus(apiRequest public.Request, status string) (events.APIGatewayProxyResponse, error) {
	accessToken, _ := getAccessToken(apiRequest)
	locationID := getLocationId(apiRequest, accessToken)
	zID, err := getZID(apiRequest)
	if err != nil {
		return clientError(http.StatusInternalServerError)
	}

	connection := httputil.ConnectionRequest("https://app.ring.com/api/v1/rs/connections", locationID, accessToken)
	wsutil.Status(zID, status, connection)
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "{\"message\": \"Success\"}",
	}, nil
}

func getMetaData(apiRequest public.Request) (events.APIGatewayProxyResponse, error) {
	accessToken, _ := getAccessToken(apiRequest)
	locationID := getLocationId(apiRequest, accessToken)
	zID, err := getZID(apiRequest)
	if err != nil {
		return clientError(http.StatusInternalServerError)
	}
	result, err := json.Marshal(public.RingMetaData{locationID, zID})
	if err != nil {
		log.Println(err)
		return clientError(http.StatusInternalServerError)
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(result),
	}, nil

}

// Handler is your Lambda function handler
// It uses Amazon API Gateway request/responses provided by the aws-lambda-go/events package,
// However you could use other event sources (S3, Kinesis etc), or JSON-decoded primitive types such as 'string'.
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("Version 2.3")
	var apiRequest public.Request
	err := json.Unmarshal([]byte(request.Body), &apiRequest)
	if err != nil {
		return clientError(http.StatusUnprocessableEntity)
	}

	pathParams := request.PathParameters
	action := pathParams["ring-action"]
	switch action {
	case "status":
		return getStatus(apiRequest)
	case "home":
		return setStatus(apiRequest, "some")
	case "away":
		return setStatus(apiRequest, "all")
	case "off":
		return setStatus(apiRequest, "none")
	case "meta":
		return getMetaData(apiRequest)
	default:
		return clientError(http.StatusUnprocessableEntity)
	}
}

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		cmd.Execute()
	} else {
		lambda.Start(Handler)
	}
}
