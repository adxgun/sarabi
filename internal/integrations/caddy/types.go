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
	TLS  TLS  `json:"tls,omitempty"`
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

type TLS struct {
	Automation Automation `json:"automation"`
}

type Automation struct {
	Policies []Policies `json:"policies"`
}

type Policies struct {
	Subjects []string `json:"subjects"`
	Issuer   Issuer   `json:"issuer"`
}

type Issuer struct {
	Module string `json:"module"`
}
