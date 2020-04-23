package sigma

import "fmt"

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
	rx := s.Fields()
	ast := &Tree{}
	r := <-rx
	root, err := newRuleMatcherFromIdent(&r, false)
	if err != nil {
		return nil, err
	}
	ast.Root = root
	return ast, nil
}
