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

type channelMessage struct {
	message interface{}
	cmd     string
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

func (s *switchWebHandler) sender(conn *websocket.Conn, toSender chan channelMessage, senderToValidator chan channelMessage, allToMainLoop chan bool) {
	for {
		m := <-toSender
		jsonMessage, err := json.Marshal(m.message)
		if err != nil {
			glog.Errorf(s.switchName+": Can't marshal "+m.cmd+" message: %v\n", err)
			//todo: send websocket.Close() message to gateway before conn.Close()
			allToMainLoop <- true
			break
		}
		conn.WriteMessage(websocket.BinaryMessage, jsonMessage)
		glog.Infof(s.switchName + ": " + m.cmd + " message sent\n")
		switch m.cmd {
		case "switch/check_in":
			senderToValidator <- m
		case "switch/config_msg":
			senderToValidator <- m
		case "switch/add_mapping":
			senderToValidator <- m
		default: //todo: add other response to server's command
		}
	}
}

func (s *switchWebHandler) receiver(conn *websocket.Conn, receiverToValidator chan channelMessage, allToMainLoop chan bool) {
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if messageType == websocket.CloseMessage {
				glog.Infof(s.switchName + ": websocket.Close() message received, close webocket gracefully\n")
				allToMainLoop <- true
				break
			} else {
				glog.Errorf(s.switchName+": Can't read websocket message: %v\n", err)
				//todo: send websocket.Close() message to gateway before conn.Close()
				allToMainLoop <- true
				break
			}
		}
		var serverMessage ServerMessage //for the cmd value
		err = json.Unmarshal(message, &serverMessage)
		if err != nil {
			glog.Errorf(s.switchName+": Can't unmarshal websocket message: %v\n", err)
			allToMainLoop <- true //todo: send websocket.Close() message to gateway before conn.CLose()
			break
		}
		switch serverMessage.Cmd {
		case "switch/check_in":
			channelMessage := channelMessage{serverMessage, serverMessage.Cmd}
			receiverToValidator <- channelMessage
		case "switch/config_msg":
			channelMessage := channelMessage{serverMessage, serverMessage.Cmd}
			receiverToValidator <- channelMessage
		case "switch/add_mapping":
			channelMessage := channelMessage{serverMessage, serverMessage.Cmd}
			receiverToValidator <- channelMessage
		default: //todo: add other server's command --> generate response -->forward to sender
		}
		glog.Infof(s.switchName + ": switchCheckInMessage OK response received\n")
	}
}

func (s *switchWebHandler) validator(senderToValidator chan channelMessage, receiverToValidator chan channelMessage, allToMainLoop chan bool) {
}

func (s *switchWebHandler) WebSocketRequest() bool { //return false if websocket creation fails
	conn, _, err := s.websocketDialer.Dial(s.gatewayWssURL.String(), nil)
	if err != nil {
		glog.Errorf(s.switchName+": Can't make websocket: %v\n", err)
		return false
	}
	glog.Infof(s.switchName + ": Websocket established\n")
	//conn.SetReadDeadline(time.Now().Add(time.Minute))

	toSender := make(chan channelMessage, 10)
	//toReceiver := make(chan channelMessage, 10)
	senderToValidator := make(chan channelMessage, 10)
	receiverToValidator := make(chan channelMessage, 10)
	allToMainLoop := make(chan bool, 10)

	go s.sender(conn, toSender, senderToValidator, allToMainLoop)
	go s.receiver(conn, receiverToValidator, allToMainLoop)
	go s.validator(senderToValidator, receiverToValidator, allToMainLoop)

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
