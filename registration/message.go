package registration

type SwitchRegistration struct {
	Serial string `json:"serial"`
	Crt    string `json:"crt"`
}

type SwitchMessage struct {
	Cmd      string `json:"cmd"`
	SwitchID string `json:"switchId"`
}

type ServerMessage struct {
	ResponseCode int    `json:"responseCode"`
	Cmd          string `json:"cmd"`
	// Data: ignore data field, use more specific structs if Data is needed
}

type SwitchCheckInMessage struct {
	Cmd      string `json:"cmd"`
	SwitchID string `json:"switchId"`
	Data     struct {
		AgentVersion string `json:"agentVersion"`
		Capability   string `json:"capability"`
		GatewayUUID  string `json:"gateway_uuid"`
		ImageName    string `json:"imageName"`
		IP           string `json:"ip"`
		ModTs        string `json:"modTs"`
		Role         string `json:"role"`
		State        string `json:"state"`
		Status       string `json:"status"`
		SwitchName   string `json:"switch_name"`
		SystemUpTime string `json:"systemUpTime"`
	} `json:"data"`
}

type CollectorBucket struct {
	Lo        int    `json:"lo"`
	Hi        int    `json:"hi"`
	Primary   string `json:"primary"`
	Secondary string `json:"secondary"`
}

type CollectorMessage struct {
	Decommissioned bool   `json:"decommissioned"`
	IP             string `json:"ip"`
	Name           string `json:"name"`
	UpdatedAt      int    `json:"updated_at"`
	CollectorID    int    `json:"collector_id"`
	Healthy        bool   `json:"healthy"`
	SpineUDPPort   int    `json:"spine_udp_port"`
	UDPPort        int    `json:"udp_port"`
}

type ServerConfigMessage struct {
	ResponseCode int    `json:"responseCode"`
	Cmd          string `json:"cmd"`
	Data         struct {
		Buckets         []CollectorBucket  `json:"buckets"`
		Active          []CollectorMessage `json:"active"`
		Deactivated     []CollectorMessage `json:"deactivated"`
		DataPathDisable bool               `json:"dataPathDisable"`
		CfgOpts         struct {
			ExportIntervalMs int `json:"exportIntervalMs"`
		} `json:"cfgOpts"`
		HwSensors []struct {
			Dn         string `json:"dn"`
			ExporterID int    `json:"exporter_id"`
			SrcPort    int    `json:"src_port"`
			State      string `json:"state"`
		} `json:"hwSensors"`
	} `json:"data"`
}
