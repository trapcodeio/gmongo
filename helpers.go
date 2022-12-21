package gmongo

import (
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

func DateTimeNow() primitive.DateTime {
	return primitive.NewDateTimeFromTime(time.Now())
}

func NewId() primitive.ObjectID {
	return primitive.NewObjectID()
}

func NewUUid() string {
	// use uuid version 4
	return uuid.NewString()
}
