package gmongo

import (
	"context"
	"fmt"
	"github.com/gookit/goutil/arrutil"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ModelData interface {
	// GetID - A function that returns the ID of the model
	GetID() primitive.ObjectID
}

// Number constraint for allowed numeric types
type Number interface {
	~int | ~int32 | ~int64 | ~float32 | ~float64
}

type Model[T ModelData] struct {
	CollectionName string
	PublicFields   []string
	Native         func() *mongo.Collection
}

// CreateModel - Create a new model with default values
// Note: Created model will not have a collection and will throw an error if `.Native()` is called
func CreateModel[T ModelData](collectionName string) *Model[T] {
	return &Model[T]{
		CollectionName: collectionName,
		PublicFields:   []string{},
		Native: func() *mongo.Collection {
			// Throw connection not linked error
			panic(fmt.Sprintf("Model is not linked to a database. Collection name: [%s]", collectionName))
		},
	}
}

// MakeModel - Create a new model with a database connection
func MakeModel[T ModelData](db *mongo.Database, collectionName string) Model[T] {
	// check if collection name is empty
	if collectionName == "" {
		panic("Collection name is empty")
	}

	collection := db.Collection(collectionName)

	return Model[T]{
		CollectionName: collectionName,
		PublicFields:   []string{},
		Native: func() *mongo.Collection {
			return collection
		},
	}
}

// LinkModel - Link model to a database
func LinkModel[T ModelData](model *Model[T], db *mongo.Database) {
	// check if collection name is empty
	if model.CollectionName == "" {
		panic("Collection name is empty")
	}

	collection := db.Collection(model.CollectionName)

	// Replace the native function with the actual collection
	model.Native = func() *mongo.Collection {
		return collection
	}
}

// FindOneAs - Find one document and decode it into a different struct
func (coll *Model[T]) FindOneAs(result interface{}, filter interface{}, opts ...*options.FindOneOptions) error {
	err := coll.Native().FindOne(context.TODO(), filter, opts...).Decode(result)
	return err
}

// FindOne - Find one document and decode it into the same struct
func (coll *Model[T]) FindOne(filter interface{}, opts ...*options.FindOneOptions) (T, error) {
	var result T

	err := coll.FindOneAs(&result, filter, opts...)
	if err != nil {
		return result, err
	}

	return result, nil
}

// FindOneById - Find one document by ID
func (coll *Model[T]) FindOneById(id primitive.ObjectID, opts ...*options.FindOneOptions) (T, error) {
	return coll.FindOne(bson.M{"_id": id}, opts...)
}

// DeleteOne Delete - Delete model from database
func (coll *Model[T]) DeleteOne(filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	return coll.Native().DeleteOne(context.TODO(), filter, opts...)
}

// UpdateOne - Update model in database
func (coll *Model[T]) UpdateOne(filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	return coll.Native().UpdateOne(context.TODO(), filter, update, opts...)
}

// Count - Count documents in database
func (coll *Model[T]) Count(filter interface{}, opts ...*options.CountOptions) (int64, error) {
	return coll.Native().CountDocuments(context.TODO(), filter, opts...)
}

// Exists - Check if document exists
func (coll *Model[T]) Exists(filter interface{}) (bool, error) {
	var res bson.M

	// Project only ID so that mongodb doesn't have to read disk.
	// only relevant if query is ID
	err := coll.FindOneAs(&res, filter, options.FindOne().SetProjection(bson.M{"_id": 1}))
	if err != nil {
		if IsNoDocumentsError(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// CountAggregate - Count documents in database using an aggregation pipeline
func (coll *Model[T]) CountAggregate(pipeline []interface{}, opts ...*options.AggregateOptions) (int64, error) {
	// Append a $count stage to the pipeline
	countPipeline := append(pipeline, bson.D{{"$count", "count"}})

	// Run the aggregation
	cursor, err := coll.Native().Aggregate(context.TODO(), countPipeline, opts...)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(context.TODO())

	// Get the result
	var result struct {
		Count int64 `bson:"count"`
	}
	if cursor.Next(context.TODO()) {
		err = cursor.Decode(&result)
		if err != nil {
			return 0, err
		}
		return result.Count, nil
	}

	// If no documents matched, return 0
	if err := cursor.Err(); err != nil {
		return 0, err
	}
	return 0, nil
}

// ProjectPublicFields - Project public fields
func (coll *Model[T]) ProjectPublicFields() bson.M {
	return Projection.OmitIdAndPick(coll.PublicFields)
}

// ProjectPublicFieldsAnd - Project public fields including some keys
func (coll *Model[T]) ProjectPublicFieldsAnd(keys []string) bson.M {
	return Projection.OmitIdAndPick(append(coll.PublicFields, keys...))
}

// ProjectPublicFieldsWithout - Project public fields excluding some keys
func (coll *Model[T]) ProjectPublicFieldsWithout(keys []string) bson.M {
	var newKeys []string

	for _, key := range coll.PublicFields {
		if arrutil.Contains(keys, key) {
			continue
		}
		newKeys = append(newKeys, key)
	}

	return Projection.OmitIdAndPick(newKeys)
}

// GetPublicFields - Get public fields
func (coll *Model[T]) GetPublicFields(model ModelData) bson.M {
	modelMap := structToMapWithTags(model, "bson")
	return lo.PickByKeys(modelMap, coll.PublicFields)
}

// GetPublicFieldsAnd - Get public fields
func (coll *Model[T]) GetPublicFieldsAnd(model ModelData, interceptor func(data bson.M) bson.M) bson.M {
	return interceptor(coll.GetPublicFields(model))
}

// Helpers - get model helper
func (coll *Model[T]) Helpers(model T) *ModelHelper[T] {
	return GetModelHelper(coll, &model)
}

// Aggregate - Aggregate
func (coll *Model[T]) Aggregate(pipeline interface{}, opts ...*options.AggregateOptions) ([]bson.M, error) {
	var results = make([]bson.M, 0)
	cursor, err := coll.Native().Aggregate(context.TODO(), pipeline, opts...)
	if err != nil {
		return results, err
	}

	if err = cursor.All(context.TODO(), &results); err != nil {
		return results, err
	}

	return results, nil
}

// Find - Find documents
func (coll *Model[T]) Find(filter interface{}, opts ...*options.FindOptions) ([]T, error) {
	var results = make([]T, 0)
	cursor, err := coll.Native().Find(context.TODO(), filter, opts...)
	if err != nil {
		return results, err
	}

	if err = cursor.All(context.TODO(), &results); err != nil {
		return results, err
	}

	return results, nil
}

// FindAs - Find documents and decode it into a different struct
func (coll *Model[T]) FindAs(result interface{}, filter interface{}, opts ...*options.FindOptions) error {
	cursor, err := coll.Native().Find(context.TODO(), filter, opts...)
	if err != nil {
		return err
	}

	if err = cursor.All(context.TODO(), result); err != nil {
		return err
	}

	return nil
}

// FindOneAsHelper - Find one document and decode it into the same struct
func (coll *Model[T]) FindOneAsHelper(filter interface{}, opts ...*options.FindOneOptions) (*ModelHelper[T], error) {
	result, err := coll.FindOne(filter, opts...)

	if err != nil {
		return nil, err
	}

	return GetModelHelper(coll, &result), nil
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
//	sum, _ := UserModel.SumMany(int(0), []string{"credit", "debit"})
//	// sum will be {credit: 300, debit: 700}
func (coll *Model[T]) SumMany(resType interface{}, keys []string, filter interface{}) (bson.M, error) {
	switch resType.(type) {
	case int:
		return SumMany[int](coll, keys, filter)
	case int32:
		return SumMany[int32](coll, keys, filter)
	case int64:
		return SumMany[int64](coll, keys, filter)
	case float32:
		return SumMany[float32](coll, keys, filter)
	case float64:
		return SumMany[float64](coll, keys, filter)
	default:
		return bson.M{}, fmt.Errorf("unsupported type")
	}
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
// sum, _ := UserModel.Sum(float64(0), "credit", nil)
// // sum will be 300
func (coll *Model[T]) Sum(key string, filter interface{}) (int, error) {
	return Sum[int](coll, key, filter)
}

// SumFloat - Sum documents and return float
//
// Same as Sum but returns float
func (coll *Model[T]) SumFloat(key string, filter interface{}) (float64, error) {
	return Sum[float64](coll, key, filter)
}
