package main

import (
    "log"
    "net/http"
    "os"

    ghandlers "github.com/gorilla/handlers"
    "github.com/gorilla/mux"
    "github.com/joho/godotenv"
    "github.com/kwabena369/scrapper/internal/db"
    "github.com/kwabena369/scrapper/internal/handlers"
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

    // FeedFollower routes
    followerProtected := routerV1.PathPrefix("/feed-followers").Subrouter()
    followerProtected.Use(handlers.AuthMiddleware)
    followerProtected.HandleFunc("", handlers.FollowFeed(client)).Methods("POST")
    followerProtected.HandleFunc("/{id}", handlers.UnfollowFeed(client)).Methods("DELETE")

    // Add CORS middleware
    corsHandler := ghandlers.CORS(
        ghandlers.AllowedOrigins([]string{"http://localhost:3000"}), // Adjust if needed
        ghandlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
        ghandlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
        ghandlers.AllowCredentials(),
    )(router)

    log.Printf("Starting server on port %s", port)
    if err := http.ListenAndServe(":"+port, corsHandler); err != nil {
        log.Fatalf("Could not start server: %s\n", err.Error())
    }
}