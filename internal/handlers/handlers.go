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
    "github.com/kwabena369/scrapper/internal/email"
    "github.com/kwabena369/scrapper/internal/models"
    "github.com/kwabena369/scrapper/internal/rss"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
)

// Feed is re-exported for use in main.go
type Feed = models.Feed

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

func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            RespondWithError(w, http.StatusUnauthorized, "No token provided")
            return
        }
        tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
        if tokenStr == authHeader {
            RespondWithError(w, http.StatusUnauthorized, "Invalid token format")
            return
        }
        user, err := db.AuthClient.VerifyIDToken(context.Background(), tokenStr)
        if err != nil {
            RespondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
            return
        }
        // Convert *auth.Token to *db.UserClaims
        fullUser, err := db.AuthClient.GetUser(context.Background(), user.UID)
        if err != nil {
            RespondWithError(w, http.StatusInternalServerError, "Failed to get user details")
            return
        }
        claims := &db.UserClaims{
            UID:   user.UID,
            Email: fullUser.Email,
        }
        ctx := context.WithValue(r.Context(), "user", claims)
        next.ServeHTTP(w, r.WithContext(ctx))
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
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

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
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
        feedFollowerCollection := client.Database("hope").Collection("feed_followers")
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        _, err = feedFollowerCollection.DeleteMany(ctx, bson.M{"feed_id": objectID})
        if err != nil {
            log.Printf("Failed to delete feed followers: %v", err)
        }

        _, err = collection.DeleteOne(ctx, bson.M{"_id": objectID})
        if err != nil {
            RespondWithError(w, http.StatusInternalServerError, "Failed to delete feed")
            return
        }
        RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Feed deleted"})
    }
}

func GetAllFeeds(client *mongo.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        collection := client.Database("hope").Collection("feeds")
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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
        user := r.Context().Value("user").(*db.UserClaims)
        if user == nil {
            RespondWithError(w, http.StatusUnauthorized, "User not authenticated")
            return
        }

        var input struct {
            FeedID string `json:"feed_id"`
            UserID string `json:"user_id"`
        }
        if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
            RespondWithError(w, http.StatusBadRequest, "Invalid input")
            return
        }

        feedID, err := primitive.ObjectIDFromHex(input.FeedID)
        if err != nil {
            RespondWithError(w, http.StatusBadRequest, "Invalid Feed ID")
            return
        }

        if user.UID != input.UserID {
            RespondWithError(w, http.StatusForbidden, "Unauthorized user")
            return
        }

        collection := client.Database("hope").Collection("feed_followers")
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        // Check if already following
        count, err := collection.CountDocuments(ctx, bson.M{"feed_id": feedID, "user_id": input.UserID})
        if err != nil {
            RespondWithError(w, http.StatusInternalServerError, "Failed to check follow status")
            return
        }
        if count > 0 {
            RespondWithError(w, http.StatusConflict, "Already following this feed")
            return
        }

        follower := models.FeedFollower{
            ID:        primitive.NewObjectID(),
            FeedID:    feedID,
            UserID:    input.UserID, // Store as string (Firebase UID)
            CreatedAt: time.Now(),
        }

        _, err = collection.InsertOne(ctx, follower)
        if err != nil {
            RespondWithError(w, http.StatusInternalServerError, "Failed to follow feed")
            return
        }
        RespondWithJSON(w, http.StatusCreated, follower)
    }
}

func UnfollowFeed(client *mongo.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        user := r.Context().Value("user").(*db.UserClaims)
        if user == nil {
            RespondWithError(w, http.StatusUnauthorized, "User not authenticated")
            return
        }

        feedIDStr := mux.Vars(r)["id"]
        feedID, err := primitive.ObjectIDFromHex(feedIDStr)
        if err != nil {
            RespondWithError(w, http.StatusBadRequest, "Invalid Feed ID")
            return
        }

        collection := client.Database("hope").Collection("feed_followers")
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        result, err := collection.DeleteOne(ctx, bson.M{"feed_id": feedID, "user_id": user.UID})
        if err != nil {
            RespondWithError(w, http.StatusInternalServerError, "Failed to unfollow feed")
            return
        }
        if result.DeletedCount == 0 {
            RespondWithError(w, http.StatusNotFound, "Follow relationship not found")
            return
        }
        RespondWithJSON(w, http.StatusOK, map[string]string{"message": "Unfollowed feed"})
    }
}

func GetFollowedFeeds(client *mongo.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        user := r.Context().Value("user").(*db.UserClaims)
        if user == nil {
            RespondWithError(w, http.StatusUnauthorized, "User not authenticated")
            return
        }

        collection := client.Database("hope").Collection("feed_followers")
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()

        cursor, err := collection.Find(ctx, bson.M{"user_id": user.UID})
        if err != nil {
            RespondWithError(w, http.StatusInternalServerError, "Failed to fetch followed feeds")
            return
        }
        defer cursor.Close(ctx)

        var followers []models.FeedFollower
        if err = cursor.All(ctx, &followers); err != nil {
            RespondWithError(w, http.StatusInternalServerError, "Failed to decode followed feeds")
            return
        }
        RespondWithJSON(w, http.StatusOK, followers)
    }
}

func ScrapeFeedLogic(client *mongo.Client, feedID string) (int, []models.FeedItem, error) {
    startTime := time.Now()
    log.Printf("Starting ScrapeFeedLogic for feed %s", feedID)

    objectID, err := primitive.ObjectIDFromHex(feedID)
    if err != nil {
        return 0, nil, err
    }

    // Fetch feed
    collection := client.Database("hope").Collection("feeds")
    ctxFeed, cancelFeed := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancelFeed()

    var feed models.Feed
    err = collection.FindOne(ctxFeed, bson.M{"_id": objectID}).Decode(&feed)
    if err != nil {
        log.Printf("Failed to fetch feed %s: %v", feedID, err)
        return 0, nil, err
    }
    log.Printf("Fetched feed %s in %v", feedID, time.Since(startTime))

    // Fetch existing items
    itemCollection := client.Database("hope").Collection("feed_items")
    ctxFetch, cancelFetch := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancelFetch()

    fetchStart := time.Now()
    cursor, err := itemCollection.Find(ctxFetch, bson.M{"feed_id": objectID})
    if err != nil {
        log.Printf("Failed to fetch existing items for feed %s: %v", feedID, err)
        return 0, nil, err
    }
    defer cursor.Close(ctxFetch)

    existingLinks := make(map[string]bool)
    var existingItems []models.FeedItem
    if err = cursor.All(ctxFetch, &existingItems); err != nil {
        log.Printf("Failed to decode existing items for feed %s: %v", feedID, err)
        return 0, nil, err
    }
    for _, item := range existingItems {
        existingLinks[item.Link] = true
    }
    log.Printf("Fetched %d existing items for feed %s in %v", len(existingItems), feedID, time.Since(fetchStart))

    // Fetch RSS items
    items, err := rss.FetchRSS(feed.Url)
    if err != nil {
        log.Printf("Failed to fetch RSS for feed %s: %v", feedID, err)
        return 0, nil, err
    }
    log.Printf("Fetched %d RSS items for feed %s in %v", len(items), feedID, time.Since(fetchStart))

    // Prepare new items for batch insert
    var newItems []interface{}
    var newFeedItems []models.FeedItem
    for _, item := range items {
        if existingLinks[item.Link] {
            continue
        }

        pubDate, err := rss.ParsePubDate(item.PubDate)
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
        newItems = append(newItems, feedItem)
        newFeedItems = append(newFeedItems, feedItem)
    }

    // Batch insert new items
    newItemsCount := len(newItems)
    if newItemsCount > 0 {
        insertStart := time.Now()
        ctxInsert, cancelInsert := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancelInsert()

        _, err = itemCollection.InsertMany(ctxInsert, newItems)
        if err != nil {
            log.Printf("Failed to batch insert %d items for feed %s: %v", newItemsCount, feedID, err)
            return 0, nil, err
        }
        log.Printf("Batch inserted %d new items for feed %s in %v", newItemsCount, feedID, time.Since(insertStart))

        // Notify followers
        go notifyFollowers(client, feed, newFeedItems)
    } else {
        log.Printf("No new items to insert for feed %s", feedID)
    }

    log.Printf("Completed ScrapeFeedLogic for feed %s in %v", feedID, time.Since(startTime))
    return newItemsCount, newFeedItems, nil
}

func notifyFollowers(client *mongo.Client, feed models.Feed, newItems []models.FeedItem) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Fetch followers
    followerCollection := client.Database("hope").Collection("feed_followers")
    cursor, err := followerCollection.Find(ctx, bson.M{"feed_id": feed.ID})
    if err != nil {
        log.Printf("Failed to fetch followers for feed %s: %v", feed.ID.Hex(), err)
        return
    }
    defer cursor.Close(ctx)

    var followers []models.FeedFollower
    if err = cursor.All(ctx, &followers); err != nil {
        log.Printf("Failed to decode followers for feed %s: %v", feed.ID.Hex(), err)
        return
    }

    // Fetch user emails
    userCollection := client.Database("hope").Collection("users")
    for _, follower := range followers {
        var user models.User
        err = userCollection.FindOne(ctx, bson.M{"firebase_uid": follower.UserID}).Decode(&user)
        if err != nil {
            log.Printf("Failed to fetch user %s for notification: %v", follower.UserID, err)
            continue
        }

        // Send email notification
        err = email.SendFeedUpdateEmail(user.Email, user.Username, feed.Name, newItems)
        if err != nil {
            log.Printf("Failed to send email to %s: %v", user.Email, err)
        } else {
            log.Printf("Sent notification email to %s for feed %s", user.Email, feed.ID.Hex())
        }
    }
}

func ScrapeFeed(client *mongo.Client) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        id := mux.Vars(r)["id"]
        newItemsCount, _, err := ScrapeFeedLogic(client, id)
        if err != nil {
            RespondWithError(w, http.StatusInternalServerError, "Failed to scrape feed: "+err.Error())
            return
        }

        RespondWithJSON(w, http.StatusOK, map[string]interface{}{
            "message":    "Feed scraped successfully",
            "new_items":  newItemsCount,
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
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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