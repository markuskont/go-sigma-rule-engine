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
	TextPatternAll
)

func isValidSpecifier(in string) bool {
	return in == "contains" ||
		in == "endswith" ||
		in == "startswith"
}

// NumMatcher is an atomic pattern for numeric item or list of items
type NumMatcher interface {
	// NumMatch implements NumMatcher
	NumMatch(int) bool
}

// NumMatchers holds multiple numeric matchers
type NumMatchers []NumMatcher

// NumMatch implements NumMatcher
func (n NumMatchers) NumMatch(val int) bool {
	for _, v := range n {
		if v.NumMatch(val) {
			return true
		}
	}
	return false
}

func NewNumMatcher(patterns ...int) (NumMatcher, error) {
	if patterns == nil || len(patterns) == 0 {
		return nil, fmt.Errorf("no patterns defined for matcher object")
	}
	matcher := make(NumMatchers, 0)
	for _, p := range patterns {
		matcher = append(matcher, NumPattern{Val: p})
	}

	return func() NumMatcher {
		if len(matcher) == 1 {
			return matcher[0]
		}
		return matcher
	}(), nil
}

// StringMatcher is an atomic pattern that could implement glob, literal or regex matchers
type StringMatcher interface {
	// StringMatch implements StringMatcher
	StringMatch(string) bool
}

func NewStringMatcher(
	mod TextPatternModifier,
	lower, all bool,
	patterns ...string,
) (StringMatcher, error) {
	if patterns == nil || len(patterns) == 0 {
		return nil, fmt.Errorf("no patterns defined for matcher object")
	}
	matcher := make([]StringMatcher, 0)
	for _, p := range patterns {
		if strings.HasPrefix(p, "/") && strings.HasSuffix(p, "/") {
			re, err := regexp.Compile(strings.TrimLeft(strings.TrimRight(p, "/"), "/"))
			if err != nil {
				return nil, err
			}
			matcher = append(matcher, RegexPattern{Re: re})
		} else if strings.Contains(p, "*") {
			matcher = append(matcher, GlobPattern{Token: p, Lowercase: lower})
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
		if all {
			return StringMatchersConj(matcher).Optimize()
		}
		return StringMatchers(matcher).Optimize()
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
	return optimizeStringMatchers(s)
}

// StringMatchersConj is similar to StringMatcher but elements are joined with
// conjunction, i.e. all patterns must match
// used to implement "all" specifier for selection types
type StringMatchersConj []StringMatcher

// StringMatch implements StringMatcher
func (s StringMatchersConj) StringMatch(msg string) bool {
	for _, m := range s {
		if !m.StringMatch(msg) {
			return false
		}
	}
	return true
}

// Optimize creates a new StringMatchers slice ordered by matcher type
// First match wins, thus we can optimize by making sure fast string patterns
// are executed first, then globs, and finally slow regular expressions
func (s StringMatchersConj) Optimize() StringMatchersConj {
	return optimizeStringMatchers(s)
}

func optimizeStringMatchers(s []StringMatcher) []StringMatcher {
	globs := make([]StringMatcher, 0)
	re := make([]StringMatcher, 0)
	literals := make([]StringMatcher, 0)
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
	return append(literals, append(globs, re...)...)
}

// ContentPattern is a token for literal content matching
type ContentPattern struct {
	Token     string
	Lowercase bool
}

// StringMatch implements StringMatcher
func (c ContentPattern) StringMatch(msg string) bool {
	return strings.Contains(
		lowerCaseIfNeeded(msg, c.Lowercase),
		lowerCaseIfNeeded(c.Token, c.Lowercase),
	)
}

// PrefixPattern is a token for literal content matching
type PrefixPattern struct {
	Token     string
	Lowercase bool
}

// StringMatch implements StringMatcher
func (c PrefixPattern) StringMatch(msg string) bool {
	return strings.HasPrefix(
		lowerCaseIfNeeded(c.Token, c.Lowercase),
		lowerCaseIfNeeded(msg, c.Lowercase),
	)
}

// SuffixPattern is a token for literal content matching
type SuffixPattern struct {
	Token     string
	Lowercase bool
}

// StringMatch implements StringMatcher
func (c SuffixPattern) StringMatch(msg string) bool {
	return strings.HasSuffix(
		lowerCaseIfNeeded(msg, c.Lowercase),
		lowerCaseIfNeeded(c.Token, c.Lowercase),
	)
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
	return glob.Glob(
		lowerCaseIfNeeded(g.Token, g.Lowercase),
		lowerCaseIfNeeded(msg, g.Lowercase),
	)
}

// SimplePattern is a reference type to illustrate StringMatcher
type SimplePattern struct {
	Token string
}

// StringMatch implements StringMatcher
func (s SimplePattern) StringMatch(msg string) bool {
	return strings.Contains(msg, s.Token)
}

func lowerCaseIfNeeded(str string, lower bool) string {
	if lower {
		return strings.ToLower(str)
	}
	return str
}

// NumPattern matches on numeric value
type NumPattern struct {
	Val int
}

// NumMatch implements NumMatcher
func (n NumPattern) NumMatch(val int) bool {
	return n.Val == val
}
