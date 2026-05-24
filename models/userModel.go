package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type User struct {
	ID           bson.ObjectID `bson:"_id, omitempty"`
	Name         string        `json:"name" bson:"name" validate:"required, min=4, max=100"`
	Username     string        `json:"username" bson:"username" validate:"required, min=4, max=100"`
	Password     string        `json:"password" bson:"password" validate:"required, min=8"`
	Email        string        `json:"email" bson:"email" validate:"required,email"`
	Token        string        `json:"token" bson:"token"`
	UserType     string        `json:"user_type" bson:"user_type" validate:"required,eq=ADMIN|eq=USER"`
	RefreshToken string        `json:"refresh_token" bson:"refresh_token"`
	CreatedAt    time.Time     `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at" bson:"updated_at"`
}
