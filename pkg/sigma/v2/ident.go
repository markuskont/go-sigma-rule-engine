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
	matcher, err := NewStringMatcher(mod, lower, patterns...)
	if err != nil {
		return nil, err
	}
	return &Keyword{S: matcher}, nil
}

type SelectionStringItem struct {
	Key     string
	Pattern StringMatcher
}

type Selection struct {
	S []SelectionStringItem
}

// Match implements Matcher
func (s Selection) Match(msg Event) bool {
	for _, v := range s.S {
		val, ok := msg.Select(v.Key)
		if !ok {
			return false
		}
		if val, ok := val.(string); ok {
			if !v.Pattern.StringMatch(val) {
				return false
			}
		}
	}
	return true
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
