package sarabi

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net"
)

// StrContains returns true if "str" is in "values"
// e.g "a" in "a,b,c" => true
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

const (
	charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

type (
	RandomIdGenerator interface {
		Generate(n int) (string, error)
	}
)

type randomIdGenerator struct {
}

func newRandomIdGenerator() RandomIdGenerator {
	return &randomIdGenerator{}
}

func (p randomIdGenerator) Generate(n int) (string, error) {
	result := make([]byte, n)
	charsetLength := byte(len(charset))

	for i := range result {
		randomByte, err := rand.Int(rand.Reader, big.NewInt(int64(charsetLength)))
		if err != nil {
			return "", err
		}
		result[i] = charset[randomByte.Int64()]
	}

	return string(result), nil
}

var (
	DefaultRandomIdGenerator = newRandomIdGenerator()
)
