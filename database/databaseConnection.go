package database

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)


// Hàm này sẽ thiết lập cấu hình tự xóa (TTL Index) cho Token
func CreateTTLIndex(client *mongo.Client) {
	collection := client.Database("horashi-api").Collection("refresh_tokens")

	// Tạo một Index trên trường "expires_at"
	indexModel := mongo.IndexModel{
		Keys: bson.D{{Key: "expires_at", Value: 1}},
		// ExpireAfterSeconds = 0 nghĩa là MongoDB sẽ tự xóa bản ghi đúng vào mốc thời gian lưu trong cột expires_at
		Options: options.Index().SetExpireAfterSeconds(0),
	}

	_, err := collection.Indexes().CreateOne(context.TODO(), indexModel)
	if err != nil {
		fmt.Println("Lỗi khi tạo TTL Index cho Refresh Token:", err.Error())
	} else {
		fmt.Println("Đã thiết lập thành công TTL Index cho Refresh Token!")
	}
}


func StartDB() *mongo.Client {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	MongoDb := os.Getenv("MONGOURI")

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(MongoDb).SetServerAPIOptions(serverAPI)

	client, err := mongo.Connect(opts)
	if err != nil {
		panic(err)
	}

	// Sends a ping to confirm a successful connection
	var result bson.M
	if err := client.Database("admin").RunCommand(context.TODO(), bson.D{{Key: "ping", Value: 1}}).Decode(&result); err != nil {
		panic(err)
	}

	fmt.Println("Pinged your deployment. You successfully connected to MongoDB!")

	return client
}

var Client = StartDB()

func OpenCollection(client *mongo.Client, collectionName string) *mongo.Collection {
	var collection = client.Database("horashi-api").Collection(collectionName)
	return collection
}
