package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type WatchHistory struct {
	Id        bson.ObjectID `json:"id" bson:"_id"`
	UserId    string        `json:"user_id" bson:"user_id"`
	MovieId   string        `json:"movie_id" bson:"movie_id"`
	WatchedAt time.Time     `json:"watched_at" bson:"watched_at"`
}