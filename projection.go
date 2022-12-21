package gmongo

import (
	"github.com/samber/lo"
	"go.mongodb.org/mongo-driver/bson"
)

/**
 * Convert array of keys to object of keys and value.
 */
func keysToMap[T any](keys []string, value T) map[string]T {
	var m = make(map[string]T)
	for _, key := range keys {
		m[key] = value
	}
	return m
}

// OmitKeys - Omit fields from the result.
func omitKeys(keys []string) map[string]any {
	return keysToMap[any](keys, 0)
}

// PickKeys - Pick fields from the result.
func pickKeys(keys []string) map[string]any {
	return keysToMap[any](keys, 1)
}

// OmitIdAnd - Omit fields from the result including _id.
func omitIdAnd(keys []string) map[string]any {
	var KeysToMap = omitKeys(keys)
	KeysToMap["_id"] = 0
	return KeysToMap
}

// OmitIdAndPick - Omit _id and pick fields.
func omitIdAndPick(keys []string) map[string]any {
	var KeysToMap = pickKeys(keys)
	KeysToMap["_id"] = 0
	return KeysToMap
}

// In order to avoid polluting the xmongo package with a lot of functions
// We group them into a struct

var Projection = struct {
	OmitKeys      func(keys []string) map[string]any
	PickKeys      func(keys []string) map[string]any
	OmitIdAnd     func(keys []string) map[string]any
	OmitIdAndPick func(keys []string) map[string]any
}{
	OmitKeys:      omitKeys,
	PickKeys:      pickKeys,
	OmitIdAnd:     omitIdAnd,
	OmitIdAndPick: omitIdAndPick,
}

func (coll *Model[T]) ToBsonMap(data ModelData) bson.M {
	return structToMapWithTags(data, "bson")
}

func (coll *Model[T]) ToJsonMap(data ModelData) bson.M {
	return structToMapWithTags(data, "json")
}

func (coll *Model[T]) Pick(data ModelData, keys []string) bson.M {
	return lo.PickByKeys(coll.ToBsonMap(data), keys)
}
