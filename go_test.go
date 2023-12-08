package gmongo

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_RemoveStringFromStringIfExists(t *testing.T) {
	// arrange
	str := "hello world"
	remove := "world"

	// act
	res := removeStringFromStringIfExists(str, remove)

	assert.Equal(t, "hello ", res)
}

func Test_StructToMapWithTags(t *testing.T) {
	type Child struct {
		Phone string `bson:"phone"`
		Email string `bson:"email"`
	}
	// arrange
	type TestStruct struct {
		Name     string `bson:"name" json:"Name"`
		Age      int    `bson:"age" json:"Age"`
		Verified bool   `bson:"verified" json:"Verified"`
		Child    Child  `bson:"contact" json:"Contact"`
	}

	testStruct := TestStruct{
		Name:     "John",
		Age:      20,
		Verified: true,
		Child: Child{
			Phone: "123456789",
			Email: "app@example.com",
		},
	}

	t.Run("With Bson Tag", func(t *testing.T) {
		bsonRes := structToMapWithTags(testStruct, "bson")
		assert.Equal(t, map[string]interface{}{
			"name":     testStruct.Name,
			"age":      testStruct.Age,
			"verified": testStruct.Verified,
			"contact":  structToMapWithTags(testStruct.Child, "bson"),
		}, bsonRes)
	})

	t.Run("With Json Tag", func(t *testing.T) {
		jsonRes := structToMapWithTags(testStruct, "json")
		assert.Equal(t, map[string]interface{}{
			"Name":     testStruct.Name,
			"Age":      testStruct.Age,
			"Verified": testStruct.Verified,
			"Contact":  structToMapWithTags(testStruct.Child, "json"),
		}, jsonRes)
	})
}
