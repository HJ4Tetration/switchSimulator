package registration

import(
	"flag"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	"net/http"
	"net/url"
	"crypto/tls"
	"time"
	"bytes"
	"encoding/json"
)

type switchWebHandler struct{
	switchName string
	gatewayRegisterURL url.URL
	gatewayWssURL url.URL
	httpClient *http.Client
	websocketDialer websocket.Dialer
}

type gateWay struct{
	IP string
}

func (gateway *gateWay)getGatewayRegisterIP() string{
	return gateway.IP
}

func (gateway *gateWay)getGatewayWebsocketIP() string{
	return gateway.IP
}

func NewSwitchWebHandler(gateway *gateWay,switchName string) *switchWebHandler{
	httpClient:=&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify:true,
			},
		},
	}
	websocketDialer:=websocket.Dialer{
		HandshakeTimeout: 60*time.Minute,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify:true,
		},
	}
	return &switchWebHandler{
		switchName: switchName,
		gatewayRegisterURL: url.URL{Scheme:"https",Host:gateway.getGatewayRegisterIP(),Path:"/switch_register"},
		gatewayWssURL: url.URL{Scheme:"wss",Host:gateway.getGatewayWebsocketIP(),Path:"/switch_wss"},
		httpClient:httpClient,
		websocketDialer: websocketDialer,
	}
}



func (s *switchWebHandler)WebSocketRequest(switchName string) bool{//return false if websocket creation fails
	con,_,err:=s.websocketDialer.Dial(s.gatewayWssURL.String(),nil)
	if err!=nil{
		glog.Errorf("Can't make websocket for switch "+s.switchName+": %v\n",err)
		return false
	}
	glog.Infof("Websocket established for switch "+s.switchName+"\n")
	switchCheckInMessage:=""//getCheckInMessage()
	switchConfigMessage:=""//getConfigMessage()
	return true
}


func (s *switchWebHandler)httpsRequest() bool{//return false if registration fails
	switchRegistration:=SwitchRegistration{Serial:s.switchName,Crt:""}//Solenoid replaces its cert with switch cert
	jsonSwitchRegistration,err:=json.Marshal(switchRegistration)
	if err!=nil{
		glog.Errorf("can't marshal https registration request for switch "+s.switchName+": %v\n",err)
		return false
	}
	_,err=s.httpClient.Post(s.gatewayRegisterURL.String(),"application/json",bytes.NewReader(jsonSwitchRegistration))
	if err!=nil{
		glog.Errorf("error getting https response for switch "+s.switchName+": %v\n",err)
		return false
	}
	return true
}


func main(){
	flag.Parse()
	flag.Lookup("logtostderr").Value.Set("true")
	gateway:=gateWay{"172.21.92.97"}
	numberOfSwitches := 1
	switchTitle:="harojianSwitchSimulator"
	s:=make([]*switchWebHandler,numberOfSwitches)
	glog.Infof("Start switch registration for %d switches\n", numberOfSwitches)
	for i:=0;i<numberOfSwitches;i++{
		s[i]=NewSwitchWebHandler(&gateway,switchTitle+string(i))
		glog.Info("Sending https request for switch "+s[i].switchName+"\n")
		if !s[i].httpsRequest(){
			glog.Info("https request for switch "+s[i].switchName+" failed\n")
		}else{
			glog.Info("https request for switch "+s[i].switchName+" succeeded\n")
		}
	}
	glog.Infof("Registration procedure all done\n")
	glog.Infof("Start sending UDP packets\n")
	for {

	}
}