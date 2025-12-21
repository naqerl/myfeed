# myFeed

The goal of this project is to generate a personal news paper in a PDF based on the provided RSS feeds.

Configuration is made via TOML file where resources and corresponding parsers could be configured.

## Features

- **Multiple feed sources**: RSS feeds, Telegram channels, YouTube transcripts
- **Content parsing**: Extract readable content from web pages
- **AI-powered agents**: Post-process content with Gemini (summarization, etc.)
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

## Used resources

- [PDF from HTML](https://www.reddit.com/r/webdev/comments/1gztdzm/building_a_pdf_with_html_crazy/)
- [genkit (fork)](https://github.com/naqerl/genkit) - AI framework with embedded dotprompt support
