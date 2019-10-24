// Package condition provides library to parse sigma rule condition field into rule tokens
package condition

import (
	"strings"
	"unicode"
)

type Item struct {
	T   Token
	Val string
}

func checkKeyWord(in string) Token {
	if len(in) == 0 {
		return TokErr
	}
	switch strings.ToLower(in) {
	case KeywordAnd.Literal():
		return KeywordAnd
	case KeywordOr.Literal():
		return KeywordOr
	case KeywordNot.Literal():
		return KeywordNot
	case "sum", "min", "max", "count", "avg":
		return KeywordAgg
	case KeywordThem.Literal():
		return KeywordThem
	default:
		if strings.Contains(in, "*") {
			return IdentifierWithWildcard
		}
		return Identifier
	}
}

var eof = rune(0)

// stateFn is a function that is specific to a state within the string.
type stateFn func(*lexer) stateFn

func lexStatement(l *lexer) stateFn {
	return lexText
}

func lexOneOf(l *lexer) stateFn {
	l.position += len(StOne.Literal())
	l.emit(StOne)
	return lexText
}

func lexAllOf(l *lexer) stateFn {
	l.position += len(StAll.Literal())
	l.emit(StAll)
	return lexText
}

func lexAggs(l *lexer) stateFn {
	return nil
}

// lexText scans what is expected to be text.
func lexText(l *lexer) stateFn {
	for {
		if strings.HasPrefix(l.todo(), StOne.Literal()) {
			return lexOneOf
		}
		if strings.HasPrefix(l.todo(), StAll.Literal()) {
			return lexAllOf
		}
		r := l.next()
		switch {
		case r == eof:
			if l.position > l.start {
				l.emit(checkKeyWord(l.collected()))
			}
			return nil
		case r == SepRpar.Rune():
			// emit any text we've accumulated.
			if l.position > l.start {
				l.emit(checkKeyWord(l.collected()))
			}
			l.emit(SepRpar)
			// TODO - entering a subsection resets the whole lookup order
			return lexText
		case r == SepLpar.Rune():
			l.emit(SepLpar)
			// TODO - entering a subsection resets the whole lookup order
			return lexText
		case r == SepPipe.Rune():
			l.emit(SepPipe)
			return lexAggs
		case unicode.IsSpace(r):
			l.backup()
			// emit any text we've accumulated.
			if l.position > l.start {
				l.emit(checkKeyWord(l.collected()))
			}
			return lexWhitespace
		}
	}
}

// lexWhitespace scans what is expected to be whitespace.
func lexWhitespace(l *lexer) stateFn {
	for {
		r := l.next()
		switch {
		case r == eof:
			return nil
		case !unicode.IsSpace(r):
			l.backup()
			return lexText
		default:
			l.ignore()
		}
	}
}
