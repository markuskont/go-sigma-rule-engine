package sigma

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gobwas/glob"
)

type TextPatternModifier int

const (
	TextPatternNone TextPatternModifier = iota
	TextPatternContains
	TextPatternPrefix
	TextPatternSuffix
	TextPatternAll
	TextPatternRegex
	TextPatternKeyword
)

// func isValidSpecifier(in string) bool {
// 	return in == "contains" ||
// 		in == "endswith" ||
// 		in == "startswith"
// }

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
	if len(patterns) == 0 {
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

var gWSCollapse = regexp.MustCompile(`\s+`)

// handleWhitespace takes str and if the global configuration for collapsing whitespace is NOT turned off
// returns the string with whitespace collapsed (1+ spaces, tabs, etc... become single space); otherwise
//just returns the unmodified str; this only applies to non-regex rules and data hitting non-regex rules
func handleWhitespace(str string, noCollapseWS bool) string {
	if noCollapseWS { //do we collapse whitespace or not?  See config.NoCollapseWS (we collapse by default)
		return str
	}
	return gWSCollapse.ReplaceAllString(str, " ")
}

func NewStringMatcher(
	mod TextPatternModifier,
	lower, all, noCollapseWS bool,
	patterns ...string,
) (StringMatcher, error) {
	if len(patterns) == 0 {
		return nil, fmt.Errorf("no patterns defined for matcher object")
	}
	matcher := make([]StringMatcher, 0)
	for _, p := range patterns {
		//process modifiers first
		switch mod {
		case TextPatternRegex: //regex per spec
			re, err := regexp.Compile(p)
			if err != nil {
				return nil, err
			}
			matcher = append(matcher, RegexPattern{Re: re})
		case TextPatternContains: //contains: puts * wildcards around the values, such that the value is matched anywhere in the field.
			p = handleWhitespace(p, noCollapseWS)
			p = "*" + p + "*"
			globNG, err := glob.Compile(p)
			if err != nil {
				return nil, err
			}
			matcher = append(matcher, GlobPattern{Glob: &globNG, NoCollapseWS: noCollapseWS})
		case TextPatternSuffix:
			p = handleWhitespace(p, noCollapseWS)
			matcher = append(matcher, SuffixPattern{Token: p, Lowercase: lower, NoCollapseWS: noCollapseWS})
		case TextPatternPrefix:
			p = handleWhitespace(p, noCollapseWS)
			matcher = append(matcher, PrefixPattern{Token: p, Lowercase: lower, NoCollapseWS: noCollapseWS})
		default:
			//no (supported) modifiers, handle non-spec regex, globs and regular values
			if strings.HasPrefix(p, "/") && strings.HasSuffix(p, "/") {
				re, err := regexp.Compile(strings.TrimLeft(strings.TrimRight(p, "/"), "/"))
				if err != nil {
					return nil, err
				}
				matcher = append(matcher, RegexPattern{Re: re})
			} else if mod == TextPatternKeyword {
				//this is a bit hacky, basically if the pattern coming in is a keyword and did not appear
				//to be a regex, always process it as a 'contains' style glob (can appear anywhere...)
				//this is due, I believe, on how keywords are generally handled, where it is likely a random
				//string or event long message that may have additional detail/etc...
				p = handleWhitespace(p, noCollapseWS)
				p = "*" + p + "*"
				globNG, err := glob.Compile(p)
				if err != nil {
					return nil, err
				}
				matcher = append(matcher, GlobPattern{Glob: &globNG, NoCollapseWS: noCollapseWS})
			} else if strings.Contains(p, "*") {
				p = handleWhitespace(p, noCollapseWS)
				globNG, err := glob.Compile(p)
				if err != nil {
					return nil, err
				}
				matcher = append(matcher, GlobPattern{Glob: &globNG, NoCollapseWS: noCollapseWS})
			} else {
				p = handleWhitespace(p, noCollapseWS)
				matcher = append(matcher, ContentPattern{Token: p, Lowercase: lower, NoCollapseWS: noCollapseWS})
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
		//I thought about a type assertion here for handling whitespace
		//however, as we're dealing with non-pointer types, that may cause
		//some added overhead that we can avoid by just implementing where need to
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
	Token        string
	Lowercase    bool
	NoCollapseWS bool
}

// StringMatch implements StringMatcher
func (c ContentPattern) StringMatch(msg string) bool {
	msg = handleWhitespace(msg, c.NoCollapseWS)
	return lowerCaseIfNeeded(msg, c.Lowercase) == lowerCaseIfNeeded(c.Token, c.Lowercase)
}

// PrefixPattern is a token for literal content matching
type PrefixPattern struct {
	Token        string
	Lowercase    bool
	NoCollapseWS bool
}

// StringMatch implements StringMatcher
func (c PrefixPattern) StringMatch(msg string) bool {
	msg = handleWhitespace(msg, c.NoCollapseWS)
	return strings.HasPrefix(
		lowerCaseIfNeeded(msg, c.Lowercase),
		lowerCaseIfNeeded(c.Token, c.Lowercase),
	)
}

// SuffixPattern is a token for literal content matching
type SuffixPattern struct {
	Token        string
	Lowercase    bool
	NoCollapseWS bool
}

// StringMatch implements StringMatcher
func (c SuffixPattern) StringMatch(msg string) bool {
	msg = handleWhitespace(msg, c.NoCollapseWS)
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
	Glob         *glob.Glob
	NoCollapseWS bool
}

// StringMatch implements StringMatcher
func (g GlobPattern) StringMatch(msg string) bool {
	msg = handleWhitespace(msg, g.NoCollapseWS)
	return (*g.Glob).Match(msg)
}

// SimplePattern is a reference type to illustrate StringMatcher
type SimplePattern struct {
	Token        string
	NoCollapseWS bool
}

// StringMatch implements StringMatcher
func (s SimplePattern) StringMatch(msg string) bool {
	msg = handleWhitespace(msg, s.NoCollapseWS)
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
