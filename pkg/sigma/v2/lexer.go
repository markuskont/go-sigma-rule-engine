package sigma

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

type lexer struct {
	input    string    // we'll store the string being parsed
	start    int       // the position we started scanning
	position int       // the current position of our scan
	width    int       // we'll be using runes which can be double byte
	items    chan Item // the channel we'll use to communicate between the lexer and the parser
}

// lex creates a lexer and starts scanning the provided input.
func lex(input string) *lexer {
	l := &lexer{
		input: input,
		items: make(chan Item, 0),
	}
	go l.scan()
	return l
}

// ignore resets the start position to the current scan position effectively
// ignoring any input.
func (l *lexer) ignore() {
	l.start = l.position
}

// next advances the lexer state to the next rune.
func (l *lexer) next() (r rune) {
	if l.position >= len(l.input) {
		l.width = 0
		return eof
	}

	r, l.width = utf8.DecodeRuneInString(l.todo())
	l.position += l.width
	return r
}

// backup allows us to step back one rune which is helpful when you've crossed
// a boundary from one state to another.
func (l *lexer) backup() {
	l.position = l.position - 1
}

// scan will step through the provided text and execute state functions as
// state changes are observed in the provided input.
func (l *lexer) scan() {
	// When we begin processing, let's assume we're going to process text.
	// One state function will return another until `nil` is returned to signal
	// the end of our process.
	for fn := lexCondition; fn != nil; {
		fn = fn(l)
	}
	close(l.items)
}

func (l *lexer) unsuppf(format string, args ...interface{}) stateFn {
	msg := fmt.Sprintf(format, args...)
	l.items <- Item{T: TokUnsupp, Val: msg}
	return nil
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	msg := fmt.Sprintf(format, args...)
	l.items <- Item{T: TokErr, Val: msg}
	return nil
}

// emit sends a item over the channel so the parser can collect and manage
// each segment.
func (l *lexer) emit(k Token) {
	i := Item{T: k, Val: l.input[l.start:l.position]}
	l.items <- i
	l.ignore() // reset our scanner now that we've dispatched a segment
}

func (l lexer) collected() string { return l.input[l.start:l.position] }
func (l lexer) todo() string      { return l.input[l.position:] }

// stateFn is a function that is specific to a state within the string.
type stateFn func(*lexer) stateFn

// lexCondition scans what is expected to be text.
func lexCondition(l *lexer) stateFn {
	for {
		// TODO - run these cheks only if we have accumulated a word, not on every char
		if strings.HasPrefix(l.todo(), TokStOne.Literal()) {
			return lexOneOf
		}
		if strings.HasPrefix(l.todo(), TokStAll.Literal()) {
			return lexAllOf
		}
		switch r := l.next(); {
		case r == eof:
			return lexEOF
		case r == TokSepRpar.Rune():
			return lexRparWithTokens
		case r == TokSepLpar.Rune():
			return lexLpar
		case r == TokSepPipe.Rune():
			return lexPipe
		case unicode.IsSpace(r):
			return lexAccumulateBeforeWhitespace
		}
	}
}

func lexStatement(l *lexer) stateFn {
	return lexCondition
}

func lexOneOf(l *lexer) stateFn {
	l.position += len(TokStOne.Literal())
	l.emit(TokStOne)
	return lexCondition
}

func lexAllOf(l *lexer) stateFn {
	l.position += len(TokStAll.Literal())
	l.emit(TokStAll)
	return lexCondition
}

func lexAggs(l *lexer) stateFn {
	return l.unsuppf("aggregation not supported yet [%s]", l.input)
}

func lexEOF(l *lexer) stateFn {
	if l.position > l.start {
		l.emit(checkKeyWord(l.collected()))
	}
	l.emit(TokLitEof)
	return nil
}

func lexPipe(l *lexer) stateFn {
	l.emit(TokSepPipe)
	return lexAggs
}

func lexLpar(l *lexer) stateFn {
	l.emit(TokSepLpar)
	return lexCondition
}

func lexRparWithTokens(l *lexer) stateFn {
	// emit any text we've accumulated.
	if l.position > l.start {
		l.backup()
		// There may be N whitespace chars between token RPAR
		// TODO - may be a more concise way to do this, right now loops like this are everywhere

		if t := checkKeyWord(l.collected()); t != TokNil {
			l.emit(t)
		}

		for {
			switch r := l.next(); {
			case r == eof:
				return lexEOF
			case unicode.IsSpace(r):
				l.ignore()
			default:
				return lexRpar
			}
		}
	}
	return lexRpar
}

func lexRpar(l *lexer) stateFn {
	l.emit(TokSepRpar)
	return lexCondition
}

func lexAccumulateBeforeWhitespace(l *lexer) stateFn {
	l.backup()
	// emit any text we've accumulated.
	if l.position > l.start {
		l.emit(checkKeyWord(l.collected()))
	}
	return lexWhitespace
}

// lexWhitespace scans what is expected to be whitespace.
func lexWhitespace(l *lexer) stateFn {
	for {
		switch r := l.next(); {
		case r == eof:
			return lexEOF
		case !unicode.IsSpace(r):
			l.backup()
			return lexCondition
		default:
			l.ignore()
		}
	}
}

func checkKeyWord(in string) Token {
	if len(in) == 0 {
		return TokNil
	}
	switch strings.ToLower(in) {
	case TokKeywordAnd.Literal():
		return TokKeywordAnd
	case TokKeywordOr.Literal():
		return TokKeywordOr
	case TokKeywordNot.Literal():
		return TokKeywordNot
	case "sum", "min", "max", "count", "avg":
		return TokKeywordAgg
	case TokIdentifierAll.Literal():
		return TokIdentifierAll
	default:
		if strings.Contains(in, "*") {
			return TokIdentifierWithWildcard
		}
		return TokIdentifier
	}
}
