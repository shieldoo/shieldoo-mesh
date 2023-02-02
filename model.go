package main

import "time"

type NebulaClientYamlConfig struct {
	AccessId                  int    `yaml:"accessid"`
	PublicIP                  string `yaml:"publicip"`
	Uri                       string `yaml:"uri"`
	Secret                    string `yaml:"secret"`
	Debug                     bool   `yaml:"debug"`
	SendInterval              int    `yaml:"sendinterval"`
	LocalUDPPort              int    `yaml:"localudpport"`
	RunAsDeskServiceRPC       bool   `yaml:"-"`
	RestrictedNetwork         bool   `yaml:"-"`
	RPCClientID               string `yaml:"-"`
	WindowsFW                 bool   `yaml:"-"`                         //windows firewall
	AutoUpdate                bool   `yaml:"-"`                         // autoupdate enabled
	AutoUpdateIntervalMinutes int64  `yaml:"autoupdateintervalminutes"` // autoupdate interval
	AutoUpdateChannel         string `yaml:"autoupdatechannel"`         // autoupdate channel
}

type NebulaLocalYamlConfig struct {
	ConfigHash string                    `json:"config_hash"`
	ConfigData *ManagementResponseConfig `json:"config_data"`
	Loaded     bool                      `json:"-"`
}

type OAuthLoginRequest struct {
	AccessID      int    `json:"access_id"`
	Timestamp     int64  `json:"timestamp"`
	Key           string `json:"key"`
	ClientID      string `json:"clientid"`
	ClientOS      string `json:"clientos"`
	ClientInfo    string `json:"clientinfo"`
	ClientVersion string `json:"clientversion"`
}

type OAuthLoginResponse struct {
	JWTToken string    `json:"jwt"`
	ValidTo  time.Time `json:"valid_to"`
}

type ManagementRequest struct {
	AccessID      int       `json:"access_id"`
	ClientID      string    `json:"clientid"`
	ConfigHash    string    `json:"confighash"`
	DnsHash       string    `json:"dnshash"`
	Timestamp     time.Time `json:"timestamp"`
	LogData       string    `json:"log_data"`
	IsConnected   bool      `json:"is_connected"`
	OverWebSocket bool      `json:"over_websocket"`
}

type ManagementResponseConfigData struct {
	Data      string `json:"config"`
	Hash      string `json:"hash"`
	IPAddress string `json:"ipaddress"`
}

type ManagementResponseConfig struct {
	AccessID                  int                          `json:"accessid"`
	UPN                       string                       `json:"upn"`
	Name                      string                       `json:"name"`
	ConfigData                ManagementResponseConfigData `json:"config"`
	NebulaPunchBack           bool                         `json:"nebulapunchback"`
	NebulaRestrictiveNetwork  bool                         `json:"nebularestrictivenetwork"`
	Autoupdate                bool                         `json:"autoupdate"`
	WebSocketUrl              string                       `json:"websocketurl"`
	WebSocketIPs              []string                     `json:"websocketips"`
	WebSocketUsernamePassword string                       `json:"websocketusernamepassword"`
	ApplianceListeners        []ManagementResponseListener `json:"listeners"`
	NebulaCIDR                string                       `json:"nebulacidr"`
}

type ManagementResponseListener struct {
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`
	ForwardPort int    `json:"forwardport"`
	ForwardHost string `json:"forwardhost"`
}

type ManagementResponse struct {
	Status     string                    `json:"status"`
	ConfigData *ManagementResponseConfig `json:"config_data"`
	Dns        *ManagementResponseDNS    `json:"dns"`
}

type ManagementResponseDNS struct {
	DnsRecords []string `json:"dnsrecords"`
	DnsHash    string   `json:"dnshash"`
}
