package helpers

import (
	"context"
	"fmt"
	"golang-mongodb/database"
	"log"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type JwtSignedDetails struct {
	Email    string `json:"email" bson:"email"`
	Name     string `json:"name" bson:"name"`
	Username string `json:"username" bson:"username"`
	Uid      string `json:"uid" bson:"uid"`
	UserType string `json:"user_type" bson:"user_type"`
	jwt.StandardClaims
}

var UserCollection = database.OpenCollection(database.Client, "users")
var SecretKey = []byte(os.Getenv("SECRET_KEY"))

func ValidateToken(signedToken string) (claims *JwtSignedDetails, msg string) {
	token, err := jwt.ParseWithClaims(signedToken, &JwtSignedDetails{}, func(token *jwt.Token) (interface{}, error) {
		return SecretKey, nil
	})
	if err != nil {
		msg = "Token is invalid" + err.Error()
		return
	}
	claims, ok := token.Claims.(*JwtSignedDetails)
	if !ok {
		msg = fmt.Sprintf("This token is incorrect. Sorry!")
		return
	}

	if claims.ExpiresAt < time.Now().Local().Unix() {
		msg = fmt.Sprintf("Ooops looks like your token has expired")
		return
	}

	return claims, msg
}

func UpdateTokens(signedToken string, signedRefreshToken string, userId string) {
	var ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	var updateToken bson.D
	defer cancel()
	objId, _ := bson.ObjectIDFromHex(userId)

	updateToken = append(updateToken, bson.E{Key: "token", Value: signedToken})
	updateToken = append(updateToken, bson.E{Key: "refresh_token", Value: signedRefreshToken})

	UpdatedAt, _ := time.Parse(time.RFC3339, time.Now().Format(time.RFC3339))
	updateToken = append(updateToken, bson.E{Key: "updated_at", Value: UpdatedAt})
	filter := bson.M{"_id": objId}
	// Dùng hàm Builder của MongoDB để tạo config Upsert = true
	opts := options.UpdateOne().SetUpsert(true)
	// Update lai token voi user
	_, err := UserCollection.UpdateOne(ctx, filter, bson.M{"$set": updateToken}, opts)
	if err != nil {
		log.Panic(err)
	}
	return
}

func GenerateAllToken(
	email string,
	name string,
	username string,
	userType string,
	uid string) (signedToken string, signedRefreshToken string, err error) {
	claims := &JwtSignedDetails{
		Email:    email,
		Name:     name,
		Username: username,
		Uid:      uid,
		UserType: userType,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Local().Add(time.Hour * time.Duration(3)).Unix(),
		},
	}
	refreshClaims := &JwtSignedDetails{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Local().Add(time.Hour * time.Duration(100)).Unix(),
		},
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(SecretKey)
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(SecretKey)
	if err != nil {
		log.Panic(err)
		return
	}
	return token, refreshToken, err
}
