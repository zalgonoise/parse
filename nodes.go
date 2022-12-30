package parse

import (
	"github.com/zalgonoise/lex"
)

// Node is a generic tree data structure unit. It is presented as a bidirectional
// tree knowledge graph that starts with a root Node (one without a parent) that
// can have zero-to-many children.
//
// It holds a reference to its parent (so that ParseFns can return to the correct
// point in the tree), the item's (joined) lexemes, and edges (if any)
//
// Edges (child nodes) are defined in a list containing the same lexical order as
// received. This allows safely nesting one or mode nodes without losing context of
// the overall structure of the Nodes in the Tree
type Node[C comparable, T any] struct {
	lex.Item[C, T]
	Parent *Node[C, T]
	Edges  []*Node[C, T]
}

// Node creates a new node from the input Item `item`, returning the created Node
//
// This action updates the tree's position the the new node's ID, and increments the
// tree's `nextID` reference number
//
// It also automatically sets the parent to the node at the previous position index,
// adding the new node as an edge of it.
//
// Note: since creating a Node nests it under the previous Node, it is the responsibility
// of the caller to move back to the parent if that is the intention
func (t *Tree[C, T]) Node(item lex.Item[C, T]) *Node[C, T] {
	n := &Node[C, T]{
		Item:   item,
		Parent: t.node,
		Edges:  []*Node[C, T]{},
	}
	if t.node != nil {
		t.node.Edges = append(t.node.Edges, n)
	}
	t.node = n
	return n
}

// Store places the current node in the input BackupSlot `slot`, in the parse.Tree
//
// If the current position is invalid, the root node (index zero) will be placed instead;
// if that fails too, an error is returned
func (t *Tree[C, T]) Store(slot BackupSlot) {
	t.backup[slot] = t.node
}

// Load returns the node stored in the input BackupSlot `slot`, or nil if either its ID is
// invalid or if the slot is empty
//
// If siduccessful, this action will also clear the BackupSlot `slot`
func (t *Tree[C, T]) Load(slot BackupSlot) *Node[C, T] {
	n := t.backup[slot]
	delete(t.backup, slot)
	return n
}

// Jump sets the current position in the tree to the node ID loaded from the BackupSlot `slot`,
// returning an OK boolean and an error in case the node does not exist
//
// If successful, this action will also clear the BackupSlot `slot`
func (t *Tree[C, T]) Jump(slot BackupSlot) (bool, error) {
	t.node = t.backup[slot]
	delete(t.backup, slot)
	return true, nil
}

// Set places the input node's position as the current one in the Tree
func (t *Tree[C, T]) Set(n *Node[C, T]) error {
	if !t.exists(n) {
		return ErrNotFound
	}
	t.node = n
	return nil
}

// List returns all top-level nodes, under the Tree's Root
func (t *Tree[C, T]) List() []*Node[C, T] {
	return t.Root.Edges
}

// Cur returns the node at the current position in the tree
func (t *Tree[C, T]) Cur() *Node[C, T] {
	return t.node
}

// Parent returns the node that is parent to the one at the current position in the tree
func (t *Tree[C, T]) Parent() *Node[C, T] {
	return t.node.Parent
}

func (t *Tree[C, T]) exists(n *Node[C, T]) bool {
	if n == nil {
		return false
	}
	var p = n
	for p.Parent != nil {
		p = p.Parent
	}
	return p == t.Root
}
