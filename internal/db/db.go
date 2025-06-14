package db

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/api/option"
)

var Client *mongo.Client
var AuthClient *auth.Client

type UserClaims struct {
	UID   string
	Email string
}

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

func ConnectFirebase() {
	ctx := context.Background()

	credPath := os.Getenv("FIREBASE_CRED_PATH")
	if credPath == "" {
		log.Fatal("FIREBASE_CRED_PATH not set in .env")
	}

	absPath, err := filepath.Abs(credPath)
	if err != nil {
		log.Fatalf("Failed to resolve Firebase credentials path: %v", err)
	}

	opt := option.WithCredentialsFile(absPath)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		log.Fatalf("Firebase initialization error: %v", err)
	}

	authClient, err := app.Auth(ctx)
	if err != nil {
		log.Fatalf("Auth client error: %v", err)
	}
	AuthClient = authClient
	log.Println("Connected to Firebase Auth successfully")
}

func DisconnectMongo() {
	if Client != nil {
		if err := Client.Disconnect(context.Background()); err != nil {
			log.Printf("Failed to disconnect from MongoDB: %v", err)
		} else {
			log.Println("Disconnected from MongoDB")
		}
	}
}