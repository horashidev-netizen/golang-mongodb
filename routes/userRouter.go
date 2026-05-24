package routes

import (
	"golang-mongodb/controllers"
	"golang-mongodb/middleware"

	"github.com/gin-gonic/gin"
)

func UserRouter(incomingRoutes *gin.Engine) {
	incomingRoutes.Use(middleware.Authenticate())
	incomingRoutes.GET("/users/:user_id", controllers.GetUser())
	incomingRoutes.GET("/users", controllers.GetUsers())
}
