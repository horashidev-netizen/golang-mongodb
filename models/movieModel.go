package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Movie struct {
	Id        bson.ObjectID `json:"id" bson:"_id"`
	Name      string        `json:"name" bson:"name" validate:"required"`
	Topic     string        `json:"topic" bson:"topic" validate:"required"`
	GenreId   string        `json:"genre_id" bson:"genre_id"`
	MovieURL  string        `json:"movie_url" bson:"movie_url" validate:"required"`
	CreatedAt time.Time     `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time     `json:"updated_at" bson:"updated_at"`
}
