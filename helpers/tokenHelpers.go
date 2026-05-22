package helpers

import (
	"log"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
)

type JwtSignedDetails struct {
	Email    string `json:"email" bson:"email"`
	Name     string `json:"name" bson:"name"`
	Username string `json:"username" bson:"username"`
	Uid      string `json:"uid" bson:"uid"`
	UserType string `json:"user_type" bson:"user_type"`
	jwt.StandardClaims
}

var SecretKey = []byte(os.Getenv("SECRET_KEY"))

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
			ExpiresAt: time.Now().Local().Add(time.Hour * time.Duration(12)).Unix(),
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
