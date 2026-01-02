-- name: GetFeed :one
SELECT
    url,
    title,
    last_processed_at
FROM
    feed
WHERE
    url = ?;

-- name: UpdateLastProcessedAt :exec
INSERT
    OR REPLACE INTO feed (url, title, last_processed_at)
VALUES
    (?, ?, ?);

-- name: SaveGenerationHistory :exec
INSERT INTO
    generation_history (feed_url, last_processed_at, created_at)
VALUES
    (?, ?, ?);

-- name: GetLatestGenerationTimestamp :one
SELECT
    last_processed_at
FROM
    generation_history
WHERE
    feed_url = ?
ORDER BY
    created_at DESC
LIMIT
    1;

-- name: DeleteLatestGeneration :exec
DELETE FROM
    generation_history
WHERE
    created_at = (
        SELECT
            created_at
        FROM
            generation_history
        ORDER BY
            created_at DESC
        LIMIT
            1
    );
