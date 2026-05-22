package controllers

import (
	"context"
	"golang-mongodb/helpers"
	"golang-mongodb/models"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/horashidev-netizen/golang-mongodb/database"
	"go.mongodb.org/mongo-driver/v2/bson"
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
	}
}
