-- name: GetFeed :one
SELECT url, title, last_processed_at
FROM feed
WHERE url = ?;

-- name: UpdateLastProcessedAt :exec
INSERT OR REPLACE INTO feed (url, title, last_processed_at)
VALUES (?, ?, ?);
