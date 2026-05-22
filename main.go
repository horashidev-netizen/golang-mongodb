package main

import (
	"golang-mongodb/routes"

	"github.com/horashidev-netizen/golang-mongodb/database"

	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	port := os.Getenv("PORT")
	// Create a Gin router with default middleware (logger and recovery)
	if port == "" {
		port = "8000"
	}
	router := gin.Default()

	//run database
	database.StartDB()

	//Log events
	router.Use(gin.Logger())
	//Register app routes
	routes.AuthRoutes(router)
	// Start server on port 8080 (default) or 9000 from env
	// Server will listen on 0.0.0.0:8080 (localhost:8080 or 9000 on Windows)
	router.GET("/api", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"success": "Welcome to horashi api!"})
	})

	err = router.Run(":" + port)
	if err != nil {
		return
	}

}
