package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Genre struct {
	Id        bson.ObjectID `bson:"_id"`
	Name      string        `bson:"name" valid:"required,min=4,max=100"`
	CreatedAt time.Time     `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time     `bson:"updated_at" json:"updated_at"`
}
