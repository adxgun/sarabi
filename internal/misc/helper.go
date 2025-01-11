package sarabi

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"math/big"
	"net"
	"sarabi/internal/types"
	"strconv"
	"strings"
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

func HexToUUID(value string) (uuid.UUID, error) {
	if len(value) != 32 {
		return uuid.Nil, errors.New("UUID part must be exactly 32 hexadecimal characters")
	}

	uuidBytes, err := hex.DecodeString(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to decode UUID hex string: %v", err)
	}

	return uuid.FromBytes(uuidBytes)
}

func ParseContainerIdentity(id string, name string) (*types.ContainerIdentity, error) {
	values := strings.Split(name, "-")
	if len(values) != 3 {
		return nil, errors.New("invalid container name: " + name)
	}

	if s := values[1]; s == "" {
		return nil, errors.New("environment is empty")
	}

	deploymentID, err := HexToUUID(values[0])
	if err != nil {
		return nil, err
	}

	instanceID, err := strconv.Atoi(values[2])
	if err != nil {
		return nil, fmt.Errorf("invalid instanceID: %v", err)
	}

	return &types.ContainerIdentity{
		ID:           id,
		DeploymentID: deploymentID,
		Environment:  values[1],
		InstanceID:   instanceID,
	}, nil
}
