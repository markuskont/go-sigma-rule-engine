package condition

import (
	"fmt"

	"github.com/markuskont/go-sigma-rule-engine/pkg/types"
)

type parser struct {
	lex *lexer

	// maintain a list of collected and validated tokens
	tokens

	// memorize last token to validate proper sequence
	// for example, two identifiers have to be joined via logical AND or OR, otherwise the sequence is invalid
	previous Token

	// sigma detection map that contains condition query and relevant fields
	sigma map[string]interface{}

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
