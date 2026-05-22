package routes

import (
	"golang-mongodb/controllers"

	"github.com/gin-gonic/gin"
)

func AuthRoutes(router *gin.Engine) {
	router.POST("users/signup", controllers.Signup())
}
