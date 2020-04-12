package wsutil

import (
	"bytes"
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"text/template"
	"time"

	"github.com/asishrs/smartthings-ringalarmv2/httputil"
	"github.com/gorilla/websocket"
)

func Status(zid string, mode string, connection httputil.RingWSConnection) (string, error) {
	wssInput := "42[\n" +
		"    \"message\",\n" +
		"    {\n" +
		"        \"msg\": \"DeviceInfoSet\",\n" +
		"        \"datatype\": \"DeviceInfoSetType\",\n" +
		"        \"body\": [\n" +
		"            {\n" +
		"                \"zid\": \"" + zid + "\",\n" +
		"                \"command\": {\n" +
		"                    \"v1\": [\n" +
		"                        {\n" +
		"                            \"commandType\": \"security-panel.switch-mode\",\n" +
		"                            \"data\": {\n" +
		"                                \"mode\": \"" + mode + "\"\n" +
		"                            }\n" +
		"                        }\n" +
		"                    ]\n" +
		"                }\n" +
		"            }\n" +
		"        ],\n" +
		"        \"seq\": 2\n" +
		"    }\n" +
		"]"

	// log.Println("WS Connection " + wssInput)
	wssCall(connection, wssInput, "DataUpdate", 1)

	return "SUCCESS", nil
}

func wsConnection(connection httputil.RingWSConnection) (string, error) {
	wsConnectionTemplate := template.New("wscon")
	wsConnectionTemplate, err := wsConnectionTemplate.Parse("wss://{{.Server}}/socket.io/?authcode={{.AuthCode}}&ack=false&EIO=3&transport=websocket")
	if err != nil {
		log.Println("Parse: ", err)
		return "", err
	}
	var wsConnection bytes.Buffer
	wsConnectionTemplate.Execute(&wsConnection, connection)
	return wsConnection.String(), nil
}

func wssCall(connection httputil.RingWSConnection, wssInput string, messageType string, waitTime int) (string, error) {
	var wssResponse string

	wssUrl, err := wsConnection(connection)
	if err != nil {
		log.Println("Parse: ", err)
		return "", err
	}

	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	c, _, err := websocket.DefaultDialer.Dial(wssUrl, nil)
	if err != nil {
		log.Println("dial:", err)
		return "", err
	}
	defer c.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}

			s := string(message)
			//log.Printf("recv: %s\n", s)
			if strings.Contains(s, messageType) {
				wssResponse = s
			}
		}
	}()

	writeErr := c.WriteMessage(websocket.TextMessage, []byte(wssInput))
	if writeErr != nil {
		log.Println("write:", writeErr)
		return "", writeErr
	}

	time.Sleep(time.Duration(waitTime) * time.Second)
	log.Printf("Timeout after %d seconds", waitTime)

	// Cleanly close the connection by sending a close message and then
	// waiting (with timeout) for the server to close the connection.
	stopErr := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if stopErr != nil {
		log.Println("write close:", stopErr)
		//ignore the error
	}

	return wssResponse, nil
}

// ActiveDevices - Find all active devices in the Ring Alarm account.
func ActiveDevices(connection httputil.RingWSConnection) (*httputil.RingDeviceInfo, error) {
	wssInput := "42[\"message\",{\"msg\":\"DeviceInfoDocGetList\",\"seq\":1}]"
	wssResponse, err := wssCall(connection, wssInput, "DeviceInfoDocGetList", 3)
	if err != nil {
		log.Println("Error: ", err)
		return nil, err
	}

	if len(wssResponse) == 0 {
		log.Println("No Response: ", err)
		return nil, err
	}

	var ringDeviceInfo httputil.RingDeviceInfo
	runes := []rune(wssResponse)
	responseBody := string(runes[13 : len(wssResponse)-1])
	//log.Printf("Response: %s\n\nJSON: %s", wssResponse, responseBody)
	responseBody = responseBody[:strings.LastIndex(responseBody, "}")+1]
	// log.Printf("Response: %s", responseBody)
	err = json.Unmarshal([]byte(responseBody), &ringDeviceInfo)
	if err != nil {
		log.Println("Unable to Parse Status Response Data: ", err)
		return nil, err
	}

	return &ringDeviceInfo, nil
}
