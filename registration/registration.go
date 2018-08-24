package registration

import(
	"flag"
	"sync"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	"net/http"
	"net/url"
)

type switchWebHandler struct{
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

func NewSwitchWebHandler(gateway *gateWay) *switchWebHandler{
	return &switchWebHandler{
		gatewayRegisterURL: url.URL{Scheme:"https",Host:gateway.getGatewayRegisterIP(),Path:"/switch_register"},
		gatewayWssURL: url.URL{Scheme:"wss",Host:gateway.getGatewayWebsocketIP(),Path:"/switch_wss"},	
	}
}





func WebSocketRequest(){

}


func httpsRequest(){

}


func main(){
	flag.Parse()
	flag.Lookup("logtostderr").Value.Set("true")
	gateway:=gateWay{"172.21.92.97"}
	var wg sync.WaitGroup
	numberOfSwitch := 1
	//wg.Add(numberOfSwitch)
	glog.Infof("Start switch registration for %d switches\n", numberOfSwitch)
	wg.Wait()
	glog.Infof("Registration procedure all done\n")
	glog.Infof("Start sending UDP packets\n")
	for {

	}
}