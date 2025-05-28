package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

// User represents a user in the system
type User struct {
	ID         primitive.ObjectID `bson:"_id,omitempty"`
	FirebaseUID string            `bson:"firebase_uid" validate:"required"`
	Email       string            `bson:"email" validate:"required,email"`
	Username    string            `bson:"username" validate:"required"`
}

// Feed represents a feed entity
type Feed struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	CreatedAt time.Time          `bson:"created_at" validate:"required"`
	UpdatedAt time.Time          `bson:"updated_at" validate:"required"`
	Name      string             `bson:"name" validate:"required"`
	Url       string             `bson:"url" validate:"required,url"`
	UserID    primitive.ObjectID `bson:"user_id" validate:"required"`
}

// FeedFollower represents a user's follow relationship with a feed
type FeedFollower struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	FeedID    primitive.ObjectID `bson:"feed_id" validate:"required"`
	UserID    primitive.ObjectID `bson:"user_id" validate:"required"`
	CreatedAt time.Time          `bson:"created_at" validate:"required"`
}