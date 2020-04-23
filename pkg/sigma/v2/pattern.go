package sigma

import (
	"regexp"
	"strings"

	"github.com/ryanuber/go-glob"
)

type TextPatternType int

const (
	PatLiteral TextPatternType = iota
	PatPrefix
	PatSuffix
	PatGlob
	PatRe
)

type TextPattern struct {
	Content   string
	Re        *regexp.Regexp
	Lowercase bool

	TextPatternType
}

func (t TextPattern) Match(msg string) bool {
	if t.Lowercase {
		msg = strings.ToLower(msg)
	}
	switch t.TextPatternType {
	case PatLiteral:
		if strings.Contains(msg, t.Content) {
			return true
		}
	case PatPrefix:
		if strings.HasPrefix(msg, t.Content) {
			return true
		}
	case PatSuffix:
		if strings.HasSuffix(msg, t.Content) {
			return true
		}
	case PatGlob:
		if glob.Glob(t.Content, msg) {
			return true
		}
	case PatRe:
		if t.Re != nil && t.Re.MatchString(msg) {
			return true
		}
	}
	return false
}
