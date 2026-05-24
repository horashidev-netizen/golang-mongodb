package controllers

import (
	"context"
	"golang-mongodb/database"
	"golang-mongodb/helpers"
	"golang-mongodb/models"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var reviewCollection *mongo.Collection = database.OpenCollection(database.Client, "review")

// Add  new review
func AddReview() gin.HandlerFunc {
	return func(c *gin.Context) {
		//Logged in account must be of the type USER
		if err := helpers.VerifyUserType(c, "USER"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		var review models.Reviews
		defer cancel()

		if err := c.BindJSON(&review); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"Status":  http.StatusBadRequest,
				"Message": "error",
				"Data":    map[string]any{"data": err.Error()}})
			return
		}

		if validationError := validate.Struct(&review); validationError != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"Status":  http.StatusBadRequest,
				"Message": "error",
				"Data":    map[string]any{"data": validationError.Error()}})
			return
		}

		newReview := models.Reviews{
			Id:         bson.NewObjectID(),
			MovieId:    review.MovieId,
			Content: 	review.Content,
		}

		result, err := reviewCollection.InsertOne(ctx, newReview)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"Status":  http.StatusInternalServerError,
				"Message": "error",
				"Data":    map[string]any{"data": err.Error()}})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"Status":  http.StatusCreated,
			"Message": "success",
			"Data":    map[string]any{"data": result}})
	}
}

// Filter reviews by movie_id
func ViewAMovieReviews() gin.HandlerFunc {
	return func(c *gin.Context) {
		var searchreviews []models.Reviews
		movieId := c.Param("movie_id")
		i,erro:= strconv.Atoi(movieId)
		if erro != nil {
			// Handle error
		}
		if movieId == "" {
			log.Println("No movie id passed")
			c.Header("Content-Type", "application/json")
			c.JSON(http.StatusNotFound, gin.H{"Error": "Invalid Search Index"})
			c.Abort()
			return
		}
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		searchquerydb, err := reviewCollection.Find(ctx, bson.M{"movie_id": i})
		if err != nil {
			c.IndentedJSON(404, "something went wrong in fetching the dbquery")
			return
		}
		err = searchquerydb.All(ctx, &searchreviews)
		if err != nil {
			log.Println(err)
			c.IndentedJSON(400, "invalid")
			return
		}
		defer searchquerydb.Close(ctx)
		if err := searchquerydb.Err(); err != nil {
			log.Println(err)
			c.IndentedJSON(400, "invalid request")
			return
		}
		defer cancel()
		c.IndentedJSON(200, searchreviews)
	}
}

func DeleteReview() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		reviewId := c.Param("review_id")
		defer cancel()
		i,erro:= strconv.Atoi(reviewId)
		if erro != nil {
			// Handle error
		}

		result, err := reviewCollection.DeleteOne(ctx, bson.M{"review_id": i})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"Status":  http.StatusInternalServerError,
				"Message": "error",
				"Data":    map[string]any{"data": err.Error()}})
			return
		}

		if result.DeletedCount < 1 {
			c.JSON(http.StatusNotFound,
				gin.H{
					" Status":  http.StatusNotFound,
					" Message": "error",
					" Data":    map[string]any{"data": "Review with specified ID not found!"}},
			)
			return
		}

		c.JSON(http.StatusOK,
			gin.H{
				"Status":  http.StatusOK,
				"Message": "success",
				"Data":    map[string]any{"data": "Your review was successfully deleted!"}},
		)
	}
}


// Allow a user view all their Reviews
func AllUserReviews() gin.HandlerFunc {
	return func(c *gin.Context) {
		var searchreviews []models.Reviews
		reviewId := c.Param("reviewer_id")
		//defer cancel()
		i,erro:= strconv.Atoi(reviewId)
		if erro != nil {
			// Handle error
		}
		if reviewId == "" {
			log.Println("No reviewer id passed")
			c.Header("Content-Type", "application/json")
			c.JSON(http.StatusNotFound, gin.H{"Error": "Invalid Search Index"})
			c.Abort()
			return
		}
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		searchquerydb, err := reviewCollection.Find(ctx, bson.M{"reviewer_id":i})

		if err != nil {
			c.IndentedJSON(404, "something went wrong in fetching the dbquery")
			return
		}
		err = searchquerydb.All(ctx, &searchreviews)
		if err != nil {
			log.Println(err)
			c.IndentedJSON(400, "invalid")
			return
		}
		defer searchquerydb.Close(ctx)
		if err := searchquerydb.Err(); err != nil {
			log.Println(err)
			c.IndentedJSON(400, "invalid request")
			return
		}
		defer cancel()
		c.IndentedJSON(200, searchreviews)
	}
}

// Edit a review
func EditReview() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Phân quyền (chỉ có tài khoản USER mới được sửa đánh giá)
		if err := helpers.VerifyUserType(c, "USER"); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 2. Lấy ID từ URL và chuyển thành định dạng ObjectID chuẩn của MongoDB
		reviewId := c.Param("review_id")
		objId, errId := bson.ObjectIDFromHex(reviewId)
		if errId != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"Status":  http.StatusBadRequest,
				"Message": "Định dạng ID đánh giá không hợp lệ",
				"Data":    errId.Error(),
			})
			return
		}

		// 3. Lấy dữ liệu update từ body request
		var review models.Reviews // Sử dụng đúng tên struct Review trong models/reviewModel.go
		if err := c.BindJSON(&review); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"Status":  http.StatusBadRequest,
				"Message": "Dữ liệu JSON không hợp lệ",
				"Data":    err.Error(),
			})
			return
		}

		// 4. Validate dữ liệu cơ bản (Tránh người dùng sửa review thành chuỗi rỗng)
		if review.Content == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"Status":  http.StatusBadRequest,
				"Message": "Nội dung đánh giá (content) không được để trống",
			})
			return
		}

		// 5. Chuẩn bị query update
		filter := bson.M{"_id": objId}
		update := bson.M{
			"$set": bson.M{
				"content":    review.Content,
				"updated_at": time.Now(), // Cập nhật lại mốc thời gian sửa đổi
			},
		}

		// 6. Thực thi Update
		result, err := reviewCollection.UpdateOne(ctx, filter, update)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"Status":  http.StatusInternalServerError,
				"Message": "Lỗi server khi cập nhật đánh giá",
				"Data":    err.Error(),
			})
			return
		}

		// Nếu lệnh chạy thành công nhưng không có bản ghi nào khớp với ID
		if result.MatchedCount < 1 {
			c.JSON(http.StatusNotFound, gin.H{
				"Status":  http.StatusNotFound,
				"Message": "Không tìm thấy đánh giá này để cập nhật!",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"Status":  http.StatusOK,
			"Message": "Cập nhật đánh giá thành công!",
		})
	}
}