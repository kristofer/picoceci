package lexer_test

import (
	"testing"

	"github.com/kristofer/picoceci/pkg/lexer"
)

func TestLexer_BasicTokens(t *testing.T) {
	tests := []struct {
		src      string
		wantKind lexer.Kind
		wantLit  string
	}{
		{"42", lexer.INTEGER, "42"},
		{"3.14", lexer.FLOAT, "3.14"},
		{"'hello'", lexer.STRING, "'hello'"},
		{"#foo", lexer.SYMBOL, "#foo"},
		{"#at:put:", lexer.SYMBOL, "#at:put:"},
		{"$A", lexer.CHARACTER, "$A"},
		{"true", lexer.BOOLLIT, "true"},
		{"false", lexer.BOOLLIT, "false"},
		{"nil", lexer.NILLIT, "nil"},
		{"self", lexer.SELF, "self"},
		{"super", lexer.SUPER, "super"},
		{"object", lexer.OBJECT, "object"},
		{"interface", lexer.INTERFACE, "interface"},
		{"compose", lexer.COMPOSE, "compose"},
		{"import", lexer.IMPORT, "import"},
		{"Counter", lexer.IDENTIFIER, "Counter"},
		{"at:", lexer.KEYWORD, "at:"},
		{"ifTrue:", lexer.KEYWORD, "ifTrue:"},
		{":=", lexer.ASSIGN, ":="},
		{".", lexer.DOT, "."},
		{";", lexer.SEMICOLON, ";"},
		{"^", lexer.CARET, "^"},
		{"|", lexer.PIPE, "|"},
		{"[", lexer.LBRACKET, "["},
		{"]", lexer.RBRACKET, "]"},
		{"(", lexer.LPAREN, "("},
		{")", lexer.RPAREN, ")"},
		{"{", lexer.LBRACE, "{"},
		{"}", lexer.RBRACE, "}"},
		{"+", lexer.BINOP, "+"},
		{"<=", lexer.BINOP, "<="},
		{"~=", lexer.BINOP, "~="},
	}

	for _, tt := range tests {
		t.Run(tt.src, func(t *testing.T) {
			l := lexer.NewString(tt.src)
			tok := l.Next()
			if tok.Kind != tt.wantKind {
				t.Errorf("kind: got %v, want %v", tok.Kind, tt.wantKind)
			}
			if tok.Literal != tt.wantLit {
				t.Errorf("literal: got %q, want %q", tok.Literal, tt.wantLit)
			}
			eof := l.Next()
			if eof.Kind != lexer.EOF {
				t.Errorf("expected EOF after token, got %v", eof.Kind)
			}
		})
	}
}

func TestLexer_HexInteger(t *testing.T) {
	l := lexer.NewString("16rFF")
	tok := l.Next()
	if tok.Kind != lexer.INTEGER {
		t.Fatalf("expected INTEGER, got %v", tok.Kind)
	}
	if tok.Literal != "16rFF" {
		t.Errorf("literal: got %q, want %q", tok.Literal, "16rFF")
	}
}

func TestLexer_StringWithEscapedQuote(t *testing.T) {
	l := lexer.NewString("'it''s'")
	tok := l.Next()
	if tok.Kind != lexer.STRING {
		t.Fatalf("expected STRING, got %v", tok.Kind)
	}
}

func TestLexer_ByteArray(t *testing.T) {
	l := lexer.NewString("#[1 2 255]")
	tok := l.Next()
	if tok.Kind != lexer.BYTEARRAY {
		t.Fatalf("expected BYTEARRAY, got %v", tok.Kind)
	}
}

func TestLexer_ArrayOpen(t *testing.T) {
	l := lexer.NewString("#(")
	tok := l.Next()
	if tok.Kind != lexer.ARRAYOPEN {
		t.Fatalf("expected ARRAYOPEN, got %v", tok.Kind)
	}
}

func TestLexer_Comment(t *testing.T) {
	l := lexer.NewString("\"a comment\" 42")
	tok := l.Next()
	if tok.Kind != lexer.INTEGER || tok.Literal != "42" {
		t.Errorf("expected INTEGER 42 after comment, got %v %q", tok.Kind, tok.Literal)
	}
}

func TestLexer_MultipleTokens(t *testing.T) {
	src := "x := 3 + 4."
	l := lexer.NewString(src)
	want := []lexer.Kind{
		lexer.IDENTIFIER, lexer.ASSIGN, lexer.INTEGER, lexer.BINOP, lexer.INTEGER, lexer.DOT, lexer.EOF,
	}
	for _, k := range want {
		tok := l.Next()
		if tok.Kind != k {
			t.Errorf("expected %v, got %v (lit=%q)", k, tok.Kind, tok.Literal)
		}
	}
}

func TestLexer_NoPanic_RandomInput(t *testing.T) {
	inputs := []string{
		"",
		"   ",
		"\x00\xff",
		"'unterminated",
		"$",
		"##",
		"::=",
	}
	for _, in := range inputs {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("lexer panicked on input %q: %v", in, r)
				}
			}()
			l := lexer.NewString(in)
			for i := 0; i < 100; i++ {
				if l.Next().Kind == lexer.EOF {
					break
				}
			}
		}()
	}
}

// FuzzLexer verifies that the lexer never panics on arbitrary input.
func FuzzLexer(f *testing.F) {
	seeds := []string{
		"42 factorial.",
		"'hello' size.",
		"#foo.",
		"$A.",
		"#[1 2 3].",
		"#(1 'two' #three).",
		"object Counter { | count | init [ count := 0 ] }",
		"[ :x | x + 1 ].",
		"",
		"'unterminated",
		"$",
		"##",
		"\x00\xff",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, data string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("lexer panicked on input %q: %v", data, r)
			}
		}()
		l := lexer.NewString(data)
		for i := 0; i < 10000; i++ {
			if l.Next().Kind == lexer.EOF {
				break
			}
		}
	})
}
