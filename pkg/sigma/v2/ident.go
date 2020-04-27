package sigma

import (
	"fmt"
	"strings"
)

type identType int

func (i identType) String() string {
	switch i {
	case identKeyword:
		return "KEYWORD"
	case identSelection:
		return "SELECTION"
	default:
		return "UNK"
	}
}

const (
	identErr identType = iota
	identSelection
	identKeyword
)

func checkIdentType(item Item, data interface{}) identType {
	t := reflectIdentKind(data)
	if strings.HasPrefix(item.Val, "keyword") {
		if data == nil {
			return identKeyword
		}
		if t != identKeyword {
			return identErr
		}
	}
	return t
}

func reflectIdentKind(data interface{}) identType {
	switch data.(type) {
	case map[string]interface{}, map[interface{}]interface{}:
		return identSelection
	default:
		return identKeyword
	}
}

func newRuleFromIdent(rule interface{}, kind identType) (Branch, error) {
	switch kind {
	case identKeyword:

	case identSelection:

	}
	return nil, fmt.Errorf("Unknown rule kind, should be keyword or selection")
}

type keyword struct{}

func newKeyword(expr interface{}) (*keyword, error) {
	return nil, ErrInvalidKeywordConstruct{Expr: expr}
}

type selection struct{}

func newSelection(expr interface{}) (*selection, error) {
	return nil, ErrInvalidSelectionConstruct{Expr: expr}
}
