-- +goose Up
-- A list's export reads its subscribers in pages, in the order they joined
-- (internal/services/listexport).
CREATE INDEX IF NOT EXISTS idx_list_subscribers_export ON list_subscribers(list_id, COALESCE(subscribed_at, ''), id);

-- +goose Down
DROP INDEX IF EXISTS idx_list_subscribers_export;
