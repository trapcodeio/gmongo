package gmongo

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
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

// SumMany - Sum many documents
//
// Example:
// Data: [
//
//	{name: "John", credit: 100, debit: 400},
//	{name: "Doe", credit: 200, debit: 300},
//
// ]
//
//	sum, _ := gmongo.SumMany(Model, []string{"credit", "debit"})
//	// sum will be {credit: 300, debit: 700}
func SumMany[V Number, T ModelData](coll *Model[T], keys []string, filter interface{}) (bson.M, error) {
	group := bson.M{"_id": nil}
	result := bson.M{}

	for _, key := range keys {
		group[key] = bson.M{"$sum": fmt.Sprintf("$%s", key)}
		result[key] = V(0) // Initialize with zero value of type V
	}

	pipeline := bson.A{}
	if filter != nil {
		pipeline = append(pipeline, bson.M{"$match": filter})
	}

	pipeline = append(pipeline, bson.M{"$group": group})

	res, err := coll.Aggregate(pipeline)
	if err != nil {
		return result, err
	}

	if len(res) == 0 {
		for _, key := range keys {
			result[key] = V(0)
		}
		return result, nil
	}

	data := res[0]
	for key, value := range data {
		if _, ok := result[key]; !ok {
			continue
		}

		// Convert to requested type V
		switch v := value.(type) {
		case float64:
			result[key] = V(v)
		case int32:
			result[key] = V(v)
		case int64:
			result[key] = V(v)
		case int:
			result[key] = V(v)
		}
	}

	return result, nil
}

// Sum - Sum documents
//
// Example:
// Data: [
//
//	{name: "John", credit: 100, debit: 400},
//	{name: "Doe", credit: 200, debit: 300},
//
// ]
// sum, _ := UserModel.Sum(Model, "credit", nil)
// // sum will be 300
func Sum[V Number, T ModelData](coll *Model[T], key string, filter interface{}) (V, error) {
	res, err := SumMany[V, T](coll, []string{key}, filter)
	if err != nil {
		return V(0), err
	}

	return res[key].(V), nil
}
