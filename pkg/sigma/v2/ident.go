package sigma

import (
	"fmt"
	"reflect"
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

// Keyword is a container for patterns joined by logical disjunction
type Keyword struct {
	S StringMatcher
}

// Match implements Matcher
func (k Keyword) Match(msg Event) bool {
	msgs, ok := msg.Keywords()
	if !ok {
		return false
	}
	for _, m := range msgs {
		if k.S.StringMatch(m) {
			return true
		}
	}
	return false
}

func newKeyword(expr interface{}) (*Keyword, error) {
	switch val := expr.(type) {
	case []string:
		return newStringKeyword(TextPatternContains, false, val...)
	case []interface{}:
		k, ok := isSameKind(val)
		if !ok {
			return nil, ErrInvalidKind{
				Kind:     reflect.Array,
				T:        identKeyword,
				Critical: false,
				Msg:      "Mixed type slice",
			}
		}
		switch v := k; {
		case v == reflect.String:
			s, _ := castIfaceToString(val...)
			return newStringKeyword(TextPatternContains, false, s...)
		default:
			return nil, ErrInvalidKind{
				Kind:     v,
				T:        identKeyword,
				Critical: false,
				Msg:      "Unsupported data type",
			}
		}

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
			return nil, ErrUnsupportedExpression{
				Msg: "glob", T: identKeyword, Expr: patterns, Critical: true,
			}
		} else if strings.HasPrefix(p, "/") && strings.HasSuffix(p, "/") {
			return nil, ErrUnsupportedExpression{
				Msg: "regex", T: identKeyword, Expr: patterns, Critical: true,
			}
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
	return &Keyword{
		S: func() StringMatcher {
			if len(matcher) == 1 {
				return matcher[0]
			}
			return matcher
		}()}, nil
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

func isSameKind(data []interface{}) (reflect.Kind, bool) {
	var current, last reflect.Kind
	for i, d := range data {
		current = reflect.TypeOf(d).Kind()
		if i > 0 {
			if current != last {
				return current, false
			}
		}
		last = current
	}
	return current, true
}

// castIfaceToString assumes that kind check has already been done
func castIfaceToString(items ...interface{}) ([]string, int) {
	tx := make([]string, 0)
	var failed int
	for _, val := range items {
		if s, ok := val.(string); ok {
			tx = append(tx, s)
		} else {
			failed++
		}
	}
	return tx, failed
}
