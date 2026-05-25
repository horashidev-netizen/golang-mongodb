package routes

import (
	"golang-mongodb/controllers"
	"golang-mongodb/middleware"

	"github.com/gin-gonic/gin"
)

func AuthRoutes(router *gin.Engine) {
	router.POST("/users/signup", controllers.Signup())
	router.POST("/users/login", controllers.Login())
	router.Use(middleware.Authenticate())
	router.POST("/users/logout", controllers.Logout())
}
