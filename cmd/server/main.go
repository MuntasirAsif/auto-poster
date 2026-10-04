package main

import (
	"log"

	"github.com/joho/godotenv"

	"auto-poster/internal/database"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using defaults")
	}

	db, err := database.Connect()
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	if err := database.ApplyMigrations(); err != nil {
		log.Fatal(err)
	}

	log.Println("Auto Poster server started")
}