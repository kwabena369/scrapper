package models

import (
    "time"

    "go.mongodb.org/mongo-driver/bson/primitive"
)

// User represents a user in the system
type User struct {
    ID          primitive.ObjectID `bson:"_id,omitempty" validate:"required"`
    FirebaseUID string             `bson:"firebase_uid" validate:"required"`
    Username    string             `bson:"username" validate:"required"`
    Email       string             `bson:"email" validate:"required,email"`
    CreatedAt   time.Time          `bson:"created_at" validate:"required"`
    UpdatedAt   time.Time          `bson:"updated_at" validate:"required"`
}

// Feed represents an RSS feed
type Feed struct {
    ID        primitive.ObjectID `bson:"_id,omitempty" validate:"required"`
    Name      string             `bson:"name" validate:"required"`
    Url       string             `bson:"url" validate:"required"`
    UserID    primitive.ObjectID `bson:"user_id" validate:"required"`
    CreatedAt time.Time          `bson:"created_at" validate:"required"`
    UpdatedAt time.Time          `bson:"updated_at" validate:"required"`
}

// FeedItem represents an item in an RSS feed
type FeedItem struct {
    ID          primitive.ObjectID `bson:"_id,omitempty" validate:"required"`
    FeedID      primitive.ObjectID `bson:"feed_id" validate:"required"`
    Title       string             `bson:"title" validate:"required"`
    Link        string             `bson:"link" validate:"required"`
    Description string             `bson:"description"`
    PubDate     time.Time          `bson:"pub_date" validate:"required"`
}

// FeedFollower represents a user's subscription to a feed
type FeedFollower struct {
    ID        primitive.ObjectID `bson:"_id,omitempty"`
    FeedID    primitive.ObjectID `bson:"feed_id" validate:"required"`
    UserID    string             `bson:"user_id" validate:"required"` // Changed to string for Firebase UID
    CreatedAt time.Time          `bson:"created_at" validate:"required"`
}