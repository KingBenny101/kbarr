package service

import "github.com/kingbenny101/kbarr/internal/parser"

func runGuessit(filename string) parser.ParseResult {
	return parser.Parse(filename)
}
