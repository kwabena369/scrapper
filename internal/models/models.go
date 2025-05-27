// internal/models/models.go
package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
    ID         primitive.ObjectID `bson:"_id,omitempty"`
    FirebaseUID string            `bson:"firebase_uid" validate:"required"`
    Email       string            `bson:"email" validate:"required,email"`
    Username    string            `bson:"username" validate:"required"`
}