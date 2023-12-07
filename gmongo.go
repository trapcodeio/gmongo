package gmongo

import (
	"context"
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

type Model[T ModelData] struct {
	Native       func() *mongo.Collection
	PublicFields []string
}

func MakeModel[T ModelData](db *mongo.Database, collectionName string) Model[T] {
	// check if collection name is empty
	if collectionName == "" {
		panic("Collection name is empty")
	}

	collection := db.Collection(collectionName)

	return Model[T]{
		Native: func() *mongo.Collection {
			return collection
		}}
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

// CountAggregate - Count aggregate
func (coll *Model[T]) CountAggregate(pipeline interface{}, opts ...*options.AggregateOptions) (int, error) {
	cursor, err := coll.Native().Aggregate(context.TODO(), pipeline, opts...)
	if err != nil {
		return 0, err
	}

	return cursor.RemainingBatchLength(), nil
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
func (coll *Model[T]) Helpers(model T) ModelHelper[T] {
	return GetModelHelper(*coll, model)
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
func (coll *Model[T]) FindOneAsHelper(filter interface{}, opts ...*options.FindOneOptions) (ModelHelper[T], error) {
	result, err := coll.FindOne(filter, opts...)

	if err != nil {
		return ModelHelper[T]{}, err
	}

	return GetModelHelper(*coll, result), nil
}
