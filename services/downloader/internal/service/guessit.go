package service

import "github.com/kingbenny101/kbarr/shared/parser"

func runGuessit(filename string) parser.ParseResult {
	return parser.Parse(filename)
}
