package sigma

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ryanuber/go-glob"
)

type TextPatternModifier int

const (
	TextPatternContains TextPatternModifier = iota
	TextPatternPrefix
	TextPatternSuffix
)

// StringMatcher is an atomic pattern that could implement glob, literal or regex matchers
type StringMatcher interface {
	// StringMatch implements StringMatcher
	StringMatch(string) bool
}

func NewStringMatcher(mod TextPatternModifier, lower bool, patterns ...string) (StringMatcher, error) {
	if patterns == nil || len(patterns) == 0 {
		return nil, fmt.Errorf("no patterns defined for keyword match rule")
	}
	matcher := make(StringMatchers, 0)
	for _, p := range patterns {
		// TODO - check for escape sequences
		if strings.Contains(p, "*") {
			matcher = append(matcher, GlobPattern{Token: p, Lowercase: lower})
		} else if strings.HasPrefix(p, "/") && strings.HasSuffix(p, "/") {
			re, err := regexp.Compile(strings.TrimLeft(strings.TrimRight(p, "/"), "/"))
			if err != nil {
				return nil, err
			}
			matcher = append(matcher, RegexPattern{Re: re})
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
	return func() StringMatcher {
		if len(matcher) == 1 {
			return matcher[0]
		}
		return matcher.Optimize()
	}(), nil
}

// StringMatchers holds multiple atomic matchers
// Patterns are meant to be list of possibilities
// thus, objects are joined with logical disjunctions
type StringMatchers []StringMatcher

// StringMatch implements StringMatcher
func (s StringMatchers) StringMatch(msg string) bool {
	for _, m := range s {
		if m.StringMatch(msg) {
			return true
		}
	}
	return false
}

// Optimize creates a new StringMatchers slice ordered by matcher type
// First match wins, thus we can optimize by making sure fast string patterns
// are executed first, then globs, and finally slow regular expressions
func (s StringMatchers) Optimize() StringMatchers {
	globs := make(StringMatchers, 0)
	re := make(StringMatchers, 0)
	literals := make(StringMatchers, 0)
	for _, pat := range s {
		switch pat.(type) {
		case ContentPattern, PrefixPattern, SuffixPattern:
			literals = append(literals, pat)
		case GlobPattern:
			globs = append(globs, pat)
		case RegexPattern:
			re = append(re, pat)
		}
	}
	tx := append(literals, globs...)
	tx = append(tx, re...)
	return tx
}

// ContentPattern is a token for literal content matching
type ContentPattern struct {
	Token     string
	Lowercase bool
}

// StringMatch implements StringMatcher
func (c ContentPattern) StringMatch(msg string) bool {
	return strings.Contains(msg, func() string {
		if c.Lowercase {
			return strings.ToLower(c.Token)
		}
		return c.Token
	}())
}

// PrefixPattern is a token for literal content matching
type PrefixPattern struct {
	Token     string
	Lowercase bool
}

// StringMatch implements StringMatcher
func (c PrefixPattern) StringMatch(msg string) bool {
	return strings.HasPrefix(msg, func() string {
		if c.Lowercase {
			return strings.ToLower(c.Token)
		}
		return c.Token
	}())
}

// SuffixPattern is a token for literal content matching
type SuffixPattern struct {
	Token     string
	Lowercase bool
}

// StringMatch implements StringMatcher
func (c SuffixPattern) StringMatch(msg string) bool {
	return strings.HasSuffix(msg, func() string {
		if c.Lowercase {
			return strings.ToLower(c.Token)
		}
		return c.Token
	}())
}

// RegexPattern is for matching messages with regular expresions
type RegexPattern struct {
	Re *regexp.Regexp
}

// StringMatch implements StringMatcher
func (r RegexPattern) StringMatch(msg string) bool {
	return r.Re.MatchString(msg)
}

// GlobPattern is similar to ContentPattern but allows for asterisk wildcards
type GlobPattern struct {
	Token     string
	Lowercase bool
}

// StringMatch implements StringMatcher
func (g GlobPattern) StringMatch(msg string) bool {
	return glob.Glob(func() string {
		if g.Lowercase {
			return strings.ToLower(g.Token)
		}
		return g.Token
	}(), msg)
}
