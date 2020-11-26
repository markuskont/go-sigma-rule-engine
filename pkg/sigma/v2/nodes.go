package sigma

// NodeSimpleAnd is a list of matchers connected with logical conjunction
type NodeSimpleAnd []Branch

// Match implements Matcher
func (n NodeSimpleAnd) Match(e Event) (bool, bool) {
	for _, b := range n {
		match, applicable := b.Match(e)
		if !match || !applicable {
			return match, applicable
		}
	}
	return true, true
}

// Reduce cleans up unneeded slices
// Static structures can be used if node only holds one or two elements
// Avoids pointless runtime loops
func (n NodeSimpleAnd) Reduce() Branch {
	if len(n) == 1 {
		return n[0]
	}
	if len(n) == 2 {
		return &NodeAnd{L: n[0], R: n[1]}
	}
	return n
}

// NodeSimpleOr is a list of matchers connected with logical disjunction
type NodeSimpleOr []Branch

// Reduce cleans up unneeded slices
// Static structures can be used if node only holds one or two elements
// Avoids pointless runtime loops
func (n NodeSimpleOr) Reduce() Branch {
	if len(n) == 1 {
		return n[0]
	}
	if len(n) == 2 {
		return &NodeOr{L: n[0], R: n[1]}
	}
	return n
}

// Match implements Matcher
func (n NodeSimpleOr) Match(e Event) (bool, bool) {
	var oneApplicable bool
	for _, b := range n {
		match, applicable := b.Match(e)
		if match {
			return true, true
		}
		if applicable {
			oneApplicable = true
		}
	}
	return false, oneApplicable
}

// NodeNot negates a branch
type NodeNot struct {
	B Branch
}

// Match implements Matcher
func (n NodeNot) Match(e Event) (bool, bool) {
	match, applicable := n.B.Match(e)
	if !applicable {
		return match, applicable
	}
	return !match, applicable
}

// NodeAnd is a two element node of a binary tree with Left and Right branches
// connected via logical conjunction
type NodeAnd struct {
	L, R Branch
}

// Match implements Matcher
func (n NodeAnd) Match(e Event) (bool, bool) {
	lMatch, lApplicable := n.L.Match(e)
	if !lMatch {
		return false, lApplicable
	}
	rMatch, rApplicable := n.R.Match(e)
	return lMatch && rMatch, lApplicable && rApplicable
}

// NodeOr is a two element node of a binary tree with Left and Right branches
// connected via logical disjunction
type NodeOr struct {
	L, R Branch
}

// Match implements Matcher
func (n NodeOr) Match(e Event) (bool, bool) {
	lMatch, lApplicable := n.L.Match(e)
	if lMatch {
		return true, lApplicable
	}
	rMatch, rApplicable := n.R.Match(e)
	return lMatch || rMatch, lApplicable || rApplicable
}

func newNodeNotIfNegated(b Branch, negated bool) Branch {
	if negated {
		return &NodeNot{B: b}
	}
	return b
}

// TODO - use these functions to create binary trees instead of dunamic slices
func newConjunction(s NodeSimpleAnd) Branch {
	if l := len(s); l == 1 || l == 2 {
		return s.Reduce()
	}
	return &NodeAnd{
		L: s[0],
		R: newConjunction(s[1:]),
	}
}

func newDisjunction(s NodeSimpleOr) Branch {
	if l := len(s); l == 1 || l == 2 {
		return s.Reduce()
	}
	return &NodeOr{
		L: s[0],
		R: newDisjunction(s[1:]),
	}
}
