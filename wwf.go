package main

import (
	"warwolf/client"
	"warwolf/config"
	"warwolf/server"
)

func main() {
	switch config.LoadString("As") {
	case "Client":
		client.New().Listen(client.Config{})

	case "Server":
		fallthrough
	default:
		server.New().Listen(server.Config{})
	}
}
