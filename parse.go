package parse

import "github.com/zalgonoise/lex"

// ParseFn is similar to the Lexer's StateFn, as a recursive function that the Tree
// will keep calling during runtime until it runs out of items received from the Lexer
//
// The ParseFn will return another ParseFn that will keep processing the items; which
// could be done in a number of ways (switch statements, helper functions, etc). When
// `nil` is returned, the parser will stop processing lex items
type ParseFn[C comparable, T any] func(t *Tree[C, T]) ParseFn[C, T]

// ProcessFn is a function that can be executed after parsing all the items, and will
// return a known-good type for the developer to work on. This is a step taken after a
// Tree is built
type ProcessFn[C comparable, T any, R any] func(t *Tree[C, T]) (R, error)

// NodeFn is a function that can be executed against a single node, when processing the
// parse.Tree
type NodeFn[C comparable, T any, R any] func(n *Node[C, T]) (R, error)

// Run encapsulates the lexer and parser runtime into a single one-shot action
//
// The caller must supply the input data []T `input`, a lex.StateFn to kick-off the lexer,
// a ParseFn to kick-off the parser, and a ProcessFn to convert the parse.Tree into the
// desired return type (or an error)
func Run[C comparable, T any, R any](
	input []T,
	initStateFn lex.StateFn[C, T],
	initParseFn ParseFn[C, T],
	processFn ProcessFn[C, T, R],
) (R, error) {
	var rootEOF C
	l := lex.New(initStateFn, input)
	t := New(l, initParseFn, rootEOF)
	t.Parse()
	return processFn(t)
}
