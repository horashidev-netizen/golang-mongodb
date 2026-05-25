package controllers

import (
	"context"
	"golang-mongodb/database"
	"golang-mongodb/helpers"
	"golang-mongodb/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var historyCollection = database.OpenCollection(database.Client, "watch_history")
// Tái sử dụng movieCollection đã khai báo ở movieControllers.go
// Nếu bị lỗi khai báo trùng, bạn chỉ cần import dùng chung là được.

// 1. Ghi lại lịch sử xem phim
func SaveWatchHistory() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Yêu cầu đăng nhập để biết ai đang xem
		if err := helpers.VerifyUserType(c, "USER"); err != nil && helpers.VerifyUserType(c, "ADMIN") != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Vui lòng đăng nhập để lưu lịch sử"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		userId := c.GetString("uid")
		movieId := c.Param("movie_id")

		// Dùng Upsert: Nếu người dùng xem lại phim cũ thì chỉ cập nhật thời gian, chưa xem thì tạo mới
		filter := bson.M{"user_id": userId, "movie_id": movieId}
		update := bson.M{
			"$set": bson.M{
				"watched_at": time.Now(),
			},
			"$setOnInsert": bson.M{
				"_id": bson.NewObjectID(),
			},
		}

		opts := options.UpdateOne().SetUpsert(true)
		_, err := historyCollection.UpdateOne(ctx, filter, update, opts)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu lịch sử xem phim"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"Message": "Đã ghi nhận lịch sử xem phim"})
	}
}

// 2. Thuật toán Gợi ý phim
func RecommendMovies() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		userId := c.GetString("uid")

		// 1. Lấy phim GẦN NHẤT người dùng vừa xem
		var lastWatch models.WatchHistory
		findOptions := options.FindOne().SetSort(bson.D{{Key: "watched_at", Value: -1}})
		err := historyCollection.FindOne(ctx, bson.M{"user_id": userId}, findOptions).Decode(&lastWatch)
		
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"Message": "Chưa có lịch sử, trả về phim thịnh hành..."}) // Gọi API phim mặc định
			return
		}

		// Lấy Object ID của phim đó
		lastMovieObjId, _ := bson.ObjectIDFromHex(lastWatch.MovieId)

		// 2. Lấy bộ phim đó ra từ CSDL để lấy cái Vector của nó
		var targetMovie models.Movie
		err = movieCollection.FindOne(ctx, bson.M{"_id": lastMovieObjId}).Decode(&targetMovie)
		if err != nil || len(targetMovie.Embedding) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Phim gốc không có dữ liệu Vector"})
			return
		}

		// 3. SIÊU TRUY VẤN: VECTOR SEARCH
		// Tìm những phim có Vector giống với phim vừa xem nhất, NHƯNG phải bỏ qua chính bộ phim đó!
		vectorSearchStage := bson.D{
			{Key: "$vectorSearch", Value: bson.D{
				{Key: "index", Value: "vector_index"}, 
				{Key: "path", Value: "embedding"},
				{Key: "queryVector", Value: targetMovie.Embedding},
				{Key: "numCandidates", Value: 100}, 
				{Key: "limit", Value: 10},          // Chọn 10 kết quả tốt nhất
				{Key: "filter", Value: bson.D{      // Lọc bỏ chính bộ phim đã xem
					{Key: "_id", Value: bson.D{{Key: "$ne", Value: lastMovieObjId}}},
				}},
			}},
		}

		// Thực thi Aggregation
		cursor, err := movieCollection.Aggregate(ctx, mongo.Pipeline{vectorSearchStage})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi thuật toán tìm kiếm: " + err.Error()})
			return
		}

		var recommendations []models.Movie
		cursor.All(ctx, &recommendations)

		c.JSON(http.StatusOK, gin.H{
			"Status":  http.StatusOK,
			"Message": "Gợi ý AI thành công",
			"Data":    recommendations,
		})
	}
}