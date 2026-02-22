// Package parser implements the lexer and recursive descent parser
// for the Agentz DSL (.az files).
package parser

import "fmt"

// TokenType represents the type of a lexical token.
type TokenType int

const (
	// Special tokens
	TokenEOF TokenType = iota
	TokenError
	TokenNewline

	// Literals
	TokenIdent
	TokenString
	TokenNumber
	TokenBool

	// Delimiters
	TokenLBrace   // {
	TokenRBrace   // }
	TokenLBracket // [
	TokenRBracket // ]
	TokenLParen   // (
	TokenRParen   // )
	TokenComma    // ,
	TokenDot      // .

	// Keywords
	TokenPackage
	TokenVersion
	TokenLang
	TokenPrompt
	TokenSkill
	TokenAgent
	TokenBinding
	TokenUses
	TokenModel
	TokenInput
	TokenOutput
	TokenExecution
	TokenDescription
	TokenContent
	TokenDefault
	TokenAdapter
	TokenSecret
	TokenEnvironment
	TokenPolicy
	TokenPlugin
	TokenServer
	TokenClient
	TokenConnects
	TokenExposes
	TokenEnv
	TokenStore
	TokenCommand
	TokenRequire
	TokenDeny
	TokenAllow
	TokenTo
	TokenTrue
	TokenFalse
	TokenRequired
	TokenImport
	TokenTransport
	TokenURL
	TokenAuth
	TokenArgs
	TokenMetadata
)

var tokenNames = map[TokenType]string{
	TokenEOF:         "EOF",
	TokenError:       "Error",
	TokenNewline:     "Newline",
	TokenIdent:       "Ident",
	TokenString:      "String",
	TokenNumber:      "Number",
	TokenBool:        "Bool",
	TokenLBrace:      "{",
	TokenRBrace:      "}",
	TokenLBracket:    "[",
	TokenRBracket:    "]",
	TokenLParen:      "(",
	TokenRParen:      ")",
	TokenComma:       ",",
	TokenDot:         ".",
	TokenPackage:     "package",
	TokenVersion:     "version",
	TokenLang:        "lang",
	TokenPrompt:      "prompt",
	TokenSkill:       "skill",
	TokenAgent:       "agent",
	TokenBinding:     "binding",
	TokenUses:        "uses",
	TokenModel:       "model",
	TokenInput:       "input",
	TokenOutput:      "output",
	TokenExecution:   "execution",
	TokenDescription: "description",
	TokenContent:     "content",
	TokenDefault:     "default",
	TokenAdapter:     "adapter",
	TokenSecret:      "secret",
	TokenEnvironment: "environment",
	TokenPolicy:      "policy",
	TokenPlugin:      "plugin",
	TokenServer:      "server",
	TokenClient:      "client",
	TokenConnects:    "connects",
	TokenExposes:     "exposes",
	TokenEnv:         "env",
	TokenStore:       "store",
	TokenCommand:     "command",
	TokenRequire:     "require",
	TokenDeny:        "deny",
	TokenAllow:       "allow",
	TokenTo:          "to",
	TokenTrue:        "true",
	TokenFalse:       "false",
	TokenRequired:    "required",
	TokenImport:      "import",
	TokenTransport:   "transport",
	TokenURL:         "url",
	TokenAuth:        "auth",
	TokenArgs:        "args",
	TokenMetadata:    "metadata",
}

func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return fmt.Sprintf("Token(%d)", int(t))
}

var keywords = map[string]TokenType{
	"package":     TokenPackage,
	"version":     TokenVersion,
	"lang":        TokenLang,
	"prompt":      TokenPrompt,
	"skill":       TokenSkill,
	"agent":       TokenAgent,
	"binding":     TokenBinding,
	"uses":        TokenUses,
	"model":       TokenModel,
	"input":       TokenInput,
	"output":      TokenOutput,
	"execution":   TokenExecution,
	"description": TokenDescription,
	"content":     TokenContent,
	"default":     TokenDefault,
	"adapter":     TokenAdapter,
	"secret":      TokenSecret,
	"environment": TokenEnvironment,
	"policy":      TokenPolicy,
	"plugin":      TokenPlugin,
	"server":      TokenServer,
	"client":      TokenClient,
	"connects":    TokenConnects,
	"exposes":     TokenExposes,
	"env":         TokenEnv,
	"store":       TokenStore,
	"command":     TokenCommand,
	"require":     TokenRequire,
	"deny":        TokenDeny,
	"allow":       TokenAllow,
	"to":          TokenTo,
	"true":        TokenTrue,
	"false":       TokenFalse,
	"required":    TokenRequired,
	"import":      TokenImport,
	"transport":   TokenTransport,
	"url":         TokenURL,
	"auth":        TokenAuth,
	"args":        TokenArgs,
	"metadata":    TokenMetadata,
}

// LookupKeyword returns the keyword token type for ident, or TokenIdent.
func LookupKeyword(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return TokenIdent
}

// Token represents a lexical token with position information.
type Token struct {
	Type    TokenType
	Literal string
	File    string
	Line    int
	Column  int
}

func (t Token) String() string {
	if t.Literal != "" {
		return fmt.Sprintf("%s(%q) at %s:%d:%d", t.Type, t.Literal, t.File, t.Line, t.Column)
	}
	return fmt.Sprintf("%s at %s:%d:%d", t.Type, t.File, t.Line, t.Column)
}
