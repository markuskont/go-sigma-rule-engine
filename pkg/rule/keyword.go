package rule

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/markuskont/go-sigma-rule-engine/pkg/types"
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
				return k, types.ErrInvalidRegex{
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

func (k *Keyword) Match(obj types.EventChecker) bool {
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
