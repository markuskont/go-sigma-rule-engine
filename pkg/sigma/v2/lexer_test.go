package sigma

import "testing"

type LexTestCase struct {
	Expr   string
	Tokens []Token
}

var LexPosCases = []LexTestCase{
	{
		Expr:   "selection",
		Tokens: []Token{Identifier, LitEof},
	},
	{
		Expr: "selection_1 AND NOT filter_0",
		Tokens: []Token{
			Identifier,
			KeywordAnd,
			KeywordNot,
			Identifier,
			LitEof,
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
