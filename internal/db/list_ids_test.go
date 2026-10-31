package db

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListIDs_RoundTrip(t *testing.T) {
	assert.Equal(t, `["12","14"]`, EncodeListIDs([]int64{12, 14}))
	assert.Equal(t, `[]`, EncodeListIDs(nil))
	assert.Equal(t, []int64{12, 14}, DecodeListIDs(EncodeListIDs([]int64{12, 14})))
}

// Only the stored form reads as lists. The old comma-separated form is
// rewritten by migration 0012.
func TestDecodeListIDs_OnlyTheStoredForm(t *testing.T) {
	assert.Empty(t, DecodeListIDs(""))
	assert.Empty(t, DecodeListIDs("[]"))
	assert.Empty(t, DecodeListIDs("1,2"))
	assert.Equal(t, []int64{1, 2}, DecodeListIDs(`["0","1","x","2"]`))
}
