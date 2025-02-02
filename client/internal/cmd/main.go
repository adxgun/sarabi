package main

import (
	"os"
	"sarabi/client/pkg/cmd"
)

func main() {
	sarabiCmd, err := cmd.New()
	if err != nil {
		exit(err)
	}

	if err := sarabiCmd.Execute(); err != nil {
		exit(err)
	}
}

func exit(err error) {
	os.Exit(1)
}
