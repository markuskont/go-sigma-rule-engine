package sigma

import "fmt"

func newBranch(d Detection, t []Item) (Branch, error) {
	group := &group{expr: t}
	if err := group.getSub(); err != nil {
		return nil, err
	}
	if !group.hasSub() {
		for _, o := range group.splitByOr() {
			and := make([]Branch, 0)
			for _, item := range group.expr[o.start:o.end] {
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
					and = append(and, b)
				default:
					panic("TODO")
				}
			}
		}
	}
	return nil, ErrWip{}
}

// offsets denote beginning and end of a logical expresison group
type offsets struct {
	start, end int
}

type group struct {
	sub  []offsets
	expr []Item
}

func (g *group) getSub() error {
	g.sub = make([]offsets, 0)
	var balance int
	//last := len(g.expr) - 1
	var start, end int
	for i, item := range g.expr {
		switch item.T {
		case TokSepLpar:
			balance++
			start = i
		case TokSepRpar:
			balance--
			if balance == 0 {
				end = i
				g.sub = append(g.sub, offsets{
					start: start + 1,
					end:   end - 1,
				})
			}
		}
	}
	if balance != 0 {
		return fmt.Errorf("invalid expr group balance %d", balance)
	}
	return nil
}

func (g group) splitByOr() []offsets {
	out := make([]offsets, 0)
	var start int
	for i, tok := range g.expr {
		if tok.T == TokKeywordOr {
			out = append(out, offsets{
				start: start,
				end:   i - 1,
			})
			start = i + 1
		}
		if i == len(g.expr)-1 {
			out = append(out, offsets{
				start: start,
				end:   i,
			})
		}
	}
	return out
}

func (g group) hasSub() bool {
	if g.sub != nil && len(g.sub) > 0 {
		return true
	}
	return false
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
