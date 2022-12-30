package parse

import (
	"errors"

	"github.com/zalgonoise/lex"
)

const (
	maxBackup = 5
)

var (
	// ErrInvalidID is a preset error for invalid node IDs
	ErrInvalidID = errors.New("invalid node ID")
	// ErrInvalidID is a preset error for non-existing nodes
	ErrNotFound = errors.New("node was not found")
	// ErrInvalidID is a preset error for cyclical edges in the graph
	ErrCyclicalEdge = errors.New("cyclical edges are not allowed")
)

// BackupSlot is a (weightless) enum type to create defined containers
// for node IDs, so they can be reused or referenced
type BackupSlot struct{}

var (
	Slot0 BackupSlot = struct{}{}
	Slot1 BackupSlot = struct{}{}
	Slot2 BackupSlot = struct{}{}
	Slot3 BackupSlot = struct{}{}
	Slot4 BackupSlot = struct{}{}
)

// Tree is a generic tree data structure to represent a lexical tree
//
// The Tree will buffer tokens of type T from a Lexer, identified by the same
// type of comparable tokens. The parser runs in tandem with a lexer -- as the
// parser advances to the next token, it is actually consuming a token from the lexer
// by calling its `lexer.NextItem()` method.
//
// A Tree exposes methods for creating and moving around Nodes, and to consume, buffer and
// backup lex Items as it converts them into nodes in the Tree.
//
// A Tree will store every node it contains, nested within the Root node. To navigate through
// the nodes in a Tree, the Tree stores (and exports) a Root element, pointing to this Root node.
//
// The Root node contains all top-level items in lexical order, which may or may not have
// edges themselves. It is the responsibility of the caller to ensure that the Tree is
// navigated through entirely, when processing it.
type Tree[C comparable, T any] struct {
	Root *Node[C, T]

	node    *Node[C, T]
	items   []lex.Item[C, T]
	lex     lex.Lexer[C, T]
	peek    int
	backup  map[BackupSlot]*Node[C, T]
	parseFn ParseFn[C, T]
}

// New creates a parse.Tree with the input lex.Lexer `l` and ParseFn `initParse`,
// initialized with a root node with type T `typ` and values V `values`, on position `-1`.
func New[C comparable, T any](
	l lex.Lexer[C, T],
	initParse ParseFn[C, T],
	typ C,
	values ...T,
) *Tree[C, T] {
	t := &Tree[C, T]{
		items:   make([]lex.Item[C, T], maxBackup),
		lex:     l,
		peek:    0,
		backup:  map[BackupSlot]*Node[C, T]{},
		parseFn: initParse,
	}
	t.Root = t.Node(lex.NewItem(-1, typ, values...))
	t.node = t.Root
	return t
}

// Next consumes and returns the next Item from the lexer
func (t *Tree[C, T]) Next() lex.Item[C, T] {
	if t.peek > 0 {
		t.peek--
	} else {
		t.items[0] = t.lex.NextItem()
	}
	return t.items[t.peek]
}

// Peek returns but does not consume the next Item from the lexer
func (t *Tree[C, T]) Peek() lex.Item[C, T] {
	if t.peek > 0 {
		return t.items[t.peek-1]
	}
	t.peek = 1

	t.items[0] = t.lex.NextItem()
	return t.items[0]
}

// Backup backs the stream up `n` Items
//
// The zeroth Item is already there. Order must be most recent -> oldest
func (t *Tree[C, T]) Backup(items ...lex.Item[C, T]) {
	for idx, item := range items {
		if idx+1 >= maxBackup {
			break
		}
		t.items[idx+1] = item
	}
	t.peek = len(items)
}

// Parse iterates through the incoming lex Items, by calling its `ParseFn`s, until all tokens
// are consumed and parsed into the Tree
func (t *Tree[C, T]) Parse() {
	for t.parseFn != nil {
		t.parseFn = t.parseFn(t)
	}
}
