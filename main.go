package main

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/waiwen1001/bike/controller"
	"github.com/waiwen1001/bike/models"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	store, err := models.NewPostgresStore()
	if err != nil {
		log.Fatalf("Error loading postgresql config: %v", err)
	}

	if err := store.Init(); err != nil {
		log.Fatalf("Error loading init db: %v", err)
	}

	server := controller.NewAPIServer(":3000", store)
	server.Run()
}
