package sigma

import "fmt"

// Tree represents the full AST for a sigma rule
type Tree struct {
	Root Branch
}

// Match implements Matcher
func (t Tree) Match(e Event) bool {
	return t.Root.Match(e)
}

// NewTree parses rule handle into an abstract syntax tree
func NewTree(r *RuleHandle) (*Tree, error) {
	if r == nil {
		return nil, fmt.Errorf("Missing rule handle")
	}
	if r.Detection == nil {
		return nil, ErrMissingDetection{}
	}
	expr, ok := r.Detection["condition"].(string)
	if !ok {
		return nil, ErrMissingCondition{}
	}

	p := &parser{
		lex:       lex(expr),
		condition: expr,
		sigma:     r.Detection,
	}
	if err := p.run(); err != nil {
		return nil, err
	}
	t := &Tree{Root: p.result}
	return t, nil
}

// newBranch builds a binary tree from token list
// sequence and group validation should be done before invoking newBranch
func newBranch(d Detection, t []Item, depth int) (Branch, error) {
	rx := genItems(t)

	and := make(NodeSimpleAnd, 0)
	or := make(NodeSimpleOr, 0)
	var negated bool

	for item := range rx {
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
			b, err := newBranch(d, extractGroup(rx), depth+1)
			if err != nil {
				return nil, err
			}
			and = append(and, newNodeNotIfNegated(b, negated))
			negated = false
		case TokIdentifierWithWildcard, TokIdentifierAll:
			// TODO
			return nil, ErrWip{}
		case TokStAll, TokStOne:
			// TODO
			return nil, ErrWip{}
		case TokSepRpar:
			return nil, fmt.Errorf("parser error, should not see %s",
				TokSepRpar)
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
	// fn is called when newBranch hits TokSepLpar
	// it will be consumed, so balance is already 1
	balance := 1
	group := make([]Item, 0)
	for item := range rx {
		if balance > 0 {
			group = append(group, item)
		}
		switch item.T {
		case TokSepLpar:
			balance++
		case TokSepRpar:
			balance--
			if balance == 0 {
				return group[:len(group)-1]
			}
		default:
		}
	}
	return group
}
