package db

import (
	"reflect"
	"testing"
)

func TestListIDs(t *testing.T) {
	for stored, want := range map[string][]int64{
		`["1","4"]`: {1, 4},
		`[1, 4]`:    {1, 4},
		`1, 4`:      {1, 4},
		`4`:         {4},
		``:          nil,
		`[]`:        nil,
		`["x"]`:     nil,
	} {
		if got := DecodeListIDs(stored); !reflect.DeepEqual(got, want) {
			t.Errorf("%q: got %v, want %v", stored, got, want)
		}
	}
	if got := EncodeListIDs([]string{" 1", "", "4"}); got != `["1","4"]` {
		t.Errorf("encode = %s", got)
	}
}
