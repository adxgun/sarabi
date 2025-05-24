package config

import "os"

type Config struct {
	// EncryptionKey is used to encrypt all application variables and other sensitive data before storing in the DB
	EncryptionKey string

	// AccessKey is the master access key to the server. Must be kept safe and secure!
	AccessKey string

	DatabasePath string
	Domain       string
	Port         string

	RunLogCollector bool
}

func New() Config {
	port := os.Getenv("SARABI_PORT")
	if port == "" {
		port = "3646"
	}

	return Config{
		AccessKey:     os.Getenv("ACCESS_KEY"),
		EncryptionKey: os.Getenv("ENCRYPTION_KEY"),
		Domain:        os.Getenv("SARABI_DOMAIN"),
		Port:          port,
		DatabasePath:  "/var/sarabi/data/database.db",
	}
}
