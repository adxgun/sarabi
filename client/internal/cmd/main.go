package main

import (
	"log"
	"sarabi/client/pkg/cmd"
)

func main() {
	sarabiCmd, err := cmd.New()
	if err != nil {
		log.Fatal(err)
	}

	if err := sarabiCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
