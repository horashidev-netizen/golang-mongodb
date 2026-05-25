package routes

import (
	"golang-mongodb/controllers"
	"golang-mongodb/middleware"

	"github.com/gin-gonic/gin"
)

func HistoryRouter(incomingRoutes *gin.Engine) {
	// Bắt buộc đăng nhập
	incomingRoutes.Use(middleware.Authenticate())
	
	// API gọi khi user click xem phim (có thể lồng vào trang Chi tiết phim)
	incomingRoutes.POST("/history/watch/:movie_id", controllers.SaveWatchHistory())
	
	// API lấy danh sách gợi ý cho màn hình Trang chủ
	incomingRoutes.GET("/history/recommendations", controllers.RecommendMovies())
}