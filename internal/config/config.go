package config

type EngineConfig struct {
	ServerUrl    *string
	Environments []Environment
}

type Environment struct {
	Name   *string
	Domain *string
}
