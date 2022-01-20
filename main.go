package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/asishrs/smartthings-ringalarmv2/cmd"
	"github.com/asishrs/smartthings-ringalarmv2/httputil"
	"github.com/asishrs/smartthings-ringalarmv2/public"
	"github.com/asishrs/smartthings-ringalarmv2/wsutil"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var errorAccessDenied = errors.New("access_denied")

func clientError(status int) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: status,
		Body:       http.StatusText(status),
	}, nil
}

func getAccessToken(apiRequest public.Request) (string, string, error) {
	if apiRequest.RefreshToken != "" {
		log.Println("Using Refresh Token to Authenticate Ring API")
		oauthResponse, err := httputil.AuthRequestWithRefreshToken("https://oauth.ring.com/oauth/token", httputil.OAuthRequestWithRefreshToken{"ring_official_ios", "refresh_token", apiRequest.RefreshToken})
		if err != nil {
			return "", apiRequest.RefreshToken, err
		}
		if oauthResponse.Error == "access_denied" {
			return oauthResponse.AccessToken, oauthResponse.RefreshToken, errorAccessDenied
		}
		return oauthResponse.AccessToken, oauthResponse.RefreshToken, nil
	} else {
		log.Println("Using User Name & Password to Authenticate Ring API")
		oauthResponse, _ := httputil.AuthRequest("https://oauth.ring.com/oauth/token", httputil.OAuthRequest{"ring_official_ios", "password", apiRequest.Password, "client", apiRequest.User}, "")
		return oauthResponse.AccessToken, "", nil
	}
}

func getLocation(apiRequest public.Request, accessToken string) (httputil.UserLocation, error) {
	location, err := httputil.LocationRequest("https://api.ring.com/devices/v1/locations", accessToken)
	if err != nil {
		return httputil.UserLocation{}, err
	}

	return location, nil
}

func getZID(apiRequest public.Request, accessToken, locationID string) (string, error) {
	// log.Println("Reading the ZID")
	zID := apiRequest.ZID
	if len(zID) == 0 {
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

func getDevices(locationID string, accessToken string) (*httputil.RingDeviceInfo, error) {
	connection, err := httputil.ConnectionRequest("https://app.ring.com/api/v1/rs/connections", locationID, accessToken)
	if err != nil {
		return nil, err
	}
	return wsutil.ActiveDevices(connection)
}

func makeTimestamp() int64 {
	return time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
}

func getStatus(apiRequest public.Request) (events.APIGatewayProxyResponse, error) {
	log.Printf("LocationID %v", apiRequest.LocationID)

	var ringEvents []public.RingDeviceEvent
	history, err := httputil.HistoryRequest("https://app.ring.com/api/v1/rs/history", apiRequest.AccessToken, apiRequest.LocationID, strconv.Itoa(apiRequest.HistoryLimit))
	if err != nil {
		log.Println("Error while trying to get Ring devices History.")
		return sendResponse(public.ProcessError{http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)})
	}

	// Adding Refresh time Event
	ringEvents = append(ringEvents, public.RingDeviceEvent{"Ring Alarm", makeTimestamp(), "Refresh"})

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
	ringDeviceInfo, err := getDevices(apiRequest.LocationID, apiRequest.AccessToken)
	if err != nil {
		log.Println("Error while trying to get Ring Devices.")
		return sendResponse(public.ProcessError{http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)})
	}

	for i := range ringDeviceInfo.Body {
		// log.Printf("RDName: %s, Type: %s, Fault: %v, Mode: %s\n", ringDeviceInfo.Body[i].General.V2.Name, ringDeviceInfo.Body[i].General.V2.DeviceType, ringDeviceInfo.Body[i].Device.V1.Faulted, ringDeviceInfo.Body[i].Device.V1.Mode)
		deviceStatus = append(deviceStatus, public.RingDeviceStatus{ringDeviceInfo.Body[i].General.V2.ZID, ringDeviceInfo.Body[i].General.V2.Name, ringDeviceInfo.Body[i].General.V2.DeviceType, ringDeviceInfo.Body[i].Device.V1.Faulted, ringDeviceInfo.Body[i].Device.V1.Mode})
	}

	// for i := range deviceStatus {
	// 	log.Printf("DName: %s, Type: %s, Fault: %v, Mode: %s\n", deviceStatus[i].Name, deviceStatus[i].Type, deviceStatus[i].Faulted, deviceStatus[i].Mode)
	// }

	return sendResponse(public.DeviceResponse{deviceStatus, ringEvents})
}

func setStatus(apiRequest public.Request, status string) (events.APIGatewayProxyResponse, error) {
	zID, err := getZID(apiRequest, apiRequest.AccessToken, apiRequest.LocationID)
	if err != nil {
		return sendResponse(public.ProcessError{http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)})
	}

	connection, err := httputil.ConnectionRequest("https://app.ring.com/api/v1/rs/connections", apiRequest.LocationID, apiRequest.AccessToken)
	if err != nil {
		return sendResponse(public.ProcessError{http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)})
	}

	_, err = wsutil.Status(zID, status, connection)
	if err != nil {
		return sendResponse(public.ProcessError{http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)})
	}

	return sendResponse(public.ModeChangeResponse{"Success"})
}

func getMetaData(apiRequest public.Request) (events.APIGatewayProxyResponse, error) {
	location, err := getLocation(apiRequest, apiRequest.AccessToken)
	if err != nil {
		log.Println("Error while trying to get Ring Location Id.")
		return sendResponse(public.ProcessError{http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)})
	}

	zID, err := getZID(apiRequest, apiRequest.AccessToken, location.ID)
	if err != nil {
		return sendResponse(public.ProcessError{http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)})
	}
	return sendResponse(public.RingMetaDataResponse{public.Location{location.ID, location.Name,
		public.Address{location.Address.Line1, location.Address.City, location.Address.State, location.Address.ZipCode}}, zID})
}

func getRawDevices(apiRequest public.Request) (events.APIGatewayProxyResponse, error) {
	location, err := getLocation(apiRequest, apiRequest.AccessToken)
	if err != nil {
		log.Println("Error while trying to get Ring Location Id.")
		return sendResponse(public.ProcessError{http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)})
	}
	log.Printf("Location ID %v", location.ID)

	devices, err := getDevices(location.ID, apiRequest.AccessToken)
	if err != nil {
		return sendResponse(public.ProcessError{http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError)})
	}
	log.Printf("Raw Device \n%v", devices)

	return sendResponse(public.RingDevices{public.Location{location.ID, location.Name,
		public.Address{location.Address.Line1, location.Address.City, location.Address.State, location.Address.ZipCode}}, devices})
}

func sendResponse(data interface{}) (events.APIGatewayProxyResponse, error) {
	result, _ := json.Marshal(data)

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(result),
	}, nil
}

// Handler is your Lambda function handler
// It uses Amazon API Gateway request/responses provided by the aws-lambda-go/events package,
// However you could use other event sources (S3, Kinesis etc), or JSON-decoded primitive types such as 'string'.
func Handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Println("Ring Alarm - Version 3.4.0")
	var apiRequest public.Request
	err := json.Unmarshal([]byte(request.Body), &apiRequest)
	if err != nil {
		return clientError(http.StatusUnprocessableEntity)
	}

	pathParams := request.PathParameters
	action := pathParams["ring-action"]
	log.Printf("Requested Action - %v\n", action)
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
	case "devices":
		return getRawDevices(apiRequest)
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
