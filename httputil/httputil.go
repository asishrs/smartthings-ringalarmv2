package httputil

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
)

type OAuthRequest struct {
	ClientID  string `json:"client_id"`
	GrantType string `json:"grant_type"`
	Password  string `json:"password"`
	Scope     string `json:"scope"`
	Username  string `json:"username"`
}

type OAuthRequestWithRefreshToken struct {
	ClientID     string `json:"client_id"`
	GrantType    string `json:"grant_type"`
	RefreshToken string `json:"refresh_token"`
}

type OAuthResponse struct {
	AccessToken       string `json:"access_token"`
	ExpiresIn         string `json:"expires_in"`
	RefreshToken      string `json:"refresh_token"`
	Scope             string `json:"scope"`
	TokenType         string `json:"token_type"`
	Error             string `json:"error"`
	ErrorDescription  string `json:"error_description"`
	NextTimeInSeconds int32  `json:"next_time_in_secs"`
	Phone             string `json:"phone"`
}

type ExchangeRequest struct {
	AccessToken string `json:"access_token"`
}

type ExchangeResponse struct {
	AccessToken string `json:"access_token"`
}

type Address struct {
	Line1   string `json:"address1"`
	Line2   string `json:"address2"`
	City    string `json:"city"`
	State   string `json:"state"`
	ZipCode string `json:"zip_code"`
	Country string `json:"country"`
}

type UserLocation struct {
	LocationID string  `json:"location_id"`
	Name       string  `json:"name"`
	Address    Address `json:"address"`
}

type UserLocations struct {
	Location []UserLocation `json:"user_locations"`
}

type ResetChip struct {
	RequiresTrust bool `json:"reset-chip"`
}

type ResetNetwork struct {
	RequiresTrust bool `json:"reset-network"`
}

type CommandTypes struct {
	ResetChip    ResetChip    `json:"reset-chip"`
	ResetNetwork ResetNetwork `json:"reset-network"`
}

type V2 struct {
	AdapterType   string `json:"adapterType"`
	BatteryLevel  int    `json:"batteryLevel"`
	BatteryStatus string `json:"batteryStatus"`
	CatalogID     string `json:"catalogId"`
	// CategoryID          string       `json:"categoryId"`
	CommStatus          string       `json:"commStatus"`
	CommandTypes        CommandTypes `json:"commandTypes"`
	DeviceFoundTime     int64        `json:"deviceFoundTime"`
	DeviceType          string       `json:"deviceType"`
	LastCommTime        int64        `json:"lastCommTime"`
	LastUpdate          int64        `json:"lastUpdate"`
	ManagerID           string       `json:"managerId"`
	Name                string       `json:"name"`
	PollInterval        int          `json:"pollInterval"`
	RoomID              int          `json:"roomId"`
	SetupByPluginStatus string       `json:"setupByPluginStatus"`
	SetupByUserStatus   string       `json:"setupByUserStatus"`
	SubCategoryID       int          `json:"subCategoryId"`
	Tags                []string     `json:"tags"`
	TamperStatus        string       `json:"tamperStatus"`
	ZID                 string       `json:"zid"`
	AdapterZID          string       `json:"adapterZid"`
}

type FingerPrint struct {
	ManufacturerID string `json:"manufacturerId"`
	ProductID      string `json:"productId"`
	ProductType    string `json:"productType"`
}

type V1 struct {
	Channel int `json:"channel"`
	// FingerPrint FingerPrint `json:"fingerprint"`
	PanID   int    `json:"panId"`
	Faulted bool   `json:"faulted"`
	Mode    string `json:"mode"`
}

type Adapter struct {
	V1 V1 `json:"v1"`
}

type Device struct {
	V1 V1 `json:"v1"`
}

type General struct {
	V2 V2 `json:"v2"`
}

type ImpulseV1 struct {
	ImpulseType string `json:"impulseType"`
}

type Impulse struct {
	ImpulseTypes []ImpulseV1 `json:"v1"`
}

type Body struct {
	General General `json:"general"`
	Device  Device  `json:"device"`
	Adapter Adapter `json:"adapter"`
	Impulse Impulse `json:"impulse"`
}

type Context struct {
	EventID              string `json:"eventId"`
	EventOccurredTsMs    int64  `json:"eventOccurredTsMs"`
	AffectedEntityType   string `json:"affectedEntityType"`
	AffectedEntityID     string `json:"affectedEntityId"`
	AffectedEntityName   string `json:"affectedEntityName"`
	InitiatingEntityType string `json:"initiatingEntityType"`
	InitiatingEntityID   string `json:"initiatingEntityId"`
	InitiatingEntityName string `json:"initiatingEntityName"`
	InterfaceType        string `json:"interfaceType"`
	InterfaceID          string `json:"interfaceId"`
	InterfaceName        string `json:"interfaceName"`
	AffectedParentID     string `json:"affectedParentId"`
	AffectedParentName   string `json:"affectedParentName"`
	AccountID            string `json:"accountId"`
	ProgramID            string `json:"programId"`
	UserAgent            string `json:"userAgent"`
	IPAddress            string `json:"ipAddress"`
	AssetID              string `json:"assetId"`
	AssetKind            string `json:"assetKind"`
}

type History struct {
	Body     []Body  `json:"body"`
	Context  Context `json:"context"`
	DataType string  `json:"datatype"`
	Message  string  `json:"msg"`
}

type Event struct {
	DeviceName         string `json:"name"`
	DateInMilliSeconds int64  `json:"dateInMilliSeconds"`
	Type               string `json:"type"`
}

// RingWSConnection is a type for Ring Connection response API.
type RingWSConnection struct {
	Server   string `json:"server"`
	AuthCode string `json:"authCode"`
}

func AuthRequest(url string, oauthRequest OAuthRequest, code string) (OAuthResponse, error) {
	log.Printf("OAuthRequest Data: %v", oauthRequest)
	var headers map[string]string
	if code != "" {
		headers = make(map[string]string)
		headers["2fa-support"] = "true"
		headers["2fa-code"] = code
		headers["Content-Type"] = "application/json"
	}

	requestByte, _ := json.Marshal(oauthRequest)
	responseBody, err := post(url, headers, requestByte)
	if err != nil {
		return OAuthResponse{}, err
	}

	var oauthResponse OAuthResponse
	json.Unmarshal(responseBody, &oauthResponse)
	log.Println("Temp Token " + oauthResponse.AccessToken)
	return oauthResponse, nil
}

func AuthRequestWithRefreshToken(url string, oauthRequest OAuthRequestWithRefreshToken) (OAuthResponse, error) {
	log.Printf("OAuthRequestWithRefreshToken Data: %v", oauthRequest)
	requestByte, _ := json.Marshal(oauthRequest)
	responseBody, err := post(url, nil, requestByte)
	if err != nil {
		return OAuthResponse{}, err
	}

	var oauthResponse OAuthResponse
	json.Unmarshal(responseBody, &oauthResponse)
	log.Printf("Temp Token %v\n", oauthResponse)
	return oauthResponse, nil
}

func AccessTokenRequest(url string, exchangeRequest ExchangeRequest) ExchangeResponse {
	requestByte, _ := json.Marshal(exchangeRequest)
	headers := map[string]string{
		"content-type": "application/json",
	}
	responseBody, _ := post(url, headers, requestByte)
	var exchangeResponse ExchangeResponse
	json.Unmarshal(responseBody, &exchangeResponse)
	// log.Printf("Access Token %v", exchangeResponse.AccessToken)
	return exchangeResponse
}

func LocationRequest(url string, accessToken string) UserLocations {
	//log.Printf("LocationRequest Data: %v", accessToken)
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}

	responseBody := get(url, headers, nil)
	var userLocations UserLocations
	json.Unmarshal(responseBody, &userLocations)
	//log.Printf("User Locations: %v", userLocations)
	return userLocations
}

func HistoryRequest(url string, accessToken string, locationID string, limit string) []History {
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}

	params := map[string]string{
		"accountId": locationID,
		"offset":    "0",
		"limit":     limit,
		"maxLevel":  "50",
	}

	responseBody := get(url, headers, params)
	var history []History
	json.Unmarshal(responseBody, &history)
	return history
}

func ConnectionRequest(url string, locationId string, accessToken string) RingWSConnection {
	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
		"Content-Type":  "application/x-www-form-urlencoded",
	}

	responseBody, _ := post(url, headers, []byte("accountId="+locationId))
	var connection RingWSConnection
	json.Unmarshal(responseBody, &connection)
	// log.Println("Connection [" + connection.Server + ", " + connection.AuthCode + "]")
	return connection
}

func get(url string, headers map[string]string, params map[string]string) []byte {
	req, _ := http.NewRequest("GET", url, nil)
	for name, value := range headers {
		req.Header.Add(name, value)
	}

	query := req.URL.Query()
	for name, value := range params {
		query.Add(name, value)
	}
	req.URL.RawQuery = query.Encode()

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	responseBody, _ := ioutil.ReadAll(res.Body)
	return responseBody
}

func post(url string, headers map[string]string, requestBody []byte) ([]byte, error) {
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
	for name, value := range headers {
		req.Header.Add(name, value)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer res.Body.Close()
	responseBody, _ := ioutil.ReadAll(res.Body)
	// log.Printf("Url - %v, Header - %v", url, headers)
	return responseBody, nil
}
