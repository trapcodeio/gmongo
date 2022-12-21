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
	// IsModel - A function that returns true if the struct is a model
	IsModel() bool
}

type Model[T ModelData] struct {
	Native       func() *mongo.Collection
	PublicFields []string
}

type DefaultModel struct {
	ID primitive.ObjectID `bson:"_id"`
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

// DeleteOne Delete - Delete model from database
func (coll *Model[T]) DeleteOne(filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	return coll.Native().DeleteOne(context.TODO(), filter, opts...)
}

// UpdateOne - Update model in database
func (coll *Model[T]) UpdateOne(filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	return coll.Native().UpdateOne(context.TODO(), filter, update, opts...)
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

// Update
