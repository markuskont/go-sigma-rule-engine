// Package condition provides library to parse sigma rule condition field into rule tokens
package condition

import (
	"strings"
)

type Item struct {
	T   Token
	Val string
}

func checkKeyWord(in string) Token {
	if len(in) == 0 {
		return TokErr
	}
	switch strings.ToLower(in) {
	case KeywordAnd.Literal():
		return KeywordAnd
	case KeywordOr.Literal():
		return KeywordOr
	case KeywordNot.Literal():
		return KeywordNot
	case "sum", "min", "max", "count", "avg":
		return KeywordAgg
	case IdentifierAll.Literal():
		return IdentifierAll
	default:
		if strings.Contains(in, "*") {
			return IdentifierWithWildcard
		}
		return Identifier
	}
}

var eof = rune(0)
