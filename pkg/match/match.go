package match

import "github.com/markuskont/go-sigma-rule-engine/pkg/types"

type Condition int

// Branch implements types.Matcher with additional methods for walking and debugging the tree
type Branch interface {
	types.Matcher
	// Self returns Node or final rule object for debugging and/or walking the tree
	// Must be type switched externally
	Self() interface{}
	Identifier
}

// Identifier implements ID retreival and modification mechanisms for adding elements to and finding them from a tree
// That way, a node may be created and kept as interface yet knowing where to place it in a tree
type Identifier interface {
	// GetID implements Identifier
	GetID() int
	// SetID implements Identifier
	SetID(int)
}

const (
	None Condition = iota
	And
	Or
	Not
	Null
)

type Node struct {
	ID int

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

func (n Node) Self() interface{} { return n }

type Tree struct {
	root *Node
}

func (t Tree) Find(id int) interface{} {
	n := t.root.Self()
	run := true
	for run {
		switch n.(type) {
		case Node:
		default:
			run = false
		}
	}
	return n
}

/*
   def find(self, value):
       current_node = self.root
       while current_node is not None:
           if(current_node.v == value):
               return True
           elif(value < current_node.v):
               current_node = current_node.l
           else:
               current_node = current_node.r
       return False
*/

func NewTree(root *Node) *Tree {
	if root == nil {
		root = &Node{
			ID: 0,
		}
	}
	return &Tree{
		root: root,
	}
}

func (t Tree) Match(obj types.EventChecker) bool { return t.root.Match(obj) }
func (t Tree) Self() interface{}                 { return t.root }
