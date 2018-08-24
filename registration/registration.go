package registration

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"net/http"
	"net/url"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"
)

type switchWebHandler struct {
	switchName         string
	gatewayRegisterURL url.URL
	gatewayWssURL      url.URL
	httpClient         *http.Client
	websocketDialer    websocket.Dialer
}

type gateWay struct {
	IP string
}

func (gateway *gateWay) getGatewayRegisterIP() string {
	return gateway.IP
}

func (gateway *gateWay) getGatewayWebsocketIP() string {
	return gateway.IP
}

func NewSwitchWebHandler(gateway *gateWay, switchName string) *switchWebHandler {
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	websocketDialer := websocket.Dialer{
		HandshakeTimeout: 60 * time.Minute,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	return &switchWebHandler{
		switchName:         switchName,
		gatewayRegisterURL: url.URL{Scheme: "https", Host: gateway.getGatewayRegisterIP(), Path: "/switch_register"},
		gatewayWssURL:      url.URL{Scheme: "wss", Host: gateway.getGatewayWebsocketIP(), Path: "/switch_wss"},
		httpClient:         httpClient,
		websocketDialer:    websocketDialer,
	}
}

func (s *switchWebHandler) WebSocketRequest() bool { //return false if websocket creation fails
	conn, _, err := s.websocketDialer.Dial(s.gatewayWssURL.String(), nil)
	if err != nil {
		glog.Errorf(s.switchName+": Can't make websocket: %v\n", err)
		return false
	}
	glog.Infof(s.switchName + ": Websocket established\n")
	//conn.SetReadDeadline(time.Now().Add(time.Minute))
	//c := make(chan []byte, 10)

	/*go func(chan []byte) {
		for {
			messageType, message, err := conn.ReadMessage()
			if messageType == websocket.CloseMessage {
				c <- message
				break
			}
		}
	}(c)*/
	var switchCheckInMessage SwitchCheckInMessage //todo: add getCheckInMessage()
	jsonSwitchCheckInMessage, err := json.Marshal(switchCheckInMessage)
	if err != nil {
		glog.Errorf(s.switchName+": Can't marshal switchCheckInMessage: %v\n", err)
		defer conn.Close() //todo: send websocket.Close() message to gateway before conn.CLose()
		return false
	}
	conn.WriteMessage(websocket.BinaryMessage, jsonSwitchCheckInMessage)
	glog.Infof(s.switchName + ": switchCheckInMessage sent\n")

	conn.SetReadDeadline(time.Now().Add(time.Minute))
	messageType, message, err := conn.ReadMessage()
	if err != nil {
		defer conn.Close() //todo: send websocket.Close() message to gateway before conn.CLose()
		if messageType == websocket.CloseMessage {
			glog.Infof(s.switchName + ": websocket.Close() message received, close webocket gracefully\n")
			return false
		} else {
			glog.Errorf(s.switchName+": Can't read websocket message: %v\n", err)
			return false
		}
	}
	var jsonMessage ServerMessage
	err = json.Unmarshal(message, &jsonMessage)
	if err != nil {
		glog.Errorf(s.switchName+": Can't unmarshal websocket message: %v\n", err)
		defer conn.Close() //todo: send websocket.Close() message to gateway before conn.CLose()
		return false
	}
	if jsonMessage.Cmd != switchCheckInMessage.Cmd {
		glog.Infof(s.switchName + ": Can't get OK response for SwitchCheckInMessage: %v\n")
		defer conn.Close() //todo: send websocket.Close() message to gateway before conn.CLose()
		return false
	}
	glog.Infof(s.switchName + ": switchCheckInMessage OK response received\n")

	var switchConfigMessage SwitchConfigMessage //todo: add getConfigMessage()
	jsonSwitchConfigMessage, err := json.Marshal(switchConfigMessage)
	if err != nil {
		glog.Errorf(s.switchName+": Can't marshal switchConfigMessage: %v\n", err)
		defer conn.Close() //todo: send websocket.Close() message to gateway before conn.CLose()
		return false
	}
	conn.WriteMessage(websocket.BinaryMessage, jsonSwitchConfigMessage)
	glog.Infof(s.switchName + ": switchConfigMessage sent\n")

	return true
}

func (s *switchWebHandler) httpsRequest() bool { //return false if registration fails
	switchRegistration := SwitchRegistration{Serial: s.switchName, Crt: ""} //Solenoid replaces its cert with switch cert
	jsonSwitchRegistration, err := json.Marshal(switchRegistration)
	if err != nil {
		glog.Errorf("can't marshal https registration request for switch "+s.switchName+": %v\n", err)
		return false
	}
	_, err = s.httpClient.Post(s.gatewayRegisterURL.String(), "application/json", bytes.NewReader(jsonSwitchRegistration))
	if err != nil {
		glog.Errorf("error getting https response for switch "+s.switchName+": %v\n", err)
		return false
	}
	return true
}

func main() {
	flag.Parse()
	flag.Lookup("logtostderr").Value.Set("true")
	gateway := gateWay{"172.21.92.97"}
	numberOfSwitches := 1
	switchTitle := "harojianSwitchSimulator"
	s := make([]*switchWebHandler, numberOfSwitches)
	glog.Infof("Start switch registration for %d switches\n", numberOfSwitches)
	for i := 0; i < numberOfSwitches; i++ {
		s[i] = NewSwitchWebHandler(&gateway, switchTitle+string(i))
		glog.Info("Sending https request for switch " + s[i].switchName + "\n")
		if !s[i].httpsRequest() {
			glog.Info("https request for switch " + s[i].switchName + " failed\n")
		} else {
			glog.Info("https request for switch " + s[i].switchName + " succeeded\n")
		}
	}
	glog.Infof("Registration procedure all done\n")
	glog.Infof("Start sending UDP packets\n")
	for {

	}
}
