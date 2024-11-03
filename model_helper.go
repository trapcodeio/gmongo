package gmongo

import (
	"context"
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// ModelHelper - A helper for models, this struct includes all the functions for a model instance
type ModelHelper[T ModelData] struct {
	Data  *T
	Model *Model[T]
}

// GetModelHelper - Get a model helper for a model instance
func GetModelHelper[T ModelData](model *Model[T], data *T) *ModelHelper[T] {
	return &ModelHelper[T]{
		Data:  data,
		Model: model,
	}
}

// GetPublicFields - Get the public fields of a model instance
func (m ModelHelper[T]) GetPublicFields() bson.M {
	modelMap := structToMapWithTags(*m.Data, "bson")
	return lo.PickByKeys(modelMap, m.Model.PublicFields)
}

// GetID - Get the ID of a model instance
func (m ModelHelper[T]) GetID() primitive.ObjectID {
	return (*m.Data).GetID()
}

// UpdateRaw - Update a model instance with raw data
func (m ModelHelper[T]) UpdateRaw(update bson.M) (*mongo.UpdateResult, error) {
	return m.Model.Native().UpdateOne(context.TODO(), bson.M{"_id": m.GetID()}, update)
}

// Update - Update a model instance
func (m ModelHelper[T]) Update(set bson.M) (*mongo.UpdateResult, error) {
	return m.UpdateRaw(bson.M{"$set": set})
}

// Delete - Delete a model instance
func (m ModelHelper[T]) Delete() (*mongo.DeleteResult, error) {
	return m.Model.Native().DeleteOne(context.TODO(), bson.M{"_id": m.GetID()})
}
