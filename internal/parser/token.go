// Package parser implements the lexer and recursive descent parser
// for the IntentLang DSL (.ias/.az files).
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

	// IntentLang 2.0 keywords
	TokenTool
	TokenDeploy
	TokenTarget
	TokenPipeline
	TokenStep
	TokenDelegate
	TokenTypeKw // "type" keyword (TokenType is taken by Go)
	TokenStrategy
	TokenMaxTurns
	TokenTimeout
	TokenTokenBudget
	TokenTemperature
	TokenStream
	TokenParallel
	TokenDependsOn
	TokenFrom
	TokenWhen
	TokenHealth
	TokenAutoscale
	TokenResources
	TokenMemory
	TokenOnError
	TokenMaxRetries
	TokenFallback
	TokenVariables
	TokenEnum
	TokenList

	// Tool block variant keywords
	TokenMCP
	TokenHTTP
	TokenInline
	TokenMethod
	TokenHeaders
	TokenBodyTemplate
	TokenBinary
	TokenLanguage
	TokenCode

	// Deploy block attribute keywords
	TokenPort
	TokenNamespace
	TokenReplicas
	TokenImage
	TokenSecrets

	// IntentLang 3.0 keywords
	TokenAs        // import alias
	TokenIf        // conditional
	TokenElse      // conditional
	TokenFor       // for each loop
	TokenEach      // for each loop
	TokenIn        // for each x in collection
	TokenConfig    // config block
	TokenValidate  // validate block
	TokenEval      // eval block
	TokenRule      // validation rule
	TokenCase      // eval case
	TokenOn        // on input block
	TokenUse       // use skill
	TokenWith      // use skill with { params }
	TokenRespond   // respond expression
	TokenScoring   // eval case scoring method
	TokenThreshold // eval case threshold
	TokenTags      // eval case tags
	TokenQuery     // http tool query params
	TokenLoop      // agent loop strategy

	// Expression operators (IntentLang 3.0)
	TokenColon     // :
	TokenEqual     // ==
	TokenNotEqual  // !=
	TokenGreater   // >
	TokenGreaterEq // >=
	TokenLess      // <
	TokenLessEq    // <=
)

var tokenNames = map[TokenType]string{
	TokenEOF:          "EOF",
	TokenError:        "Error",
	TokenNewline:      "Newline",
	TokenIdent:        "Ident",
	TokenString:       "String",
	TokenNumber:       "Number",
	TokenBool:         "Bool",
	TokenLBrace:       "{",
	TokenRBrace:       "}",
	TokenLBracket:     "[",
	TokenRBracket:     "]",
	TokenLParen:       "(",
	TokenRParen:       ")",
	TokenComma:        ",",
	TokenDot:          ".",
	TokenPackage:      "package",
	TokenVersion:      "version",
	TokenLang:         "lang",
	TokenPrompt:       "prompt",
	TokenSkill:        "skill",
	TokenAgent:        "agent",
	TokenBinding:      "binding",
	TokenUses:         "uses",
	TokenModel:        "model",
	TokenInput:        "input",
	TokenOutput:       "output",
	TokenExecution:    "execution",
	TokenDescription:  "description",
	TokenContent:      "content",
	TokenDefault:      "default",
	TokenAdapter:      "adapter",
	TokenSecret:       "secret",
	TokenEnvironment:  "environment",
	TokenPolicy:       "policy",
	TokenPlugin:       "plugin",
	TokenServer:       "server",
	TokenClient:       "client",
	TokenConnects:     "connects",
	TokenExposes:      "exposes",
	TokenEnv:          "env",
	TokenStore:        "store",
	TokenCommand:      "command",
	TokenRequire:      "require",
	TokenDeny:         "deny",
	TokenAllow:        "allow",
	TokenTo:           "to",
	TokenTrue:         "true",
	TokenFalse:        "false",
	TokenRequired:     "required",
	TokenImport:       "import",
	TokenTransport:    "transport",
	TokenURL:          "url",
	TokenAuth:         "auth",
	TokenArgs:         "args",
	TokenMetadata:     "metadata",
	TokenTool:         "tool",
	TokenDeploy:       "deploy",
	TokenTarget:       "target",
	TokenPipeline:     "pipeline",
	TokenStep:         "step",
	TokenDelegate:     "delegate",
	TokenTypeKw:       "type",
	TokenStrategy:     "strategy",
	TokenMaxTurns:     "max_turns",
	TokenTimeout:      "timeout",
	TokenTokenBudget:  "token_budget",
	TokenTemperature:  "temperature",
	TokenStream:       "stream",
	TokenParallel:     "parallel",
	TokenDependsOn:    "depends_on",
	TokenFrom:         "from",
	TokenWhen:         "when",
	TokenHealth:       "health",
	TokenAutoscale:    "autoscale",
	TokenResources:    "resources",
	TokenMemory:       "memory",
	TokenOnError:      "on_error",
	TokenMaxRetries:   "max_retries",
	TokenFallback:     "fallback",
	TokenVariables:    "variables",
	TokenEnum:         "enum",
	TokenList:         "list",
	TokenMCP:          "mcp",
	TokenHTTP:         "http",
	TokenInline:       "inline",
	TokenMethod:       "method",
	TokenHeaders:      "headers",
	TokenBodyTemplate: "body_template",
	TokenBinary:       "binary",
	TokenLanguage:     "language",
	TokenCode:         "code",
	TokenPort:         "port",
	TokenNamespace:    "namespace",
	TokenReplicas:     "replicas",
	TokenImage:        "image",
	TokenSecrets:      "secrets",
	TokenAs:           "as",
	TokenIf:           "if",
	TokenElse:         "else",
	TokenFor:          "for",
	TokenEach:         "each",
	TokenIn:           "in",
	TokenConfig:       "config",
	TokenValidate:     "validate",
	TokenEval:         "eval",
	TokenRule:         "rule",
	TokenCase:         "case",
	TokenOn:           "on",
	TokenUse:          "use",
	TokenWith:         "with",
	TokenRespond:      "respond",
	TokenScoring:      "scoring",
	TokenThreshold:    "threshold",
	TokenTags:         "tags",
	TokenQuery:        "query",
	TokenLoop:         "loop",
	TokenColon:        ":",
	TokenEqual:        "==",
	TokenNotEqual:     "!=",
	TokenGreater:      ">",
	TokenGreaterEq:    ">=",
	TokenLess:         "<",
	TokenLessEq:       "<=",
}

func (t TokenType) String() string {
	if name, ok := tokenNames[t]; ok {
		return name
	}
	return fmt.Sprintf("Token(%d)", int(t))
}

var keywords = map[string]TokenType{
	"package":       TokenPackage,
	"version":       TokenVersion,
	"lang":          TokenLang,
	"prompt":        TokenPrompt,
	"skill":         TokenSkill,
	"agent":         TokenAgent,
	"binding":       TokenBinding,
	"uses":          TokenUses,
	"model":         TokenModel,
	"input":         TokenInput,
	"output":        TokenOutput,
	"execution":     TokenExecution,
	"description":   TokenDescription,
	"content":       TokenContent,
	"default":       TokenDefault,
	"adapter":       TokenAdapter,
	"secret":        TokenSecret,
	"environment":   TokenEnvironment,
	"policy":        TokenPolicy,
	"plugin":        TokenPlugin,
	"server":        TokenServer,
	"client":        TokenClient,
	"connects":      TokenConnects,
	"exposes":       TokenExposes,
	"env":           TokenEnv,
	"store":         TokenStore,
	"command":       TokenCommand,
	"require":       TokenRequire,
	"deny":          TokenDeny,
	"allow":         TokenAllow,
	"to":            TokenTo,
	"true":          TokenTrue,
	"false":         TokenFalse,
	"required":      TokenRequired,
	"import":        TokenImport,
	"transport":     TokenTransport,
	"url":           TokenURL,
	"auth":          TokenAuth,
	"args":          TokenArgs,
	"metadata":      TokenMetadata,
	"tool":          TokenTool,
	"deploy":        TokenDeploy,
	"target":        TokenTarget,
	"pipeline":      TokenPipeline,
	"step":          TokenStep,
	"delegate":      TokenDelegate,
	"type":          TokenTypeKw,
	"strategy":      TokenStrategy,
	"max_turns":     TokenMaxTurns,
	"timeout":       TokenTimeout,
	"token_budget":  TokenTokenBudget,
	"temperature":   TokenTemperature,
	"stream":        TokenStream,
	"parallel":      TokenParallel,
	"depends_on":    TokenDependsOn,
	"from":          TokenFrom,
	"when":          TokenWhen,
	"health":        TokenHealth,
	"autoscale":     TokenAutoscale,
	"resources":     TokenResources,
	"memory":        TokenMemory,
	"on_error":      TokenOnError,
	"max_retries":   TokenMaxRetries,
	"fallback":      TokenFallback,
	"variables":     TokenVariables,
	"enum":          TokenEnum,
	"list":          TokenList,
	"mcp":           TokenMCP,
	"http":          TokenHTTP,
	"inline":        TokenInline,
	"method":        TokenMethod,
	"headers":       TokenHeaders,
	"body_template": TokenBodyTemplate,
	"binary":        TokenBinary,
	"language":      TokenLanguage,
	"code":          TokenCode,
	"port":          TokenPort,
	"namespace":     TokenNamespace,
	"replicas":      TokenReplicas,
	"image":         TokenImage,
	"secrets":       TokenSecrets,
	"as":            TokenAs,
	"if":            TokenIf,
	"else":          TokenElse,
	"for":           TokenFor,
	"each":          TokenEach,
	"in":            TokenIn,
	"config":        TokenConfig,
	"validate":      TokenValidate,
	"eval":          TokenEval,
	"rule":          TokenRule,
	"case":          TokenCase,
	"on":            TokenOn,
	"use":           TokenUse,
	"with":          TokenWith,
	"respond":       TokenRespond,
	"scoring":       TokenScoring,
	"threshold":     TokenThreshold,
	"tags":          TokenTags,
	"query":         TokenQuery,
	"loop":          TokenLoop,
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
