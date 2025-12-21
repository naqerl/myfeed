package cache

import (
	"encoding/json"
	"fmt"

	"github.com/scipunch/myfeed/parser"
	"github.com/scipunch/myfeed/parser/telegram"
	"github.com/scipunch/myfeed/parser/web"
	"github.com/scipunch/myfeed/parser/youtube"
)

// CachedResponse wraps parser responses for serialization
type CachedResponse struct {
	ParserType string          `json:"parser_type"`
	Data       json.RawMessage `json:"data"`
}

// SerializeParserResponse converts parser.Response to JSON bytes
func SerializeParserResponse(parserType string, resp parser.Response) ([]byte, error) {
	var data []byte
	var err error

	switch parserType {
	case parser.Web:
		webResp, ok := resp.(web.Response)
		if !ok {
			return nil, fmt.Errorf("expected web.Response, got %T", resp)
		}
		data, err = json.Marshal(webResp)

	case parser.YouTube:
		ytResp, ok := resp.(youtube.Response)
		if !ok {
			return nil, fmt.Errorf("expected youtube.Response, got %T", resp)
		}
		data, err = json.Marshal(ytResp)

	case parser.Telegram:
		tgResp, ok := resp.(telegram.Response)
		if !ok {
			return nil, fmt.Errorf("expected telegram.Response, got %T", resp)
		}
		data, err = json.Marshal(tgResp)

	default:
		return nil, fmt.Errorf("unknown parser type: %s", parserType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to marshal parser response: %w", err)
	}

	cached := CachedResponse{
		ParserType: parserType,
		Data:       data,
	}

	return json.Marshal(cached)
}

// DeserializeParserResponse converts JSON bytes back to parser.Response
func DeserializeParserResponse(parserType string, data []byte) (parser.Response, error) {
	var cached CachedResponse
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached response: %w", err)
	}

	if cached.ParserType != parserType {
		return nil, fmt.Errorf("parser type mismatch: cached=%s, expected=%s", cached.ParserType, parserType)
	}

	switch parserType {
	case parser.Web:
		var resp web.Response
		if err := json.Unmarshal(cached.Data, &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal web response: %w", err)
		}
		return resp, nil

	case parser.YouTube:
		var resp youtube.Response
		if err := json.Unmarshal(cached.Data, &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal youtube response: %w", err)
		}
		return resp, nil

	case parser.Telegram:
		var resp telegram.Response
		if err := json.Unmarshal(cached.Data, &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal telegram response: %w", err)
		}
		return resp, nil

	default:
		return nil, fmt.Errorf("unknown parser type: %s", parserType)
	}
}
