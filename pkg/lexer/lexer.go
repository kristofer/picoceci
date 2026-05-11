// Package lexer implements the picoceci lexical analyser (tokenizer).
//
// The lexer converts picoceci source text (UTF-8) into a stream of Token
// values.  It follows the lexical rules in LANGUAGE_SPEC.md §2 and the
// formal grammar in docs/grammar.ebnf.
//
// Usage:
//
//	l := lexer.New(source)
//	for {
//	    tok := l.Next()
//	    if tok.Kind == lexer.EOF { break }
//	    fmt.Println(tok)
//	}
package lexer

import (
	"fmt"
	"unicode/utf8"
)

// Kind identifies the type of a lexical token.
type Kind uint8

const (
	// Literals
	INTEGER   Kind = iota // 42  16rFF  2r1010
	FLOAT                 // 3.14  1.5e-3
	STRING                // 'hello'
	SYMBOL                // #hello  #at:put:
	CHARACTER             // $A  $\n
	BYTEARRAY             // #[ 1 2 3 ]  (the whole literal)
	ARRAYOPEN             // #(
	BOOLLIT               // true / false (reserved word)
	NILLIT                // nil (reserved word)

	// Names
	IDENTIFIER // counter  Counter  x
	KEYWORD    // at:  put:  ifTrue:

	// Operators
	BINOP // +  -  *  /  <  <=  =  ~=  ,  @  ...

	// Punctuation
	DOT       // .
	SEMICOLON // ;
	CARET     // ^
	ASSIGN    // :=
	PIPE      // |
	LBRACKET  // [
	RBRACKET  // ]
	LPAREN    // (
	RPAREN    // )
	LBRACE    // {
	RBRACE    // }
	COLON     // :  (inside parameter list)

	// Reserved words (not IDENTIFIER)
	SELF        // self
	SUPER       // super
	THISCONTEXT // thisContext
	OBJECT      // object
	INTERFACE   // interface
	COMPOSE     // compose
	IMPORT      // import

	// Sentinel
	EOF
	ILLEGAL
)

// Token is a single lexical element with its source position.
type Token struct {
	Kind    Kind
	Literal string // raw source text for this token
	Line    int    // 1-based line number
	Col     int    // 1-based column (byte offset within line)
}

func (t Token) String() string {
	return fmt.Sprintf("Token{%s %q L%d:C%d}", kindName(t.Kind), t.Literal, t.Line, t.Col)
}

func kindName(k Kind) string {
	names := [...]string{
		"INTEGER", "FLOAT", "STRING", "SYMBOL", "CHARACTER",
		"BYTEARRAY", "ARRAYOPEN", "BOOLLIT", "NILLIT",
		"IDENTIFIER", "KEYWORD", "BINOP",
		"DOT", "SEMICOLON", "CARET", "ASSIGN", "PIPE",
		"LBRACKET", "RBRACKET", "LPAREN", "RPAREN", "LBRACE", "RBRACE", "COLON",
		"SELF", "SUPER", "THISCONTEXT", "OBJECT", "INTERFACE", "COMPOSE", "IMPORT",
		"EOF", "ILLEGAL",
	}
	if int(k) < len(names) {
		return names[k]
	}
	return "UNKNOWN"
}

// Lexer holds the state of the tokenizer.
type Lexer struct {
	src  []byte
	pos  int // current byte position
	line int
	col  int
}

// New creates a Lexer for the given source bytes.
func New(src []byte) *Lexer {
	return &Lexer{src: src, pos: 0, line: 1, col: 1}
}

// NewString creates a Lexer from a string.
func NewString(src string) *Lexer {
	return New([]byte(src))
}

// Next reads and returns the next token from the source.
// After EOF is returned, subsequent calls also return EOF.
func (l *Lexer) Next() Token {
	l.skipWhitespaceAndComments()

	if l.pos >= len(l.src) {
		return l.tok(EOF, "")
	}

	startLine, startCol := l.line, l.col
	_ = startLine
	_ = startCol

	ch := l.src[l.pos]

	switch {
	case ch == '\'':
		return l.readString()
	case ch == '$':
		return l.readCharacter()
	case ch == '#':
		return l.readHash()
	case ch == ':':
		return l.readColon()
	case ch == '^':
		l.advance()
		return l.tok(CARET, "^")
	case ch == '.':
		l.advance()
		return l.tok(DOT, ".")
	case ch == ';':
		l.advance()
		return l.tok(SEMICOLON, ";")
	case ch == '|':
		l.advance()
		return l.tok(PIPE, "|")
	case ch == '[':
		l.advance()
		return l.tok(LBRACKET, "[")
	case ch == ']':
		l.advance()
		return l.tok(RBRACKET, "]")
	case ch == '(':
		l.advance()
		return l.tok(LPAREN, "(")
	case ch == ')':
		l.advance()
		return l.tok(RPAREN, ")")
	case ch == '{':
		l.advance()
		return l.tok(LBRACE, "{")
	case ch == '}':
		l.advance()
		return l.tok(RBRACE, "}")
	case isDigit(ch):
		return l.readNumber()
	case isLetter(ch):
		return l.readIdentifierOrKeyword()
	case isBinChar(ch):
		return l.readBinOp()
	default:
		l.advance()
		return l.tok(ILLEGAL, string(ch))
	}
}

// Peek returns the next token without consuming it.
func (l *Lexer) Peek() Token {
	saved := *l
	tok := l.Next()
	*l = saved
	return tok
}

// --- internal helpers -------------------------------------------------------

func (l *Lexer) tok(k Kind, lit string) Token {
	return Token{Kind: k, Literal: lit, Line: l.line, Col: l.col}
}

func (l *Lexer) advance() byte {
	if l.pos >= len(l.src) {
		return 0
	}
	ch := l.src[l.pos]
	l.pos++
	if ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return ch
}

func (l *Lexer) current() byte {
	if l.pos >= len(l.src) {
		return 0
	}
	return l.src[l.pos]
}

func (l *Lexer) skipWhitespaceAndComments() {
	for l.pos < len(l.src) {
		ch := l.src[l.pos]
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			l.advance()
		} else if ch == '"' {
			// comment: skip until closing "
			l.advance()
			for l.pos < len(l.src) && l.src[l.pos] != '"' {
				l.advance()
			}
			if l.pos < len(l.src) {
				l.advance() // consume closing "
			}
		} else {
			break
		}
	}
}

func (l *Lexer) readString() Token {
	start := l.pos
	l.advance() // consume opening '
	var buf []byte
	for l.pos < len(l.src) {
		ch := l.src[l.pos]
		if ch == '\'' {
			l.advance()
			if l.pos < len(l.src) && l.src[l.pos] == '\'' {
				buf = append(buf, '\'')
				l.advance()
				continue
			}
			break
		}
		buf = append(buf, ch)
		l.advance()
	}
	return Token{Kind: STRING, Literal: string(l.src[start:l.pos]), Line: l.line, Col: l.col}
}

func (l *Lexer) readCharacter() Token {
	start := l.pos
	l.advance() // consume $
	if l.pos >= len(l.src) {
		return l.tok(ILLEGAL, string(l.src[start:l.pos]))
	}
	if l.src[l.pos] == '\\' {
		l.advance()
		if l.pos < len(l.src) {
			l.advance()
		}
	} else {
		_, size := utf8.DecodeRune(l.src[l.pos:])
		for i := 0; i < size; i++ {
			l.advance()
		}
	}
	return Token{Kind: CHARACTER, Literal: string(l.src[start:l.pos]), Line: l.line, Col: l.col}
}

func (l *Lexer) readHash() Token {
	start := l.pos
	l.advance() // consume #
	if l.pos >= len(l.src) {
		return Token{Kind: ILLEGAL, Literal: "#", Line: l.line, Col: l.col}
	}
	ch := l.src[l.pos]
	switch {
	case ch == '[':
		return l.readByteArray(start)
	case ch == '(':
		l.advance()
		return Token{Kind: ARRAYOPEN, Literal: "#(", Line: l.line, Col: l.col}
	case ch == '\'':
		// #'symbol with spaces'
		tok := l.readString()
		return Token{Kind: SYMBOL, Literal: string(l.src[start:l.pos]), Line: tok.Line, Col: tok.Col}
	case isLetter(ch):
		return l.readSymbol(start)
	default:
		return Token{Kind: ILLEGAL, Literal: string(l.src[start : l.pos+1]), Line: l.line, Col: l.col}
	}
}

func (l *Lexer) readByteArray(start int) Token {
	l.advance() // consume [
	for l.pos < len(l.src) && l.src[l.pos] != ']' {
		l.advance()
	}
	if l.pos < len(l.src) {
		l.advance() // consume ]
	}
	return Token{Kind: BYTEARRAY, Literal: string(l.src[start:l.pos]), Line: l.line, Col: l.col}
}

func (l *Lexer) readSymbol(start int) Token {
	// consume identifier chars and colons (for keyword symbols like #at:put:)
	for l.pos < len(l.src) && (isLetter(l.src[l.pos]) || isDigit(l.src[l.pos]) || l.src[l.pos] == '_' || l.src[l.pos] == ':') {
		l.advance()
	}
	return Token{Kind: SYMBOL, Literal: string(l.src[start:l.pos]), Line: l.line, Col: l.col}
}

func (l *Lexer) readColon() Token {
	l.advance() // consume :
	if l.pos < len(l.src) && l.src[l.pos] == '=' {
		l.advance()
		return l.tok(ASSIGN, ":=")
	}
	return l.tok(COLON, ":")
}

func (l *Lexer) readNumber() Token {
	start := l.pos
	for l.pos < len(l.src) && isDigit(l.src[l.pos]) {
		l.advance()
	}
	// base#value notation (e.g. 16rFF)
	if l.pos < len(l.src) && l.src[l.pos] == 'r' {
		l.advance()
		for l.pos < len(l.src) && isAlNum(l.src[l.pos]) {
			l.advance()
		}
		return Token{Kind: INTEGER, Literal: string(l.src[start:l.pos]), Line: l.line, Col: l.col}
	}
	// float
	if l.pos < len(l.src) && l.src[l.pos] == '.' {
		// look ahead to see if next char is digit
		if l.pos+1 < len(l.src) && isDigit(l.src[l.pos+1]) {
			l.advance() // .
			for l.pos < len(l.src) && isDigit(l.src[l.pos]) {
				l.advance()
			}
			// exponent
			if l.pos < len(l.src) && (l.src[l.pos] == 'e' || l.src[l.pos] == 'E') {
				l.advance()
				if l.pos < len(l.src) && (l.src[l.pos] == '+' || l.src[l.pos] == '-') {
					l.advance()
				}
				for l.pos < len(l.src) && isDigit(l.src[l.pos]) {
					l.advance()
				}
			}
			return Token{Kind: FLOAT, Literal: string(l.src[start:l.pos]), Line: l.line, Col: l.col}
		}
	}
	return Token{Kind: INTEGER, Literal: string(l.src[start:l.pos]), Line: l.line, Col: l.col}
}

func (l *Lexer) readIdentifierOrKeyword() Token {
	start := l.pos
	for l.pos < len(l.src) && (isLetter(l.src[l.pos]) || isDigit(l.src[l.pos]) || l.src[l.pos] == '_') {
		l.advance()
	}
	lit := string(l.src[start:l.pos])

	// keyword: identifier followed immediately by ':'
	if l.pos < len(l.src) && l.src[l.pos] == ':' && (l.pos+1 >= len(l.src) || l.src[l.pos+1] != '=') {
		l.advance() // consume ':'
		lit += ":"
		return Token{Kind: KEYWORD, Literal: lit, Line: l.line, Col: l.col}
	}

	// reserved words
	switch lit {
	case "true", "false":
		return Token{Kind: BOOLLIT, Literal: lit, Line: l.line, Col: l.col}
	case "nil":
		return Token{Kind: NILLIT, Literal: lit, Line: l.line, Col: l.col}
	case "self":
		return Token{Kind: SELF, Literal: lit, Line: l.line, Col: l.col}
	case "super":
		return Token{Kind: SUPER, Literal: lit, Line: l.line, Col: l.col}
	case "thisContext":
		return Token{Kind: THISCONTEXT, Literal: lit, Line: l.line, Col: l.col}
	case "object":
		return Token{Kind: OBJECT, Literal: lit, Line: l.line, Col: l.col}
	case "interface":
		return Token{Kind: INTERFACE, Literal: lit, Line: l.line, Col: l.col}
	case "compose":
		return Token{Kind: COMPOSE, Literal: lit, Line: l.line, Col: l.col}
	case "import":
		return Token{Kind: IMPORT, Literal: lit, Line: l.line, Col: l.col}
	}

	return Token{Kind: IDENTIFIER, Literal: lit, Line: l.line, Col: l.col}
}

func (l *Lexer) readBinOp() Token {
	start := l.pos
	if l.pos+1 < len(l.src) {
		// Keep generic delimiters and channel receive as distinct tokens so
		// nested generic types like Queue<<Channel<<Int>>>> remain parseable.
		ch0, ch1 := l.src[l.pos], l.src[l.pos+1]
		switch {
		case ch0 == '<' && ch1 == '<':
			fallthrough
		case ch0 == '>' && ch1 == '>':
			fallthrough
		case ch0 == '<' && ch1 == '-':
			l.advance()
			l.advance()
			return Token{Kind: BINOP, Literal: string(l.src[start:l.pos]), Line: l.line, Col: l.col}
		}
	}
	for l.pos < len(l.src) && isBinChar(l.src[l.pos]) {
		l.advance()
	}
	return Token{Kind: BINOP, Literal: string(l.src[start:l.pos]), Line: l.line, Col: l.col}
}

// --- character class helpers ------------------------------------------------

func isDigit(ch byte) bool { return ch >= '0' && ch <= '9' }

func isLetter(ch byte) bool {
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || ch == '_' || ch > 127
}

func isAlNum(ch byte) bool { return isDigit(ch) || isLetter(ch) }

// isBinChar returns true for characters that can appear in a binary operator.
// Explicitly excludes punctuation that has dedicated token kinds.
func isBinChar(ch byte) bool {
	switch ch {
	case '+', '-', '*', '/', '<', '>', '=', '~',
		'@', ',', '&', '%', '?', '!', '\\':
		return true
	}
	return false
}
