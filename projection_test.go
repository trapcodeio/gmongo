package gmongo

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// ==============================================
// ========== TEST PRIVATE FUNCTIONS ============
// ==============================================

func Test_keysToMap(t *testing.T) {
	keys := []string{"name", "email"}
	Map := keysToMap[any](keys, 0)

	assert.Equal(t, Map, map[string]any{"name": 0, "email": 0})

	Map = keysToMap[any](keys, 1)
	assert.Equal(t, Map, map[string]any{"name": 1, "email": 1})
}

func Test_omitKeys(t *testing.T) {
	keys := []string{"name", "email"}
	Map := omitKeys(keys)

	assert.Equal(t, Map, map[string]any{"name": 0, "email": 0})
}

func Test_pickKeys(t *testing.T) {
	keys := []string{"name", "email"}
	Map := pickKeys(keys)

	assert.Equal(t, Map, map[string]any{"name": 1, "email": 1})
}

func Test_omitIdAnd(t *testing.T) {
	keys := []string{"name", "email"}
	Map := omitIdAnd(keys)

	assert.Equal(t, Map, map[string]any{"_id": 0, "name": 0, "email": 0})
}

func Test_omitIdAndPick(t *testing.T) {
	keys := []string{"name", "email"}
	Map := omitIdAndPick(keys)

	assert.Equal(t, Map, map[string]any{"_id": 0, "name": 1, "email": 1})
}
