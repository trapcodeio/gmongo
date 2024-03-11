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
	ID       primitive.ObjectID `bson:"_id"`
	Name     string             `bson:"name"`
	Age      int                `bson:"age"`
	Verified bool               `bson:"verified"`
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

	UserModel.PublicFields = []string{"name", "verified"}

	var DeleteAllUsers = func() {
		// delete all users
		_, err := UserModel.Native().DeleteMany(context.TODO(), bson.M{})
		if err != nil {
			t.Error(err)
		}
	}

	var CreateTestUserWithoutDeletingOldUsers = func() User {

		// create user
		newUser := User{
			ID:       NewId(),
			Name:     "John",
			Age:      20,
			Verified: true,
		}

		_, err := UserModel.Native().InsertOne(context.TODO(), newUser)
		if err != nil {
			t.Error(err)
		}

		return newUser
	}

	var CreateTestUser = func() User {
		DeleteAllUsers()
		return CreateTestUserWithoutDeletingOldUsers()
	}

	newUser := CreateTestUser()
	var newUserMap bson.M = structToMapWithTags(newUser, "bson")

	var CreateMultipleTestUsers = func(length int) {
		newUser = CreateTestUser()
		for i := 0; i < (length - 1); i++ {
			CreateTestUserWithoutDeletingOldUsers()
		}
	}

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

	// Test `Count`
	t.Run("Count", func(t *testing.T) {
		count, err := UserModel.Count(bson.M{})
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, count, int64(1))

		CreateTestUserWithoutDeletingOldUsers()
		CreateTestUserWithoutDeletingOldUsers()

		// new count must be 3
		count, err = UserModel.Count(bson.M{})
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, count, int64(3))
	})

	// Test `CountAggregate`
	t.Run("Count Aggregate", func(t *testing.T) {
		newUser = CreateTestUser()

		count, err := UserModel.CountAggregate(bson.A{})
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, count, 1)

		CreateTestUserWithoutDeletingOldUsers()
		CreateTestUserWithoutDeletingOldUsers()
		CreateTestUserWithoutDeletingOldUsers()

		// new count must be 5
		count, err = UserModel.CountAggregate(bson.A{})
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, count, 4)
	})

	// Test `Exists`
	t.Run("Exists", func(t *testing.T) {
		exists, err := UserModel.Exists(bson.M{"_id": newUser.ID})
		if err != nil {
			t.Error(err)
		}

		assert.True(t, exists)

		exists, err = UserModel.Exists(bson.M{"_id": NewId()})
		if err != nil {
			t.Error(err)
		}

		assert.False(t, exists)
	})

	// Test `ProjectPublicFields`
	t.Run("Project Public Fields", func(t *testing.T) {
		projection := UserModel.ProjectPublicFields()
		assert.EqualValues(t, projection, bson.M{"_id": 0, "name": 1, "verified": 1})
	})

	// Test `ProjectPublicFieldsAnd`
	t.Run("Project Public Fields And", func(t *testing.T) {
		projection := UserModel.ProjectPublicFieldsAnd([]string{"age"})
		assert.EqualValues(t, projection, bson.M{"_id": 0, "name": 1, "verified": 1, "age": 1})
	})

	// Test `ProjectPublicFieldsWithout`
	t.Run("Project Public Fields Without", func(t *testing.T) {
		projection := UserModel.ProjectPublicFieldsWithout([]string{"verified"})
		assert.EqualValues(t, projection, bson.M{"_id": 0, "name": 1})
	})

	// Test `GetPublicFields`
	t.Run("Get Public Fields", func(t *testing.T) {
		publicFields := UserModel.GetPublicFields(&newUser)
		assert.EqualValues(t, publicFields, bson.M{"name": "John", "verified": true})
	})

	// Test `GetPublicFieldsAnd`
	t.Run("Get Public Fields And", func(t *testing.T) {
		publicFields := UserModel.GetPublicFieldsAnd(&newUser, func(data bson.M) bson.M {
			data["age"] = 20
			return data
		})
		assert.EqualValues(t, publicFields, bson.M{"name": "John", "verified": true, "age": 20})
	})

	// Test `Helpers`
	t.Run("Helpers", func(t *testing.T) {
		userHelper := UserModel.Helpers(&newUser)
		assert.IsType(t, ModelHelper[*User]{}, userHelper)
	})

	// Test `Aggregate`
	t.Run("Aggregate", func(t *testing.T) {
		newUser = CreateTestUser()

		// aggregate
		aggregate, err := UserModel.Aggregate(bson.A{
			bson.M{"$match": bson.M{"_id": newUser.ID}},
			bson.M{"$project": Projection.OmitIdAndPick([]string{"name"})},
		})

		if err != nil {
			t.Error(err)
		}

		assert.EqualValues(t, aggregate[0], bson.M{"name": "John"})
	})

	// Test `Find`
	t.Run("Find", func(t *testing.T) {
		newUser = CreateTestUser()

		// find
		results, err := UserModel.Find(bson.M{"_id": newUser.ID})
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, results[0], &User{
			ID:       newUser.ID,
			Name:     "John",
			Age:      20,
			Verified: true,
		})
	})

	// Test `FindAs`
	t.Run("Find As", func(t *testing.T) {
		newUser = CreateTestUser()
		type NameOnly struct {
			Name string `bson:"name"`
		}

		// find
		var results []NameOnly
		err := UserModel.FindAs(
			&results,
			bson.M{"_id": newUser.ID},
			options.Find().SetProjection(Projection.OmitIdAndPick([]string{"name"})),
		)

		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, results[0], NameOnly{"John"})
	})

	// Test `Paginate`
	t.Run("Paginate", func(t *testing.T) {
		CreateMultipleTestUsers(10)

		// paginate
		paginated, err := UserModel.Paginate(1, 5, bson.M{})
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, paginated.Meta, PaginatedMeta{
			Total:    10,
			PerPage:  5,
			Page:     1,
			LastPage: 2,
		})
	})

	// Test `PaginateAggregate`
	t.Run("Paginate Aggregate", func(t *testing.T) {
		CreateMultipleTestUsers(10)

		// paginate
		paginated, err := UserModel.PaginateAggregate(1, 5, bson.A{})
		if err != nil {
			t.Error(err)
		}

		assert.Equal(t, paginated.Meta, PaginatedMeta{
			Total:    10,
			PerPage:  5,
			Page:     1,
			LastPage: 2,
		})
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
			assert.EqualValues(t, publicFields, bson.M{"name": "John", "verified": true})
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
