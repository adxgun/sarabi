package initcmd

import "fmt"

type Config struct {
	Servers []Server `yaml:"servers"`
}

type Mode string

const (
	Manager Mode = "manager"
	Worker  Mode = "worker"
)

func (m Mode) String() string {
	switch m {
	case Manager, Worker:
		return string(m)
	default:
		return string(Worker)
	}
}

type Server struct {
	IP          string `yaml:"ip"`
	SSHPort     int    `yaml:"ssh_port"`
	Mode        Mode   `yaml:"mode"`
	Auth        Auth   `yaml:"auth"`
	BlockDevice string `yaml:"block_device"`
}

func (s Server) SwarmListenAddr() string {
	return fmt.Sprintf("%s:%d", s.IP, 2377)
}

func (s Server) AdsAddr() string {
	return s.SwarmListenAddr()
}

func (s Server) port() string {
	if s.SSHPort == 0 {
		return "22"
	}
	return fmt.Sprintf("%d", s.SSHPort)
}

func (s Server) SSHConnectionString() string {
	return s.IP + ":" + s.port()
}

type Auth struct {
	Type     string `yaml:"type"`
	Path     string `yaml:"path"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}
