# parse

*a generic parser library written in Go*

_______________

## Concept

`parse` is a parser library for Go, based on the concept of the [`text/template`](https://pkg.go.dev/text/template) lexer, as a generic implementation. The logic behind this parser is mostly based off of [Rob Pike](https://github.com/robpike)'s talk about [Lexical Scanning in Go](https://www.youtube.com/watch?v=HxaD_trXwRE), which is also seen in the standard library (in [`text/template/parse/parse.go`](https://cs.opensource.google/go/go/+/refs/tags/go1.19.4:src/text/template/parse/parse.go)).


The lexer is a state-machine that analyzes input data (split into single units) by traversing the slice and classifying *blobs* of the slice (as lexemes) with a certain token. The lexer emits items, which are composed of three main elements:
- the starting position in the slice where the lexeme starts
- the (comparable type) token classifying this piece of data
- a slice containing the actual data 

The lexer is usually in-sync with a parser (also a state-machine, running in tandem with a lexer), that will consume the items emited by the lexer to build a parse tree. The parse tree, as the name implies, is a tree graph data structure that will layout the received tokens with a certain path / structure, with some logic in mind. It is finally able to output the processed tree as an output type, configurable by the developer, too.

## Overview


The idea behind implementing a generic algorithm for a lexer and parser came from trying to build a graph (data structure) representing the logic blocks in a Go file. Watching the talk above was a breath of fresh air when it came to the design of the lexer and its simple approach. So, it would be nice to leverage this algorithm for the Go code graph idea from before. By making the logic generic, one could implement an `Item` type to hold a defined token type, and a set of (any type of) values and `StateFn` state-functions to tokenize input data. In concept this works for any given type, as the point is to label elements of a slice with identifying tokens, that will be processed into a parse tree (with a specific parser implementation).

The parser is a state-machine that works exactly the same way as the lexer, but with a slightly different direction. The parser consumes items emitted by a lexer one by one (with look-ahead capabilities), and builds a parse tree from them. The way this works is by running `ParseFn`s on the parser until all items from the lexer are depleted. This is the similarity with the lexer's logic, in terms of developer extensibility and the implementation of these functions.

Parsers will also be able to (optionally) convert the parse tree into a meaningful type, with the help of `ProcessFn`s and `NodeFn`s. Like `ParseFn`s and `StateFn`s, these will take in a `parse.Tree` and a `parse.Node` respectively, returning a generic type R (any type) and an error. The developer can either implement these processor functions or simply work with the `parse.Tree`.

## Installation 

> Note: this library is not ready out-of-the box! You will need to implement your own `ParseFn` parse-functions and optionally your own `ProcessFn` processor-functions with defined types. This repo will expose simple examples to understand the flow of the parser, below.

You can add this library to your Go project with a `go get` command:

```
go get github.com/zalgonoise/parse
```

## Features

### Entities

#### Tree

The Tree is the state-machine holding all the nodes as lexemes are parsed. It stores a (tree) graph data structure by storing (and exporting) the Root Node -- one that is created when the Tree is initialized -- which may or may not contain additional nodes as edges.

A Tree buffers the items emitted by a lexer in a slice (`Tree.items`) that is initialized with a certain size. It buffers the items because the slice only populates more than the first slot when looking-ahead; making this the least expensive approach possible.

The Tree also holds the `lex.Lexer` lexer, which it leverages when then next item is requested (by a `ParseFn`), as it is actually calling the lexer's `l.NextItem()` method.

Also, the Tree has a `map[BackupSlot]*Node[C, T]` field representing backup slots for nodes. The library exposes 5 `BackupSlot`s as an exported type, so the caller can store and load positions in the Tree when processing it.

Last but not least, just like the `lex.Lexer`, it holds a `ParseFn` that is called until `nil` is returned (similar to the `lex.StateFn`).

```go
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
```

#### Node 

A Node stores the `lex.Item` received from a lexer, modified or not. It will also store a pointer to its parent Node, as well as a list of edges, or child Nodes.

```go
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
```

#### ParseFn

Similar to the lexer's `lex.StateFn`, it is a recursive function called by the Tree which should consume the items received by the lexer, and organizing them in the parse Tree.

It is a defined type and the consumer of the library must implement their own logic to consume the lexemes emitted by the lexer.

```go
// ParseFn is similar to the Lexer's StateFn, as a recursive function that the Tree
// will keep calling during runtime until it runs out of items received from the Lexer
//
// The ParseFn will return another ParseFn that will keep processing the items; which
// could be done in a number of ways (switch statements, helper functions, etc). When
// `nil` is returned, the parser will stop processing lex items
type ParseFn[C comparable, T any] func(t *Tree[C, T]) ParseFn[C, T]
```

#### ProcessFn

The `ProcessFn` is a post-parsing function (a processor-function) that can be implemented in order to convert the input parse Tree into a meaningful data type R (any type).

In the context of parsing a string, type `T` would be a `rune`, and type `R` would be a `string`. It returns an error for seamless error handling.

```go
// ProcessFn is a function that can be executed after parsing all the items, and will
// return a known-good type for the developer to work on. This is a step taken after a
// Tree is built
type ProcessFn[C comparable, T any, R any] func(t *Tree[C, T]) (R, error)
```

#### NodeFn

A `NodeFn` is a function called by a `ProcessFn` implementation, to process a Node. Similar to `ProcessFn`, it is an optional type that serves as a building block for more complex or structured parsing and processing.

In the context of parsing Markdown into HTML, the Node `n` could contain *h1 item* or a *hyperlink item*; that could be reused in a Markdown-to-HTML converter.

```go
// NodeFn is a function that can be executed against a single node, when processing the
// parse.Tree
type NodeFn[C comparable, T any, R any] func(n *Node[C, T]) (R, error)
```

### Helpers

#### Run function

Run simplifies a few actions when running a converter, so that the consumer of the library only needs to provide:
- the data
- the lexer's `StateFn`
- the parser's `ParseFn`
- the parser's `ProcessFn`

```go
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
```

## Implementing

**Note**: *Example and tests can be found in the [`impl`](./impl/) directory; from the lexer to the parser*

________

### Token type

Implementing a Lexer requires considering the format of the input data and how it can be tokenized. For this example, the input data is a string, where the lexeme units will be runes.

> The `TemplateItem` will be a comparable (unique) TextToken type, where the lexemes will be runes

```go
// TemplateItem represents the lex.Item for a runes lexer based on TextToken identifiers
type TemplateItem[C TextToken, I rune] lex.Item[C, I]
```

For this, the developer needs to define a token type (with an enumeration of expected tokens, where the zero-value for the type is EOF).

> A set of expected tokens are enumerated. In this case the text template will take text
> between single braces (like `{this}`), and ...replace the braces with double angle-brackets
> (like `>>this<<`). Not very fancy but serves as an example.

```go
// TextToken is a unique identifier for this text template implementation
type TextToken int

const (
	TokenEOF TextToken = iota
	TokenError
	TokenIDENT
	TokenTEMPL
	TokenLBRACE
	TokenRBRACE
)
```


After defining the type, a (set of) `StateFn`(s) need to be created, in context of the input data and how it should be tokenized. Each `StateFn` will hold the responsibility of tokenizing a certain lexeme, and each `StateFn` will have a different flow and responsibility.

### Lexer and state functions

> `initState` switches on the next lexable unit's value, to either emit an item or simply return a new state. This state should be able to listen to all types of (supported) symbols since this example supports so (a user could start a template in the very first char, and end it on the last one)
>
> The checks for `l.Width() > 0` ensures that an existing *stack* is being pushed before 
> advancing to the next token in a different procedure (e.g., consider all identifier tokens
> before going into the `stateBRACE` routine)


```go
// initState describes the StateFn to kick off the lexer. It is also the default fallback StateFn
// for any other StateFn
func initState[C TextToken, T rune](l lex.Lexer[C, T]) lex.StateFn[C, T] {
	switch l.Next() {
	case '}':
		if l.Width() > 0 {
			l.Prev()
			l.Emit((C)(TokenIDENT))
		}
		l.Ignore()
		return stateRBRACE[C, T]
	case '{':
		if l.Width() > 0 {
			l.Prev()
			l.Emit((C)(TokenIDENT))
		}
		l.Ignore()
		return stateLBRACE[C, T]
	case 0:
		return nil
	default:
		return stateIDENT[C, T]
	}
}
```

> `stateIDENT` absorbs all text characters until it hits a `{`, `}` or EOF. Then, if the following 
> character is a `{`, or a `}` it returns the `stateLBRACE` or `stateRBRACE` routine, respectively. 
> If it hits EOF, it will return a EOF token and a nil `StateFn`.


```go
// stateIDENT describes the StateFn to parse text tokens.
func stateIDENT[C TextToken, T rune](l lex.Lexer[C, T]) lex.StateFn[C, T] {
	l.AcceptRun(func(item T) bool {
		return item != '}' && item != '{' && item != 0
	})
	switch l.Next() {
	case '}':
		if l.Width() > 0 {
			l.Prev()
			l.Emit((C)(TokenIDENT))
		}
		return stateRBRACE[C, T]
	case '{':
		if l.Width() > 0 {
			l.Prev()
			l.Emit((C)(TokenIDENT))
		}
		return stateLBRACE[C, T]
	default:
		if l.Width() > 0 {
			l.Emit((C)(TokenIDENT))
		}
		l.Emit((C)(TokenEOF))
		return nil
	}
}
```

> `stateLBRACE` tokenizes the `{` character, returning the initial state after skipping this character

```go
// stateLBRACE describes the StateFn to check for and emit an LBRACE token
func stateLBRACE[C TextToken, T rune](l lex.Lexer[C, T]) lex.StateFn[C, T] {
	l.Next() // skip this symbol
	l.Emit((C)(TokenLBRACE))
	return initState[C, T]
}
```

> Similarly, `stateRBRACE` tokenizes the `}` character:

```go
// stateRBRACE describes the StateFn to check for and emit an RBRACE token
func stateRBRACE[C TextToken, T rune](l lex.Lexer[C, T]) lex.StateFn[C, T] {
	l.Next() // skip this symbol
	l.Emit((C)(TokenRBRACE))
	return initState[C, T]
}
```

> Finally `stateError` tokenizes an error if found (none in this lexer's example)

```go
// stateError describes an errored state in the lexer / parser, ignoring this set of tokens and emitting an
// error item
func stateError[C TextToken, T rune](l lex.Lexer[C, T]) lex.StateFn[C, T] {
	l.Backup()
	l.Prev() // mark the previous char as erroring token
	l.Emit((C)(TokenError))
	return initState[C, T]
}
```

### Parser

#### Parse functions

> Just like the lexer, start by defining a top-level ParseFn that will scan for all expected tokens
>
> This function will peek into the next item from the lexer and return the appropriate ParseFn before actually consuming the token

```go
// initParse describes the ParseFn to kick off the parser. It is also the default fallback 
// for any other ParseFn
func initParse[C TextToken, T rune](t *parse.Tree[C, T]) parse.ParseFn[C, T] {
	for t.Peek().Type != C(TokenEOF) {
		switch t.Peek().Type {
		case (C)(TokenIDENT):
			return parseText[C, T]
		case (C)(TokenLBRACE), (C)(TokenRBRACE):
			return parseTemplate[C, T]
		}
	}
	return nil
}
```

> `parseText` simply consumes the item as a new node under the current.

```go
// parseText consumes the next item as a text token, creating a node for it under the
// current one in the tree. 
func parseText[C TextToken, T rune](t *parse.Tree[C, T]) parse.ParseFn[C, T] {
	t.Node(t.Next())
	return initParse[C, T]
}
```

> `parseTemplate` is a state where we're about to consume either a `{` or a `}`. 
>
> For `{` tokens, a template Node is created, as it will be a wrapper for one or more text or template items. Returns the initial state.
>
> For `}` tokens, the node that is parent to the `{` is set as the current position (closing the template)

```go
// parseTemplate creates a node for a template item, for which it expects both a text item edge
// that which also needs to contain an end-template edge.
//
// If it encounters a `}` token to close the template, it sets the position up three levels
// (back to the template's parent)
func parseTemplate[C TextToken, T rune](t *parse.Tree[C, T]) parse.ParseFn[C, T] {
	switch t.Peek().Type {
	case (C)(TokenLBRACE):
		t.Set(t.Parent())
		t.Node(t.Next())
	case (C)(TokenRBRACE):
		t.Node(t.Next())
		t.Set(t.Parent().Parent.Parent)
	}
	return initParse[C, T]
}
```

#### Process functions

> `processFn` is the *top-level* processor function, that will consume the nodes in the Tree.
>
> It will use a strings.Builder to create the returned string, and iterate through the Tree's
> root Node's edges and switching on its Type.
>
> The content written to the strings.Builder comes from the appropriate `NodeFn` for the Node type.

```go
// processFn is the ProcessFn that will process the Tree's Nodes, returning a string and an error
func processFn[C TextToken, T rune, R string](t *parse.Tree[C, T]) (R, error) {
	var sb = new(strings.Builder)
	for _, n := range t.List() {
		switch n.Type {
		case (C)(TokenIDENT):
			proc, err := processText[C, T, R](n)
			if err != nil {
				return (R)(sb.String()), err
			}
			sb.WriteString((string)(proc))
		case (C)(TokenLBRACE):
			proc, err := processTemplate[C, T, R](n)
			if err != nil {
				return (R)(sb.String()), err
			}
			sb.WriteString((string)(proc))
		}
	}

	return (R)(sb.String()), nil
}
```

> for text it's straight-forward, it just casts the T-type values as rune, and returns a string value of it

```go
// processText converts the T-type items into runes, and returns a string value of it
func processText[C TextToken, T rune, R string](n *parse.Node[C, T]) (R, error) {
	var val = make([]rune, len(n.Value), len(n.Value))
	for idx, r := range n.Value {
		val[idx] = (rune)(r)
	}
	return (R)(val), nil
}
```

> for templates, a few checks need to be made -- in this particular example it is to ensure that templates are terminated.
> 
> the `processTemplate ProcessFn` does that exactly -- it replaces the wrapper text with the appropriate content, adds in the text in the next node, and looks into that text node's edges for a `}` item (to mark the template as closed). Otherwise returns an error:

```go
// processTemplate prcesses the text within two template nodes
//
// Returns an error if a template is not terminated appropriately
func processTemplate[C TextToken, T rune, R string](n *parse.Node[C, T]) (R, error) {
	var sb = new(strings.Builder)
	var ended bool

	sb.WriteString(">>")
	for _, node := range n.Edges {
		switch node.Type {
		case (C)(TokenIDENT):
			proc, err := processText[C, T, R](node)
			if err != nil {
				return (R)(sb.String()), err
			}
			for _, e := range node.Edges {
				if e.Type == (C)(TokenRBRACE) {
					ended = true
				}
			}
			sb.WriteString((string)(proc))
		case (C)(TokenLBRACE):
			proc, err := processTemplate[C, T, R](node)
			if err != nil {
				return (R)(sb.String()), err
			}
			sb.WriteString((string)(proc))
		}
	}
	if !ended {
		return (R)(sb.String()), fmt.Errorf("parse error on line: %d", n.Pos)
	}

	sb.WriteString("<<")
	return (R)(sb.String()), nil
}
```

#### Wrapper

> Perfect! Now all components are wired-up among themselves, and it just needs a simple entrypoint function
>
> For this, we can use the template `Parse` function to run it all at once:

```go
// Run parses the input templated data (a string as []rune), returning
// a processed string and an error
func Run[C TextToken, T rune, R string](s []T) (R, error) {
	return parse.Run(
		s,
		initState[C, T],
		initParse[C, T],
		processFn[C, T, R],
	)
}
```

## Benchmarks

Performance is critical in a lexer or parser. I will add benchmarks (and performance improvements) very soon :)