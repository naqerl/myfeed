package parser

import "fmt"

type Type = string

var (
	Web      = Type("web")
	Telegram = Type("telegram")
	Torrent  = Type("torrent")
	YouTube  = Type("youtube")
)

type Parser interface {
	Parse(uri string) (Response, error)
}

type Response interface {
	fmt.Stringer
}
