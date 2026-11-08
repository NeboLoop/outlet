-- +goose Up
-- Platform roles (users.role) are super_admin and agent, the role of everyone
-- else: what they may do is their role in each brand. admin and manager meant
-- nothing more than agent.
UPDATE users SET role = 'agent' WHERE role IN ('admin', 'manager');

-- A webhook's events are stored comma-separated, the way the dispatcher reads
-- them. The admin app stored a JSON array, so those webhooks never matched.
UPDATE webhooks
SET events = COALESCE((SELECT group_concat(value, ',') FROM json_each(webhooks.events)), '')
WHERE json_valid(events) AND json_type(events) = 'array';

-- +goose Down
SELECT 1;
