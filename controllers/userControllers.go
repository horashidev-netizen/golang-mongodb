package controllers

import (
	"context"
	"fmt"
	"golang-mongodb/helpers"
	"golang-mongodb/models"
	"log"
	"net/http"
	"strconv"
	"time"

	"golang-mongodb/database"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"golang.org/x/crypto/bcrypt"
)

var UserCollection = database.OpenCollection(database.Client, "users")

func MaskPassword(password string) string {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	if err != nil {
		log.Fatal(err)
	}
	return string(bytes)
}

func ConfirmPassword(userPassword string, passwordEntered string) (bool, string) {
	err := bcrypt.CompareHashAndPassword([]byte(userPassword), []byte(passwordEntered))
	check := true
	msg := ""
	if err != nil {
		msg = "Look like you entered a wrong password"
		check = false
	}
	return check, msg
}
func Login() gin.HandlerFunc {
	return func(c *gin.Context) {
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var user models.User
		defer cancel()
		var retrievedUser models.User
		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		err := UserCollection.FindOne(ctx, bson.M{"email": user.Email}).Decode(&retrievedUser)
		if err != nil {
			fmt.Println("LỖI FIND ONE DATABASE:", err.Error())
			c.JSON(http.StatusInternalServerError, gin.H{"error": "your email or password is incorrect"})
			return
		}
		//fmt.Println(">>>> user:", user)
		//fmt.Println(">>>> retrievedUser:", retrievedUser)
		passwordIsValid, msg := ConfirmPassword(retrievedUser.Password, user.Password)
		if passwordIsValid != true {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "your email or password is incorrect" + msg})
			return
		}

		token, refreshToken, _ := helpers.GenerateAllToken(retrievedUser.Email, retrievedUser.Name, retrievedUser.Username, retrievedUser.UserType, retrievedUser.ID.Hex())
		helpers.UpdateTokens(token, refreshToken, retrievedUser.ID.Hex())
		// update token directly to memory and return instantly to retrievedUser
		retrievedUser.Token = token
		retrievedUser.RefreshToken = refreshToken
		c.JSON(http.StatusOK, retrievedUser)
	}
}

func Signup() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		var user models.User
		defer cancel()
		if err := c.BindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"Status":  http.StatusBadRequest,
				"Message": "error",
				"Data":    map[string]interface{}{"data": err.Error()},
			})
			return
		}
		//Check to see if name exists

		emailMatch := bson.M{"$regex": bson.Regex{Pattern: user.Email, Options: "i"}}
		emailCount, emailErr := UserCollection.CountDocuments(ctx, bson.M{"email": emailMatch})
		usernameMatch := bson.M{"$regex": bson.Regex{Pattern: user.Username, Options: "i"}}
		usernameCount, usernameErr := UserCollection.CountDocuments(ctx, bson.M{"username": usernameMatch})
		if emailErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi DB: " + emailErr.Error()})
			return
		}
		if emailCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Looks like this email already exists", "count": emailCount})
			return
		}
		if usernameErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "error occurred while checking for this email / username"})
			return
		}
		if usernameCount > 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Looks like this username already exists", "count": usernameCount})
			return
		}

		user.Password = MaskPassword(user.Password)
		user.CreatedAt, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		user.UpdatedAt, _ = time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
		user.ID = bson.NewObjectID()
		//Sign details to token
		token, refreshToken, _ := helpers.GenerateAllToken(
			user.Email,
			user.Name,
			user.Username,
			user.UserType,
			user.ID.Hex(),
		)
		user.Token = token
		user.RefreshToken = refreshToken
		//Save refTok to redis 100h
		//errRedis := database.RedisClient.Set(context.TODO(), user.UserId.Hex(), refreshToken, time.Duration(100)*time.Hour).Err()
		//if errRedis != nil {
		//	c.JSON(http.StatusInternalServerError, gin.H{
		//		"Status":  http.StatusInternalServerError,
		//		"Message": "error",
		//		"error":   "Cannot save to Redis",
		//	})
		//}

		// save user to MongoDB
		resultInsertionNumber, insertErr := UserCollection.InsertOne(ctx, user)
		if insertErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lưu user vào database"})
			return
		}

		// 2. return success and show info of user via ...
		c.JSON(http.StatusOK, gin.H{
			"Status":  http.StatusOK,
			"message": "Đăng ký tài khoản thành công!",
			"user_id": resultInsertionNumber.InsertedID,
			"data":    user,
		})
	}
}

func GetUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.Param("user_id")
		var ctx, cancel = context.WithTimeout(context.Background(), 100*time.Second)
		var user models.User
		defer cancel()
		objId, errId := bson.ObjectIDFromHex(userId)
		if errId != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ID must be hex 24 format" + errId.Error()})
		}
		err := UserCollection.FindOne(ctx, bson.M{"_id": objId}).Decode(&user)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, user)
	}
}

func GetUsers() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Phân quyền Admin (Tuyệt vời, bạn đã giữ cái này rất tốt!)
		if err := helpers.VerifyUserType(c, "ADMIN"); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second) // Giảm xuống 10s cho an toàn, 100s là quá dài
		defer cancel()

		// ==========================================
		// 2. XỬ LÝ PHÂN TRANG (PAGINATION)
		// ==========================================
		recordPerPage, err := strconv.Atoi(c.Query("recordPerPage"))
		if err != nil || recordPerPage <= 0 {
			recordPerPage = 10
		}

		page, err1 := strconv.Atoi(c.Query("page"))
		if err1 != nil || page <= 0 {
			page = 1
		}

		startIndex := (page - 1) * recordPerPage

		// CHỈ ghi đè startIndex NẾU người dùng thực sự truyền nó trên URL
		if queryStartIndex := c.Query("startIndex"); queryStartIndex != "" {
			if parsedStart, err := strconv.Atoi(queryStartIndex); err == nil {
				startIndex = parsedStart
			}
		}

		// ==========================================
		// 3. MONGODB PIPELINE - TỐI ƯU HÓA BẰNG $facet
		// ==========================================
		matchStage := bson.D{{Key: "$match", Value: bson.D{{}}}}

		// Phân luồng: Đếm tổng và Cắt dữ liệu chạy song song
		facetStage := bson.D{{Key: "$facet", Value: bson.D{
			{Key: "metadata", Value: bson.A{
				bson.D{{Key: "$count", Value: "total_count"}},
			}},
			{Key: "user_items", Value: bson.A{
				bson.D{{Key: "$skip", Value: startIndex}},
				bson.D{{Key: "$limit", Value: recordPerPage}}, // Lọc lấy đúng lượng User cần thiết
			}},
		}}}

		// Format lại kết quả json cho đẹp
		projectStage := bson.D{{Key: "$project", Value: bson.D{
			{Key: "total_count", Value: bson.D{{Key: "$arrayElemAt", Value: []interface{}{"$metadata.total_count", 0}}}},
			{Key: "user_items", Value: 1},
		}}}

		result, err := UserCollection.Aggregate(ctx, mongo.Pipeline{matchStage, facetStage, projectStage})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi tổng hợp dữ liệu tài khoản: " + err.Error()})
			return
		}

		var allUsers []bson.M
		if err = result.All(ctx, &allUsers); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi khi giải mã kết quả: " + err.Error()})
			return
		}

		// ==========================================
		// 4. BẢO VỆ CHỐNG PANIC KHI DỮ LIỆU RỖNG
		// ==========================================
		if len(allUsers) == 0 {
			c.JSON(http.StatusOK, gin.H{
				"total_count": 0,
				"user_items":  []interface{}{},
			})
			return
		}

		// Trả kết quả chuẩn xác
		c.JSON(http.StatusOK, allUsers[0])
	}
}
