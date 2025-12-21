package parser

import (
	"fmt"

	"github.com/scipunch/myfeed/fetcher/types"
)

type Type = string

var (
	Web      = Type("web")
	Telegram = Type("telegram")
	Torrent  = Type("torrent")
	YouTube  = Type("youtube")
)

type Parser interface {
	Parse(item types.FeedItem) (Response, error)
}

type Response interface {
	fmt.Stringer
}
