package condition

import (
	"fmt"

	"github.com/markuskont/go-sigma-rule-engine/pkg/match"
	"github.com/markuskont/go-sigma-rule-engine/pkg/types"
)

func parseSearch(t tokens, data types.Detection) (match.Branch, error) {
	fmt.Printf("Parsing %+v\n", t)

	// seek to LPAR -> store offset set balance as 1
	// seek from offset to end -> increment balance when encountering LPAR, decrement when encountering RPAR
	// increment group count on every decrement
	// stop when balance is 0, error of EOF if balance is positive or negative
	// if group count is > 0, fill sub brances via recursion
	// finally, build branch from identifiers and logic statements

	var balance, groups int
	var from, to int
	to = len(t) - 1
	for i, item := range t {
		switch item.T {
		case SepLpar:
			balance++
			if groups == 0 {
				from = i
			}
		case SepRpar:
			balance--
			if balance == 0 {
				groups++
				to = i
			}
		}
	}

	if balance > 0 || balance < 0 {
		return nil, fmt.Errorf("Broken rule group")
	}
	fmt.Printf("got %d groups between %d and %d\n", groups, from, to)

	return nil, nil
}

type parser struct {
	lex *lexer

	// maintain a list of collected and validated tokens
	tokens

	// memorize last token to validate proper sequence
	// for example, two identifiers have to be joined via logical AND or OR, otherwise the sequence is invalid
	previous Token

	// sigma detection map that contains condition query and relevant fields
	sigma types.Detection

	// for debug
	condition string

	// sigma condition rules
	rules []interface{}
}

func (p *parser) run() error {
	if p.lex == nil {
		return fmt.Errorf("cannot run condition parser, lexer not initialized")
	}
	// Pass 1: collect tokens, do basic sequence validation and collect sigma fields
	if err := p.collectAndValidateTokenSequences(); err != nil {
		return err
	}
	// Pass 2: find groups
	fmt.Println("------------------")
	parseSearch(p.tokens, p.sigma)
	return nil
}

func (p *parser) collectAndValidateTokenSequences() error {
	for item := range p.lex.items {

		if item.T == TokUnsupp {
			return types.ErrUnsupportedToken{Msg: item.Val}
		}
		if !validTokenSequence(p.previous, item.T) {
			return fmt.Errorf(
				"invalid token sequence %s -> %s. Value: %s",
				p.previous,
				item.T,
				item.Val,
			)
		}
		if item.T != LitEof {
			p.tokens = append(p.tokens, item)
		}
		p.previous = item.T
	}
	if p.previous != LitEof {
		return fmt.Errorf("last element should be EOF, got %s", p.previous.String())
	}
	return nil
}
