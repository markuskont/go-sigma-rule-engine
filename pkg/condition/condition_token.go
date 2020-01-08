package condition

import "fmt"

type Token int

const (
	TokErr Token = iota

	// Helpers for internal stuff
	TokUnsupp
	TokBegin
	TokNil

	// user-defined word
	Identifier
	IdentifierWithWildcard
	IdentifierAll

	// Literals
	LitEof

	// Separators
	SepLpar
	SepRpar
	SepPipe

	// Operators
	OpEq
	OpGt
	OpGte
	OpLt
	OpLte

	// Keywords
	KeywordAnd
	KeywordOr
	KeywordNot
	KeywordAgg

	// TODO
	KeywordNear
	KeywordBy

	// Statements
	StOne
	StAll
)

func (t Token) String() string {
	switch t {
	case Identifier, IdentifierWithWildcard:
		return "IDENT"
	case IdentifierAll:
		return "THEM"
	case SepLpar:
		return "LPAR"
	case SepRpar:
		return "RPAR"
	case SepPipe:
		return "PIPE"
	case OpEq:
		return "EQ"
	case OpGt:
		return "GT"
	case OpGte:
		return "GTE"
	case OpLt:
		return "LT"
	case OpLte:
		return "LTE"
	case KeywordAnd:
		return "AND"
	case KeywordOr:
		return "OR"
	case KeywordNot:
		return "NOT"
	case StAll:
		return "ALL"
	case StOne:
		return "ONE"
	case KeywordAgg:
		return "AGG"
	case LitEof:
		return "EOF"
	case TokErr:
		return "ERR"
	case TokUnsupp:
		return "UNSUPPORTED"
	case TokBegin:
		return "BEGINNING"
	case TokNil:
		return "NIL"
	default:
		return "Unk"
	}
}

func (t Token) Literal() string {
	switch t {
	case Identifier, IdentifierWithWildcard:
		return "keywords"
	case IdentifierAll:
		return "them"
	case SepLpar:
		return "("
	case SepRpar:
		return ")"
	case SepPipe:
		return "|"
	case OpEq:
		return "="
	case OpGt:
		return ">"
	case OpGte:
		return ">="
	case OpLt:
		return "<"
	case OpLte:
		return "<="
	case KeywordAnd:
		return "and"
	case KeywordOr:
		return "or"
	case KeywordNot:
		return "not"
	case StAll:
		return "all of"
	case StOne:
		return "1 of"
	case LitEof, TokNil:
		return ""
	default:
		return "Err"
	}
}

func (t Token) Rune() rune {
	switch t {
	case SepLpar:
		return '('
	case SepRpar:
		return ')'
	case SepPipe:
		return '|'
	default:
		return eof
	}
}

// detect invalid token sequences
func validTokenSequence(t1, t2 Token) bool {
	switch t2 {
	case StAll, StOne:
		switch t1 {
		case TokBegin, SepLpar, KeywordAnd, KeywordOr, KeywordNot:
			return true
		}
	case IdentifierAll:
		switch t1 {
		case StAll, StOne:
			return true
		}
	case Identifier, IdentifierWithWildcard:
		switch t1 {
		case SepLpar, TokBegin, KeywordAnd, KeywordOr, KeywordNot, StOne, StAll:
			return true
		}
	case KeywordAnd, KeywordOr:
		switch t1 {
		case Identifier, IdentifierAll, IdentifierWithWildcard, SepRpar:
			return true
		}
	case KeywordNot:
		switch t1 {
		case KeywordAnd, KeywordOr, SepLpar, TokBegin:
			return true
		}
	case SepLpar:
		switch t1 {
		case KeywordAnd, KeywordOr, KeywordNot, TokBegin:
			return true
		}
	case SepRpar:
		switch t1 {
		case Identifier, IdentifierAll, IdentifierWithWildcard, SepLpar:
			return true
		}
	case LitEof:
		switch t1 {
		case Identifier, IdentifierAll, IdentifierWithWildcard, SepRpar:
			return true
		}
	case SepPipe:
		switch t1 {
		case Identifier, IdentifierAll, IdentifierWithWildcard, SepRpar:
			return true
		}
	}
	return false
}

type tokens []Item

func (t tokens) splitByOr() []tokens {
	var start int

	rules := make([]tokens, 0)
	last := len(t) - 1

	if t.contains(KeywordOr) {
		for pos, item := range t {
			if item.T == KeywordOr || pos == last {
				switch pos {
				case last:
					rules = append(rules, func() tokens {
						if last > 0 && t[pos-1].T == KeywordNot {
							return t[pos-1:]
						}
						return t[pos:]
					}())
				default:
					rules = append(rules, t[start:pos])
					start = pos + 1
				}
			}
		}
	} else {
		rules = append(rules, t)
	}
	return rules
}

func (t tokens) len() int { return len(t) }
func (t tokens) lastIdx() int {
	return t.len() - 1
}
func (t tokens) tail(i int) tokens {
	if i < 0 || i > t.lastIdx() {
		return t
	}
	return t[i:]
}

func (t tokens) head(i int) tokens {
	if i < 0 || i > t.lastIdx() {
		return t
	}
	return t[:i]
}

func (t tokens) nextKeyword() (int, Token) {
	for i, item := range t {
		if item.T == KeywordAnd || item.T == KeywordOr {
			return i, item.T
		}
	}
	return -1, TokNil
}

func (t tokens) getTokenIndices(tok Token) []int {
	out := make([]int, 0)
	for i, item := range t {
		if item.T == tok {
			out = append(out, i)
		}
	}
	return out
}

func (t tokens) countAnd() int {
	var c int
	for _, item := range t {
		if item.T == KeywordAnd {
			c++
		}
	}
	return c
}

func (t tokens) countOr() int {
	var c int
	for _, item := range t {
		if item.T == KeywordOr {
			c++
		}
	}
	return c
}

func (t tokens) count(tok ...Token) []int {
	c := make([]int, len(t))
	for _, item := range t {
		for i, token := range tok {
			if item.T == token {
				c[i]++
			}
		}
	}
	return c
}

func (t tokens) isNegated() bool {
	if len(t) > 1 && t[0].T == KeywordNot {
		return true
	}
	return false
}

func (t tokens) index(tok Token) int {
	for i, item := range t {
		if item.T == tok {
			return i
		}
	}
	return -1
}

func (t tokens) reverseIndex(tok Token) int {
	for i := len(t) - 1; i > 0; i-- {
		if t[i].T == tok {
			return i
		}
	}
	return -1
}

func (t tokens) contains(tok Token) bool {
	for _, item := range t {
		if item.T == tok {
			return true
		}
	}
	return false
}

type offsets struct {
	From, To int
}

func (o *offsets) SetFrom(i int) *offsets {
	o.From = i
	return o
}
func (o *offsets) SetTo(i int) *offsets {
	o.To = i
	return o
}

func newGroupOffsetInTokens(t tokens) ([]offsets, bool, error) {
	if t == nil || t.len() == 0 {
		return nil, false, nil
	}
	if !t.contains(SepLpar) {
		return nil, false, nil
	}
	groups := make([]offsets, 0)
	var balance, found int
	for i, item := range t {
		switch item.T {
		case SepLpar:
			if balance == 0 {
				groups = append(groups, offsets{From: i + 1, To: -1})
			}
			balance++
		case SepRpar:
			balance--
			if balance == 0 {
				groups[found].SetTo(i)
				found++
			}
		}
	}
	if balance > 0 || balance < 0 {
		return groups, false, fmt.Errorf("Broken rule group")
	}
	return groups, true, nil
}
