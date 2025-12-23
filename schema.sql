-- Main feed tracking table
CREATE TABLE IF NOT EXISTS feed (
    url TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    last_processed_at INTEGER NOT NULL
);

-- Parser cache: stores parsed content by URL and parser type
CREATE TABLE IF NOT EXISTS parser_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    url TEXT NOT NULL,
    parser_type TEXT NOT NULL,
    output_data TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    accessed_at INTEGER NOT NULL,
    UNIQUE(url, parser_type)
);

CREATE INDEX IF NOT EXISTS idx_parser_cache_lookup ON parser_cache(url, parser_type);

CREATE INDEX IF NOT EXISTS idx_parser_cache_accessed ON parser_cache(accessed_at);

-- Agent cache: stores final agent pipeline output by URL, parser type, and agent pipeline
CREATE TABLE IF NOT EXISTS agent_cache (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    url TEXT NOT NULL,
    parser_type TEXT NOT NULL,
    agent_pipeline TEXT NOT NULL,
    output_data TEXT NOT NULL,
    created_at INTEGER NOT NULL,
    accessed_at INTEGER NOT NULL,
    UNIQUE(url, parser_type, agent_pipeline)
);

CREATE INDEX IF NOT EXISTS idx_agent_cache_lookup ON agent_cache(url, parser_type, agent_pipeline);

CREATE INDEX IF NOT EXISTS idx_agent_cache_accessed ON agent_cache(accessed_at);
