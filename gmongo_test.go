package gmongo

import (
	"context"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
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

func (a *User) GetID() primitive.ObjectID {
	return a.ID
}

func TestModel(t *testing.T) {
	client := testConnectToDb()

	// create model
	UserModel := MakeModel[*User](client.Database, "users")
	if UserModel.Native() == nil {
		t.Error("Model is nil")
	}

	UserModel.PublicFields = []string{"name"}

	var CreateTestUser = func() User {
		// delete all users
		_, err := UserModel.Native().DeleteMany(context.TODO(), bson.M{})
		if err != nil {
			t.Error(err)
		}

		// create user
		newUser := User{
			ID:   NewId(),
			Name: "John",
			Age:  20,
		}

		_, err = UserModel.Native().InsertOne(context.TODO(), newUser)
		if err != nil {
			t.Error(err)
		}

		return newUser
	}

	newUser := CreateTestUser()
	var newUserMap bson.M = structToMapWithTags(newUser, "bson")

	// Test `FindOne`
	t.Run("Find One", func(t *testing.T) {
		user, err := UserModel.FindOneById(newUser.ID)
		if err != nil {
			t.Error(err)
		}

		assert.EqualValues(t, UserModel.ToBsonMap(user), newUserMap)

	})

	// Test `FindOneAs`
	t.Run("Find One As", func(t *testing.T) {
		type UserResult struct {
			Age int `bson:"age"`
		}

		var userResult UserResult
		err := UserModel.FindOneAs(
			&userResult,
			bson.M{"_id": newUser.ID},
			options.FindOne().SetProjection(Projection.OmitIdAndPick([]string{"age"})),
		)

		if err != nil {
			t.Error(err)
		}

		userResultMap := structToMapWithTags(userResult, "bson")

		assert.NotEqualValues(t, userResultMap, newUserMap)
		assert.EqualValues(t, userResultMap, bson.M{"age": 20})
	})

	// Test `DeleteOne`
	t.Run("Delete One", func(t *testing.T) {
		// delete user
		deleted, err := UserModel.DeleteOne(bson.M{"_id": newUser.ID})
		if err != nil {
			t.Error(err)
		}

		assert.EqualValues(t, deleted.DeletedCount, 1)

		// check if user is deleted
		_, err = UserModel.FindOne(bson.M{"_id": newUser.ID})
		assert.True(t, IsNoDocumentsError(err))

		// recreate user
		newUser = CreateTestUser()
	})

	// Test `UpdateOne`
	t.Run("Update One", func(t *testing.T) {
		// update user
		updated, err := UserModel.UpdateOne(
			bson.M{"_id": newUser.ID},
			bson.M{"$set": bson.M{"name": "Jack"}},
		)

		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, updated.ModifiedCount, int64(1))

		// check if user is updated
		updatedUser, err := UserModel.FindOne(bson.M{"_id": newUser.ID})
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, updatedUser.Name, "Jack")

		// recreate test user
		newUser = CreateTestUser()
	})

	// Test `FindOneAsHelper`
	t.Run("Find One As Helper", func(t *testing.T) {
		newUser = CreateTestUser()
		user, err := UserModel.FindOneAsHelper(bson.M{"_id": newUser.ID})

		if err != nil {
			t.Error(err)
		}

		// user must be type of ModelHelper
		assert.IsType(t, ModelHelper[*User]{}, user)

		// Test `GetPublicFields`
		t.Run("Get Public Fields", func(t *testing.T) {
			publicFields := user.GetPublicFields()
			assert.EqualValues(t, publicFields, bson.M{"name": "John"})
		})

		// Test `GetID`
		t.Run("Get ID", func(t *testing.T) {
			assert.Equal(t, user.GetID(), newUser.ID)
		})

		// Test `UpdateRaw`
		t.Run("Update Raw", func(t *testing.T) {
			updated, err := user.UpdateRaw(bson.M{"$set": bson.M{"name": "Jack"}})
			if err != nil {
				t.Error(err)
			}

			assert.EqualValues(t, updated.ModifiedCount, 1)

			// check if user is updated
			updatedUser, err := UserModel.FindOne(bson.M{"_id": newUser.ID})
			if err != nil {
				t.Error(err)
			}

			assert.EqualValues(t, updatedUser.Name, "Jack")
		})

		// Test `Update`
		t.Run("Update", func(t *testing.T) {
			updated, err := user.Update(bson.M{"name": "Jude"})
			if err != nil {
				t.Error(err)
			}

			assert.EqualValues(t, updated.ModifiedCount, 1)

			// check if user is updated
			updatedUser, err := UserModel.FindOne(bson.M{"_id": newUser.ID})
			if err != nil {
				t.Error(err)
			}

			assert.EqualValues(t, updatedUser.Name, "Jude")
		})

		// Test `Delete`
		t.Run("Delete", func(t *testing.T) {
			deleted, err := user.Delete()
			if err != nil {
				t.Error(err)
			}

			assert.EqualValues(t, deleted.DeletedCount, 1)

			// check if user is deleted
			_, err = UserModel.FindOne(bson.M{"_id": newUser.ID})
			assert.True(t, IsNoDocumentsError(err))
		})
	})
}
