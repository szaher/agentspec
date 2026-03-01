package parser

import (
	"fmt"
	"strconv"
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
loop:
	for !p.isAtEnd() && !p.check(TokenNewline) && !p.isAtEnd() {
		switch {
		case p.check(TokenVersion):
			p.advance()
			pkg.Version = p.expectString("version")
		case p.check(TokenLang):
			p.advance()
			pkg.LangVersion = p.expectString("lang version")
		default:
			break loop
		}
	}

	if pkg.LangVersion == "1.0" {
		p.addError("IntentLang 1.0 is no longer supported",
			"run 'agentspec migrate --to-v2' to upgrade to IntentLang 2.0")
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
	case p.check(TokenDeploy):
		return p.parseDeployTarget()
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
	case p.check(TokenPipeline):
		return p.parsePipeline()
	case p.check(TokenTypeKw):
		return p.parseTypeDef()
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
		case p.check(TokenVariables):
			p.advance()
			prompt.Variables = p.parseVariablesBlock()
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
		case p.check(TokenTool):
			p.advance()
			skill.ToolConfig = p.parseToolConfig()
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
			switch {
			case p.check(TokenPrompt):
				p.advance()
				ref := &ast.Ref{
					Kind:     "prompt",
					Name:     p.expectString("prompt reference"),
					StartPos: p.currentPos(),
				}
				ref.EndPos = p.currentPos()
				agent.Prompt = ref
			case p.check(TokenSkill):
				p.advance()
				ref := &ast.Ref{
					Kind:     "skill",
					Name:     p.expectString("skill reference"),
					StartPos: p.currentPos(),
				}
				ref.EndPos = p.currentPos()
				agent.Skills = append(agent.Skills, ref)
			default:
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
		case p.check(TokenStrategy):
			p.advance()
			agent.Strategy = p.expectString("strategy")
		case p.check(TokenMaxTurns):
			p.advance()
			agent.MaxTurns = p.expectInt("max_turns")
		case p.check(TokenTimeout):
			p.advance()
			agent.Timeout = p.expectString("timeout")
		case p.check(TokenTokenBudget):
			p.advance()
			agent.TokenBudget = p.expectInt("token_budget")
		case p.check(TokenTemperature):
			p.advance()
			agent.Temperature = p.expectFloat("temperature")
			agent.HasTemp = true
		case p.check(TokenStream):
			p.advance()
			val := p.expectBool("stream")
			agent.Stream = &val
		case p.check(TokenOnError):
			p.advance()
			agent.OnError = p.expectString("on_error")
		case p.check(TokenMaxRetries):
			p.advance()
			agent.MaxRetries = p.expectInt("max_retries")
		case p.check(TokenFallback):
			p.advance()
			agent.Fallback = p.expectString("fallback agent")
		case p.check(TokenMemory):
			p.advance()
			agent.MemoryCfg = p.parseMemoryConfig()
		case p.check(TokenDelegate):
			p.advance()
			del := p.parseDelegate()
			if del != nil {
				agent.Delegates = append(agent.Delegates, del)
			}
		case p.check(TokenMetadata):
			p.advance()
			p.parseMetadataBlock(agent.Metadata)
		// IntentLang 3.0: prompt shorthand (same as uses prompt)
		case p.check(TokenPrompt):
			p.advance()
			ref := &ast.Ref{
				Kind:     "prompt",
				Name:     p.expectStringOrIdent("prompt reference"),
				StartPos: p.currentPos(),
			}
			ref.EndPos = p.currentPos()
			agent.Prompt = ref
		// IntentLang 3.0: config, validate, eval, on input, loop
		case p.check(TokenConfig):
			p.advance()
			agent.ConfigParams = p.parseConfigBlock()
		case p.check(TokenValidate):
			p.advance()
			agent.ValidationRules = p.parseValidateBlock()
		case p.check(TokenEval):
			p.advance()
			agent.EvalCases = p.parseEvalBlock()
		case p.check(TokenOn):
			p.advance()
			agent.OnInput = p.parseOnInputBlock()
		case p.check(TokenLoop):
			p.advance()
			agent.Strategy = p.expectString("loop strategy")
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
			switch {
			case p.check(TokenTrue):
				p.advance()
				binding.Default = true
			case p.check(TokenFalse):
				p.advance()
				binding.Default = false
			default:
				binding.Default = true
			}
		case p.check(TokenAdapter):
			p.advance()
			binding.Adapter = p.expectString("adapter name")
		default:
			// Config key-value pair
			key := p.current().Literal
			p.advance()
			binding.Config[key] = p.current().Literal
			p.advance()
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

	imp := &ast.Import{
		Path:     p.expectString("import path"),
		StartPos: startPos,
	}
	if p.check(TokenVersion) {
		p.advance()
		imp.Version = p.expectString("import version")
	}
	if p.check(TokenAs) {
		p.advance()
		imp.Alias = p.expectStringOrIdent("import alias")
	}
	imp.EndPos = p.currentPos()
	return imp
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

func (p *Parser) expectInt(context string) int {
	if p.check(TokenNumber) {
		lit := p.advance().Literal
		n, err := strconv.Atoi(lit)
		if err != nil {
			p.addError(fmt.Sprintf("invalid integer for %s: %q", context, lit), "")
			return 0
		}
		return n
	}
	p.addError(fmt.Sprintf("expected number for %s, got %s", context, p.current().Type), "")
	if !p.isAtEnd() {
		p.advance()
	}
	return 0
}

func (p *Parser) expectFloat(context string) float64 {
	if p.check(TokenNumber) {
		lit := p.advance().Literal
		f, err := strconv.ParseFloat(lit, 64)
		if err != nil {
			p.addError(fmt.Sprintf("invalid float for %s: %q", context, lit), "")
			return 0
		}
		return f
	}
	p.addError(fmt.Sprintf("expected number for %s, got %s", context, p.current().Type), "")
	if !p.isAtEnd() {
		p.advance()
	}
	return 0
}

func (p *Parser) expectBool(context string) bool {
	if p.check(TokenTrue) {
		p.advance()
		return true
	}
	if p.check(TokenFalse) {
		p.advance()
		return false
	}
	p.addError(fmt.Sprintf("expected true/false for %s, got %s", context, p.current().Type), "")
	if !p.isAtEnd() {
		p.advance()
	}
	return false
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

// parseToolConfig parses the tool block variants: mcp, http, command, inline.
// Called after 'tool' keyword has been consumed.
func (p *Parser) parseToolConfig() *ast.ToolConfig {
	tc := &ast.ToolConfig{StartPos: p.currentPos()}

	switch {
	case p.check(TokenMCP):
		p.advance()
		tc.Type = "mcp"
		tc.ServerTool = p.expectString("mcp server/tool reference")

	case p.check(TokenHTTP):
		p.advance()
		tc.Type = "http"
		p.expectToken(TokenLBrace)
		p.skipNewlines()
		for !p.check(TokenRBrace) && !p.isAtEnd() {
			p.skipNewlines()
			if p.check(TokenRBrace) {
				break
			}
			switch {
			case p.check(TokenMethod):
				p.advance()
				tc.Method = p.expectString("http method")
			case p.check(TokenURL):
				p.advance()
				tc.URL = p.expectString("http url")
			case p.check(TokenHeaders):
				p.advance()
				tc.Headers = make(map[string]string)
				p.parseMetadataBlock(tc.Headers)
			case p.check(TokenBodyTemplate):
				p.advance()
				tc.BodyTemplate = p.expectString("body template")
			case p.check(TokenTimeout):
				p.advance()
				tc.Timeout = p.expectString("timeout")
			default:
				p.addError(fmt.Sprintf("unexpected token %s in tool http block", p.current().Type), "")
				p.advance()
			}
			p.skipNewlines()
		}
		p.expectToken(TokenRBrace)

	case p.check(TokenCommand):
		p.advance()
		tc.Type = "command"
		p.expectToken(TokenLBrace)
		p.skipNewlines()
		for !p.check(TokenRBrace) && !p.isAtEnd() {
			p.skipNewlines()
			if p.check(TokenRBrace) {
				break
			}
			switch {
			case p.check(TokenBinary):
				p.advance()
				tc.Binary = p.expectString("binary path")
			case p.check(TokenArgs):
				p.advance()
				tc.Args = p.parseStringList()
			case p.check(TokenTimeout):
				p.advance()
				tc.Timeout = p.expectString("timeout")
			case p.check(TokenEnv):
				p.advance()
				tc.Env = make(map[string]string)
				p.parseMetadataBlock(tc.Env)
			case p.check(TokenSecrets):
				p.advance()
				tc.Secrets = make(map[string]string)
				p.parseMetadataBlock(tc.Secrets)
			default:
				p.addError(fmt.Sprintf("unexpected token %s in tool command block", p.current().Type), "")
				p.advance()
			}
			p.skipNewlines()
		}
		p.expectToken(TokenRBrace)

	case p.check(TokenInline):
		p.advance()
		tc.Type = "inline"
		p.expectToken(TokenLBrace)
		p.skipNewlines()
		for !p.check(TokenRBrace) && !p.isAtEnd() {
			p.skipNewlines()
			if p.check(TokenRBrace) {
				break
			}
			switch {
			case p.check(TokenLanguage):
				p.advance()
				tc.Language = p.expectString("language")
			case p.check(TokenCode):
				p.advance()
				tc.Code = p.expectString("code")
			case p.check(TokenTimeout):
				p.advance()
				tc.Timeout = p.expectString("timeout")
			case p.check(TokenMemory):
				p.advance()
				tc.MemoryLimit = p.expectString("memory limit")
			default:
				p.addError(fmt.Sprintf("unexpected token %s in tool inline block", p.current().Type), "")
				p.advance()
			}
			p.skipNewlines()
		}
		p.expectToken(TokenRBrace)

	default:
		p.addError(fmt.Sprintf("expected tool type (mcp, http, command, inline), got %s", p.current().Type), "")
		p.advance()
	}

	tc.EndPos = p.currentPos()
	return tc
}

// parseDeployTarget parses: deploy "name" target "type" { ... }
func (p *Parser) parseDeployTarget() *ast.DeployTarget {
	startPos := p.currentPos()
	p.expect(TokenDeploy)

	dt := &ast.DeployTarget{
		StartPos: startPos,
		Env:      make(map[string]string),
		Secrets:  make(map[string]string),
	}
	dt.Name = p.expectString("deploy name")
	p.registerName("DeployTarget", dt.Name, startPos)

	// Expect 'target' keyword
	if p.check(TokenTarget) {
		p.advance()
		dt.Target = p.expectString("deploy target type")
	} else {
		p.addError("expected 'target' keyword in deploy block", "")
	}

	p.expectToken(TokenLBrace)
	p.skipNewlines()

	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}

		switch {
		case p.check(TokenPort):
			p.advance()
			dt.Port = p.expectInt("port")
		case p.check(TokenDefault):
			p.advance()
			dt.Default = p.expectBool("default")
		case p.check(TokenNamespace):
			p.advance()
			dt.Namespace = p.expectString("namespace")
		case p.check(TokenReplicas):
			p.advance()
			dt.Replicas = p.expectInt("replicas")
		case p.check(TokenImage):
			p.advance()
			dt.Image = p.expectString("image")
		case p.check(TokenResources):
			p.advance()
			dt.Resources = p.parseResourceLimits()
		case p.check(TokenHealth):
			p.advance()
			dt.Health = p.parseHealthConfig()
		case p.check(TokenAutoscale):
			p.advance()
			dt.Autoscale = p.parseAutoscaleConfig()
		case p.check(TokenEnv):
			p.advance()
			p.parseMetadataBlock(dt.Env)
		case p.check(TokenSecrets):
			p.advance()
			p.parseMetadataBlock(dt.Secrets)
		default:
			p.addError(fmt.Sprintf("unexpected token %s in deploy block", p.current().Type), "")
			p.advance()
		}
		p.skipNewlines()
	}
	p.expectToken(TokenRBrace)
	dt.EndPos = p.currentPos()
	return dt
}

// parseMemoryConfig parses: { strategy "sliding_window" max_messages 50 }
func (p *Parser) parseMemoryConfig() *ast.MemoryConfig {
	mc := &ast.MemoryConfig{StartPos: p.currentPos()}

	p.expectToken(TokenLBrace)
	p.skipNewlines()

	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}

		switch {
		case p.check(TokenStrategy):
			p.advance()
			mc.Strategy = p.expectString("memory strategy")
		case p.check(TokenIdent) && p.current().Literal == "max_messages":
			p.advance()
			mc.MaxMessages = p.expectInt("max_messages")
		default:
			p.addError(fmt.Sprintf("unexpected token %s in memory block", p.current().Type), "")
			p.advance()
		}
		p.skipNewlines()
	}
	p.expectToken(TokenRBrace)
	mc.EndPos = p.currentPos()
	return mc
}

// parseResourceLimits parses: { cpu "0.5" memory "512m" }
func (p *Parser) parseResourceLimits() *ast.ResourceLimits {
	rl := &ast.ResourceLimits{StartPos: p.currentPos()}

	p.expectToken(TokenLBrace)
	p.skipNewlines()

	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}

		switch {
		case p.current().Literal == "cpu":
			p.advance()
			rl.CPU = p.expectString("cpu")
		case p.check(TokenMemory):
			p.advance()
			rl.Memory = p.expectString("memory")
		default:
			p.addError(fmt.Sprintf("unexpected token %s in resources block", p.current().Type), "")
			p.advance()
		}
		p.skipNewlines()
	}
	p.expectToken(TokenRBrace)
	rl.EndPos = p.currentPos()
	return rl
}

// parseHealthConfig parses: { path "/healthz" interval "10s" [timeout "5s"] }
func (p *Parser) parseHealthConfig() *ast.HealthConfig {
	hc := &ast.HealthConfig{StartPos: p.currentPos()}

	p.expectToken(TokenLBrace)
	p.skipNewlines()

	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}

		switch {
		case p.current().Literal == "path":
			p.advance()
			hc.Path = p.expectString("health path")
		case p.current().Literal == "interval":
			p.advance()
			hc.Interval = p.expectString("health interval")
		case p.check(TokenTimeout):
			p.advance()
			hc.Timeout = p.expectString("health timeout")
		default:
			p.addError(fmt.Sprintf("unexpected token %s in health block", p.current().Type), "")
			p.advance()
		}
		p.skipNewlines()
	}
	p.expectToken(TokenRBrace)
	hc.EndPos = p.currentPos()
	return hc
}

// parseAutoscaleConfig parses: { min 2 max 10 metric "cpu" target 80 }
func (p *Parser) parseAutoscaleConfig() *ast.AutoscaleConfig {
	ac := &ast.AutoscaleConfig{StartPos: p.currentPos()}

	p.expectToken(TokenLBrace)
	p.skipNewlines()

	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}

		switch {
		case p.current().Literal == "min":
			p.advance()
			ac.MinReplicas = p.expectInt("min replicas")
		case p.current().Literal == "max":
			p.advance()
			ac.MaxReplicas = p.expectInt("max replicas")
		case p.current().Literal == "metric":
			p.advance()
			ac.Metric = p.expectString("autoscale metric")
		case p.check(TokenTarget):
			p.advance()
			ac.Target = p.expectInt("autoscale target")
		default:
			p.addError(fmt.Sprintf("unexpected token %s in autoscale block", p.current().Type), "")
			p.advance()
		}
		p.skipNewlines()
	}
	p.expectToken(TokenRBrace)
	ac.EndPos = p.currentPos()
	return ac
}

// parseVariablesBlock parses: { var_name type required default "value" ... }
func (p *Parser) parseVariablesBlock() []*ast.Variable {
	var vars []*ast.Variable

	p.expectToken(TokenLBrace)
	p.skipNewlines()

	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}

		v := &ast.Variable{StartPos: p.currentPos()}
		v.Name = p.current().Literal
		p.advance()

		// Type is required
		v.Type = p.current().Literal
		p.advance()

		// Optional: required and default
	varLoop:
		for !p.check(TokenRBrace) && !p.isAtEnd() && !p.check(TokenNewline) {
			switch {
			case p.current().Literal == "required":
				p.advance()
				v.Required = true
			case p.current().Literal == "default" || p.check(TokenDefault):
				p.advance()
				v.Default = p.expectString("default value")
			default:
				break varLoop
			}
		}

		v.EndPos = p.currentPos()
		vars = append(vars, v)
		p.skipNewlines()
	}
	p.expectToken(TokenRBrace)
	return vars
}

// parseDelegate parses: to agent "name" when "condition"
func (p *Parser) parseDelegate() *ast.Delegate {
	del := &ast.Delegate{StartPos: p.currentPos()}

	// expect "to"
	if p.check(TokenTo) {
		p.advance()
	}
	// expect "agent"
	if p.check(TokenAgent) {
		p.advance()
	}
	del.AgentRef = p.expectString("delegate agent name")

	// expect "when"
	if p.check(TokenWhen) {
		p.advance()
		del.Condition = p.expectString("delegate condition")
	}

	del.EndPos = p.currentPos()
	return del
}

// parseTypeDef parses: type "name" { field1 string required ... } or type "name" enum ["a", "b"] or type "name" list string
func (p *Parser) parseTypeDef() *ast.TypeDef {
	startPos := p.currentPos()
	p.expect(TokenTypeKw)

	td := &ast.TypeDef{StartPos: startPos}
	td.Name = p.expectString("type name")
	p.registerName("Type", td.Name, startPos)

	switch {
	case p.check(TokenEnum):
		p.advance()
		td.EnumVals = p.parseStringList()
	case p.check(TokenList):
		p.advance()
		td.ListOf = p.current().Literal
		p.advance()
	case p.check(TokenLBrace):
		p.expectToken(TokenLBrace)
		p.skipNewlines()

		for !p.check(TokenRBrace) && !p.isAtEnd() {
			p.skipNewlines()
			if p.check(TokenRBrace) {
				break
			}

			field := &ast.TypeField{StartPos: p.currentPos()}
			field.Name = p.current().Literal
			p.advance()
			field.Type = p.current().Literal
			p.advance()

			// Optional: required, default
		fieldLoop:
			for !p.check(TokenRBrace) && !p.isAtEnd() && !p.check(TokenNewline) {
				switch {
				case p.current().Literal == "required":
					p.advance()
					field.Required = true
				case p.current().Literal == "default" || p.check(TokenDefault):
					p.advance()
					field.Default = p.expectString("default value")
				default:
					break fieldLoop
				}
			}

			field.EndPos = p.currentPos()
			td.Fields = append(td.Fields, field)
			p.skipNewlines()
		}
		p.expectToken(TokenRBrace)
	default:
		p.addError("expected '{', 'enum', or 'list' after type name", "")
	}

	td.EndPos = p.currentPos()
	return td
}

// parsePipeline parses: pipeline "name" { step "name" { ... } ... }
func (p *Parser) parsePipeline() *ast.Pipeline {
	startPos := p.currentPos()
	p.expect(TokenPipeline)

	pipeline := &ast.Pipeline{StartPos: startPos}
	pipeline.Name = p.expectString("pipeline name")
	p.registerName("Pipeline", pipeline.Name, startPos)

	p.expectToken(TokenLBrace)
	p.skipNewlines()

	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}

		if p.check(TokenStep) {
			p.advance()
			step := p.parsePipelineStep()
			pipeline.Steps = append(pipeline.Steps, step)
		} else {
			p.addError(fmt.Sprintf("unexpected token %s in pipeline block, expected 'step'", p.current().Type), "")
			p.advance()
		}
		p.skipNewlines()
	}

	p.expectToken(TokenRBrace)
	pipeline.EndPos = p.currentPos()
	return pipeline
}

// parsePipelineStep parses: "name" { agent "x" input "..." output "..." depends_on [...] parallel true when "..." }
func (p *Parser) parsePipelineStep() *ast.PipelineStep {
	step := &ast.PipelineStep{StartPos: p.currentPos()}
	step.Name = p.expectString("step name")

	p.expectToken(TokenLBrace)
	p.skipNewlines()

	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}

		switch {
		case p.check(TokenAgent):
			p.advance()
			step.Agent = p.expectString("step agent")
		case p.check(TokenInput):
			p.advance()
			step.Input = p.expectString("step input")
		case p.check(TokenOutput):
			p.advance()
			step.Output = p.expectString("step output")
		case p.check(TokenDependsOn):
			p.advance()
			step.DependsOn = p.parseStringList()
		case p.check(TokenParallel):
			p.advance()
			step.Parallel = p.expectBool("parallel")
		case p.check(TokenWhen):
			p.advance()
			step.When = p.expectString("step condition")
		default:
			p.addError(fmt.Sprintf("unexpected token %s in step block", p.current().Type), "")
			p.advance()
		}
		p.skipNewlines()
	}

	p.expectToken(TokenRBrace)
	step.EndPos = p.currentPos()
	return step
}

// ---------------------------------------------------------------------------
// IntentLang 3.0: Config, Validate, Eval, On Input, Control Flow parsers
// ---------------------------------------------------------------------------

// parseConfigBlock parses: config { name type [required] [secret] [default value] "description" ... }
// Called after 'config' keyword has been consumed.
func (p *Parser) parseConfigBlock() []*ast.ConfigParam {
	var params []*ast.ConfigParam
	p.expectToken(TokenLBrace)
	p.skipNewlines()

	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}

		param := &ast.ConfigParam{StartPos: p.currentPos()}
		param.Name = p.expectStringOrIdent("config param name")
		param.Type = p.expectStringOrIdent("config param type")

		// Parse optional modifiers: required, secret, default
	modLoop:
		for !p.isAtEnd() && !p.check(TokenNewline) && !p.check(TokenRBrace) && !p.check(TokenString) {
			switch {
			case p.check(TokenRequired):
				p.advance()
				param.Required = true
			case p.check(TokenSecret):
				p.advance()
				param.Secret = true
			case p.check(TokenDefault):
				p.advance()
				param.HasDefault = true
				switch {
				case p.check(TokenString):
					param.Default = p.advance().Literal
				case p.check(TokenNumber):
					param.Default = p.advance().Literal
				case p.check(TokenTrue):
					param.Default = p.advance().Literal
				case p.check(TokenFalse):
					param.Default = p.advance().Literal
				default:
					param.Default = p.expectStringOrIdent("default value")
				}
			default:
				break modLoop
			}
		}

		p.skipNewlines()

		// Optional description string
		if p.check(TokenString) {
			param.Description = p.advance().Literal
		}

		param.EndPos = p.currentPos()
		params = append(params, param)
		p.skipNewlines()
	}
	p.expectToken(TokenRBrace)
	return params
}

// parseValidateBlock parses: validate { rule name severity [max_retries n] "message" when expression ... }
// Called after 'validate' keyword has been consumed.
func (p *Parser) parseValidateBlock() []*ast.ValidationRule {
	var rules []*ast.ValidationRule
	p.expectToken(TokenLBrace)
	p.skipNewlines()

	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}

		if !p.check(TokenRule) {
			p.addError(fmt.Sprintf("unexpected token %s in validate block", p.current().Type), "expected 'rule'")
			p.advance()
			continue
		}
		p.advance() // consume 'rule'

		rule := &ast.ValidationRule{StartPos: p.currentPos()}
		rule.Name = p.expectStringOrIdent("rule name")
		rule.Severity = p.expectStringOrIdent("rule severity")

		// Optional max_retries
		if p.check(TokenMaxRetries) {
			p.advance()
			rule.MaxRetries = p.expectInt("max_retries")
		}

		p.skipNewlines()

		// Description string
		if p.check(TokenString) {
			rule.Message = p.advance().Literal
		}

		p.skipNewlines()

		// when expression (collects until next 'rule' or '}')
		if p.check(TokenWhen) {
			p.advance()
			rule.Expression = p.collectExpressionUntil(TokenRule, TokenRBrace)
		}

		rule.EndPos = p.currentPos()
		rules = append(rules, rule)
		p.skipNewlines()
	}
	p.expectToken(TokenRBrace)
	return rules
}

// parseEvalBlock parses: eval { case name input "..." expect "..." scoring method [threshold f] [tags [...]] ... }
// Called after 'eval' keyword has been consumed.
func (p *Parser) parseEvalBlock() []*ast.EvalCase {
	var cases []*ast.EvalCase
	p.expectToken(TokenLBrace)
	p.skipNewlines()

	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}

		if !p.check(TokenCase) {
			p.addError(fmt.Sprintf("unexpected token %s in eval block", p.current().Type), "expected 'case'")
			p.advance()
			continue
		}
		p.advance() // consume 'case'

		ec := &ast.EvalCase{
			StartPos:  p.currentPos(),
			Threshold: 0.8, // default threshold
		}
		ec.Name = p.expectStringOrIdent("eval case name")

		p.skipNewlines()

		// Parse case attributes until next 'case' or '}'
		for !p.check(TokenCase) && !p.check(TokenRBrace) && !p.isAtEnd() {
			p.skipNewlines()
			if p.check(TokenCase) || p.check(TokenRBrace) {
				break
			}

			switch {
			case p.check(TokenInput):
				p.advance()
				ec.Input = p.expectString("eval input")
			case p.current().Literal == "expect":
				p.advance()
				ec.Expected = p.expectString("expected output")
			case p.check(TokenScoring):
				p.advance()
				ec.Scoring = p.expectStringOrIdent("scoring method")
				if p.check(TokenThreshold) {
					p.advance()
					ec.Threshold = p.expectFloat("threshold")
				}
			case p.check(TokenTags):
				p.advance()
				ec.Tags = p.parseStringList()
			default:
				p.addError(fmt.Sprintf("unexpected token %s in eval case", p.current().Type), "expected 'input', 'expect', 'scoring', or 'tags'")
				p.advance()
			}
			p.skipNewlines()
		}

		ec.EndPos = p.currentPos()
		cases = append(cases, ec)
		p.skipNewlines()
	}
	p.expectToken(TokenRBrace)
	return cases
}

// parseOnInputBlock parses: on input { statements... }
// Called after 'on' keyword has been consumed.
func (p *Parser) parseOnInputBlock() *ast.OnInputBlock {
	block := &ast.OnInputBlock{StartPos: p.currentPos()}

	// 'on' already consumed, expect 'input'
	if p.check(TokenInput) {
		p.advance()
	} else {
		p.addError("expected 'input' after 'on'", "")
	}

	p.expectToken(TokenLBrace)
	p.skipNewlines()

	block.Statements = p.parseOnInputStatements()

	p.expectToken(TokenRBrace)
	block.EndPos = p.currentPos()
	return block
}

// parseOnInputStatements parses statements within an on input block.
func (p *Parser) parseOnInputStatements() []ast.OnInputStmt {
	var stmts []ast.OnInputStmt

	for !p.check(TokenRBrace) && !p.isAtEnd() {
		p.skipNewlines()
		if p.check(TokenRBrace) {
			break
		}

		switch {
		case p.check(TokenUse):
			stmts = append(stmts, p.parseUseSkillStmt())
		case p.check(TokenDelegate):
			stmts = append(stmts, p.parseDelegateToStmt())
		case p.check(TokenRespond):
			stmts = append(stmts, p.parseRespondStmt())
		case p.check(TokenIf):
			stmts = append(stmts, p.parseIfBlock())
		case p.check(TokenFor):
			stmts = append(stmts, p.parseForEachBlock())
		default:
			p.addError(fmt.Sprintf("unexpected token %s in on input block", p.current().Type),
				"expected 'use', 'delegate', 'respond', 'if', or 'for'")
			p.advance()
		}
		p.skipNewlines()
	}

	return stmts
}

// parseUseSkillStmt parses: use skill <name> [with { key: value, ... }]
func (p *Parser) parseUseSkillStmt() *ast.UseSkillStmt {
	stmt := &ast.UseSkillStmt{StartPos: p.currentPos()}
	p.expect(TokenUse)

	if p.check(TokenSkill) {
		p.advance()
	}

	stmt.SkillName = p.expectStringOrIdent("skill name")

	// Optional with block
	if p.check(TokenWith) {
		p.advance()
		stmt.Params = make(map[string]string)
		p.expectToken(TokenLBrace)
		p.skipNewlines()

		for !p.check(TokenRBrace) && !p.isAtEnd() {
			p.skipNewlines()
			if p.check(TokenRBrace) {
				break
			}

			key := p.expectStringOrIdent("param key")
			p.expectToken(TokenColon)
			value := p.collectExpressionUntil(TokenComma, TokenRBrace, TokenNewline)
			stmt.Params[key] = value

			if p.check(TokenComma) {
				p.advance()
			}
			p.skipNewlines()
		}
		p.expectToken(TokenRBrace)
	}

	stmt.EndPos = p.currentPos()
	return stmt
}

// parseDelegateToStmt parses: delegate to <agent_name>
func (p *Parser) parseDelegateToStmt() *ast.DelegateToStmt {
	stmt := &ast.DelegateToStmt{StartPos: p.currentPos()}
	p.expect(TokenDelegate)

	if p.check(TokenTo) {
		p.advance()
	}

	stmt.AgentName = p.expectStringOrIdent("agent name")
	stmt.EndPos = p.currentPos()
	return stmt
}

// parseRespondStmt parses: respond <expression>
func (p *Parser) parseRespondStmt() *ast.RespondStmt {
	stmt := &ast.RespondStmt{StartPos: p.currentPos()}
	p.expect(TokenRespond)

	if p.check(TokenString) {
		stmt.Expression = p.advance().Literal
	} else {
		stmt.Expression = p.collectExpressionUntil(TokenNewline, TokenRBrace)
	}

	stmt.EndPos = p.currentPos()
	return stmt
}

// parseIfBlock parses: if <expression> { ... } [else if <expression> { ... }]* [else { ... }]
func (p *Parser) parseIfBlock() *ast.IfBlock {
	block := &ast.IfBlock{StartPos: p.currentPos()}
	p.expect(TokenIf)

	block.Condition = p.collectExpressionUntil(TokenLBrace)

	p.expectToken(TokenLBrace)
	p.skipNewlines()
	block.Body = p.parseOnInputStatements()
	p.expectToken(TokenRBrace)

	p.skipNewlines()

	// Parse else if / else chains
	for p.check(TokenElse) {
		p.advance()

		if p.check(TokenIf) {
			// else if
			p.advance()
			elseIf := &ast.ElseIfClause{StartPos: p.currentPos()}
			elseIf.Condition = p.collectExpressionUntil(TokenLBrace)

			p.expectToken(TokenLBrace)
			p.skipNewlines()
			elseIf.Body = p.parseOnInputStatements()
			p.expectToken(TokenRBrace)
			elseIf.EndPos = p.currentPos()

			block.ElseIfs = append(block.ElseIfs, elseIf)
			p.skipNewlines()
		} else {
			// else
			p.expectToken(TokenLBrace)
			p.skipNewlines()
			block.ElseBody = p.parseOnInputStatements()
			p.expectToken(TokenRBrace)
			break
		}
	}

	block.EndPos = p.currentPos()
	return block
}

// parseForEachBlock parses: for each <variable> in <collection_expression> { ... }
func (p *Parser) parseForEachBlock() *ast.ForEachBlock {
	block := &ast.ForEachBlock{StartPos: p.currentPos()}
	p.expect(TokenFor)

	if p.check(TokenEach) {
		p.advance()
	}

	block.Variable = p.expectStringOrIdent("loop variable")

	if p.check(TokenIn) {
		p.advance()
	} else {
		p.addError("expected 'in' in for each loop", "")
	}

	block.Collection = p.collectExpressionUntil(TokenLBrace)

	p.expectToken(TokenLBrace)
	p.skipNewlines()
	block.Body = p.parseOnInputStatements()
	p.expectToken(TokenRBrace)

	block.EndPos = p.currentPos()
	return block
}

// collectExpressionUntil collects token literals into an expression string until
// one of the stop tokens is encountered. Handles dot-separated property access
// and preserves string literals with quotes.
func (p *Parser) collectExpressionUntil(stopTokens ...TokenType) string {
	var parts []string
	for !p.isAtEnd() && !p.isStopToken(stopTokens) {
		if p.check(TokenNewline) {
			p.advance()
			continue
		}
		tok := p.current()
		switch tok.Type {
		case TokenString:
			parts = append(parts, `"`+tok.Literal+`"`)
		case TokenDot:
			// Attach dot to previous part for property access
			if len(parts) > 0 {
				parts[len(parts)-1] += "."
			}
		default:
			// If previous part ends with dot, append without space
			if len(parts) > 0 && strings.HasSuffix(parts[len(parts)-1], ".") {
				parts[len(parts)-1] += tok.Literal
			} else {
				parts = append(parts, tok.Literal)
			}
		}
		p.advance()
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

// isStopToken checks if the current token matches any of the given stop tokens.
func (p *Parser) isStopToken(stops []TokenType) bool {
	for _, stop := range stops {
		if p.check(stop) {
			return true
		}
	}
	return false
}

// expectToken for TokenColon (needed for with { key: value } syntax)
// Already handled by the generic expectToken method.

// AllNames returns all registered names for duplicate checking and suggestions.
func (p *Parser) AllNames() map[string]map[string]bool {
	return p.names
}
