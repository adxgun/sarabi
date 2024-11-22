package caddy

type Config struct {
	Apps  Apps        `json:"apps"`
	Admin AdminConfig `json:"admin,omitempty"`
}

type AdminConfig struct {
	Listen string `json:"listen"`
}

type Apps struct {
	HTTP   HTTP   `json:"http"`
	Layer4 Layer4 `json:"layer4,omitempty"`
	TLS    TLS    `json:"tls,omitempty"`
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

type Layer4 struct {
	Servers map[string]*Layer4Server `json:"servers"`
}

type Layer4Route struct {
	Handle []Layer4Handle `json:"handle"`
	Match  []Layer4Match  `json:"match"`
}

type Layer4Handle struct {
	Handler   string           `json:"handler"`
	Upstreams []Layer4Upstream `json:"upstreams,omitempty"`
}

type Layer4Upstream struct {
	Dial []string `json:"dial"`
}

type Layer4Match struct {
	RemoteIP Layer4RemoteIP `json:"remote_ip"`
}

type Layer4RemoteIP struct {
	Ranges []string `json:"ranges"`
}

type Layer4Server struct {
	Listen []string      `json:"listen"`
	Routes []Layer4Route `json:"routes"`
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
