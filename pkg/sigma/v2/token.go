package sigma

var eof = rune(0)

// Item is lexical token along with respective plaintext value
// Item is communicated between lexer and parser
type Item struct {
	T   Token
	Val string
}

// Token is a lexical token extracted from condition field
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

// String documents human readable textual value of token
// For visual debugging, so symbols will be written out and everything is uppercased
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

// Literal documents plaintext values of a token
// Uses special symbols and expressions, as used in a rule
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

// Rune returns UTF-8 numeric value of symbol
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

// validTokenSequence detects invalid token sequences
// not meant to be a perfect validator, simply a quick check before parsing
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
		case KeywordAnd, KeywordOr, KeywordNot, TokBegin, SepLpar:
			return true
		}
	case SepRpar:
		switch t1 {
		case Identifier, IdentifierAll, IdentifierWithWildcard, SepLpar, SepRpar:
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