package main

import (
	"log"
	"net/http"
	"os"
	"time"

	gcontext "context"

	ghandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/kwabena369/scrapper/internal/db"
	"github.com/kwabena369/scrapper/internal/email"
	"github.com/kwabena369/scrapper/internal/handlers"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func main() {
    err := godotenv.Load()
    if err != nil {
        log.Fatal("Error loading .env file")
    }
    port := os.Getenv("PORT")
    if port == "" {
        port = "8080"
    }

    db.ConnectMongo()
    db.ConnectFirebase()
    defer db.DisconnectMongo()

    email.InitEmailClient()

    router := mux.NewRouter()
    router.Use(handlers.TheLoggingMiddleware)

    routerV1 := router.PathPrefix("/v1").Subrouter()
    client := db.Client

    // Public route
    routerV1.HandleFunc("/users", handlers.CreateUser(client)).Methods("POST")

    // Protected routes
    protected := routerV1.PathPrefix("/users").Subrouter()
    protected.Use(handlers.AuthMiddleware)
    protected.HandleFunc("/{id}", handlers.GetUser(client)).Methods("GET")
    protected.HandleFunc("/{id}", handlers.UpdateUser(client)).Methods("PUT")
    protected.HandleFunc("/{id}", handlers.DeleteUser(client)).Methods("DELETE")

    // Feed CRUD routes
    feedProtected := routerV1.PathPrefix("/feeds").Subrouter()
    feedProtected.Use(handlers.AuthMiddleware)
    feedProtected.HandleFunc("", handlers.CreateFeed(client)).Methods("POST")
    feedProtected.HandleFunc("/{id}", handlers.GetFeed(client)).Methods("GET")
    feedProtected.HandleFunc("/{id}", handlers.UpdateFeed(client)).Methods("PUT")
    feedProtected.HandleFunc("/{id}", handlers.DeleteFeed(client)).Methods("DELETE")
    feedProtected.HandleFunc("", handlers.GetAllFeeds(client)).Methods("GET")
    feedProtected.HandleFunc("/{id}/scrape", handlers.ScrapeFeed(client)).Methods("POST")
    feedProtected.HandleFunc("/{id}/items", handlers.GetFeedItems(client)).Methods("GET")

    // FeedFollower routes
    followerProtected := routerV1.PathPrefix("/feed-followers").Subrouter()
    followerProtected.Use(handlers.AuthMiddleware)
    followerProtected.HandleFunc("", handlers.GetFollowedFeeds(client)).Methods("GET")
    followerProtected.HandleFunc("", handlers.FollowFeed(client)).Methods("POST")
    followerProtected.HandleFunc("/{id}", handlers.UnfollowFeed(client)).Methods("DELETE")

    // Start cron job for periodic scraping
    go startCronJob(client)

    // Add CORS middleware
    corsHandler := ghandlers.CORS(
        ghandlers.AllowedOrigins([]string{"http://localhost:3001"}),
        ghandlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
        ghandlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
        ghandlers.AllowCredentials(),
    )(router)

    log.Printf("Starting server on port %s", port)
    if err := http.ListenAndServe(":"+port, corsHandler); err != nil {
        log.Fatalf("Could not start server: %s\n", err.Error())
    }
}

func startCronJob(client *mongo.Client) {
    ticker := time.NewTicker(1 * time.Hour) // Run every hour
    defer ticker.Stop()

    for range ticker.C {
        log.Println("Running scheduled scrape job")
        feeds, err := getAllFeeds(client)
        if err != nil {
            log.Printf("Failed to fetch feeds for cron job: %v", err)
            continue
        }
        for _, feed := range feeds {
            log.Printf("Scraping feed %s", feed.ID.Hex())
            newItemsCount, _, err := handlers.ScrapeFeedLogic(client, feed.ID.Hex())
            if err != nil {
                log.Printf("Failed to scrape feed %s: %v", feed.ID.Hex(), err)
                continue
            }
            log.Printf("Scraped feed %s, added %d new items", feed.ID.Hex(), newItemsCount)
        }
    }
}

func getAllFeeds(client *mongo.Client) ([]handlers.Feed, error) {
    collection := client.Database("hope").Collection("feeds")
    ctx, cancel := gcontext.WithTimeout(gcontext.Background(), 5*time.Second)
    defer cancel()

    cursor, err := collection.Find(ctx, bson.M{})
    if err != nil {
        return nil, err
    }
    defer cursor.Close(ctx)

    var feeds []handlers.Feed
    if err = cursor.All(ctx, &feeds); err != nil {
        return nil, err
    }
    return feeds, nil
}