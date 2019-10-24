package condition

import (
	"fmt"
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
	for fn := lexText; fn != nil; {
		fn = fn(l)
	}
	close(l.items)
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
