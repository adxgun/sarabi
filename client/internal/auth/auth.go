package auth

import "github.com/zalando/go-keyring"

const (
	appName = "sarabi"
	keyName = "access-key"
)

func Save(key string) error {
	return keyring.Set(appName, keyName, key)
}

func Get() (string, error) {
	return keyring.Get(appName, keyName)
}
