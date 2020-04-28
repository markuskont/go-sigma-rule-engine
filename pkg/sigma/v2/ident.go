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

type Keyword struct {
}

// Match implements Matcher
func (k Keyword) Match(msg Event) bool {
	panic("not implemented") // TODO: Implement
}

func newKeyword(expr interface{}) (*Keyword, error) {
	switch val := expr.(type) {
	case []string:
		return newStringKeyword(TextPatternContains, false, val...)
	case []interface{}:
		panic("TODO - keyword interface slice")
	default:
		// TODO
		return nil, ErrInvalidKeywordConstruct{Expr: expr}
	}
}

func newStringKeyword(mod TextPatternModifier, lower bool, patterns ...string) (*Keyword, error) {
	if patterns == nil || len(patterns) == 0 {
		return nil, fmt.Errorf("no patterns defined for keyword match rule")
	}
	matcher := make(StringMatchers, 0)
	for _, p := range patterns {
		if strings.Contains(p, "*") {

		} else if strings.HasPrefix(p, "/") && strings.HasSuffix(p, "/") {

		} else {
			switch mod {
			case TextPatternSuffix:
				matcher = append(matcher, SuffixPattern{Token: p, Lowercase: lower})
			case TextPatternPrefix:
				matcher = append(matcher, PrefixPattern{Token: p, Lowercase: lower})
			default:
				matcher = append(matcher, ContentPattern{Token: p, Lowercase: lower})
			}
		}
	}
	return nil, nil
}

type selection struct{}

// Keywords implements Keyworder
func (s selection) Keywords() ([]string, bool) {
	panic("not implemented") // TODO: Implement
}

// Select implements Selector
func (s selection) Select(_ string) (interface{}, bool) {
	panic("not implemented") // TODO: Implement
}
