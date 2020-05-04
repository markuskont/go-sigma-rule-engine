package sigma

import (
	"context"
	"fmt"
)

// newBranch builds a binary tree from token list
// sequence and group validation should be done before invoking newBranch
func newBranch(d Detection, t []Item) (Branch, error) {
	tx := make(chan Item, 0)
	go func(ctx context.Context) {
		defer close(tx)
		for _, item := range t {
			tx <- item
		}
	}(context.TODO())

	and := make(NodeSimpleAnd, 0)
	or := make(NodeSimpleOr, 0)
	var negated bool

	for item := range tx {
		switch item.T {
		case TokIdentifier:
			val, ok := d[item.Val]
			if !ok {
				return nil, ErrMissingConditionItem{Key: item.Val}
			}
			b, err := newRuleFromIdent(val, checkIdentType(item, val))
			if err != nil {
				return nil, err
			}
			and = append(and, newNodeNotIfNegated(b, negated))
			negated = false
		case TokKeywordAnd:
			// no need to do anything special here
		case TokKeywordOr:
			// fill OR gate with collected AND nodes
			// reduce will strip AND logic if only one token has been collected
			or = append(or, and.Reduce())
			// reset existing AND collector
			and = make(NodeSimpleAnd, 0)
		case TokKeywordNot:
			negated = true
		case TokSepLpar:
			// recursively create new branch and append to existing list
			// then skip to next token after grouping
			b, err := newBranch(d, extractGroup(tx))
			if err != nil {
				return nil, err
			}
			and = append(and, newNodeNotIfNegated(b, negated))
			negated = false
		case TokIdentifierWithWildcard, TokIdentifierAll:
			// TODO
			panic("TODO")
		case TokStAll, TokStOne:
			// TODO
			panic("TODO")
		default:
			return nil, ErrUnsupportedToken{
				Msg: fmt.Sprintf("%s | %s", item.T, item.T.Literal()),
			}
		}
	}
	or = append(or, newNodeNotIfNegated(and.Reduce(), negated))

	return or.Reduce(), nil
}

func extractGroup(rx <-chan Item) []Item {
	var balance int
	group := make([]Item, 0)
	for item := range rx {
		switch item.T {
		case TokSepLpar:
			if balance > 0 {
				group = append(group, item)
			}
			balance++
		case TokSepRpar:
			balance--
			if balance == 0 {
				return group
			} else if balance > 0 {
				group = append(group, item)
			}
		}
	}
	return group
}

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
	_, err := newBranch(p.sigma, p.tokens)
	if err != nil {
		return err
	}
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

/*
func ParseDetection(s Detection) (*Tree, error) {
	if s == nil {
		return nil, ErrMissingDetection{}
	}
	if len(s) < 3 {
		return parseSimpleScenario(s)
	}
	return parseComplexScenario(s)
}

func parseSimpleScenario(s Detection) (*Tree, error) {
	switch len(s) {
	case 1:
		// Simple case - should have only one search field, but should not have a condition field
		if c, ok := s["condition"].(string); ok {
			return nil, ErrIncompleteDetection{Condition: c}
		}
	case 2:
		// Simple case - one condition statement comprised of single IDENT that matches the second field name
		if c, ok := s["condition"].(string); !ok {
			return nil, ErrIncompleteDetection{Condition: "MISSING"}
		} else {
			if _, ok := s[c]; !ok {
				return nil, ErrIncompleteDetection{
					Condition: c,
					Msg:       fmt.Sprintf("Field %s defined in condition missing from map.", c),
					//Keys:      s.FieldSlice(),
				}
			}
		}
		delete(s, "condition")
	default:
		return nil, ErrMissingDetection{}
	}
	ast := &Tree{}
	rx := s.Fields()
	r := <-rx
	root, err := newRuleMatcherFromIdent(&r, false)
	if err != nil {
		return nil, err
	}
	ast.Root = root
	return ast, nil
}

func parseComplexScenario(s Detection) (*Tree, error) {
	// Complex case, time to build syntax tree out of condition statement
	raw, ok := s["condition"].(string)
	if !ok {
		return nil, ErrMissingCondition{}
	}
	p := &parser{
		lex:       lex(raw),
		sigma:     s,
		tokens:    make([]Item, 0),
		previous:  TokBegin,
		condition: raw,
	}
	if err := p.run(); err != nil {
		return nil, err
	}
	return &Tree{Root: p.result}, nil
}
*/
