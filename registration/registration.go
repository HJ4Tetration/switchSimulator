package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"io/ioutil"
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
	Cmd     string
	Message []byte
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

func (s *switchWebHandler) sender(conn *websocket.Conn, toSender chan channelMessage, senderToValidator chan string, allToMainLoop chan string) {
	for {
		m := <-toSender
		conn.WriteMessage(websocket.BinaryMessage, m.Message)

		glog.Infof(s.switchName + ": " + m.Cmd + " message sent\n")
		switch m.Cmd {
		case "switch/check_in":
			senderToValidator <- m.Cmd
		case "switch/config_msg":
			senderToValidator <- m.Cmd
		case "switch/add_mapping":
			senderToValidator <- m.Cmd
		default: //todo: add other response to server's command
		}
	}
}

func (s *switchWebHandler) receiver(conn *websocket.Conn, receiverToValidator chan string, allToMainLoop chan string) {
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if messageType == websocket.CloseMessage {
				glog.Infof(s.switchName + ": websocket.Close() message received, close webocket gracefully\n")
				allToMainLoop <- s.switchName
				return
			} else {
				glog.Errorf(s.switchName+": Can't read websocket message: %v\n", err)
				//todo: send websocket.Close() message to gateway before conn.Close()
				allToMainLoop <- s.switchName
				return
			}
		}
		var serverMessage ServerMessage //for the cmd value
		err = json.Unmarshal(message, &serverMessage)
		if err != nil {
			glog.Errorf(s.switchName+": Can't unmarshal websocket message: %v\n", err)
			allToMainLoop <- s.switchName //todo: send websocket.Close() message to gateway before conn.CLose()
			return
		}
		glog.Infof(s.switchName + ": Server's " + serverMessage.Cmd + " message received\n")
		switch serverMessage.Cmd {
		case "switch/check_in":
			receiverToValidator <- serverMessage.Cmd
		case "switch/config_msg":
			receiverToValidator <- serverMessage.Cmd
			/*var serverConfigMessage ServerConfigMessage
			err = json.Unmarshal(message, &serverConfigMessage)
			if err != nil {
				glog.Errorf(s.switchName+": Can't unmarshal switch/config_msg message: %v\n", err)
				allToMainLoop <- s.switchName
				return
			}*/
			fileName := s.switchName + "ConfigMessage"
			err = ioutil.WriteFile(fileName, message, 0644)
			if err != nil {
				glog.Errorf(s.switchName+": Error writing configMessage to file: %v\n", err)
				allToMainLoop <- s.switchName
				return
			}
		case "switch/add_mapping":
			receiverToValidator <- serverMessage.Cmd
		default: //todo: add other server's command --> generate response -->forward to sender
		}
	}
}

func (s *switchWebHandler) validator(senderToValidator chan string, receiverToValidator chan string, allToMainLoop chan string) {
	for {
		ms := <-senderToValidator
		timeOut := make(chan bool)
		go func() {
			time.Sleep(30 * time.Second)
			timeOut <- true
		}()
		for f := true; f == true; {
			select {
			case mr := <-receiverToValidator:
				if mr == ms {
					glog.Infof(s.switchName + ": request " + ms + " and response " + mr + " matched\n")
					f = false
					break
				} else {
					glog.Infof(s.switchName + ": Validation error! request " + ms + " and response " + mr + " unmatched\n")
					allToMainLoop <- s.switchName
					return
				}
			case <-timeOut:
				glog.Infof(s.switchName + ": Timeout waiting for " + ms + " response\n")
				allToMainLoop <- s.switchName
				return
			default:
			}
		}
	}
}

func (s *switchWebHandler) marshalMessage(cmd string, message interface{}) ([]byte, bool) {
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		glog.Errorf(s.switchName+": Can't marshal "+cmd+" message: %v\n", err)
		return nil, false
	} else {
		return jsonMessage, true
	}
}

func (s *switchWebHandler) WebSocketRequest(allToMainLoop chan string) bool { //return false if websocket creation fails
	conn, _, err := s.websocketDialer.Dial(s.gatewayWssURL.String(), nil)
	if err != nil {
		glog.Errorf(s.switchName+": Can't make websocket: %v\n", err)
		return false
	}
	glog.Infof(s.switchName + ": Websocket established\n")
	//conn.SetReadDeadline(time.Now().Add(time.Minute))

	toSender := make(chan channelMessage, 10)
	senderToValidator := make(chan string, 10)
	receiverToValidator := make(chan string, 10)

	go s.sender(conn, toSender, senderToValidator, allToMainLoop)
	go s.receiver(conn, receiverToValidator, allToMainLoop)
	go s.validator(senderToValidator, receiverToValidator, allToMainLoop)

	cm := channelMessage{"switch/check_in", s.getCheckInMessage()}
	glog.Infof(s.switchName + ": forwarding switch/check_in message to sender\n")
	toSender <- cm
	cm = channelMessage{"switch/config_msg", s.getConfigMessage()}
	glog.Infof(s.switchName + ": forwarding switch/config_msg message to sender\n")
	toSender <- cm
	cm = channelMessage{"switch/add_mapping", s.getAddMappingMessageVRF()}
	glog.Infof(s.switchName + ": forwarding switch/add_mapping message to sender\n")
	toSender <- cm
	cm = channelMessage{"switch/add_mapping", s.getAddMappingMessagePort()}
	glog.Infof(s.switchName + ": forwarding switch/add_mapping message to sender\n")
	toSender <- cm
	cm = channelMessage{"switch/add_mapping", s.getAddMappingMessagePortToVRF()}
	glog.Infof(s.switchName + ": forwarding switch/add_mapping message to sender\n")
	toSender <- cm
	/*checkInMessage := s.getCheckInMessage()
	cmd := "switch/check_in"
	jsonCheckInMessage, flag := s.marshalMessage(cmd, checkInMessage)
	if !flag {
		glog.Infof(s.switchName + ": closing websocket\n")
		defer conn.Close()
		allToMainLoop <- s.switchName
		return false
	}
	message := channelMessage{cmd, jsonCheckInMessage}
	glog.Infof(s.switchName + ": forwarding " + cmd + " message to sender\n")
	toSender <- message

	configMessage := s.getConfigMessage()
	cmd = "switch/config_msg"
	jsonConfigMessage, flag := s.marshalMessage(cmd, configMessage)
	if !flag {
		glog.Infof(s.switchName + ": closing websocket\n")
		defer conn.Close()
		allToMainLoop <- s.switchName
		return false
	}
	message = channelMessage{cmd, jsonConfigMessage}
	glog.Infof(s.switchName + ": forwarding " + cmd + " message to sender\n")
	toSender <- message*/

	return true
}

func (s *switchWebHandler) httpsRequest() bool { //return false if registration fails
	switchRegistration := SwitchRegistration{Serial: s.switchName, Crt: ""} //Solenoid replaces its cert with switch cert
	jsonSwitchRegistration, err := json.Marshal(switchRegistration)
	if err != nil {
		glog.Errorf(s.switchName+": Can't marshal https registration request: %v\n", err)
		return false
	}
	_, err = s.httpClient.Post(s.gatewayRegisterURL.String(), "application/json", bytes.NewReader(jsonSwitchRegistration))
	if err != nil {
		glog.Errorf(s.switchName+": Error getting https reponse: %v\n", err)
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
	c := make(chan string, numberOfSwitches)
	glog.Infof("Start switch registration for %d switches\n", numberOfSwitches)

	for i := 0; i < numberOfSwitches; i++ {
		s[i] = NewSwitchWebHandler(&gateway, switchTitle+string(i))
		glog.Infof(s[i].switchName + ": Sending https request\n")
		if !s[i].httpsRequest() {
			glog.Infof(s[i].switchName + ": https request failed\n")
		} else {
			glog.Infof(s[i].switchName + ": https request succeeded\n")
			glog.Infof(s[i].switchName + ": Sending websocket request\n")
			if !s[i].WebSocketRequest(c) {
				glog.Infof(s[i].switchName + ": Websocket request failed\n")
			}
		}
	}
	glog.Infof("Registration procedure all done\n")
	for n := numberOfSwitches; n > 0; {
		switchName := <-c
		n--
		glog.Infof(switchName + ": websocket closed\n")
	}
	glog.Infof("All websockets closed, quit main loop\n")
}
