// internal/db/db.go
package db

import (
    "context"
    "log"
    "os"
    "time"

    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

var Client *mongo.Client

func ConnectMongo() {
    uri := os.Getenv("MONGO_URI")
    if uri == "" {
        log.Fatal("MONGO_URI not set in .env")
    }

    client, err := mongo.Connect(context.Background(), options.Client().ApplyURI(uri))
    if err != nil {
        log.Fatalf("Failed to connect to MongoDB: %v", err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    err = client.Ping(ctx, nil)
    if err != nil {
        log.Fatalf("Failed to ping MongoDB: %v", err)
    }

    Client = client
    log.Println("Connected to MongoDB successfully")
}

func DisconnectMongo() {
    if Client != nil {
        if err := Client.Disconnect(context.Background()); err != nil {
            log.Fatalf("Failed to disconnect from MongoDB: %v", err)
        }
    }
}