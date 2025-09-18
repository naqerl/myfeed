package factory

import (
	"fmt"
	"log"

	"github.com/scipunch/myfeed/parser"
	"github.com/scipunch/myfeed/parser/web"
	"github.com/scipunch/myfeed/parser/youtube"
)

func Init(types []parser.Type) (map[parser.Type]parser.Parser, error) {
	res := make(map[parser.Type]parser.Parser)
	for _, parserT := range types {
		if res[parserT] != nil {
			continue
		}
		var (
			p   parser.Parser
			err error
		)
		switch parserT {
		case parser.Web:
			p, err = web.New()
		case parser.YouTube:
			p, err = youtube.New()
		default:
			log.Fatalf("parser with type %s not implemented", parserT)
		}
		if err != nil {
			return res, fmt.Errorf("failed to initialize parser for %s with %w", parserT, err)
		}
		res[parserT] = p
	}
	return res, nil
}
