package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Review struct {
	Id        bson.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	MovieId   string        `json:"movie_id" bson:"movie_id" validate:"required"`
	Content   string        `json:"content" bson:"content" validate:"required,min=5"`
	CreatedAt time.Time     `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time     `json:"updated_at" bson:"updated_at"`
}
