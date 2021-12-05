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
	switch v := data.(type) {
	case map[string]interface{}, map[interface{}]interface{}:
		return identSelection
	case []interface{}:
		k, ok := isSameKind(v)
		if !ok {
			return identErr
		}
		switch k {
		case reflect.Map:
			return identSelection
		default:
			return identKeyword
		}
	default:
		return identKeyword
	}
}

func newRuleFromIdent(rule interface{}, kind identType) (Branch, error) {
	switch kind {
	case identKeyword:
		return NewKeyword(rule)
	case identSelection:
		return NewSelectionBranch(rule)
	}
	return nil, fmt.Errorf("Unknown rule kind, should be keyword or selection")
}

// Keyword is a container for patterns joined by logical disjunction
type Keyword struct {
	S StringMatcher
	Stats
}

// Match implements Matcher
func (k Keyword) Match(msg Event) (bool, bool) {
	msgs, ok := msg.Keywords()
	if !ok {
		return false, false
	}
	for _, m := range msgs {
		if k.S.StringMatch(m) {
			return true, true
		}
	}
	return false, true
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
	matcher, err := NewStringMatcher(mod, lower, false, patterns...)
	if err != nil {
		return nil, err
	}
	return &Keyword{S: matcher}, nil
}

type SelectionNumItem struct {
	Key     string
	Pattern NumMatcher
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
func (s Selection) Match(msg Event) (bool, bool) {
	for _, v := range s.N {
		val, ok := msg.Select(v.Key)
		if !ok {
			return false, false
		}
		switch vt := val.(type) {
		case string:
			n, err := strconv.Atoi(vt)
			if err != nil {
				// TODO - better debugging
				return false, true
			}
			if !v.Pattern.NumMatch(n) {
				return false, true
			}
		case float64:
			// JSON numbers are all by spec float64 values
			if !v.Pattern.NumMatch(int(vt)) {
				return false, true
			}
		case int:
			// JSON numbers are all by spec float64 values
			if !v.Pattern.NumMatch(vt) {
				return false, true
			}
		case int64:
			// JSON numbers are all by spec float64 values
			if !v.Pattern.NumMatch(int(vt)) {
				return false, true
			}
		case int32:
			// JSON numbers are all by spec float64 values
			if !v.Pattern.NumMatch(int(vt)) {
				return false, true
			}
		case uint:
			// JSON numbers are all by spec float64 values
			if !v.Pattern.NumMatch(int(vt)) {
				return false, true
			}
		case uint32:
			// JSON numbers are all by spec float64 values
			if !v.Pattern.NumMatch(int(vt)) {
				return false, true
			}
		case uint64:
			// JSON numbers are all by spec float64 values
			if !v.Pattern.NumMatch(int(vt)) {
				return false, true
			}
		}
	}
	for _, v := range s.S {
		val, ok := msg.Select(v.Key)
		if !ok {
			return false, false
		}
		switch vt := val.(type) {
		case string:
			if !v.Pattern.StringMatch(vt) {
				return false, true
			}
		case float64:
			// TODO - tmp hack that also loses floating point accuracy
			if !v.Pattern.StringMatch(strconv.Itoa(int(vt))) {
				return false, true
			}
		default:
			s.incrementMismatchCount()
			return false, true
		}
	}
	return true, true
}

func (s *Selection) incrementMismatchCount() *Selection {
	s.Stats.TypeMismatchCount++
	return s
}

func newSelectionFromMap(expr map[string]interface{}) (*Selection, error) {
	sel := &Selection{S: make([]SelectionStringItem, 0)}
	for key, pattern := range expr {
		var mod TextPatternModifier
		var all bool
		if strings.Contains(key, "|") {
			bits := strings.Split(key, "|")
			if length := len(bits); length < 2 || length > 3 {
				return nil, fmt.Errorf(
					"selection key %s invalid. Specifier should result in 2 or 3 sections", key)
			}
			if !isValidSpecifier(bits[1]) {
				return nil, fmt.Errorf("selection key %s specifier %s invalid",
					key, bits[1])
			}
			switch bits[1] {
			case "startswith":
				mod = TextPatternPrefix
			case "endswith":
				mod = TextPatternSuffix
			case "contains":
				if len(bits) == 3 && bits[2] == "all" {
					all = true
				}
			}
		}
		switch pat := pattern.(type) {
		case string:
			m, err := NewStringMatcher(mod, false, all, pat)
			if err != nil {
				return nil, err
			}
			sel.S = append(sel.S, SelectionStringItem{Key: key, Pattern: m})
		case int:
			m, err := NewNumMatcher(pat)
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
				m, err := NewStringMatcher(mod, false, all, castIfaceToString(pat)...)
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

func NewSelectionBranch(expr interface{}) (Branch, error) {
	switch v := expr.(type) {
	case []interface{}:
		selections := make([]Branch, 0)
		for _, item := range v {
			b, err := NewSelectionBranch(item)
			if err != nil {
				return nil, err
			}
			selections = append(selections, b)
		}
		return NodeSimpleOr(selections).Reduce(), nil
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
		cType := reflect.TypeOf(d)
		if cType == nil {
			return reflect.Invalid, false
		}
		current = cType.Kind()
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
