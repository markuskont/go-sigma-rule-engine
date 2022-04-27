package sigma

import "testing"

type LexTestCase struct {
	Expr   string
	Tokens []Token
}

var LexPosCases = []LexTestCase{
	{
		Expr:   "selection",
		Tokens: []Token{TokIdentifier, TokLitEof},
	},
	{
		Expr: "selection_1 and not filter_0",
		Tokens: []Token{
			TokIdentifier, TokKeywordAnd, TokKeywordNot, TokIdentifier, TokLitEof,
		},
	},
	{
		Expr: "((selection_1 and not filter_0) OR (keyword_0 and not filter1)) or idontcare",
		Tokens: []Token{
			TokSepLpar, TokSepLpar, TokIdentifier, TokKeywordAnd, TokKeywordNot, TokIdentifier,
			TokSepRpar, TokKeywordOr, TokSepLpar, TokIdentifier, TokKeywordAnd, TokKeywordNot,
			TokIdentifier, TokSepRpar, TokSepRpar, TokKeywordOr, TokIdentifier, TokLitEof,
		},
	},
	{
		Expr: "all of selection* and not 1 of filter* | count() > 10",
		Tokens: []Token{
			TokStAll, TokIdentifierWithWildcard, TokKeywordAnd, TokKeywordNot, TokStOne,
			TokIdentifierWithWildcard, TokSepPipe, TokUnsupp, TokIdentifier, TokLitEof,
		},
	},
}

func TestLex(t *testing.T) {
	for j, c := range LexPosCases {
		l := lex(c.Expr)
		var i int
		for item := range l.items {
			if item.T != c.Tokens[i] {
				t.Fatalf(
					"lex case %d expr %s failed on item %d expected %s got %s",
					j, c.Expr, i, c.Tokens[i].String(), item.T.String())
			}
			i++
		}
	}
}
