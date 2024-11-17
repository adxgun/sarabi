package sarabi

import (
	"fmt"
	"net"
)

func StrContains(str string, values []string) bool {
	for _, next := range values {
		if str == next {
			return true
		}
	}
	return false
}

type (
	PortGenerator interface {
		Generate() (string, error)
	}
)

type portGenerator struct {
}

func newPortGenerator() PortGenerator {
	return &portGenerator{}
}

func (p portGenerator) Generate() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}

	defer listener.Close()
	addr := listener.Addr().(*net.TCPAddr)
	return fmt.Sprintf("%d", addr.Port), nil
}

var (
	DefaultPortGenerator = newPortGenerator()
)
