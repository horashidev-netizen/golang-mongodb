package routes

import (
	"golang-mongodb/controllers"
	"golang-mongodb/middleware"

	"github.com/gin-gonic/gin"
)

func MovieRoutes(router *gin.Engine) {
	router.GET("/movies/search/:movieName", controllers.SearchMovieByQuery())
	router.GET("/movies/:movie_id", controllers.GetMovie())
	router.Use(middleware.Authenticate())
	router.POST("/movies/createmovie", controllers.CreateMovie())
	router.POST("/movies/updatemovie/:movie_id", controllers.UpdateMovie())
	router.GET("/movies", controllers.GetMovies())
	router.GET("/movies/filter/:genreID", controllers.SearchMovieByGenre())
	router.DELETE("/movies/:movie_id", controllers.DeleteMovie())
}
