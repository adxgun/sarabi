package caddy

type Config struct {
	Apps  Apps        `json:"apps"`
	Admin AdminConfig `json:"admin,omitempty"`
}

type AdminConfig struct {
	Listen string `json:"listen"`
}

type Apps struct {
	HTTP HTTP `json:"http"`
}

type HTTP struct {
	Servers map[string]*Server `json:"servers"`
}

type Server struct {
	Listen []string `json:"listen"`
	Routes []Route  `json:"routes"`
}

type Route struct {
	Handle []Handle `json:"handle"`
	Match  []Match  `json:"match"`
}

type Handle struct {
	Handler   string     `json:"handler"`
	Upstreams []Upstream `json:"upstreams,omitempty"`
	Root      string     `json:"root,omitempty"`
}

type Upstream struct {
	Dial string `json:"dial"`
}

type Match struct {
	Host []string `json:"host"`
}
