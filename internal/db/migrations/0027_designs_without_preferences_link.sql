-- +goose Up
-- The starter Branded design linked to {{preferences_url}}, a page Outlet
-- doesn't have: the tag went out unfilled. Its link comes out of the designs
-- that still hold it as seeded.
UPDATE email_designs
SET html_body = REPLACE(html_body, ' ·
        <a href="{{preferences_url}}" style="color: #6b7280;">Email Preferences</a>', '')
WHERE html_body LIKE '%{{preferences_url}}%';

-- +goose Down
SELECT 1;
