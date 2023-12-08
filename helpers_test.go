package gmongo

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"testing"
	"time"
)

// Test `DateTimeNow` function
func Test_DateTimeNow(t *testing.T) {
	assert.Equal(t, DateTimeNow(), primitive.NewDateTimeFromTime(time.Now()))
}

// Test `NewId` function
func Test_NewId(t *testing.T) {
	id := NewId()

	// Check if the ID is valid
	assert.True(t, primitive.IsValidObjectID(id.Hex()))
	assert.False(t, id.IsZero())
}

// Test `NewUUid` function
func Test_NewUUid(t *testing.T) {
	id := NewUUid()

	// Check if the UUID is valid
	assert.True(t, len(id) == 36)
}

// Test `IsNoDocumentsError` function
func Test_IsNoDocumentsError(t *testing.T) {
	assert.False(t, IsNoDocumentsError(nil))
	assert.True(t, IsNoDocumentsError(mongo.ErrNoDocuments))
}

// Test `IsFindOneError` function
func Test_IsFindOneError(t *testing.T) {
	assert.False(t, IsFindOneError(nil))
	assert.False(t, IsFindOneError(mongo.ErrNoDocuments))
	assert.True(t, IsFindOneError(errors.New("some error")))
}
