package routes

import (
	"golang-mongodb/controllers"
	"golang-mongodb/middleware"

	"github.com/gin-gonic/gin"
)

func GenreRouter(incomingRoutes *gin.Engine) {
	incomingRoutes.GET("/genres/search", controllers.SearchGenresByName())
	incomingRoutes.Use(middleware.Authenticate())
	incomingRoutes.POST("/genres/creategenre", controllers.CreateGenre())
	incomingRoutes.GET("/genres/:genre_id", controllers.GetGenre())
	incomingRoutes.GET("/genres", controllers.GetGenres())
	incomingRoutes.PUT("/genres/:genre_id", controllers.EditGenre())
	incomingRoutes.DELETE("/genres/:genre_id", controllers.DeleteGenre())
}
