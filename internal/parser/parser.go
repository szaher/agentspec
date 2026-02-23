package parser

import (
	"fmt"
	"strings"

	"github.com/szaher/designs/agentz/internal/ast"
)

// ParseError represents a parse error with position information.
type ParseError struct {
	File    string
	Line    int
	Column  int
	Message string
	Hint    string
}

func (e *ParseError) Error() string {
	s := fmt.Sprintf("%s:%d:%d: error: %s", e.File, e.Line, e.Column, e.Message)
	if e.Hint != "" {
		s += "\n  hint: " + e.Hint
	}
	return s
}

// Parser performs recursive descent parsing of IntentLang (.ias/.az) token streams.
type Parser struct {
	tokens []Token
	pos    int
	file   string
	errors []*ParseError
	names  map[string]map[string]bool // kind -> name -> exists (for duplicate detection)
}

// Parse parses the given IntentLang (.ias/.az) source and returns an AST File.
func Parse(input, file string) (*ast.File, []*ParseError) {
	lexer := NewLexer(input, file)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, []*ParseError{{File: file, Line: 1, Column: 1, Message: err.Error()}}
	}

	p := &Parser{
		tokens: tokens,
		file:   file,
		names:  make(map[string]map[string]bool),
	}

	f := p.parseFile()
	if len(p.errors) > 0 {
		return f, p.errors
	}
	return f, nil
}

func (p *Parser) parseFile() *ast.File {
	f := &ast.File{
		Path:     p.file,
		StartPos: p.currentPos(),
	}

	p.skipNewlines()

	// Parse package header (required)
	if p.check(TokenPackage) {
		f.Package = p.parsePackage()
	} else {
		p.addError("expected 'package' declaration at start of file", "")
	}

	// Parse top-level statements
	for !p.isAtEnd() {
		p.skipNewlines()
		if p.isAtEnd() {
			break
		}

		stmt := p.parseStatement()
		if stmt != nil {
			f.Statements = append(f.Statements, stmt)
		}
	}

	f.EndPos = p.currentPos()
	return f
}

func (p *Parser) parsePackage() *ast.Package {
	startPos := p.currentPos()
	p.expect(TokenPackage)

	pkg := &ast.Package{StartPos: startPos}

	pkg.Name = p.expectString("package name")

	// Parse optional package attributes
	for !p.isAtEnd() && !p.check(TokenNewline) && !p.isAtEnd() {
		if p.check(TokenVersion) {
			p.advance()
			pkg.Version = p.expectString("version")
		} else if p.check(TokenLang) {
			p.advance()
			pkg.LangVersion = p.expectString("lang version")
		} else {
			break
		}
	}

	pkg.EndPos = p.currentPos()
	return pkg
}

func (p *Parser) parseStatement() ast.Statement {
	p.skipNewlines()
	if p.isAtEnd() {
		return nil
	}

	switch {
	case p.check(TokenPrompt):
		return p.parsePrompt()
	case p.check(TokenSkill):
		return p.parseSkill()
	case p.check(TokenAgent):
		return p.parseAgent()
	case p.check(TokenBinding):
		return p.parseBinding()
	case p.check(TokenSecret):
		return p.parseSecret()
	case p.check(TokenEnvironment):
		return p.parseEnvironment()
	case p.check(TokenPolicy):
		return p.parsePolicy()
	case p.check(TokenPlugin):
		return p.parsePlugin()
	case p.check(TokenServer):
		return p.parseMCPServer()
	case p.check(TokenClient):
		return p.parseMCPClient()
	case p.check(TokenImport):
		return p.parseImportAsStmt()
	default:
		tok := p.current()
		p.addError(fmt.Sprintf("unexpected token %s", tok.Type), "expected a resource declaration (agent, prompt, skill, etc.)")
		p.advance()
		return nil
	}
}

func (p *Parser) parsePrompt() *ast.Prompt {
	startPos := p.currentPos()
	p.expect(TokenPrompt)

	prompt := &ast.Prompt{
		StartPos: startPos,
		Metadata: make(map[string]string),
	}
	prompt.Name = p.expectString("prompt name")
	p.registerName("Prompt", prompt.Name, startPos)

	p.expectToken(TokenLBrace)
	p.skipNewlines()

	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}

		switch {
		case p.check(TokenContent):
			p.advance()
			prompt.Content = p.expectString("prompt content")
		case p.check(TokenVersion):
			p.advance()
			prompt.Version = p.expectString("version")
		case p.check(TokenMetadata):
			p.advance()
			p.parseMetadataBlock(prompt.Metadata)
		default:
			p.addError(fmt.Sprintf("unexpected token %s in prompt block", p.current().Type), "")
			p.advance()
		}
		p.skipNewlines()
	}
	p.expectToken(TokenRBrace)
	prompt.EndPos = p.currentPos()
	return prompt
}

func (p *Parser) parseSkill() *ast.Skill {
	startPos := p.currentPos()
	p.expect(TokenSkill)

	skill := &ast.Skill{
		StartPos: startPos,
		Metadata: make(map[string]string),
	}
	skill.Name = p.expectString("skill name")
	p.registerName("Skill", skill.Name, startPos)

	p.expectToken(TokenLBrace)
	p.skipNewlines()

	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}

		switch {
		case p.check(TokenDescription):
			p.advance()
			skill.Description = p.expectString("description")
		case p.check(TokenInput):
			p.advance()
			skill.Input = p.parseFieldBlock()
		case p.check(TokenOutput):
			p.advance()
			skill.Output = p.parseFieldBlock()
		case p.check(TokenExecution):
			p.advance()
			skill.Execution = p.parseExecution()
		case p.check(TokenMetadata):
			p.advance()
			p.parseMetadataBlock(skill.Metadata)
		default:
			p.addError(fmt.Sprintf("unexpected token %s in skill block", p.current().Type), "")
			p.advance()
		}
		p.skipNewlines()
	}
	p.expectToken(TokenRBrace)
	skill.EndPos = p.currentPos()
	return skill
}

func (p *Parser) parseAgent() *ast.Agent {
	startPos := p.currentPos()
	p.expect(TokenAgent)

	agent := &ast.Agent{
		StartPos:   startPos,
		Parameters: make(map[string]string),
		Metadata:   make(map[string]string),
	}
	agent.Name = p.expectString("agent name")
	p.registerName("Agent", agent.Name, startPos)

	p.expectToken(TokenLBrace)
	p.skipNewlines()

	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}

		switch {
		case p.check(TokenUses):
			p.advance()
			if p.check(TokenPrompt) {
				p.advance()
				ref := &ast.Ref{
					Kind:     "prompt",
					Name:     p.expectString("prompt reference"),
					StartPos: p.currentPos(),
				}
				ref.EndPos = p.currentPos()
				agent.Prompt = ref
			} else if p.check(TokenSkill) {
				p.advance()
				ref := &ast.Ref{
					Kind:     "skill",
					Name:     p.expectString("skill reference"),
					StartPos: p.currentPos(),
				}
				ref.EndPos = p.currentPos()
				agent.Skills = append(agent.Skills, ref)
			} else {
				p.addError("expected 'prompt' or 'skill' after 'uses'", "")
			}
		case p.check(TokenModel):
			p.advance()
			agent.Model = p.expectString("model")
		case p.check(TokenConnects):
			p.advance()
			if p.check(TokenTo) {
				p.advance()
			}
			if p.check(TokenClient) {
				p.advance()
			}
			ref := &ast.Ref{
				Kind:     "client",
				Name:     p.expectString("client reference"),
				StartPos: p.currentPos(),
			}
			ref.EndPos = p.currentPos()
			agent.Client = ref
		case p.check(TokenMetadata):
			p.advance()
			p.parseMetadataBlock(agent.Metadata)
		default:
			p.addError(fmt.Sprintf("unexpected token %s in agent block", p.current().Type), "")
			p.advance()
		}
		p.skipNewlines()
	}
	p.expectToken(TokenRBrace)
	agent.EndPos = p.currentPos()
	return agent
}

func (p *Parser) parseBinding() *ast.Binding {
	startPos := p.currentPos()
	p.expect(TokenBinding)

	binding := &ast.Binding{
		StartPos: startPos,
		Config:   make(map[string]string),
	}
	binding.Name = p.expectString("binding name")
	p.registerName("Binding", binding.Name, startPos)

	// Parse optional inline "adapter <name>"
	if p.check(TokenAdapter) {
		p.advance()
		binding.Adapter = p.expectString("adapter name")
	}

	p.expectToken(TokenLBrace)
	p.skipNewlines()

	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}

		switch {
		case p.check(TokenDefault):
			p.advance()
			if p.check(TokenTrue) {
				p.advance()
				binding.Default = true
			} else if p.check(TokenFalse) {
				p.advance()
				binding.Default = false
			} else {
				binding.Default = true
			}
		case p.check(TokenAdapter):
			p.advance()
			binding.Adapter = p.expectString("adapter name")
		default:
			// Config key-value pair
			key := p.current().Literal
			p.advance()
			if p.check(TokenString) {
				binding.Config[key] = p.current().Literal
				p.advance()
			} else if p.check(TokenNumber) {
				binding.Config[key] = p.current().Literal
				p.advance()
			} else {
				binding.Config[key] = p.current().Literal
				p.advance()
			}
		}
		p.skipNewlines()
	}
	p.expectToken(TokenRBrace)
	binding.EndPos = p.currentPos()
	return binding
}

func (p *Parser) parseSecret() *ast.Secret {
	startPos := p.currentPos()
	p.expect(TokenSecret)

	secret := &ast.Secret{StartPos: startPos}
	secret.Name = p.expectString("secret name")
	p.registerName("Secret", secret.Name, startPos)

	p.expectToken(TokenLBrace)
	p.skipNewlines()

	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}

		switch {
		case p.check(TokenEnv):
			p.advance()
			secret.Source = "env"
			if p.check(TokenLParen) {
				p.advance()
				secret.Key = p.expectStringOrIdent("env key")
				p.expectToken(TokenRParen)
			} else {
				secret.Key = p.expectStringOrIdent("env key")
			}
		case p.check(TokenStore):
			p.advance()
			secret.Source = "store"
			if p.check(TokenLParen) {
				p.advance()
				secret.Key = p.expectStringOrIdent("store path")
				p.expectToken(TokenRParen)
			} else {
				secret.Key = p.expectStringOrIdent("store path")
			}
		default:
			p.addError(fmt.Sprintf("unexpected token %s in secret block", p.current().Type), "expected 'env' or 'store'")
			p.advance()
		}
		p.skipNewlines()
	}
	p.expectToken(TokenRBrace)
	secret.EndPos = p.currentPos()
	return secret
}

func (p *Parser) parseEnvironment() *ast.Environment {
	startPos := p.currentPos()
	p.expect(TokenEnvironment)

	env := &ast.Environment{StartPos: startPos}
	env.Name = p.expectString("environment name")
	p.registerName("Environment", env.Name, startPos)

	p.expectToken(TokenLBrace)
	p.skipNewlines()

	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}

		// Parse resource overrides: resource "ref" { attribute value }
		resourceKind := p.current().Literal
		p.advance()
		resourceName := p.expectString("resource name")
		p.expectToken(TokenLBrace)
		p.skipNewlines()

		for !p.check(TokenRBrace) && !p.isAtEnd() {
			p.skipNewlines()
			if p.check(TokenRBrace) {
				break
			}
			attr := p.current().Literal
			p.advance()
			val := p.expectStringOrIdent("override value")
			env.Overrides = append(env.Overrides, &ast.Override{
				Resource:  resourceKind + "/" + resourceName,
				Attribute: attr,
				Value:     val,
				StartPos:  startPos,
				EndPos:    p.currentPos(),
			})
			p.skipNewlines()
		}
		p.expectToken(TokenRBrace)
		p.skipNewlines()
	}
	p.expectToken(TokenRBrace)
	env.EndPos = p.currentPos()
	return env
}

func (p *Parser) parsePolicy() *ast.Policy {
	startPos := p.currentPos()
	p.expect(TokenPolicy)

	policy := &ast.Policy{StartPos: startPos}
	policy.Name = p.expectString("policy name")
	p.registerName("Policy", policy.Name, startPos)

	p.expectToken(TokenLBrace)
	p.skipNewlines()

	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}

		rule := &ast.Rule{StartPos: p.currentPos()}
		switch {
		case p.check(TokenDeny):
			rule.Action = "deny"
			p.advance()
		case p.check(TokenAllow):
			rule.Action = "allow"
			p.advance()
		case p.check(TokenRequire):
			rule.Action = "require"
			p.advance()
		default:
			p.addError(fmt.Sprintf("unexpected token %s in policy block", p.current().Type), "expected 'deny', 'allow', or 'require'")
			p.advance()
			continue
		}

		// Collect the rest of the rule as subject
		var parts []string
		for !p.check(TokenNewline) && !p.check(TokenRBrace) && !p.isAtEnd() {
			parts = append(parts, p.current().Literal)
			p.advance()
		}
		if len(parts) > 0 {
			// First part may be resource pattern
			rule.Resource = parts[0]
			if len(parts) > 1 {
				rule.Subject = strings.Join(parts[1:], " ")
			}
		}
		rule.EndPos = p.currentPos()
		policy.Rules = append(policy.Rules, rule)
		p.skipNewlines()
	}
	p.expectToken(TokenRBrace)
	policy.EndPos = p.currentPos()
	return policy
}

func (p *Parser) parsePlugin() *ast.Plugin {
	startPos := p.currentPos()
	p.expect(TokenPlugin)

	plugin := &ast.Plugin{StartPos: startPos}
	plugin.Name = p.expectString("plugin name")

	if p.check(TokenVersion) {
		p.advance()
		plugin.Version = p.expectString("plugin version")
	}

	plugin.EndPos = p.currentPos()
	return plugin
}

func (p *Parser) parseMCPServer() *ast.MCPServer {
	startPos := p.currentPos()
	p.expect(TokenServer)

	server := &ast.MCPServer{
		StartPos: startPos,
		Env:      make(map[string]string),
		Metadata: make(map[string]string),
	}
	server.Name = p.expectString("server name")
	p.registerName("MCPServer", server.Name, startPos)

	p.expectToken(TokenLBrace)
	p.skipNewlines()

	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}

		switch {
		case p.check(TokenTransport):
			p.advance()
			server.Transport = p.expectStringOrIdent("transport type")
		case p.check(TokenCommand):
			p.advance()
			server.Command = p.expectString("command")
		case p.check(TokenArgs):
			p.advance()
			server.Args = p.parseStringList()
		case p.check(TokenURL):
			p.advance()
			server.URL = p.expectString("url")
		case p.check(TokenAuth):
			p.advance()
			ref := &ast.Ref{
				Kind:     "secret",
				Name:     p.expectString("auth secret reference"),
				StartPos: p.currentPos(),
			}
			ref.EndPos = p.currentPos()
			server.Auth = ref
		case p.check(TokenExposes):
			p.advance()
			if p.check(TokenSkill) {
				p.advance()
			}
			ref := &ast.Ref{
				Kind:     "skill",
				Name:     p.expectString("skill reference"),
				StartPos: p.currentPos(),
			}
			ref.EndPos = p.currentPos()
			server.Skills = append(server.Skills, ref)
		case p.check(TokenEnv):
			p.advance()
			p.parseMetadataBlock(server.Env)
		case p.check(TokenMetadata):
			p.advance()
			p.parseMetadataBlock(server.Metadata)
		default:
			p.addError(fmt.Sprintf("unexpected token %s in server block", p.current().Type), "")
			p.advance()
		}
		p.skipNewlines()
	}
	p.expectToken(TokenRBrace)
	server.EndPos = p.currentPos()
	return server
}

func (p *Parser) parseMCPClient() *ast.MCPClient {
	startPos := p.currentPos()
	p.expect(TokenClient)

	client := &ast.MCPClient{
		StartPos: startPos,
		Metadata: make(map[string]string),
	}
	client.Name = p.expectString("client name")
	p.registerName("MCPClient", client.Name, startPos)

	p.expectToken(TokenLBrace)
	p.skipNewlines()

	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}

		switch {
		case p.check(TokenConnects):
			p.advance()
			if p.check(TokenTo) {
				p.advance()
			}
			if p.check(TokenServer) {
				p.advance()
			}
			ref := &ast.Ref{
				Kind:     "server",
				Name:     p.expectString("server reference"),
				StartPos: p.currentPos(),
			}
			ref.EndPos = p.currentPos()
			client.Servers = append(client.Servers, ref)
		case p.check(TokenMetadata):
			p.advance()
			p.parseMetadataBlock(client.Metadata)
		default:
			p.addError(fmt.Sprintf("unexpected token %s in client block", p.current().Type), "")
			p.advance()
		}
		p.skipNewlines()
	}
	p.expectToken(TokenRBrace)
	client.EndPos = p.currentPos()
	return client
}

func (p *Parser) parseImportAsStmt() ast.Statement {
	startPos := p.currentPos()
	p.expect(TokenImport)

	pkg := p.parseFile().Package
	if pkg == nil {
		return nil
	}

	imp := &ast.Import{
		Path:     p.expectString("import path"),
		StartPos: startPos,
	}
	if p.check(TokenVersion) {
		p.advance()
		imp.Version = p.expectString("import version")
	}
	imp.EndPos = p.currentPos()

	// Wrap in a PluginRef to satisfy Statement interface
	pluginRef := &ast.PluginRef{
		Name:     imp.Path,
		Version:  imp.Version,
		StartPos: imp.StartPos,
		EndPos:   imp.EndPos,
	}
	return pluginRef
}

// Helper methods

func (p *Parser) parseFieldBlock() []*ast.Field {
	var fields []*ast.Field
	p.expectToken(TokenLBrace)
	p.skipNewlines()

	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}

		field := &ast.Field{StartPos: p.currentPos()}
		field.Name = p.current().Literal
		p.advance()

		field.Type = p.current().Literal
		p.advance()

		if p.check(TokenRequired) {
			field.Required = true
			p.advance()
		}
		field.EndPos = p.currentPos()
		fields = append(fields, field)
		p.skipNewlines()
	}
	p.expectToken(TokenRBrace)
	return fields
}

func (p *Parser) parseExecution() *ast.Execution {
	exec := &ast.Execution{StartPos: p.currentPos()}
	exec.Type = p.current().Literal
	p.advance()
	exec.Value = p.expectString("execution value")
	exec.EndPos = p.currentPos()
	return exec
}

func (p *Parser) parseMetadataBlock(m map[string]string) {
	if !p.check(TokenLBrace) {
		return
	}
	p.expectToken(TokenLBrace)
	p.skipNewlines()

	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}
		key := p.current().Literal
		p.advance()
		val := p.expectStringOrIdent("metadata value")
		m[key] = val
		p.skipNewlines()
	}
	p.expectToken(TokenRBrace)
}

func (p *Parser) parseStringList() []string {
	var items []string
	if p.check(TokenLBracket) {
		p.advance()
		for !p.check(TokenRBracket) && !p.isAtEnd() {
			if p.check(TokenComma) {
				p.advance()
				continue
			}
			items = append(items, p.expectString("list item"))
		}
		p.expectToken(TokenRBracket)
	} else {
		items = append(items, p.expectString("argument"))
	}
	return items
}

func (p *Parser) registerName(kind, name string, pos ast.Pos) {
	if _, ok := p.names[kind]; !ok {
		p.names[kind] = make(map[string]bool)
	}
	if p.names[kind][name] {
		p.addError(fmt.Sprintf("duplicate %s name %q", kind, name), "each resource must have a unique name within its kind")
	}
	p.names[kind][name] = true
}

// Token navigation

func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF, File: p.file}
	}
	return p.tokens[p.pos]
}

func (p *Parser) check(t TokenType) bool {
	return p.current().Type == t
}

func (p *Parser) advance() Token {
	tok := p.current()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}

func (p *Parser) expect(t TokenType) Token {
	if !p.check(t) {
		p.addError(fmt.Sprintf("expected %s, got %s", t, p.current().Type), "")
	}
	return p.advance()
}

func (p *Parser) expectToken(t TokenType) {
	p.skipNewlines()
	if !p.check(t) {
		p.addError(fmt.Sprintf("expected %s, got %s", t, p.current().Type), "")
		return
	}
	p.advance()
}

func (p *Parser) expectString(context string) string {
	if p.check(TokenString) {
		return p.advance().Literal
	}
	p.addError(fmt.Sprintf("expected string for %s, got %s", context, p.current().Type), "")
	if !p.isAtEnd() {
		return p.advance().Literal
	}
	return ""
}

func (p *Parser) expectStringOrIdent(context string) string {
	if p.check(TokenString) || p.check(TokenIdent) {
		return p.advance().Literal
	}
	// Accept keywords as identifiers in value position
	if p.current().Type >= TokenPackage {
		return p.advance().Literal
	}
	p.addError(fmt.Sprintf("expected string or identifier for %s, got %s", context, p.current().Type), "")
	if !p.isAtEnd() {
		return p.advance().Literal
	}
	return ""
}

func (p *Parser) skipNewlines() {
	for p.check(TokenNewline) {
		p.advance()
	}
}

func (p *Parser) isAtEnd() bool {
	return p.current().Type == TokenEOF
}

func (p *Parser) currentPos() ast.Pos {
	tok := p.current()
	return ast.Pos{File: tok.File, Line: tok.Line, Column: tok.Column}
}

func (p *Parser) addError(msg, hint string) {
	tok := p.current()
	p.errors = append(p.errors, &ParseError{
		File:    tok.File,
		Line:    tok.Line,
		Column:  tok.Column,
		Message: msg,
		Hint:    hint,
	})
}

// AllNames returns all registered names for duplicate checking and suggestions.
func (p *Parser) AllNames() map[string]map[string]bool {
	return p.names
}
