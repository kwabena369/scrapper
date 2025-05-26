// internal/handlers/handlers.go
package handlers

import (
    "context"
    "encoding/json"
    "log"
    "net/http"
    "time"

    "github.com/gorilla/mux"
    "github.com/kwabena369/scrapper/internal/models"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
)

func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
    dat, err := json.Marshal(payload) // Changed to json.Marshal for JSON compatibility
    if err != nil {
        log.Printf("Failed to marshal JSON response: %v", payload)
        w.WriteHeader(500)
        return
    }
    w.Header().Add("Content-Type", "application/json")
    w.WriteHeader(code)
    w.Write(dat)
}

func RespondWithError(w http.ResponseWriter, code int, msg string) {
    if code > 499 {
        log.Printf("Server error: %s", msg)
    }
    RespondWithJSON(w, code, map[string]string{"error": msg})
}

func TheLoggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        log.Printf("%s %s", r.Method, r.URL.Path)
        next.ServeHTTP(w, r)
    })
}

func CreateUser(client *mongo.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var user models.User
        if err := json.NewDecoder(r.Body).Decode(&user); err != nil { // Fixed decoding
            RespondWithError(w, http.StatusBadRequest, "Invalid input")
            return
        }
        user.ID = primitive.NewObjectID()
        collection := client.Database("hope").Collection("users")
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        _, err := collection.InsertOne(ctx, user)
        if err != nil {
            RespondWithError(w, http.StatusInternalServerError, "Failed to create user")
            return
        }
        RespondWithJSON(w, http.StatusCreated, user)
    }
}

func GetUser(client *mongo.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        id := mux.Vars(r)["id"]
        objectID, err := primitive.ObjectIDFromHex(id)
        if err != nil {
            RespondWithError(w, http.StatusBadRequest, "Invalid ID")
            return
        }
        collection := client.Database("hope").Collection("users")
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        var user models.User
        err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&user)
        if err != nil {
            RespondWithError(w, http.StatusNotFound, "User not found")
            return
        }
        RespondWithJSON(w, http.StatusOK, user)
    }
}

func UpdateUser(client *mongo.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        id := mux.Vars(r)["id"]
        objectID, err := primitive.ObjectIDFromHex(id)
        if err != nil {
            RespondWithError(w, http.StatusBadRequest, "Invalid ID")
            return
        }
        var user models.User
        if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
            RespondWithError(w, http.StatusBadRequest, "Invalid input")
            return
        }
        collection := client.Database("hope").Collection("users")
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        _, err = collection.UpdateOne(ctx, bson.M{"_id": objectID}, bson.M{"$set": user})
        if err != nil {
            RespondWithError(w, http.StatusInternalServerError, "Failed to update user")
            return
        }
        RespondWithJSON(w, http.StatusOK, user)
    }
}

func DeleteUser(client *mongo.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        id := mux.Vars(r)["id"]
        objectID, err := primitive.ObjectIDFromHex(id)
        if err != nil {
            RespondWithError(w, http.StatusBadRequest, "Invalid ID")
            return
        }
        collection := client.Database("hope").Collection("users")
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        _, err = collection.DeleteOne(ctx, bson.M{"_id": objectID})
        if err != nil {
            RespondWithError(w, http.StatusInternalServerError, "Failed to delete user")
            return
        }
        RespondWithJSON(w, http.StatusOK, map[string]string{"message": "User deleted"})
    }
}