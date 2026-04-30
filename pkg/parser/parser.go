// Package parser implements a recursive-descent parser for picoceci.
//
// It consumes a stream of tokens from pkg/lexer and produces an AST
// rooted at *ast.Program.  See docs/grammar.ebnf for the formal grammar
// and LANGUAGE_SPEC.md §4 and §5 for the language constructs.
//
// Usage:
//
//	p := parser.New(lexer.NewString(src))
//	prog, err := p.ParseProgram()
package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kristofer/picoceci/pkg/ast"
	"github.com/kristofer/picoceci/pkg/lexer"
)

// ParseError describes a syntax error found during parsing.
type ParseError struct {
	Pos     ast.Pos
	Message string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("parse error at %s: %s", e.Pos, e.Message)
}

// Parser holds parser state.
type Parser struct {
	lex    *lexer.Lexer
	cur    lexer.Token
	peek   lexer.Token
	errors []*ParseError
}

// New creates a Parser wrapping the given Lexer.
func New(l *lexer.Lexer) *Parser {
	p := &Parser{lex: l}
	p.advance() // fill cur
	p.advance() // fill peek
	return p
}

// Errors returns all parse errors collected during parsing.
func (p *Parser) Errors() []*ParseError { return p.errors }

// ParseProgram parses an entire picoceci source file.
func (p *Parser) ParseProgram() (*ast.Program, error) {
	prog := &ast.Program{Pos: pos(p.cur)}
	for p.cur.Kind != lexer.EOF {
		stmt := p.parseTopLevel()
		if stmt != nil {
			prog.Statements = append(prog.Statements, stmt)
		}
		p.consumeOptional(lexer.DOT)
	}
	if len(p.errors) > 0 {
		return prog, p.errors[0]
	}
	return prog, nil
}

// --- top-level --------------------------------------------------------------

func (p *Parser) parseTopLevel() ast.Node {
	switch p.cur.Kind {
	case lexer.IMPORT:
		return p.parseImport()
	case lexer.OBJECT:
		return p.parseObjectDecl()
	case lexer.INTERFACE:
		return p.parseInterfaceDecl()
	default:
		return p.parseStatement()
	}
}

func (p *Parser) parseImport() *ast.ImportDecl {
	n := &ast.ImportDecl{Pos: pos(p.cur)}
	p.expect(lexer.IMPORT)
	if p.cur.Kind != lexer.STRING {
		p.errorf("expected string after import, got %s", p.cur.Literal)
		return n
	}
	n.Path = unquoteString(p.cur.Literal)
	p.advance()
	return n
}

func (p *Parser) parseObjectDecl() *ast.ObjectDecl {
	n := &ast.ObjectDecl{Pos: pos(p.cur)}
	p.expect(lexer.OBJECT)
	if p.cur.Kind != lexer.IDENTIFIER {
		p.errorf("expected object name")
		return n
	}
	n.Name = p.cur.Literal
	p.advance()
	p.expect(lexer.LBRACE)

	for p.cur.Kind != lexer.RBRACE && p.cur.Kind != lexer.EOF {
		switch p.cur.Kind {
		case lexer.COMPOSE:
			p.advance()
			if p.cur.Kind == lexer.IDENTIFIER {
				n.Composes = append(n.Composes, p.cur.Literal)
				p.advance()
			}
			p.consumeOptional(lexer.DOT)
		case lexer.PIPE:
			vd := p.parseVarDecl()
			n.Slots = append(n.Slots, vd.Names...)
		default:
			m := p.parseMethodDef()
			if m != nil {
				n.Methods = append(n.Methods, m)
			}
		}
	}
	p.expect(lexer.RBRACE)
	return n
}

func (p *Parser) parseMethodDef() *ast.MethodDef {
	m := &ast.MethodDef{Pos: pos(p.cur)}
	// selector
	switch p.cur.Kind {
	case lexer.IDENTIFIER:
		m.Selector = p.cur.Literal
		p.advance()
	case lexer.BINOP:
		m.Selector = p.cur.Literal
		p.advance()
		if p.cur.Kind == lexer.IDENTIFIER {
			m.Params = append(m.Params, p.cur.Literal)
			p.advance()
		}
	case lexer.KEYWORD:
		for p.cur.Kind == lexer.KEYWORD {
			m.Selector += p.cur.Literal
			p.advance()
			if p.cur.Kind == lexer.IDENTIFIER {
				m.Params = append(m.Params, p.cur.Literal)
				p.advance()
			}
		}
	default:
		p.errorf("expected method selector, got %s", p.cur.Literal)
		return nil
	}

	p.expect(lexer.LBRACKET)
	if p.cur.Kind == lexer.PIPE {
		vd := p.parseVarDecl()
		m.Locals = vd.Names
	}
	m.Body = p.parseStatements(lexer.RBRACKET)
	p.expect(lexer.RBRACKET)
	return m
}

func (p *Parser) parseInterfaceDecl() *ast.InterfaceDecl {
	n := &ast.InterfaceDecl{Pos: pos(p.cur)}
	p.expect(lexer.INTERFACE)
	if p.cur.Kind != lexer.IDENTIFIER {
		p.errorf("expected interface name")
		return n
	}
	n.Name = p.cur.Literal
	p.advance()
	p.expect(lexer.LBRACE)
	for p.cur.Kind != lexer.RBRACE && p.cur.Kind != lexer.EOF {
		sig := ""
		switch p.cur.Kind {
		case lexer.IDENTIFIER:
			sig = p.cur.Literal
			p.advance()
		case lexer.BINOP:
			sig = p.cur.Literal
			p.advance()
			if p.cur.Kind == lexer.IDENTIFIER {
				p.advance() // param name (ignored in sig)
			}
		case lexer.KEYWORD:
			for p.cur.Kind == lexer.KEYWORD {
				sig += p.cur.Literal
				p.advance()
				if p.cur.Kind == lexer.IDENTIFIER {
					p.advance()
				}
			}
		}
		if sig != "" {
			n.Sigs = append(n.Sigs, sig)
		}
		p.consumeOptional(lexer.DOT)
	}
	p.expect(lexer.RBRACE)
	return n
}

// --- statements -------------------------------------------------------------

func (p *Parser) parseStatement() ast.Node {
	if p.cur.Kind == lexer.CARET {
		return p.parseReturn()
	}
	if p.cur.Kind == lexer.PIPE {
		return p.parseVarDecl()
	}
	return p.parseExpression()
}

func (p *Parser) parseReturn() *ast.Return {
	n := &ast.Return{Pos: pos(p.cur)}
	p.expect(lexer.CARET)
	n.Value = p.parseExpression()
	return n
}

func (p *Parser) parseVarDecl() *ast.VarDecl {
	n := &ast.VarDecl{Pos: pos(p.cur)}
	p.expect(lexer.PIPE)
	for p.cur.Kind == lexer.IDENTIFIER {
		n.Names = append(n.Names, p.cur.Literal)
		p.advance()
	}
	p.expect(lexer.PIPE)
	return n
}

func (p *Parser) parseStatements(stop lexer.Kind) []ast.Node {
	var stmts []ast.Node
	for p.cur.Kind != stop && p.cur.Kind != lexer.EOF {
		s := p.parseStatement()
		if s != nil {
			stmts = append(stmts, s)
		}
		p.consumeOptional(lexer.DOT)
	}
	return stmts
}

// --- expressions ------------------------------------------------------------

func (p *Parser) parseExpression() ast.Node {
	// Assignment: IDENTIFIER :=
	if p.cur.Kind == lexer.IDENTIFIER && p.peek.Kind == lexer.ASSIGN {
		n := &ast.Assign{Pos: pos(p.cur), Name: p.cur.Literal}
		p.advance() // identifier
		p.advance() // :=
		n.Value = p.parseExpression()
		return n
	}
	return p.parseCascade()
}

func (p *Parser) parseCascade() ast.Node {
	recv := p.parseKeywordExpr()
	if p.cur.Kind != lexer.SEMICOLON {
		return recv
	}
	n := &ast.Cascade{Pos: pos(p.cur), Receiver: recv}
	for p.cur.Kind == lexer.SEMICOLON {
		p.advance()
		switch {
		case p.cur.Kind == lexer.IDENTIFIER:
			msg := &ast.UnaryMsg{Pos: pos(p.cur), Selector: p.cur.Literal}
			p.advance()
			n.Messages = append(n.Messages, msg)
		case p.cur.Kind == lexer.BINOP:
			msg := &ast.BinaryMsg{Pos: pos(p.cur), Op: p.cur.Literal}
			p.advance()
			msg.Arg = p.parseUnaryExpr()
			n.Messages = append(n.Messages, msg)
		case p.cur.Kind == lexer.KEYWORD:
			n.Messages = append(n.Messages, p.parseKeywordMsg(nil))
		}
	}
	return n
}

func (p *Parser) parseKeywordExpr() ast.Node {
	recv := p.parseBinaryExpr()
	if p.cur.Kind != lexer.KEYWORD {
		return recv
	}
	return p.parseKeywordMsg(recv)
}

func (p *Parser) parseKeywordMsg(recv ast.Node) *ast.KeywordMsg {
	n := &ast.KeywordMsg{Pos: pos(p.cur), Receiver: recv}
	for p.cur.Kind == lexer.KEYWORD {
		n.Keywords = append(n.Keywords, p.cur.Literal)
		p.advance()
		n.Args = append(n.Args, p.parseBinaryExpr())
	}
	return n
}

func (p *Parser) parseBinaryExpr() ast.Node {
	left := p.parseUnaryExpr()
	for p.cur.Kind == lexer.BINOP {
		n := &ast.BinaryMsg{Pos: pos(p.cur), Receiver: left, Op: p.cur.Literal}
		p.advance()
		n.Arg = p.parseUnaryExpr()
		left = n
	}
	return left
}

func (p *Parser) parseUnaryExpr() ast.Node {
	n := p.parsePrimary()
	for p.cur.Kind == lexer.IDENTIFIER && p.peek.Kind != lexer.ASSIGN && p.peek.Kind != lexer.COLON {
		// Only consume as unary if next token is not ':' (which would make it a keyword)
		msg := &ast.UnaryMsg{Pos: pos(p.cur), Receiver: n, Selector: p.cur.Literal}
		p.advance()
		n = msg
	}
	return n
}

func (p *Parser) parsePrimary() ast.Node {
	switch p.cur.Kind {
	case lexer.NILLIT:
		n := &ast.NilLit{Pos: pos(p.cur)}
		p.advance()
		return n
	case lexer.BOOLLIT:
		n := &ast.BoolLit{Pos: pos(p.cur), Value: p.cur.Literal == "true"}
		p.advance()
		return n
	case lexer.INTEGER:
		return p.parseInt()
	case lexer.FLOAT:
		return p.parseFloat()
	case lexer.STRING:
		n := &ast.StringLit{Pos: pos(p.cur), Value: unquoteString(p.cur.Literal)}
		p.advance()
		return n
	case lexer.SYMBOL:
		n := &ast.SymbolLit{Pos: pos(p.cur), Value: parseSymbol(p.cur.Literal)}
		p.advance()
		return n
	case lexer.CHARACTER:
		n := &ast.CharLit{Pos: pos(p.cur), Value: parseChar(p.cur.Literal)}
		p.advance()
		return n
	case lexer.BYTEARRAY:
		return p.parseByteArrayLit()
	case lexer.ARRAYOPEN:
		return p.parseArrayLit()
	case lexer.SELF:
		n := &ast.SelfExpr{Pos: pos(p.cur)}
		p.advance()
		return n
	case lexer.SUPER:
		n := &ast.SuperExpr{Pos: pos(p.cur)}
		p.advance()
		return n
	case lexer.IDENTIFIER:
		n := &ast.Ident{Pos: pos(p.cur), Name: p.cur.Literal}
		p.advance()
		return n
	case lexer.LBRACKET:
		return p.parseBlock()
	case lexer.LPAREN:
		p.advance()
		e := p.parseExpression()
		p.expect(lexer.RPAREN)
		return e
	default:
		p.errorf("unexpected token %q in primary expression", p.cur.Literal)
		p.advance()
		return &ast.NilLit{Pos: pos(p.cur)}
	}
}

func (p *Parser) parseInt() *ast.IntLit {
	n := &ast.IntLit{Pos: pos(p.cur), Raw: p.cur.Literal}
	raw := p.cur.Literal
	if idx := strings.IndexByte(raw, 'r'); idx >= 0 {
		base, _ := strconv.ParseInt(raw[:idx], 10, 64)
		// Clamp base to the valid range for strconv.ParseInt (2–36).
		if base < 2 {
			base = 2
		} else if base > 36 {
			base = 36
		}
		val, _ := strconv.ParseInt(raw[idx+1:], int(base), 64)
		n.Value = val
	} else {
		n.Value, _ = strconv.ParseInt(raw, 10, 64)
	}
	p.advance()
	return n
}

func (p *Parser) parseFloat() *ast.FloatLit {
	n := &ast.FloatLit{Pos: pos(p.cur), Raw: p.cur.Literal}
	n.Value, _ = strconv.ParseFloat(p.cur.Literal, 64)
	p.advance()
	return n
}

func (p *Parser) parseByteArrayLit() *ast.ByteArrayLit {
	n := &ast.ByteArrayLit{Pos: pos(p.cur)}
	// Literal is  #[ 1 2 3 ]  — re-lex the content from the raw token
	raw := p.cur.Literal // e.g. "#[1 2 3]"
	p.advance()
	inner := strings.TrimPrefix(raw, "#[")
	inner = strings.TrimSuffix(inner, "]")
	for _, part := range strings.Fields(inner) {
		v, _ := strconv.ParseInt(part, 10, 16)
		// Clamp to valid byte range 0–255.
		if v < 0 {
			v = 0
		} else if v > 255 {
			v = 255
		}
		n.Bytes = append(n.Bytes, byte(v))
	}
	return n
}

func (p *Parser) parseArrayLit() *ast.ArrayLit {
	n := &ast.ArrayLit{Pos: pos(p.cur)}
	p.advance() // consume #(
	for p.cur.Kind != lexer.RPAREN && p.cur.Kind != lexer.EOF {
		elem := p.parsePrimary()
		n.Elements = append(n.Elements, elem)
	}
	p.expect(lexer.RPAREN)
	return n
}

func (p *Parser) parseBlock() *ast.Block {
	n := &ast.Block{Pos: pos(p.cur)}
	p.expect(lexer.LBRACKET)

	// Parameters: :p :q |
	if p.cur.Kind == lexer.COLON {
		for p.cur.Kind == lexer.COLON {
			p.advance()
			if p.cur.Kind == lexer.IDENTIFIER {
				n.Params = append(n.Params, p.cur.Literal)
				p.advance()
			}
		}
		p.consumeOptional(lexer.PIPE)
	}
	// Locals
	if p.cur.Kind == lexer.PIPE {
		vd := p.parseVarDecl()
		n.Locals = vd.Names
	}
	n.Body = p.parseStatements(lexer.RBRACKET)
	p.expect(lexer.RBRACKET)
	return n
}

// --- helpers ----------------------------------------------------------------

func (p *Parser) advance() {
	p.cur = p.peek
	p.peek = p.lex.Next()
}

func (p *Parser) expect(k lexer.Kind) {
	if p.cur.Kind != k {
		p.errorf("expected %s, got %q", kindName(k), p.cur.Literal)
		return
	}
	p.advance()
}

func (p *Parser) consumeOptional(k lexer.Kind) {
	if p.cur.Kind == k {
		p.advance()
	}
}

func (p *Parser) errorf(format string, args ...interface{}) {
	p.errors = append(p.errors, &ParseError{
		Pos:     pos(p.cur),
		Message: fmt.Sprintf(format, args...),
	})
}

func pos(t lexer.Token) ast.Pos { return ast.Pos{Line: t.Line, Col: t.Col} }

func kindName(k lexer.Kind) string {
	return fmt.Sprintf("token<%d>", k)
}

func unquoteString(s string) string {
	// Remove surrounding quotes and unescape ''
	if len(s) >= 2 && s[0] == '\'' && s[len(s)-1] == '\'' {
		s = s[1 : len(s)-1]
	}
	return strings.ReplaceAll(s, "''", "'")
}

func parseSymbol(s string) string {
	// Remove leading #
	s = strings.TrimPrefix(s, "#")
	// If quoted: #'...'
	if len(s) >= 2 && s[0] == '\'' {
		s = s[1 : len(s)-1]
		s = strings.ReplaceAll(s, "''", "'")
	}
	return s
}

func parseChar(s string) rune {
	// $A  or $\n
	if len(s) < 2 {
		return 0
	}
	if s[1] == '\\' && len(s) >= 3 {
		switch s[2] {
		case 'n':
			return '\n'
		case 't':
			return '\t'
		case 'r':
			return '\r'
		case '\\':
			return '\\'
		case '\'':
			return '\''
		case '0':
			return 0
		}
	}
	r, _ := firstRune(s[1:])
	return r
}

func firstRune(s string) (rune, int) {
	if len(s) == 0 {
		return 0, 0
	}
	return rune(s[0]), 1
}
