package main

import (
	"log"

	cmd "hris-core-service/cmd"
)

//go:generate go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g main.go -o docs --parseInternal

// @title Mapatu Backend API
// @version 1.0
// @description Mapatu HRIS operational API and global identity authentication foundation.
// @BasePath /

// @contact.name Juan Descka
// @contact.email juandescka@gmail.com

// @schemes http

// @tag.name Operations
// @tag.description Public health and readiness probes

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Opaque bearer session token. Enter: Bearer <access_token>. This authenticates global identity, without tenant permissions.

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
