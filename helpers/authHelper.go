package helpers

import (
	"errors"

	"github.com/gin-gonic/gin"
)

func VerifyUserType(c *gin.Context, role string) (err error) {
	userType := c.GetString("user_type")
	err = nil
	if userType != role {
		err = errors.New("you are not authorised to access this resource")
		return err
	}
	return err
}

func MatchToUid(c *gin.Context, userId string) (err error) {
	userType := c.GetString("user_type")
	uid := c.GetString("uid")
	err = nil
	if userType == "USER" && uid != userId {
		err = errors.New("you are not authorised to access this resource")
	}
	err = VerifyUserType(c, userType)
	return err
}
