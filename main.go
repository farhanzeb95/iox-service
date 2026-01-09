package main

import (
	"log"
	"os"

	"iox-service/database"
	router "iox-service/routers"
)

func main() {
	// App port
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "9000"
	}

	log.Println("🚀 Starting application")

	// Connect to MongoDB
	database.Connect("mongodb://localhost:27017")

	// Setup Gin router
	r := router.Setup()

	log.Printf("🟢 App is running on port %s\n", port)
	r.Run(":" + port) // start HTTP server
}
