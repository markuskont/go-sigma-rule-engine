package sigma

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ryanuber/go-glob"
)

type stringPatterns struct {
	literals []string
	re       []*regexp.Regexp
	globs    []string
}

func newStringPatterns(patterns ...string) (*stringPatterns, error) {
	k := &stringPatterns{}
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if strings.HasPrefix(p, "/") && strings.HasSuffix(p, "/") {
			if k.re == nil {
				k.re = make([]*regexp.Regexp, 0)
			}
			p = strings.TrimLeft(p, "/")
			p = strings.TrimRight(p, "/")
			re, err := regexp.Compile(p)
			if err != nil {
				return k, ErrInvalidRegex{
					Pattern: p,
					Err:     err,
				}
			}
			k.re = append(k.re, re)
		} else if strings.Contains(p, "*") {
			if k.globs == nil {
				k.globs = make([]string, 0)
			}
			k.globs = append(k.globs, p)
		} else {
			if k.literals == nil {
				k.literals = make([]string, 0)
			}
			k.literals = append(k.literals, p)
		}
	}
	return k, nil
}

type Keyword struct {
	stringPatterns
	toLower bool
	Stats
}

func NewKeywordFromInterface(lowercase bool, expr interface{}) (*Keyword, error) {
	switch v := expr.(type) {
	case []string:
		return NewKeyword(lowercase, v...)
	case []interface{}:
		slc := make([]string, 0)
		for _, item := range v {
			switch cast := item.(type) {
			case string:
				slc = append(slc, cast)
			case int:
				slc = append(slc, strconv.Itoa(cast))
			case float64:
				slc = append(slc, strconv.Itoa(int(cast)))
			}
		}
		return NewKeyword(lowercase, slc...)
	case map[string]interface{}:
		if patterns, ok := v["Message"].([]string); ok {
			return NewKeyword(lowercase, patterns...)
		}
	case map[interface{}]interface{}:
		if vals, ok := v["Message"]; ok {
			switch data := vals.(type) {
			case []interface{}:
				slc := make([]string, 0)
				for _, item := range data {
					switch cast := item.(type) {
					case string:
						slc = append(slc, cast)
					case int:
						slc = append(slc, strconv.Itoa(cast))
					case float64:
						slc = append(slc, strconv.Itoa(int(cast)))
					}
				}
				return NewKeyword(lowercase, slc...)
			}
		}
	}
	return nil, fmt.Errorf(
		"Invalid type for parsing keyword expression. Should be slice of strings or a funky one element map where value is slice of strings. Or other stuff. Got |%+v| with type |%s|", expr, reflect.TypeOf(expr).String(),
	)
}

func NewKeyword(lowercase bool, patterns ...string) (*Keyword, error) {
	if patterns == nil || len(patterns) == 0 {
		return nil, fmt.Errorf("no patterns defined for keyword match rule")
	}
	k := &Keyword{
		toLower: lowercase,
		Stats:   Stats{},
	}
	for i, pat := range patterns {
		if lowercase {
			patterns[i] = strings.ToLower(pat)
		}
	}
	p, err := newStringPatterns(patterns...)
	if err != nil {
		return k, err
	}
	k.stringPatterns = *p
	return k, nil
}

func (k *Keyword) Match(obj EventChecker) bool {
	k.Total++
	return matchKeywords(k.stringPatterns, k.toLower, obj.GetMessage()...)
}

func (k Keyword) Self() interface{} { return k }

func matchKeywords(k stringPatterns, lowercase bool, fields ...string) bool {
	if fields == nil || len(fields) == 0 {
		return false
	}
	for _, field := range fields {
		if lowercase {
			field = strings.ToLower(field)
		}
		if k.literals != nil && len(k.literals) > 0 {
			for _, pattern := range k.literals {
				if strings.Contains(field, pattern) {
					return true
				}
			}
		}
		if k.re != nil && len(k.re) > 0 {
			for _, re := range k.re {
				if re.MatchString(field) {
					return true
				}
			}
		}
		if k.globs != nil && len(k.globs) > 0 {
			for _, g := range k.globs {
				if glob.Glob(g, field) {
					return true
				}
			}
		}
	}
	return false
}

type RuleConfig struct {
	LowerCase   bool
	NumToString bool
}

type FieldsList []*Fields

// Match implements sigma Matcher
func (f FieldsList) Match(obj EventChecker) bool {
	for _, rule := range f {
		if rule.Match(obj) {
			return true
		}
	}
	return false
}

// Self returns Node or final rule object for debugging and/or walking the tree
// Must be type switched externally
//Identifier
func (f FieldsList) Self() interface{} {
	return f
}

// JSON numbers are by spec all float64 values
type numPatterns []float64

func (p numPatterns) match(n float64) bool {
	for _, val := range p {
		if n == val {
			return true
		}
	}
	return false
}

func (p numPatterns) matchNumber(num interface{}, tryString bool) bool {
	switch nu := num.(type) {
	case float64:
		return p.match(nu)
	case int:
		return p.match(float64(nu))
	case int64:
		return p.match(float64(nu))
	case uint:
		return p.match(float64(nu))
	case uint64:
		return p.match(float64(nu))
	case int32:
		return p.match(float64(nu))
	case uint32:
		return p.match(float64(nu))
	case string:
		if tryString {
			if val, err := strconv.Atoi(nu); err == nil {
				return p.match(float64(val))
			}
		}
		return false
	default:
		return false
	}
}

func (p numPatterns) len() int    { return len(p) }
func (p numPatterns) empty() bool { return p.len() == 0 }

type Fields struct {
	sPatterns map[string]stringPatterns
	nPatterns map[string]numPatterns

	toLower         bool
	tryStingNumbers bool
	Stats
}

func NewFields(raw map[string]interface{}, lowercase, stringnum bool) (*Fields, error) {
	if raw == nil || len(raw) == 0 {
		return nil, fmt.Errorf("wrong interface type for rule condition, only map[string]interface{} supported")
	}
	f := &Fields{
		toLower:         lowercase,
		tryStingNumbers: stringnum,
	}
	// Thank you pythonistas
	for k, v := range raw {
		// TODO - key might also have a piped specification for substring placement
		// TODO - cont. example is : Image|endswith and CommandLine|contains
		// TODO - this is a temporary hack, use HasPrefix/HasSuffix/Contains methods instead
		if strings.Contains(k, "|") {
			k = strings.Split(k, "|")[0]
		}
		switch condition := v.(type) {
		case nil:
			// TODO
			// Boolean pattern is a thing
			// should handle in case json parser picks null value from rule
		case string:
			if f.sPatterns == nil {
				f.sPatterns = make(map[string]stringPatterns)
			}
			patterns, err := newStringPatterns(condition)
			if err != nil {
				return f, err
			}
			f.sPatterns[k] = *patterns
		case float64:
			if f.nPatterns == nil {
				f.nPatterns = make(map[string]numPatterns)
			}
			f.nPatterns[k] = []float64{condition}
		case int:
			if f.nPatterns == nil {
				f.nPatterns = make(map[string]numPatterns)
			}
			f.nPatterns[k] = []float64{float64(condition)}
		case []string:
			if f.sPatterns == nil {
				f.sPatterns = make(map[string]stringPatterns)
			}
			patterns, err := newStringPatterns(condition...)
			if err != nil {
				return f, err
			}
			f.sPatterns[k] = *patterns
		case []interface{}:
			var t reflect.Kind
			var stringAndNumber bool
			// make sure all list types are the same
		loop:
			for i, item := range condition {
				if i > 0 {
					if t2 := reflect.TypeOf(item).Kind(); t2 != t && (t == reflect.String || t2 == reflect.String) {
						// Just convert all values to string if a single item happens to be one
						stringAndNumber = true
						t = reflect.String
						break loop
					} else if t2 != t {
						return f, fmt.Errorf(
							"Selection/field rule parse fail for key %s list contains %s and %s",
							k,
							t.String(),
							t2.String(),
						)
					}
				}
				t = reflect.TypeOf(item).Kind()
			}
			switch t {
			case reflect.String:
				if f.sPatterns == nil {
					f.sPatterns = make(map[string]stringPatterns)
				}
				patterns, err := newStringPatterns(func() []string {
					str := make([]string, len(condition))
					for i, item := range condition {
						// This should already be checked
						if stringAndNumber {
							switch cast := v.(type) {
							case string:
								str[i] = cast
							case int:
								str[i] = strconv.Itoa(cast)
							case float64:
								str[i] = strconv.Itoa(int(cast))
							default:

							}
						} else {
							str[i] = item.(string)
						}
					}
					return str
				}()...)
				if err != nil {
					return f, err
				}
				f.sPatterns[k] = *patterns
			case reflect.Float64:
				if f.nPatterns == nil {
					f.nPatterns = make(map[string]numPatterns)
				}
				if f.nPatterns == nil {
					f.nPatterns = make(map[string]numPatterns)
				}
				f.nPatterns[k] = func() []float64 {
					flt := make([]float64, len(condition))
					for i, item := range condition {
						// this should already be checked
						flt[i] = item.(float64)
					}
					return flt
				}()
			case reflect.Int:
				if f.nPatterns == nil {
					f.nPatterns = make(map[string]numPatterns)
				}
				if f.nPatterns == nil {
					f.nPatterns = make(map[string]numPatterns)
				}
				f.nPatterns[k] = func() []float64 {
					flt := make([]float64, len(condition))
					for i, item := range condition {
						// this should already be checked
						flt[i] = float64(item.(int))
					}
					return flt
				}()
			default:
				return f, fmt.Errorf("unsupported type for key %s, got %s but expected string or float64", k, t.String())
			}
		default:
			return nil, fmt.Errorf(
				"wrong rule type for [%+v], field [%s], got %T, only support string, float64, or their respective sliced versions",
				raw, k, v,
			)
		}
	}
	return f, nil
}

func (f *Fields) Match(obj EventChecker) bool {
	f.Total++
	if f.sPatterns != nil && len(f.sPatterns) > 0 {
	and1:
		for field, patterns := range f.sPatterns {
			if val, ok := obj.GetField(field); ok {
				if str, ok := val.(string); ok {
					if matchKeywords(patterns, f.toLower, str) {
						continue and1
					}
				}
				return false
			}
			return false
		}
	}
	if f.nPatterns != nil && len(f.nPatterns) > 0 {
	and2:
		for field, patterns := range f.nPatterns {
			if val, ok := obj.GetField(field); ok {
				if patterns.matchNumber(val, f.tryStingNumbers) {
					continue and2
				}
			}
			return false
		}
	}
	return true
}

func (f Fields) Self() interface{} { return f }

type Stats struct {
	Hits, Total int64
	Took
}

type Took struct {
	Min, Max, Avg time.Duration
	ringBuffer    chan time.Duration
}
