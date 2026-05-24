package routes

import (
	"golang-mongodb/controllers"
	"golang-mongodb/middleware"

	"github.com/gin-gonic/gin"
)

func ReviewRouter(incomingRoutes *gin.Engine) {
	// Các API CÓ THỂ KHÔNG CẦN đăng nhập (Ai cũng có thể xem review)
	// Lấy danh sách review của một bộ phim
	incomingRoutes.GET("/reviews/movie/:movie_id", controllers.ViewAMovieReviews())
	// Lấy danh sách review của một người dùng
	incomingRoutes.GET("/reviews/user/:reviewer_id", controllers.AllUserReviews())

	// ==========================================
	// Các API BẮT BUỘC ĐĂNG NHẬP (Thêm, Xóa, Sửa)
	// ==========================================
	incomingRoutes.Use(middleware.Authenticate())
	
	// Thêm review mới
	incomingRoutes.POST("/reviews/create", controllers.AddReview())
	// Xóa một review
	incomingRoutes.DELETE("/reviews/:review_id", controllers.DeleteReview())
	incomingRoutes.PUT("/reviews/:review_id", controllers.EditReview())
}