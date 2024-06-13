package aregion

type Config struct {
	AuthServer   *AuthServer   `json:"auth_server,omitempty"`
	RegionServer *RegionServer `json:"region_server,omitempty"`
}

type AuthServer struct {
	Location *Location `json:"location,omitempty"`
}

type RegionServer struct {
	Location *Location `json:"location,omitempty"`
}

type Location struct {
	Address string `json:"address,omitempty"`
	Port    int    `json:"port,omitempty"`
	Path    string `json:"path,omitempty"`
	AK      string `json:"ak,omitempty"`
	SK      string `json:"sk,omitempty"`
}
