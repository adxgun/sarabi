package config

import (
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
	"io"
	"os"
)

const (
	Path = ".sarabi.yml"
)

type (
	Config struct {
		ApplicationID uuid.UUID `yaml:"applicationID"`
		Frontend      string
		Backend       string
	}
)

func Parse() (Config, error) {
	c := Config{}
	fi, err := os.Open(Path)
	if err != nil {
		return c, err
	}

	value, err := io.ReadAll(fi)
	if err != nil {
		return c, err
	}

	if err = yaml.Unmarshal(value, &c); err != nil {
		return c, err
	}

	return c, nil
}
