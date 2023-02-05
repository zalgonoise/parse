package impl

import (
	"testing"

	"github.com/zalgonoise/lex"
	"github.com/zalgonoise/parse"
)

func BenchmarkLexParseAndProcess(b *testing.B) {
	b.Run("Simple", func(b *testing.B) {
		input := []rune(`with {tmpl}.`)
		var tree *parse.Tree[TextToken, rune]

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l := lex.New(initState[TextToken, rune], input)
			tree = parse.New(
				(lex.Emitter[TextToken, rune])(l),
				initParse[TextToken, rune],
				TokenEOF,
			)

			tree.Parse()
		}
		_ = tree
	})
	b.Run("Complex", func(b *testing.B) {
		input := []rune(`string with {template} in it even { in {twice} out } in a row, or {even} { more {examples} if necessary}.`)
		var tree *parse.Tree[TextToken, rune]

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			l := lex.New(initState[TextToken, rune], input)
			tree = parse.New(
				(lex.Emitter[TextToken, rune])(l),
				initParse[TextToken, rune],
				TokenEOF,
			)

			tree.Parse()
		}
		_ = tree
	})

}
