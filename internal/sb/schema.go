package sb

type PortBinding struct {
	UUID        string  `ovsdb:"_uuid"`
	LogicalPort string  `ovsdb:"logical_port"`
	Type        string  `ovsdb:"type"`
	Datapath    string  `ovsdb:"datapath"`
	TunnelKey   int     `ovsdb:"tunnel_key"`
	Chassis     *string `ovsdb:"chassis"`
	Up          *bool   `ovsdb:"up"`

	Options map[string]string `ovsdb:"options"`
}
