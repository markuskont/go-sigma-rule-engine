package condition

import (
	"fmt"
	"strings"
)

func interfaceMapToStringInterfaceMap(m map[interface{}]interface{}) (map[string]interface{}, error) {
	m2 := make(map[string]interface{})
	for k, v := range m {
		sk, ok := k.(string)
		if !ok {
			return m2, fmt.Errorf("failed to create selection rule from interface")
		}
		m2[sk] = v
	}
	return m2, nil
}

func isKeywords(s string) bool { return strings.HasPrefix(s, "keywords") }

func checkKeyWord(in string) Token {
	if len(in) == 0 {
		return TokNil
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
