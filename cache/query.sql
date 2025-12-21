-- Parser Cache Queries

-- name: GetParserOutput :one
SELECT output_data
FROM parser_cache
WHERE url = ? AND parser_type = ?;

-- name: SetParserOutput :exec
INSERT OR REPLACE INTO parser_cache
(url, parser_type, output_data, created_at, accessed_at)
VALUES (?, ?, ?, ?, ?);

-- name: UpdateParserAccessTime :exec
UPDATE parser_cache
SET accessed_at = ?
WHERE url = ? AND parser_type = ?;

-- name: DeleteParserCache :exec
DELETE FROM parser_cache;

-- name: CountParserEntries :one
SELECT COUNT(*) FROM parser_cache;


-- Agent Cache Queries

-- name: GetAgentOutput :one
SELECT output_data
FROM agent_cache
WHERE url = ? AND parser_type = ? AND agent_pipeline = ?;

-- name: SetAgentOutput :exec
INSERT OR REPLACE INTO agent_cache
(url, parser_type, agent_pipeline, output_data, created_at, accessed_at)
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateAgentAccessTime :exec
UPDATE agent_cache
SET accessed_at = ?
WHERE url = ? AND parser_type = ? AND agent_pipeline = ?;

-- name: DeleteAgentCache :exec
DELETE FROM agent_cache;

-- name: CountAgentEntries :one
SELECT COUNT(*) FROM agent_cache;


-- Statistics Queries

-- name: GetOldestCacheEntry :one
SELECT MIN(created_at) as oldest
FROM (
    SELECT created_at FROM parser_cache
    UNION ALL
    SELECT created_at FROM agent_cache
);
