package config

import "os"

type Config struct {
	// EncryptionKey is used to encrypt all application variables and other sensitive data before storing in the DB
	EncryptionKey string

	// AccessKey is the master access key to the server. Must be kept safe and secure!
	AccessKey string

	ServerSSLCertFile, ServerSSLKeyFile string

	DatabasePath string
}

func New() Config {
	return Config{
		AccessKey:         os.Getenv("ACCESS_KEY"),
		EncryptionKey:     os.Getenv("ENCRYPTION_KEY"),
		ServerSSLCertFile: os.Getenv("SERVER_SSL_KEY_FILE"),
		ServerSSLKeyFile:  os.Getenv("SERVER_SSL_CERT_FILE"),
		DatabasePath:      "/var/sarabi/data/database.db",
	}
}

func (c Config) HasTLSConfig() bool {
	return c.ServerSSLCertFile != "" && c.ServerSSLKeyFile != ""
}
