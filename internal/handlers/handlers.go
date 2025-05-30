package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/kwabena369/scrapper/internal/db"
	"github.com/kwabena369/scrapper/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
    "github.com/kwabena369/scrapper/internal/rss"
)

func RespondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
    dat, err := json.Marshal(payload)
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

// AuthMiddleware checks for a valid Firebase token in the Authorization header
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            RespondWithError(w, http.StatusUnauthorized, "No token provided")
            return
        }
        tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
        if tokenStr == authHeader { // No Bearer prefix
            RespondWithError(w, http.StatusUnauthorized, "Invalid token format")
            return
        }
        _, err := db.AuthClient.VerifyIDToken(context.Background(), tokenStr)
        if err != nil {
            RespondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
            return
        }
        next.ServeHTTP(w, r)
    })
}

func CreateUser(client *mongo.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var user models.User
        if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
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
        feedFollowerCollection := client.Database("hope").Collection("feed_followers")
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        // Cascade delete: Remove associated feed followers
        _, err = feedFollowerCollection.DeleteMany(ctx, bson.M{"user_id": objectID})
        if err != nil {
            log.Printf("Failed to delete feed followers: %v", err)
        }

        _, err = collection.DeleteOne(ctx, bson.M{"_id": objectID})
        if err != nil {
            RespondWithError(w, http.StatusInternalServerError, "Failed to delete user")
            return
        }
        RespondWithJSON(w, http.StatusOK, map[string]string{"message": "User deleted"})
    }
}

// Feed CRUD Handlers
func CreateFeed(client *mongo.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        log.Println("Received request to create feed")

        var feed models.Feed
        if err := json.NewDecoder(r.Body).Decode(&feed); err != nil {
            log.Printf("Error decoding request body: %v", err)
            RespondWithError(w, http.StatusBadRequest, "Invalid input")
            return
        }
        log.Printf("Decoded feed: %+v", feed)

        // Validate required fields
        if feed.Name == "" || feed.Url == "" || feed.UserID.IsZero() {
            log.Println("Missing required fields in feed")
            RespondWithError(w, http.StatusBadRequest, "Missing required fields")
            return
        }

        feed.ID = primitive.NewObjectID()
        feed.CreatedAt = time.Now()
        feed.UpdatedAt = time.Now()
        log.Printf("Saving feed with ID: %s", feed.ID.Hex())

        collection := client.Database("hope").Collection("feeds")
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        _, err := collection.InsertOne(ctx, feed)
        if err != nil {
            log.Printf("Error saving feed to MongoDB: %v", err)
            RespondWithError(w, http.StatusInternalServerError, "Failed to create feed")
            return
        }
        log.Println("Feed created successfully")
        RespondWithJSON(w, http.StatusCreated, feed)
    }
}

func GetFeed(client *mongo.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        id := mux.Vars(r)["id"]
        objectID, err := primitive.ObjectIDFromHex(id)
        if err != nil {
            RespondWithError(w, http.StatusBadRequest, "Invalid ID")
            return
        }
        collection := client.Database("hope").Collection("feeds")
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        var feed models.Feed
        err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&feed)
        if err != nil {
            RespondWithError(w, http.StatusNotFound, "Feed not found")
            return
        }
        RespondWithJSON(w, http.StatusOK, feed)
    }
}

func UpdateFeed(client *mongo.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        id := mux.Vars(r)["id"]
        objectID, err := primitive.ObjectIDFromHex(id)
        if err != nil {
            RespondWithError(w, http.StatusBadRequest, "Invalid ID")
            return
        }
        var feed models.Feed
        if err := json.NewDecoder(r.Body).Decode(&feed); err != nil {
            RespondWithError(w, http.StatusBadRequest, "Invalid input")
            return
        }
        feed.ID = objectID
        feed.UpdatedAt = time.Now()
        collection := client.Database("hope").Collection("feeds")
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        _, err = collection.UpdateOne(ctx, bson.M{"_id": objectID}, bson.M{"$set": feed})
        if err != nil {
            RespondWithError(w, http.StatusInternalServerError, "Failed to update feed")
            return
        }
        RespondWithJSON(w, http.StatusOK, feed)
    }
}

func DeleteFeed(client *mongo.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        id := mux.Vars(r)["id"]
        objectID, err := primitive.ObjectIDFromHex(id)
        if err != nil {
            RespondWithError(w, http.StatusBadRequest, "Invalid ID")
            return
        }
        collection := client.Database("hope").Collection("feeds")
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        _, err = collection.DeleteOne(ctx, bson.M{"_id": objectID})
        if err != nil {
            RespondWithError(w, http.StatusInternalServerError, "Failed to delete feed")
            return
        }
        RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Feed deleted"})
    }
}

// Optional: Get all feeds (for listing purposes)
func GetAllFeeds(client *mongo.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        collection := client.Database("hope").Collection("feeds")
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        cursor, err := collection.Find(ctx, bson.M{})
        if err != nil {
            RespondWithError(w, http.StatusInternalServerError, "Failed to fetch feeds")
            return
        }
        defer cursor.Close(ctx)

        var feeds []models.Feed
        if err = cursor.All(ctx, &feeds); err != nil {
            RespondWithError(w, http.StatusInternalServerError, "Failed to decode feeds")
            return
        }
        RespondWithJSON(w, http.StatusOK, feeds)
    }
}


func FollowFeed(client *mongo.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var follower models.FeedFollower
        if err := json.NewDecoder(r.Body).Decode(&follower); err != nil {
            RespondWithError(w, http.StatusBadRequest, "Invalid input")
            return
        }
        follower.ID = primitive.NewObjectID()
        follower.CreatedAt = time.Now()
        collection := client.Database("hope").Collection("feed_followers")
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        _, err := collection.InsertOne(ctx, follower)
        if err != nil {
            RespondWithError(w, http.StatusInternalServerError, "Failed to follow feed")
            return
        }
        RespondWithJSON(w, http.StatusCreated, follower)
    }
}

func UnfollowFeed(client *mongo.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        id := mux.Vars(r)["id"] // Assuming ID is the FeedFollower ID
        objectID, err := primitive.ObjectIDFromHex(id)
        if err != nil {
            RespondWithError(w, http.StatusBadRequest, "Invalid ID")
            return
        }
        collection := client.Database("hope").Collection("feed_followers")
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        _, err = collection.DeleteOne(ctx, bson.M{"_id": objectID})
        if err != nil {
            RespondWithError(w, http.StatusInternalServerError, "Failed to unfollow feed")
            return
        }
        RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Unfollowed feed"})
    }
}
func ScrapeFeed(client *mongo.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        id := mux.Vars(r)["id"]
        objectID, err := primitive.ObjectIDFromHex(id)
        if err != nil {
            RespondWithError(w, http.StatusBadRequest, "Invalid Feed ID")
            return
        }

        // Fetch the feed from MongoDB
        collection := client.Database("hope").Collection("feeds")
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        var feed models.Feed
        err = collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&feed)
        if err != nil {
            RespondWithError(w, http.StatusNotFound, "Feed not found")
            return
        }

        // Fetch existing items to avoid duplicates
        itemCollection := client.Database("hope").Collection("feed_items")
        cursor, err := itemCollection.Find(ctx, bson.M{"feed_id": objectID})
        if err != nil {
            RespondWithError(w, http.StatusInternalServerError, "Failed to fetch existing items")
            return
        }
        defer cursor.Close(ctx)

        existingLinks := make(map[string]bool)
        var existingItems []models.FeedItem
        if err = cursor.All(ctx, &existingItems); err != nil {
            RespondWithError(w, http.StatusInternalServerError, "Failed to decode existing items")
            return
        }
        for _, item := range existingItems {
            existingLinks[item.Link] = true
        }

        // Scrape RSS items
        items, err := rss.FetchRSS(feed.Url)
        if err != nil {
            RespondWithError(w, http.StatusInternalServerError, "Failed to scrape RSS feed: "+err.Error())
            return
        }

        // Store new items only
        var newItemsCount int
        for _, item := range items {
            if existingLinks[item.Link] {
                continue // Skip existing items
            }

            pubDate, err := time.Parse(time.RFC1123Z, item.PubDate)
            if err != nil {
                log.Printf("Failed to parse pubDate for item %s: %v", item.Title, err)
                continue
            }

            feedItem := models.FeedItem{
                ID:          primitive.NewObjectID(),
                FeedID:      feed.ID,
                Title:       item.Title,
                Link:        item.Link,
                Description: item.Description,
                PubDate:     pubDate,
            }

            _, err = itemCollection.InsertOne(ctx, feedItem)
            if err != nil {
                log.Printf("Failed to save feed item %s: %v", item.Title, err)
                continue
            }
            newItemsCount++
        }

        RespondWithJSON(w, http.StatusOK, map[string]interface{}{
            "message": "Feed scraped successfully",
            "new_items": newItemsCount,
        })
    }
}


func GetFeedItems(client *mongo.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]
		objectID, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			RespondWithError(w, http.StatusBadRequest, "Invalid Feed ID")
			return
		}

		collection := client.Database("hope").Collection("feed_items")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cursor, err := collection.Find(ctx, bson.M{"feed_id": objectID})
		if err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Failed to fetch feed items")
			return
		}
		defer cursor.Close(ctx)

		var items []models.FeedItem
		if err = cursor.All(ctx, &items); err != nil {
			RespondWithError(w, http.StatusInternalServerError, "Failed to decode feed items")
			return
		}

		RespondWithJSON(w, http.StatusOK, items)
	}
}