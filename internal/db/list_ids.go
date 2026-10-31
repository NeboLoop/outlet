package db

import (
	"encoding/json"
	"strconv"
)

// A campaign keeps its target and excluded lists (list_ids and
// exclude_list_ids) as a JSON array of list IDs, like ["12","14"].
// EncodeListIDs and DecodeListIDs are the one way to write and read it.

// EncodeListIDs writes list IDs in the stored form.
func EncodeListIDs(ids []int64) string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, strconv.FormatInt(id, 10))
	}
	b, _ := json.Marshal(out)
	return string(b)
}

// DecodeListIDs reads list IDs in the stored form. Anything else reads as
// no lists.
func DecodeListIDs(s string) []int64 {
	var raw []string
	if s == "" || json.Unmarshal([]byte(s), &raw) != nil {
		return nil
	}
	ids := make([]int64, 0, len(raw))
	for _, r := range raw {
		if id, err := strconv.ParseInt(r, 10, 64); err == nil && id > 0 {
			ids = append(ids, id)
		}
	}
	return ids
}
