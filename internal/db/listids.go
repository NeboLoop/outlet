package db

import (
	"encoding/json"
	"strconv"
	"strings"
)

// Campaign list_ids and exclude_list_ids are stored as a JSON array of list
// IDs, e.g. ["1","4"]. EncodeListIDs writes that form; DecodeListIDs reads it
// and also comma-separated values, which older MCP tool calls stored.

// EncodeListIDs returns the stored form of a set of list IDs.
func EncodeListIDs(ids []string) string {
	clean := []string{}
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" {
			clean = append(clean, id)
		}
	}
	b, _ := json.Marshal(clean)
	return string(b)
}

// DecodeListIDs returns the numeric list IDs in a stored value.
func DecodeListIDs(stored string) []int64 {
	stored = strings.TrimSpace(stored)
	if stored == "" {
		return nil
	}
	var parts []string
	if strings.HasPrefix(stored, "[") {
		var raw []any
		if err := json.Unmarshal([]byte(stored), &raw); err != nil {
			return nil
		}
		for _, v := range raw {
			switch t := v.(type) {
			case string:
				parts = append(parts, t)
			case float64:
				parts = append(parts, strconv.FormatInt(int64(t), 10))
			}
		}
	} else {
		parts = strings.Split(stored, ",")
	}
	var ids []int64
	for _, p := range parts {
		if id, err := strconv.ParseInt(strings.TrimSpace(p), 10, 64); err == nil && id > 0 {
			ids = append(ids, id)
		}
	}
	return ids
}
