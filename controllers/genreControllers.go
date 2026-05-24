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

var genreCollection = database.OpenCollection(database.Client, "genre")

func CreateGenre() gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := helpers.VerifyUserType(c, "ADMIN"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		var genre models.Genre
		defer cancel()
		if err := c.BindJSON(&genre); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"Status":  http.StatusBadRequest,
				"Message": err.Error(),
				"Data":    map[string]any{"data": err.Error()},
			})
			return
		}
		//Check to see if name exists
		nameMatch := bson.M{"$regex": bson.Regex{Pattern: genre.Name, Options: "i"}}
		count, err := genreCollection.CountDocuments(ctx, bson.M{"name": nameMatch})
		defer cancel()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "error occurred while checking for the genre name",
			})
			log.Panic(err.Error())
			return
		}
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{
				"error": "genre already exists",
				"count": count,
			})
			return
		}
		newObjId := bson.NewObjectID()
		newGenre := models.Genre{
			Id:        newObjId,
			Name:      genre.Name,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		result, err := genreCollection.InsertOne(ctx, newGenre)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"Status":  http.StatusBadRequest,
				"Message": err.Error(),
				"Data":    map[string]any{"data": err.Error()},
			})
			return
		}
		c.JSON(http.StatusCreated, gin.H{
			"Status":  http.StatusCreated,
			"Data":    result,
			"Message": "genre created successfully",
		})
	}
}

func GetGenre() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		genreID := c.Param("genre_id")
		var genre models.Genre
		objId, err := bson.ObjectIDFromHex(genreID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"Status": http.StatusBadRequest,
			})
			return
		}
		err = genreCollection.FindOne(ctx, bson.M{"_id": objId}).Decode(&genre)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"Status":  http.StatusInternalServerError,
				"Message": err.Error(),
				"Data":    map[string]any{"data": err.Error()},
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"Status": http.StatusOK,
			"Data":   map[string]any{"data": genre},
		})

	}
}

func GetGenres() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel() // GỌI LUÔN Ở ĐÂY ĐỂ TRÁNH QUÊN VÀ RÒ RỈ BỘ NHỚ

		// ==========================================
		// 1. XỬ LÝ PHÂN TRANG (PAGINATION)
		// ==========================================
		recordPerPage, err := strconv.Atoi(c.Query("recordPerPage"))
		if err != nil || recordPerPage < 1 {
			recordPerPage = 10 // Mặc định hiển thị 10 thể loại/trang
		}

		page, err1 := strconv.Atoi(c.Query("page"))
		if err1 != nil || page < 1 {
			page = 1 // Mặc định là trang 1
		}

		startIndex := (page - 1) * recordPerPage

		// CHỈ ghi đè startIndex NẾU trên Postman có truyền tham số này
		if queryStartIndex := c.Query("startIndex"); queryStartIndex != "" {
			if parsedStart, err := strconv.Atoi(queryStartIndex); err == nil {
				startIndex = parsedStart
			}
		}

		// ==========================================
		// 2. MONGODB PIPELINE - TỐI ƯU HÓA BẰNG $facet
		// ==========================================
		matchStage := bson.D{{Key: "$match", Value: bson.D{{}}}}

		// Dùng $facet để đếm tổng và phân trang song song (Giải phóng RAM)
		facetStage := bson.D{{Key: "$facet", Value: bson.D{
			// Luồng 1: Đếm tổng số lượng thể loại
			{Key: "metadata", Value: bson.A{
				bson.D{{Key: "$count", Value: "total_count"}},
			}},
			// Luồng 2: Lọc lấy đúng số lượng thể loại cần hiển thị (Cắt sớm!)
			{Key: "genre_items", Value: bson.A{
				bson.D{{Key: "$skip", Value: startIndex}},
				bson.D{{Key: "$limit", Value: recordPerPage}},
			}},
		}}}

		// Format lại kết quả json cho đẹp
		projectStage := bson.D{{Key: "$project", Value: bson.D{
			{Key: "total_count", Value: bson.D{{Key: "$arrayElemAt", Value: []any{"$metadata.total_count", 0}}}},
			{Key: "genre_items", Value: 1},
		}}}

		result, err := genreCollection.Aggregate(ctx, mongo.Pipeline{matchStage, facetStage, projectStage})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi tổng hợp dữ liệu thể loại: " + err.Error()})
			return
		}

		var allGenres []bson.M
		if err = result.All(ctx, &allGenres); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi đọc kết quả: " + err.Error()})
			return
		}

		// ==========================================
		// 3. VÒNG BẢO VỆ CHỐNG PANIC (Nếu Database rỗng)
		// ==========================================
		if len(allGenres) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"total_count": 0,
				"genre_items": []any{},
			})
			return
		}

		// Trả kết quả mượt mà
		c.JSON(http.StatusOK, allGenres[0])
	}
}

func EditGenre() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Phân quyền
		if err := helpers.VerifyUserType(c, "ADMIN"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 2. Ép kiểu ID chuẩn MongoDB (Fix lỗi Atoi)
		genreID := c.Param("genre_id")
		objId, errId := bson.ObjectIDFromHex(genreID)
		if errId != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Định dạng ID không hợp lệ"})
			return
		}

		// 3. Bind JSON và Validate (Giữ nguyên ý tưởng hay của bạn)
		var genre models.Genre
		if err := c.BindJSON(&genre); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu sai định dạng: " + err.Error()})
			return
		}

		// 4. Kiểm tra trùng lặp (Kế thừa logic $ne)
		nameMatch := bson.M{"$regex": bson.Regex{Pattern: genre.Name, Options: "i"}}
		filterCheck := bson.M{
			"name": nameMatch,
			"_id":  bson.M{"$ne": objId}, // Tìm tên giống nhưng ID phải khác
		}
		count, errCheck := genreCollection.CountDocuments(ctx, filterCheck)
		if errCheck != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi server: " + errCheck.Error()})
			return
		}
		if count > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "Tên thể loại này đã bị trùng với một thể loại khác!"})
			return
		}

		// 5. Cập nhật vào Database
		filterByID := bson.M{"_id": objId} // Tìm bằng ObjectID
		updateObj := bson.M{
			"$set": bson.M{
				"name":       genre.Name,
				"updated_at": time.Now(), // Cập nhật thời gian
			},
		}

		result, err := genreCollection.UpdateOne(ctx, filterByID, updateObj)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi cập nhật: " + err.Error()})
			return
		}

		// 6. Lấy lại dữ liệu mới nhất trả về (Giữ nguyên ý tưởng của bạn)
		var updatedGenre models.Genre
		if result.MatchedCount == 1 {
			err := genreCollection.FindOne(ctx, filterByID).Decode(&updatedGenre)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi lấy dữ liệu: " + err.Error()})
				return
			}
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy thể loại để cập nhật"})
			return
		}

		// 7. Trả kết quả thành công
		c.JSON(http.StatusOK, gin.H{
			"Status":  http.StatusOK,
			"Message": "Cập nhật thành công!",
			"Data":    updatedGenre,
		})
	}
}
func DeleteGenre() gin.HandlerFunc {
	return func(c *gin.Context) {
		// ==========================================
		// LEVEL UP 1: CHỐT CHẶN BẢO MẬT (Chỉ Admin mới được xóa)
		// ==========================================
		if err := helpers.VerifyUserType(c, "ADMIN"); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Bạn không có quyền thực hiện thao tác xóa!"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// ==========================================
		// LEVEL UP 2: ÉP KIỂU ID CHUẨN MONGODB
		// ==========================================
		genreID := c.Param("genre_id")
		objId, errId := bson.ObjectIDFromHex(genreID)
		if errId != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Định dạng ID không hợp lệ (Phải là ObjectID 24 ký tự)"})
			return
		}

		// ==========================================
		// LEVEL UP 3: THỰC THI LỆNH XÓA (Tìm bằng _id)
		// ==========================================
		result, err := genreCollection.DeleteOne(ctx, bson.M{"_id": objId})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi hệ thống khi xóa dữ liệu: " + err.Error()})
			return
		}

		// Nếu gọi lệnh xóa thành công nhưng không có cái nào bị xóa (Do ID không tồn tại)
		if result.DeletedCount < 1 {
			c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy thể loại này, có thể nó đã bị xóa từ trước!"})
			return
		}

		// ==========================================
		// LEVEL UP 4: TRẢ VỀ JSON GỌN GÀNG, SẠCH SẼ
		// ==========================================
		c.JSON(http.StatusOK, gin.H{
			"Status":  http.StatusOK,
			"Message": "Xóa thể loại thành công!",
		})
	}
}

func SearchGenresByName() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Lấy từ khóa tìm kiếm từ URL (Ví dụ: /genres/search?name=Ani => Animal)
		searchQuery := c.Query("name")

		// Bắt lỗi nếu người dùng không nhập từ khóa
		if searchQuery == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Vui lòng nhập từ khóa tìm kiếm vào tham số 'name'"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// 2. Sử dụng $regex để tìm kiếm gần đúng và không phân biệt hoa thường (Options: "i")
		filter := bson.M{
			"name": bson.M{"$regex": bson.Regex{Pattern: searchQuery, Options: "i"}},
		}

		// 3. Thực thi tìm kiếm trong Database
		// Dùng Find() thay vì FindOne() vì kết quả có thể trả về nhiều thể loại
		cursor, err := genreCollection.Find(ctx, filter)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi truy vấn dữ liệu: " + err.Error()})
			return
		}
		defer func(cursor *mongo.Cursor, ctx context.Context) {
			err := cursor.Close(ctx)
			if err != nil {

			}
		}(cursor, ctx) // Nhớ đóng cursor để giải phóng RAM

		// 4. Giải mã toàn bộ kết quả vào một mảng (Array)
		var genres []models.Genre
		if err = cursor.All(ctx, &genres); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi đọc kết quả: " + err.Error()})
			return
		}

		// (Mẹo nhỏ): Nếu không tìm thấy kết quả nào, mảng sẽ bị nil (null trong JSON).
		// Gán nó thành mảng rỗng [] để Frontend dễ xử lý vòng lặp hơn.
		if len(genres) == 0 {
			genres = []models.Genre{}
		}

		// 5. Trả kết quả về cho Frontend
		c.JSON(http.StatusOK, gin.H{
			"Status":  http.StatusOK,
			"Message": "Tìm kiếm thành công",
			"Count":   len(genres), // Báo cho Frontend biết tìm được bao nhiêu kết quả
			"Data":    genres,
		})
	}
}
