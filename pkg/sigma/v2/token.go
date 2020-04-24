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
	TokIdentifier
	TokIdentifierWithWildcard
	TokIdentifierAll

	// Literals
	TokLitEof

	// Separators
	TokSepLpar
	TokSepRpar
	TokSepPipe

	// Operators
	TokOpEq
	TokOpGt
	TokOpGte
	TokOpLt
	TokOpLte

	// Keywords
	TokKeywordAnd
	TokKeywordOr
	TokKeywordNot
	TokKeywordAgg

	// TODO
	TokKeywordNear
	TokKeywordBy

	// Statements
	TokStOne
	TokStAll
)

// String documents human readable textual value of token
// For visual debugging, so symbols will be written out and everything is uppercased
func (t Token) String() string {
	switch t {
	case TokIdentifier:
		return "IDENT"
	case TokIdentifierWithWildcard:
		return "WILDCARDIDENT"
	case TokIdentifierAll:
		return "THEM"
	case TokSepLpar:
		return "LPAR"
	case TokSepRpar:
		return "RPAR"
	case TokSepPipe:
		return "PIPE"
	case TokOpEq:
		return "EQ"
	case TokOpGt:
		return "GT"
	case TokOpGte:
		return "GTE"
	case TokOpLt:
		return "LT"
	case TokOpLte:
		return "LTE"
	case TokKeywordAnd:
		return "AND"
	case TokKeywordOr:
		return "OR"
	case TokKeywordNot:
		return "NOT"
	case TokStAll:
		return "ALL"
	case TokStOne:
		return "ONE"
	case TokKeywordAgg:
		return "AGG"
	case TokLitEof:
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
	case TokIdentifier, TokIdentifierWithWildcard:
		return "keywords"
	case TokIdentifierAll:
		return "them"
	case TokSepLpar:
		return "("
	case TokSepRpar:
		return ")"
	case TokSepPipe:
		return "|"
	case TokOpEq:
		return "="
	case TokOpGt:
		return ">"
	case TokOpGte:
		return ">="
	case TokOpLt:
		return "<"
	case TokOpLte:
		return "<="
	case TokKeywordAnd:
		return "and"
	case TokKeywordOr:
		return "or"
	case TokKeywordNot:
		return "not"
	case TokStAll:
		return "all of"
	case TokStOne:
		return "1 of"
	case TokLitEof, TokNil:
		return ""
	default:
		return "Err"
	}
}

// Rune returns UTF-8 numeric value of symbol
func (t Token) Rune() rune {
	switch t {
	case TokSepLpar:
		return '('
	case TokSepRpar:
		return ')'
	case TokSepPipe:
		return '|'
	default:
		return eof
	}
}

// validTokenSequence detects invalid token sequences
// not meant to be a perfect validator, simply a quick check before parsing
func validTokenSequence(t1, t2 Token) bool {
	switch t2 {
	case TokStAll, TokStOne:
		switch t1 {
		case TokBegin, TokSepLpar, TokKeywordAnd, TokKeywordOr, TokKeywordNot:
			return true
		}
	case TokIdentifierAll:
		switch t1 {
		case TokStAll, TokStOne:
			return true
		}
	case TokIdentifier, TokIdentifierWithWildcard:
		switch t1 {
		case TokSepLpar, TokBegin, TokKeywordAnd, TokKeywordOr, TokKeywordNot, TokStOne, TokStAll:
			return true
		}
	case TokKeywordAnd, TokKeywordOr:
		switch t1 {
		case TokIdentifier, TokIdentifierAll, TokIdentifierWithWildcard, TokSepRpar:
			return true
		}
	case TokKeywordNot:
		switch t1 {
		case TokKeywordAnd, TokKeywordOr, TokSepLpar, TokBegin:
			return true
		}
	case TokSepLpar:
		switch t1 {
		case TokKeywordAnd, TokKeywordOr, TokKeywordNot, TokBegin, TokSepLpar:
			return true
		}
	case TokSepRpar:
		switch t1 {
		case TokIdentifier, TokIdentifierAll, TokIdentifierWithWildcard, TokSepLpar, TokSepRpar:
			return true
		}
	case TokLitEof:
		switch t1 {
		case TokIdentifier, TokIdentifierAll, TokIdentifierWithWildcard, TokSepRpar:
			return true
		}
	case TokSepPipe:
		switch t1 {
		case TokIdentifier, TokIdentifierAll, TokIdentifierWithWildcard, TokSepRpar:
			return true
		}
	}
	return false
}
