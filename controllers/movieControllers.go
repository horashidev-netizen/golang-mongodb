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
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

var movieCollection = database.OpenCollection(database.Client, "movie")

var validate = validator.New()

func CreateMovie() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Phân quyền
		if err := helpers.VerifyUserType(c, "ADMIN"); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		var movie models.Movie

		// 2. Ép kiểu JSON từ Postman
		if err := c.BindJSON(&movie); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"Status":  http.StatusBadRequest,
				"Message": "Dữ liệu JSON sai định dạng",
				"Data":    map[string]interface{}{"data": err.Error()},
			})
			return
		}

		// ==========================================
		// 3. VALIDATE NGAY LẬP TỨC (Dịch chuyển lên đây)
		// ==========================================
		if validationError := validate.Struct(&movie); validationError != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"Status":  http.StatusBadRequest,
				"Message": "Thiếu dữ liệu bắt buộc",
				"Data":    map[string]interface{}{"data": validationError.Error()},
			})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel() // CHỈ GỌI 1 LẦN Ở ĐÂY LÀ ĐỦ BẢO VỆ

		// 4. Kiểm tra trùng lặp tên phim
		regexMatch := bson.M{"$regex": bson.Regex{Pattern: movie.Name, Options: "i"}}
		count, err := movieCollection.CountDocuments(ctx, bson.M{"name": regexMatch})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi kiểm tra tên phim: " + err.Error()})
			return
		}
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Tên phim này đã tồn tại trong hệ thống!"})
			return
		}

		// 5. Khởi tạo đối tượng Phim mới
		newMovie := models.Movie{
			Id:        bson.NewObjectID(),
			Name:      movie.Name,
			Topic:     movie.Topic,
			GenreId:   movie.GenreId,
			MovieURL:  movie.MovieURL,
			CreatedAt: time.Now(), // Server tự đóng mốc thời gian
			UpdatedAt: time.Now(), // Server tự đóng mốc thời gian
		}

		// 6. Lưu xuống DB
		result, err := movieCollection.InsertOne(ctx, newMovie)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"Status":  http.StatusInternalServerError,
				"Message": "Lỗi hệ thống khi lưu Database",
				"Data":    map[string]interface{}{"data": err.Error()},
			})
			return
		}

		// 7. Trả kết quả thành công
		c.JSON(http.StatusCreated, gin.H{
			"Status":  http.StatusCreated,
			"Message": "Tạo phim thành công!",
			"Data":    map[string]interface{}{"data": result},
		})
	}
}

func GetMovie() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		movieId := c.Param("movie_id")
		var movie models.Movie

		// ==========================================
		// SỬA LỖI 1: Ép kiểu string sang ObjectID
		// ==========================================
		objId, errId := bson.ObjectIDFromHex(movieId)
		if errId != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"Status":  http.StatusBadRequest,
				"Message": "Định dạng ID phim không hợp lệ",
				"Data":    map[string]interface{}{"data": errId.Error()},
			})
			return
		}

		// ==========================================
		// SỬA LỖI 2: Tìm kiếm bằng trường _id thay vì movie_id
		// ==========================================
		err := movieCollection.FindOne(ctx, bson.M{"_id": objId}).Decode(&movie)
		if err != nil {
			// Thường lỗi ở đây là do truyền đúng chuẩn ID nhưng phim không tồn tại
			c.JSON(http.StatusNotFound, gin.H{
				"Status":  http.StatusNotFound,
				"Message": "Không tìm thấy bộ phim này",
				"Data":    map[string]interface{}{"data": err.Error()},
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"Status":  http.StatusOK,
			"Message": "success",
			"Data":    map[string]interface{}{"data": movie},
		})
	}
}
func GetMovies() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel() // GỌI LUÔN Ở ĐÂY ĐỂ TRÁNH QUÊN

		// ==========================================
		// 1. XỬ LÝ PHÂN TRANG (PAGINATION)
		// ==========================================
		recordPerPage, _ := strconv.Atoi(c.Query("recordPerPage"))
		if recordPerPage < 1 {
			recordPerPage = 10
		}
		page, _ := strconv.Atoi(c.Query("page"))
		if page < 1 {
			page = 1
		}
		startIndex := (page - 1) * recordPerPage
		if queryStartIndex := c.Query("startIndex"); queryStartIndex != "" {
			if parsedStart, err := strconv.Atoi(queryStartIndex); err == nil {
				startIndex = parsedStart
			}
		}

		// ==========================================
		// 2. MONGODB PIPELINE - TỐI ƯU HÓA HIỆU SUẤT ($facet)
		// ==========================================
		matchStage := bson.D{{Key: "$match", Value: bson.D{{}}}}

		// Dùng $facet để chạy 2 luồng công việc song song, không cần gom $group nữa
		facetStage := bson.D{{Key: "$facet", Value: bson.D{
			// Luồng 1: Đếm tổng số phim (rất nhẹ)
			{Key: "metadata", Value: bson.A{
				bson.D{{Key: "$count", Value: "total_count"}},
			}},
			// Luồng 2: Lọc sớm, Cắt sớm ngay tại Database!
			{Key: "movie_items", Value: bson.A{
				bson.D{{Key: "$skip", Value: startIndex}},     // Nhảy qua số trang đã qua
				bson.D{{Key: "$limit", Value: recordPerPage}}, // Chỉ bốc đúng 10 phim
			}},
		}}}

		// Format lại kết quả cho đẹp (gỡ mảng metadata ra)
		projectStage := bson.D{{Key: "$project", Value: bson.D{
			{Key: "total_count", Value: bson.D{{Key: "$arrayElemAt", Value: []interface{}{"$metadata.total_count", 0}}}},
			{Key: "movie_items", Value: 1},
		}}}

		result, err := movieCollection.Aggregate(ctx, mongo.Pipeline{matchStage, facetStage, projectStage})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi tổng hợp dữ liệu: " + err.Error()})
			return
		}

		var allMovies []bson.M
		if err = result.All(ctx, &allMovies); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi đọc kết quả: " + err.Error()})
			return
		}

		// ... (Phần xử lý mảng rỗng ở dưới giữ nguyên) ...
		if len(allMovies) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"total_count": 0,
				"movie_items": []interface{}{},
			})
			return
		}

		// Trả kết quả
		c.JSON(http.StatusOK, allMovies[0])
	}
}

func UpdateMovie() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		movieId := c.Param("movie_id")
		var movie models.Movie
		defer cancel()
		objId, _ := bson.ObjectIDFromHex(movieId)

		if err := c.BindJSON(&movie); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"Status":  http.StatusBadRequest,
				"Message": "error",
				"Data":    map[string]interface{}{"data": err.Error()}})
			return
		}

		if validationError := validate.Struct(&movie); validationError != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"Status":  http.StatusBadRequest,
				"Message": "error",
				"Data":    map[string]interface{}{"data": validationError.Error()}})
			return
		}

		update := bson.M{
			"name":      movie.Name,
			"topic":     movie.Topic,
			"genre_id":  movie.GenreId,
			"movie_url": movie.MovieURL}
		filterByID := bson.M{"_id": bson.M{"$eq": objId}}
		result, err := movieCollection.UpdateOne(ctx, filterByID, bson.M{"$set": update})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"Status":  http.StatusInternalServerError,
				"Message": "error",
				"Data":    map[string]interface{}{"data": err.Error()}})
			return
		}

		var updatedMovie models.Movie
		if result.MatchedCount == 1 {
			err := movieCollection.FindOne(ctx, filterByID).Decode(&updatedMovie)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"Status":  http.StatusInternalServerError,
					"Message": "error",
					"Data":    map[string]interface{}{"data": err.Error()}})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"Status":  http.StatusOK,
			"Message": "movie updated successfully!",
			"Data":    updatedMovie})
	}
}

func SearchMovieByQuery() gin.HandlerFunc {
	return func(c *gin.Context) {
		var searchmovies []models.Movie
		movieName := c.Param("movieName")

		if movieName == "" {
			log.Println("movie name is empty")
			c.Header("Content-Type", "application/json")
			c.JSON(http.StatusNotFound, gin.H{"Error": "Invalid search parameter" + movieName})
			c.Abort()
			return
		}
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		searcherDB, err := movieCollection.Find(ctx, bson.M{"name": bson.M{"$regex": bson.Regex{Pattern: movieName, Options: "i"}}})
		if err != nil {
			c.IndentedJSON(404, "something went wrong")
			return
		}
		err = searcherDB.All(ctx, &searchmovies)
		if err != nil {
			log.Println(err)
			c.IndentedJSON(400, "invalid")
			return
		}
		if err := searcherDB.Err(); err != nil {
			log.Println(err)
			c.IndentedJSON(400, "invalid request")
			return
		}
		defer cancel()
		c.IndentedJSON(200, searchmovies)

	}
}

func SearchMovieByGenre() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Lấy Genre ID từ URL (Giữ nguyên định dạng chuỗi String)
		genreID := c.Param("genreID")
		if genreID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu mã thể loại (genreID) trong URL"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel() // CHỈ GỌI 1 LẦN DỰ PHÒNG TIMEOUT

		// 2. Khởi tạo mảng rỗng để chống lỗi null JSON
		var searchByGenre []models.Movie

		// 3. Tìm kiếm toàn bộ phim khớp với genre_id
		// Chú ý: Vì GenreId trong struct Movie là string, nên ta gán thẳng genreID vào tìm kiếm
		cursor, err := movieCollection.Find(ctx, bson.M{"genre_id": genreID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi truy vấn Database: " + err.Error()})
			return
		}

		// GỌI DEFER CLOSE NGAY SAU KHI FIND THÀNH CÔNG ĐỂ GIẢI PHÓNG RAM
		defer func(cursor *mongo.Cursor, ctx context.Context) {
			err := cursor.Close(ctx)
			if err != nil {

			}
		}(cursor, ctx)

		// 4. Giải mã dữ liệu từ Cursor vào mảng
		if err = cursor.All(ctx, &searchByGenre); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi đọc dữ liệu phim: " + err.Error()})
			return
		}

		// 5. Trả kết quả đồng bộ với các API khác
		c.JSON(http.StatusOK, gin.H{
			"Status":  http.StatusOK,
			"Message": "Tìm kiếm thành công",
			"Count":   len(searchByGenre), // Tính luôn số lượng phim cho Frontend
			"Data":    searchByGenre,
		})
	}
}
func DeleteMovie() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		movieId := c.Param("movie_id")
		objId, errId := bson.ObjectIDFromHex(movieId)
		if errId != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Định dạng ID phim không hợp lệ (Phải là ObjectID 24 ký tự)"})
			return
		}

		result, err := movieCollection.DeleteOne(ctx, bson.M{"_id": objId})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống khi xóa dữ liệu: " + err.Error()})
			return
		}

		// Nếu gọi lệnh xóa thành công nhưng không có bản ghi nào bị xóa (ID không tồn tại)
		if result.DeletedCount < 1 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy bộ phim này, có thể nó đã bị xóa từ trước!"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"Status":  http.StatusOK,
			"Message": "Xóa bộ phim thành công!",
		})
	}
}
