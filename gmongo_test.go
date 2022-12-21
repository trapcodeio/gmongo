package gmongo

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"testing"
)

/**
================== Define Model ==================
*/

type User struct {
	ID   primitive.ObjectID `bson:"_id"`
	Name string             `bson:"name"`
	Age  int                `bson:"age"`
}

func (a *User) IsModel() bool {
	return true
}

func TestModel(t *testing.T) {
	client := test_ConnectToDb()

	// create model
	UserModel := MakeModel[*User](client.Database, "users")
	if UserModel.Native() == nil {
		t.Error("Model is nil")
	}

	// delete all users
	_, err := UserModel.Native().DeleteMany(context.TODO(), bson.M{})
	if err != nil {
		t.Error(err)
	}

	userId := NewId()

	// Test add data
	t.Run("Test Add Data", func(t *testing.T) {
		user := User{
			ID:   userId,
			Name: "John",
			Age:  20,
		}

		inserted, err := UserModel.Native().InsertOne(context.TODO(), user)
		if err != nil {
			t.Error(err)
		}

		t.Logf("Inserted ID: %v", inserted)

		// Test find one
		t.Run("Find One", func(t *testing.T) {
			user, err := UserModel.FindOne(bson.M{"_id": userId})
			if err != nil {
				t.Error(err)
			}

			t.Logf("Found user: %v", user.Name)
		})
	})

}
