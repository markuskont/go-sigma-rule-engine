package sigma

import (
	"fmt"
)

type parser struct {
	// lexer that tokenizes input string
	lex *lexer

	// container for collected tokens and their values
	tokens []Item

	// memorize last token to validate proper sequence
	// for example, two identifiers have to be joined via logical AND or OR, otherwise the sequence is invalid
	previous Item

	// sigma detection map that contains condition query and relevant fields
	sigma Detection

	// for debug
	condition string

	// resulting rule that can be collected later
	result Branch

	// if true, stops the parser from collapsing whitespace in non-regex rules (default is false to collapse)
	// and the data that will be matched against them; default is to collapse whitespace to allow for better
	// matching in the event that a bad actor attempts to pad whitespace inot a command to fool the engine
	noCollapseWS bool
}

func (p *parser) run() error {
	if p.lex == nil {
		return fmt.Errorf("cannot run condition parser, lexer not initialized")
	}
	// Pass 1: collect tokens, do basic sequence validation and collect sigma fields
	if err := p.collect(); err != nil {
		return err
	}
	// Pass 2: find groups
	if err := p.parse(); err != nil {
		return err
	}
	return nil
}

func (p *parser) parse() error {
	res, err := newBranch(p.sigma, p.tokens, 0, p.noCollapseWS)
	if err != nil {
		return err
	}
	p.result = res
	return nil
}

// collect gathers all items from lexer and does preliminary sequence validation
func (p *parser) collect() error {
	for item := range p.lex.items {
		if item.T == TokUnsupp {
			return ErrUnsupportedToken{Msg: item.Val}
		}
		if p.previous.T != TokBegin && !validTokenSequence(p.previous.T, item.T) {
			return ErrInvalidTokenSeq{
				Prev:      p.previous,
				Next:      item,
				Collected: p.tokens,
			}
		}
		if item.T != TokLitEof {
			p.tokens = append(p.tokens, item)
		}
		p.previous = item
	}
	if p.previous.T != TokLitEof {
		return ErrIncompleteTokenSeq{
			Expression: p.condition,
			Items:      p.tokens,
			Last:       p.previous,
		}
	}
	return nil
}
