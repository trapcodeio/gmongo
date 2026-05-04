package gmongo

import (
	"testing"

	"github.com/stretchr/testify/assert"
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

func Test_StructToMapWithTags_Inline(t *testing.T) {
	type Inner struct {
		ID string `bson:"_id"`
	}

	t.Run("value-type embedded inline", func(t *testing.T) {
		type Outer struct {
			Inner `bson:",inline"`
			Name  string `bson:"name"`
		}
		got := structToMapWithTags(Outer{Inner: Inner{ID: "abc"}, Name: "n"}, "bson")
		assert.Equal(t, map[string]interface{}{"_id": "abc", "name": "n"}, got)
	})

	t.Run("non-nil pointer embedded inline", func(t *testing.T) {
		type Outer struct {
			*Inner `bson:",inline"`
			Name   string `bson:"name"`
		}
		got := structToMapWithTags(Outer{Inner: &Inner{ID: "abc"}, Name: "n"}, "bson")
		assert.Equal(t, map[string]interface{}{"_id": "abc", "name": "n"}, got)
	})

	t.Run("nil pointer embedded inline omitted", func(t *testing.T) {
		type Outer struct {
			*Inner `bson:",inline"`
			Name   string `bson:"name"`
		}
		got := structToMapWithTags(Outer{Name: "n"}, "bson")
		assert.Equal(t, map[string]interface{}{"name": "n"}, got)
	})

	t.Run("named non-anonymous inline field", func(t *testing.T) {
		type Outer struct {
			Sub  Inner  `bson:",inline"`
			Name string `bson:"name"`
		}
		got := structToMapWithTags(Outer{Sub: Inner{ID: "abc"}, Name: "n"}, "bson")
		assert.Equal(t, map[string]interface{}{"_id": "abc", "name": "n"}, got)
	})

	t.Run("inline+omitempty on nil pointer omitted", func(t *testing.T) {
		type Outer struct {
			*Inner `bson:",inline,omitempty"`
			Name   string `bson:"name"`
		}
		got := structToMapWithTags(Outer{Name: "n"}, "bson")
		assert.Equal(t, map[string]interface{}{"name": "n"}, got)
	})

	t.Run("inline on non-struct field silently skipped", func(t *testing.T) {
		type Outer struct {
			Foo  string `bson:",inline"`
			Name string `bson:"name"`
		}
		got := structToMapWithTags(Outer{Foo: "x", Name: "n"}, "bson")
		assert.Equal(t, map[string]interface{}{"name": "n"}, got)
	})

	t.Run("multiple inlines flatten without collision", func(t *testing.T) {
		type A struct {
			ID string `bson:"_id"`
		}
		type B struct {
			CreatedAt int64 `bson:"createdAt"`
		}
		type C struct {
			UpdatedAt int64 `bson:"updatedAt"`
		}
		type Outer struct {
			A    `bson:",inline"`
			B    `bson:",inline"`
			*C   `bson:",inline"`
			Name string `bson:"name"`
		}
		got := structToMapWithTags(Outer{
			A:    A{ID: "abc"},
			B:    B{CreatedAt: 1000},
			C:    &C{UpdatedAt: 2000},
			Name: "example",
		}, "bson")
		assert.Equal(t, map[string]interface{}{
			"_id":       "abc",
			"createdAt": int64(1000),
			"updatedAt": int64(2000),
			"name":      "example",
		}, got)
	})
}
