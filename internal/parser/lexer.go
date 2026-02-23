package parser

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Lexer tokenizes IntentLang (.ias/.az) source input.
type Lexer struct {
	input   string
	file    string
	pos     int
	line    int
	col     int
	tokens  []Token
	start   int
	startLn int
	startCl int
}

// NewLexer creates a new lexer for the given input.
func NewLexer(input, file string) *Lexer {
	return &Lexer{
		input: input,
		file:  file,
		line:  1,
		col:   1,
	}
}

// Tokenize scans the entire input and returns all tokens.
func (l *Lexer) Tokenize() ([]Token, error) {
	for {
		tok, err := l.nextToken()
		if err != nil {
			return nil, err
		}
		l.tokens = append(l.tokens, tok)
		if tok.Type == TokenEOF {
			break
		}
	}
	return l.tokens, nil
}

func (l *Lexer) nextToken() (Token, error) {
	l.skipWhitespaceAndComments()

	if l.pos >= len(l.input) {
		return l.makeToken(TokenEOF, ""), nil
	}

	l.start = l.pos
	l.startLn = l.line
	l.startCl = l.col

	ch := l.peek()

	switch {
	case ch == '\n':
		l.advance()
		return l.makeToken(TokenNewline, "\n"), nil
	case ch == '{':
		l.advance()
		return l.makeToken(TokenLBrace, "{"), nil
	case ch == '}':
		l.advance()
		return l.makeToken(TokenRBrace, "}"), nil
	case ch == '[':
		l.advance()
		return l.makeToken(TokenLBracket, "["), nil
	case ch == ']':
		l.advance()
		return l.makeToken(TokenRBracket, "]"), nil
	case ch == '(':
		l.advance()
		return l.makeToken(TokenLParen, "("), nil
	case ch == ')':
		l.advance()
		return l.makeToken(TokenRParen, ")"), nil
	case ch == ',':
		l.advance()
		return l.makeToken(TokenComma, ","), nil
	case ch == '.':
		l.advance()
		return l.makeToken(TokenDot, "."), nil
	case ch == '"':
		return l.scanString()
	case isDigit(ch) || (ch == '-' && l.peekAt(1) != 0 && isDigit(l.peekAt(1))):
		return l.scanNumber()
	case isIdentStart(ch):
		return l.scanIdentOrKeyword()
	default:
		l.advance()
		return Token{}, fmt.Errorf("%s:%d:%d: unexpected character %q", l.file, l.startLn, l.startCl, ch)
	}
}

func (l *Lexer) scanString() (Token, error) {
	l.advance() // consume opening quote
	var sb strings.Builder
	for l.pos < len(l.input) {
		ch := l.peek()
		if ch == '"' {
			l.advance() // consume closing quote
			return l.makeToken(TokenString, sb.String()), nil
		}
		if ch == '\\' {
			l.advance()
			if l.pos >= len(l.input) {
				return Token{}, fmt.Errorf("%s:%d:%d: unterminated string escape", l.file, l.line, l.col)
			}
			esc := l.peek()
			switch esc {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case '\\':
				sb.WriteByte('\\')
			case '"':
				sb.WriteByte('"')
			default:
				sb.WriteByte('\\')
				sb.WriteRune(esc)
			}
			l.advance()
			continue
		}
		if ch == '\n' {
			l.line++
			l.col = 1
			sb.WriteRune(ch)
			l.pos += utf8.RuneLen(ch)
			continue
		}
		sb.WriteRune(ch)
		l.advance()
	}
	return Token{}, fmt.Errorf("%s:%d:%d: unterminated string", l.file, l.startLn, l.startCl)
}

func (l *Lexer) scanNumber() (Token, error) {
	for l.pos < len(l.input) && (isDigit(l.peek()) || l.peek() == '.' || l.peek() == '-') {
		l.advance()
	}
	return l.makeToken(TokenNumber, l.input[l.start:l.pos]), nil
}

func (l *Lexer) scanIdentOrKeyword() (Token, error) {
	for l.pos < len(l.input) && isIdentPart(l.peek()) {
		l.advance()
	}
	literal := l.input[l.start:l.pos]
	tokType := LookupKeyword(literal)
	return l.makeToken(tokType, literal), nil
}

func (l *Lexer) skipWhitespaceAndComments() {
	for l.pos < len(l.input) {
		ch := l.peek()
		if ch == ' ' || ch == '\t' || ch == '\r' {
			l.advance()
			continue
		}
		// Line comments start with #
		if ch == '#' {
			for l.pos < len(l.input) && l.peek() != '\n' {
				l.advance()
			}
			continue
		}
		// Also handle // comments
		if ch == '/' && l.peekAt(1) == '/' {
			for l.pos < len(l.input) && l.peek() != '\n' {
				l.advance()
			}
			continue
		}
		break
	}
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.pos:])
	return r
}

func (l *Lexer) peekAt(offset int) rune {
	p := l.pos + offset
	if p >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[p:])
	return r
}

func (l *Lexer) advance() {
	if l.pos >= len(l.input) {
		return
	}
	r, size := utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += size
	if r == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
}

func (l *Lexer) makeToken(typ TokenType, literal string) Token {
	return Token{
		Type:    typ,
		Literal: literal,
		File:    l.file,
		Line:    l.startLn,
		Column:  l.startCl,
	}
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func isIdentStart(r rune) bool {
	return unicode.IsLetter(r) || r == '_'
}

func isIdentPart(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'
}
