package main

import (
	"golang-mongodb/database"
	"golang-mongodb/routes"
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

	database.CreateTTLIndex(database.Client)

	//Log events
	router.Use(gin.Logger())

	//Register app routes
	routes.AuthRoutes(router)
	routes.UserRouter(router)
	routes.GenreRouter(router)
	routes.MovieRoutes(router)
	routes.ReviewRouter(router)
	routes.HistoryRouter(router)
	
	router.GET("/api", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"success": "Welcome to horashi api!"})
	})

	err = router.Run(":" + port)
	if err != nil {
		return
	}

}
