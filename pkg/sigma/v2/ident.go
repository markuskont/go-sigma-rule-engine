package sigma

import (
	"fmt"
	"reflect"
	"strconv"
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

func checkIdentType(name string, data interface{}) identType {
	t := reflectIdentKind(data)
	if strings.HasPrefix(name, "keyword") {
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
		return NewKeyword(rule)
	case identSelection:
		return NewSelection(rule)
	}
	return nil, fmt.Errorf("Unknown rule kind, should be keyword or selection")
}

// Keyword is a container for patterns joined by logical disjunction
type Keyword struct {
	S StringMatcher
	Stats
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

func NewKeyword(expr interface{}) (*Keyword, error) {
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
			return newStringKeyword(TextPatternContains, false, castIfaceToString(val)...)
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

type SelectionNumItem struct {
	Key     string
	Pattern interface{}
}

type SelectionStringItem struct {
	Key     string
	Pattern StringMatcher
}

type Selection struct {
	N []SelectionNumItem
	S []SelectionStringItem
	Stats
}

// Match implements Matcher
// TODO - numeric and boolean pattern match
func (s Selection) Match(msg Event) bool {
	for _, v := range s.S {
		val, ok := msg.Select(v.Key)
		if !ok {
			return false
		}
		switch vt := val.(type) {
		case string:
			if !v.Pattern.StringMatch(vt) {
				return false
			}
		case float64:
			// TODO - tmp hack that also loses floating point accuracy
			if !v.Pattern.StringMatch(strconv.Itoa(int(vt))) {
				return false
			}
		default:
			s.incrementMismatchCount()
			return false
		}
	}
	return true
}

func (s *Selection) incrementMismatchCount() *Selection {
	s.Stats.TypeMismatchCount++
	return s
}

func newSelectionFromMap(expr map[string]interface{}) (*Selection, error) {
	sel := &Selection{S: make([]SelectionStringItem, 0)}
	for key, pattern := range expr {
		var mod TextPatternModifier
		if strings.Contains(key, "|") {
			bits := strings.Split(key, "|")
			if length := len(bits); length != 2 {
				return nil, fmt.Errorf(
					"selection key %s invalid. Specifier should result in 2 sections", key)
			}
			if !isValidSpecifier(bits[1]) {
				return nil, fmt.Errorf("selection key %s specifier %s invalid",
					key, bits[1])
			}
			switch bits[1] {
			case TextPatternPrefix.String():
				mod = TextPatternPrefix
			case TextPatternSuffix.String():
				mod = TextPatternSuffix
			}
		}
		switch pat := pattern.(type) {
		case string:
			m, err := NewStringMatcher(mod, false, pat)
			if err != nil {
				return nil, err
			}
			sel.S = append(sel.S, SelectionStringItem{Key: key, Pattern: m})
		case []interface{}:
			// TODO - move this part to separate function and reuse in NewKeyword
			k, ok := isSameKind(pat)
			if !ok {
				return nil, ErrInvalidKind{
					Kind:     reflect.Array,
					T:        identKeyword,
					Critical: false,
					Msg:      "Mixed type slice",
				}
			}
			switch k {
			case reflect.String:
				m, err := NewStringMatcher(mod, false, castIfaceToString(pat)...)
				if err != nil {
					return nil, err
				}
				sel.S = append(sel.S, SelectionStringItem{Key: key, Pattern: m})
			case reflect.Int:
				m, err := NewNumMatcher(castIfaceToInt(pat)...)
				if err != nil {
					return nil, err
				}
				sel.N = func() []SelectionNumItem {
					item := SelectionNumItem{
						Key: key, Pattern: m,
					}
					if sel.N == nil {
						sel.N = []SelectionNumItem{item}
					}
					return append(sel.N, item)
				}()
			default:
				return nil, ErrInvalidKind{
					Kind:     k,
					T:        identKeyword,
					Critical: false,
					Msg:      "Unsupported data type",
				}
			}
		default:
			if t := reflect.TypeOf(pattern); t != nil {
				return nil, ErrInvalidKind{
					Kind:     t.Kind(),
					T:        identSelection,
					Critical: true,
					Msg:      "Unsupported selection value",
				}
			}
			return nil, ErrUnableToReflect
		}
	}
	return sel, nil
}

func NewSelection(expr interface{}) (*Selection, error) {
	switch v := expr.(type) {
	case map[interface{}]interface{}:
		return newSelectionFromMap(cleanUpInterfaceMap(v))
	default:
		return nil, ErrInvalidKind{
			Kind:     reflect.TypeOf(expr).Kind(),
			T:        identSelection,
			Critical: true,
			Msg:      "Unsupported selection root container",
		}
	}
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

func castIfaceToString(items []interface{}) []string {
	tx := make([]string, 0)
	for _, val := range items {
		tx = append(tx, fmt.Sprintf("%v", val))
	}
	return tx
}

func castIfaceToInt(items []interface{}) []int {
	tx := make([]int, 0)
	for _, val := range items {
		if n, ok := val.(int); ok {
			tx = append(tx, n)
		}
	}
	return tx
}

// Yaml can have non-string keys, so go-yaml unmarshals to map[interface{}]interface{}
// really annoying
func cleanUpInterfaceMap(rx map[interface{}]interface{}) map[string]interface{} {
	tx := make(map[string]interface{})
	for k, v := range rx {
		tx[fmt.Sprintf("%v", k)] = v
	}
	return tx
}
