package match

import "github.com/markuskont/go-sigma-rule/pkg/types"

type Dummy struct{}

func (d Dummy) Match(types.EventChecker) bool { return false }
func (d Dummy) Self() interface{}             { return d }

type Condition int

// Branch implements types.Matcher with additional methods for walking and debugging the tree
type Branch interface {
	types.Matcher
	// Self returns Node or final rule object for debugging and/or walking the tree
	// Must be type switched externally
	Self() interface{}
}

const (
	None Condition = iota
	And
	Or
	Not
	Null
)

type Node struct {
	id int

	Condition

	L Branch
	R Branch
}

func (m *Node) Match(obj types.EventChecker) bool {
	if m.L != nil || m.R != nil {
		return true
	}
	return false
}

type Tree struct {
	root *Node
}

func NewTree(root *Node) *Tree {
	if root == nil {
		root = &Node{
			id: 0,
			L:  Dummy{},
			R:  Dummy{},
		}
	}
	return &Tree{
		root: root,
	}
}

func (t Tree) Match(obj types.EventChecker) bool { return t.root.Match(obj) }
func (t Tree) Self() interface{}                 { return t.root }
