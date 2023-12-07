package gmongo

import (
	"errors"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"time"
)

// DateTimeNow - Get the current date and time
func DateTimeNow() primitive.DateTime {
	return primitive.NewDateTimeFromTime(time.Now())
}

// NewId - Generate a new ID
func NewId() primitive.ObjectID {
	return primitive.NewObjectID()
}

// NewUUid - Generate a new UUID
func NewUUid() string {
	// use uuid version 4
	return uuid.NewString()
}

// IsNoDocumentsError - Check if the error exists and is a mongo.ErrNoDocuments error
func IsNoDocumentsError(err error) bool {
	return errors.Is(err, mongo.ErrNoDocuments)
}

// IsFindOneError - Check if the error exists but is not a mongo.ErrNoDocuments error
func IsFindOneError(err error) bool {
	return err != nil && !IsNoDocumentsError(err)
}
