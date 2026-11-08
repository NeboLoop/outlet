-- +goose Up
-- The MCP tool told AI assistants to save list rules as list_subscribe, a
-- trigger nothing runs: entry rules for a list are list_join.
UPDATE sequence_entry_rules SET trigger_type = 'list_join' WHERE trigger_type = 'list_subscribe';

-- +goose Down
SELECT 1;
