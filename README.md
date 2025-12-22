# myFeed

The goal of this project is to generate a personal news paper in a PDF based on the provided RSS feeds.

Configuration is made via TOML file where resources and corresponding parsers could be configured.

## Features

- **Multiple feed sources**: RSS feeds, Telegram channels, YouTube transcripts
- **Content parsing**: Extract readable content from web pages
- **AI-powered agents**: Post-process content with Gemini (summarization, etc.)
- **Smart caching**: SQLite-based cache for parsers and agents to speed up reruns
- **Flexible pipeline**: Fetch → Parse → Process → Render

## Supported resources

- [ ] Web page via [readability implementation in go](https://github.com/mackee/go-readability)
- [ ] Telegram channel via MTProto API
- [ ] Torrent files (PDF, CBR)

## Agents

Agents are AI-powered post-processors that transform content after parsing. They use Google's Gemini API via [genkit](https://github.com/naqerl/genkit) (fork with embedded dotprompt support).

### Available Agents

- **summary**: Summarizes content into concise markdown (3-5 paragraphs)

### Configuration

1. **Get Gemini API key**: Visit [Google AI Studio](https://ai.google.dev/) and create an API key

2. **Configure credentials** (`~/.config/myfeed/creds.toml`):
```toml
[gemini]
api_key = "your_gemini_api_key"
model = "gemini-2.0-flash-exp"  # or gemini-1.5-pro, gemini-1.5-flash
```

3. **Enable agents per resource** (`~/.config/myfeed/config.toml`):
```toml
[[resources]]
feed_url = "https://example.com/feed"
parser = "web"
type = "rss"
agents = ["summary"]  # Enable summarization
```

### Agent Chaining

Agents can be chained to apply multiple transformations:
```toml
agents = ["summary", "translate", "format"]  # Future: multiple agents in sequence
```

**Note**: The app will fail fast during startup if:
- Agents are configured but Gemini credentials are missing
- An unknown agent type is specified
- Embedded prompt files are missing

## Filters

Filters allow you to exclude unwanted content from your feed based on various criteria. They are defined in your config file and can be applied per resource.

### Available Filter Options

- **min_length**: Minimum character count (title + description combined)
- **min_words**: Minimum word count
- **exclude_patterns**: List of regex patterns to exclude matching items
- **require_paragraphs**: Require content to have multiple paragraphs/lines

### Filter Examples

```toml
[filters.short_posts]
min_length = 100        # Minimum character count (title + description)
min_words = 20          # Minimum word count

[filters.quality_content]
min_length = 200
min_words = 50
require_paragraphs = true

[filters.lwn_subscription]
exclude_patterns = [
    "Subscription required",
    "currently available to LWN subscribers only",
    "Please consider subscribing to LWN",
]

[filters.russian_announcements]
exclude_patterns = [
    "^[Дд]ержите.*",     # Starts with "Держите" (case insensitive)
    "^[Сс]мотрите.*",    # Starts with "Смотрите"
    "^[Вв]прочем.*",     # Starts with "Впрочем"
]
```

### Applying Filters to Resources

```toml
[[resources]]
feed_url = "https://lwn.net/headlines/rss"
parser = "web"
type = "rss"
filters = ["lwn_subscription"]  # Apply single filter

[[resources]]
feed_url = "https://t.me/techsparks"
parser = "telegram"
type = "telegram_channel"
filters = ["quality_content", "russian_announcements"]  # Apply multiple filters (pipeline)
```

### Filter Pipeline

When multiple filters are specified, they are applied as a pipeline in order. An item must pass all filters to be included:

```toml
[[resources]]
feed_url = "https://example.com/feed"
parser = "web"
type = "rss"
filters = ["short_posts", "quality_content"]  # Item must pass both filters
```

If an item fails any filter in the pipeline, it will be excluded from the final output.

## Caching

To speed up development and testing, myfeed caches parser and agent outputs in `~/.cache/myfeed/cache.db`.

### Cache Behavior

- **Parser cache**: Stores parsed content (HTML, transcriptions, formatted messages) by URL and parser type
- **Agent cache**: Stores final processed content after running the complete agent pipeline by URL, parser type, and agent list
- **Cache key**: Uses feed item URL as the primary cache key
- **Automatic invalidation**: Cache is invalidated when parser type changes or agent pipeline changes

### How Caching Works

1. **Agent cache check**: If agents are configured, first check if final processed output exists in cache
2. **Parser cache check**: If no agent cache hit, check if parsed content exists
3. **Fresh parse**: If no parser cache hit, parse the content and store in cache
4. **Agent processing**: If agents configured and no agent cache hit, run agent pipeline and store final output

### Cache Management

```bash
# Clear all cache entries (parser + agent cache)
myfeed -clean

# Normal run (uses cache automatically)
myfeed

# Check cache statistics on startup (logged automatically)
```

### Cache Statistics

On startup, myfeed displays cache statistics:
```
INFO cache initialized parser_entries=42 agent_entries=15
```

### When to Clear Cache

Clear the cache when:
- You change parser type for a resource (e.g., web → youtube)
- You change the agent pipeline for a resource
- You want to force fresh parsing/processing of all content
- Cache becomes stale or corrupted

### Cache Location

- **Default**: `~/.cache/myfeed/cache.db` (follows XDG Base Directory specification)
- **Alternative**: `$XDG_CACHE_HOME/myfeed/cache.db` if `XDG_CACHE_HOME` is set

### Performance Benefits

Caching dramatically improves rerun performance:
- **Parser cache**: Skip expensive web scraping, YouTube transcription, etc.
- **Agent cache**: Skip expensive AI API calls (Gemini)
- **Typical speedup**: 10-100x faster for cached content

## Used resources

- [PDF from HTML](https://www.reddit.com/r/webdev/comments/1gztdzm/building_a_pdf_with_html_crazy/)
- [genkit (fork)](https://github.com/naqerl/genkit) - AI framework with embedded dotprompt support
