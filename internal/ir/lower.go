package ir

import (
	"fmt"

	"github.com/szaher/designs/agentz/internal/ast"
)

// Lower converts an AST File to an IR Document by resolving
// references, flattening resources, and computing FQNs.
func Lower(f *ast.File) (*Document, error) {
	if f.Package == nil {
		return nil, fmt.Errorf("missing package declaration")
	}

	doc := &Document{
		IRVersion:   "1.0",
		LangVersion: f.Package.LangVersion,
		Package: Package{
			Name:    f.Package.Name,
			Version: f.Package.Version,
		},
	}

	pkgName := f.Package.Name

	for _, stmt := range f.Statements {
		switch s := stmt.(type) {
		case *ast.Prompt:
			r := Resource{
				Kind: "Prompt",
				Name: s.Name,
				FQN:  fmt.Sprintf("%s/Prompt/%s", pkgName, s.Name),
				Attributes: map[string]interface{}{
					"content": s.Content,
				},
			}
			if s.Version != "" {
				r.Attributes["version"] = s.Version
			}
			if len(s.Variables) > 0 {
				vars := make([]interface{}, 0, len(s.Variables))
				for _, v := range s.Variables {
					vd := map[string]interface{}{
						"name": v.Name,
						"type": v.Type,
					}
					if v.Required {
						vd["required"] = true
					}
					if v.Default != "" {
						vd["default"] = v.Default
					}
					vars = append(vars, vd)
				}
				r.Attributes["variables"] = vars
			}
			if len(s.Metadata) > 0 {
				r.Metadata = strMapToInterface(s.Metadata)
			}
			r.Hash = ComputeHash(r.Attributes)
			doc.Resources = append(doc.Resources, r)

		case *ast.Skill:
			attrs := map[string]interface{}{
				"description": s.Description,
			}
			if s.Execution != nil {
				attrs["execution"] = map[string]interface{}{
					"type":  s.Execution.Type,
					"value": s.Execution.Value,
				}
			}
			if s.ToolConfig != nil {
				tc := map[string]interface{}{
					"type": s.ToolConfig.Type,
				}
				switch s.ToolConfig.Type {
				case "mcp":
					tc["server_tool"] = s.ToolConfig.ServerTool
				case "http":
					if s.ToolConfig.Method != "" {
						tc["method"] = s.ToolConfig.Method
					}
					if s.ToolConfig.URL != "" {
						tc["url"] = s.ToolConfig.URL
					}
					if len(s.ToolConfig.Headers) > 0 {
						tc["headers"] = strMapToInterface(s.ToolConfig.Headers)
					}
					if s.ToolConfig.BodyTemplate != "" {
						tc["body_template"] = s.ToolConfig.BodyTemplate
					}
				case "command":
					if s.ToolConfig.Binary != "" {
						tc["binary"] = s.ToolConfig.Binary
					}
					if len(s.ToolConfig.Args) > 0 {
						args := make([]interface{}, len(s.ToolConfig.Args))
						for i, a := range s.ToolConfig.Args {
							args[i] = a
						}
						tc["args"] = args
					}
					if s.ToolConfig.Timeout != "" {
						tc["timeout"] = s.ToolConfig.Timeout
					}
					if len(s.ToolConfig.Env) > 0 {
						tc["env"] = strMapToInterface(s.ToolConfig.Env)
					}
					if len(s.ToolConfig.Secrets) > 0 {
						tc["secrets"] = strMapToInterface(s.ToolConfig.Secrets)
					}
				case "inline":
					if s.ToolConfig.Language != "" {
						tc["language"] = s.ToolConfig.Language
					}
					if s.ToolConfig.Code != "" {
						tc["code"] = s.ToolConfig.Code
					}
					if s.ToolConfig.MemoryLimit != "" {
						tc["memory_limit"] = s.ToolConfig.MemoryLimit
					}
				}
				if s.ToolConfig.Timeout != "" && s.ToolConfig.Type != "command" {
					tc["timeout"] = s.ToolConfig.Timeout
				}
				attrs["tool"] = tc
			}
			if len(s.Input) > 0 {
				inputs := make([]interface{}, 0, len(s.Input))
				for _, f := range s.Input {
					field := map[string]interface{}{
						"name": f.Name,
						"type": f.Type,
					}
					if f.Required {
						field["required"] = true
					}
					inputs = append(inputs, field)
				}
				attrs["input"] = inputs
			}
			if len(s.Output) > 0 {
				outputs := make([]interface{}, 0, len(s.Output))
				for _, f := range s.Output {
					field := map[string]interface{}{
						"name": f.Name,
						"type": f.Type,
					}
					if f.Required {
						field["required"] = true
					}
					outputs = append(outputs, field)
				}
				attrs["output"] = outputs
			}
			r := Resource{
				Kind:       "Skill",
				Name:       s.Name,
				FQN:        fmt.Sprintf("%s/Skill/%s", pkgName, s.Name),
				Attributes: attrs,
			}
			if len(s.Metadata) > 0 {
				r.Metadata = strMapToInterface(s.Metadata)
			}
			r.Hash = ComputeHash(r.Attributes)
			doc.Resources = append(doc.Resources, r)

		case *ast.Agent:
			attrs := map[string]interface{}{
				"model": s.Model,
			}
			var refs []string
			if s.Prompt != nil {
				attrs["prompt"] = s.Prompt.Name
				refs = append(refs, fmt.Sprintf("%s/Prompt/%s", pkgName, s.Prompt.Name))
			}
			if len(s.Skills) > 0 {
				skills := make([]interface{}, 0, len(s.Skills))
				for _, sk := range s.Skills {
					skills = append(skills, sk.Name)
					refs = append(refs, fmt.Sprintf("%s/Skill/%s", pkgName, sk.Name))
				}
				attrs["skills"] = skills
			}
			if s.Client != nil {
				attrs["client"] = s.Client.Name
				refs = append(refs, fmt.Sprintf("%s/MCPClient/%s", pkgName, s.Client.Name))
			}
			if s.Strategy != "" {
				attrs["strategy"] = s.Strategy
			}
			if s.MaxTurns > 0 {
				attrs["max_turns"] = s.MaxTurns
			}
			if s.Timeout != "" {
				attrs["timeout"] = s.Timeout
			}
			if s.TokenBudget > 0 {
				attrs["token_budget"] = s.TokenBudget
			}
			if s.HasTemp {
				attrs["temperature"] = s.Temperature
			}
			if s.Stream != nil {
				attrs["stream"] = *s.Stream
			}
			if s.OnError != "" {
				attrs["on_error"] = s.OnError
			}
			if s.MaxRetries > 0 {
				attrs["max_retries"] = s.MaxRetries
			}
			if s.Fallback != "" {
				attrs["fallback"] = s.Fallback
				refs = append(refs, fmt.Sprintf("%s/Agent/%s", pkgName, s.Fallback))
			}
			if s.MemoryCfg != nil {
				mem := map[string]interface{}{}
				if s.MemoryCfg.Strategy != "" {
					mem["strategy"] = s.MemoryCfg.Strategy
				}
				if s.MemoryCfg.MaxMessages > 0 {
					mem["max_messages"] = s.MemoryCfg.MaxMessages
				}
				attrs["memory"] = mem
			}
			if len(s.Delegates) > 0 {
				delegates := make([]interface{}, 0, len(s.Delegates))
				for _, d := range s.Delegates {
					delegates = append(delegates, map[string]interface{}{
						"agent":     d.AgentRef,
						"condition": d.Condition,
					})
					refs = append(refs, fmt.Sprintf("%s/Agent/%s", pkgName, d.AgentRef))
				}
				attrs["delegates"] = delegates
			}
			r := Resource{
				Kind:       "Agent",
				Name:       s.Name,
				FQN:        fmt.Sprintf("%s/Agent/%s", pkgName, s.Name),
				Attributes: attrs,
				References: refs,
			}
			if len(s.Metadata) > 0 {
				r.Metadata = strMapToInterface(s.Metadata)
			}
			r.Hash = ComputeHash(r.Attributes)
			doc.Resources = append(doc.Resources, r)

		case *ast.MCPServer:
			attrs := map[string]interface{}{
				"transport": s.Transport,
			}
			if s.Command != "" {
				attrs["command"] = s.Command
			}
			if len(s.Args) > 0 {
				args := make([]interface{}, len(s.Args))
				for i, a := range s.Args {
					args[i] = a
				}
				attrs["args"] = args
			}
			if s.URL != "" {
				attrs["url"] = s.URL
			}
			if len(s.Env) > 0 {
				attrs["env"] = strMapToInterface(s.Env)
			}

			var refs []string
			if s.Auth != nil {
				attrs["auth"] = s.Auth.Name
				refs = append(refs, fmt.Sprintf("%s/Secret/%s", pkgName, s.Auth.Name))
			}
			for _, sk := range s.Skills {
				refs = append(refs, fmt.Sprintf("%s/Skill/%s", pkgName, sk.Name))
			}

			r := Resource{
				Kind:       "MCPServer",
				Name:       s.Name,
				FQN:        fmt.Sprintf("%s/MCPServer/%s", pkgName, s.Name),
				Attributes: attrs,
				References: refs,
			}
			if len(s.Metadata) > 0 {
				r.Metadata = strMapToInterface(s.Metadata)
			}
			r.Hash = ComputeHash(r.Attributes)
			doc.Resources = append(doc.Resources, r)

		case *ast.MCPClient:
			attrs := map[string]interface{}{}
			var refs []string
			if len(s.Servers) > 0 {
				servers := make([]interface{}, 0, len(s.Servers))
				for _, srv := range s.Servers {
					servers = append(servers, srv.Name)
					refs = append(refs, fmt.Sprintf("%s/MCPServer/%s", pkgName, srv.Name))
				}
				attrs["servers"] = servers
			}
			r := Resource{
				Kind:       "MCPClient",
				Name:       s.Name,
				FQN:        fmt.Sprintf("%s/MCPClient/%s", pkgName, s.Name),
				Attributes: attrs,
				References: refs,
			}
			if len(s.Metadata) > 0 {
				r.Metadata = strMapToInterface(s.Metadata)
			}
			r.Hash = ComputeHash(r.Attributes)
			doc.Resources = append(doc.Resources, r)

		case *ast.Secret:
			attrs := map[string]interface{}{
				"source": s.Source,
				"key":    s.Key,
			}
			r := Resource{
				Kind:       "Secret",
				Name:       s.Name,
				FQN:        fmt.Sprintf("%s/Secret/%s", pkgName, s.Name),
				Attributes: attrs,
			}
			r.Hash = ComputeHash(r.Attributes)
			doc.Resources = append(doc.Resources, r)

		case *ast.Environment:
			attrs := map[string]interface{}{
				"name": s.Name,
			}
			overrides := make([]interface{}, 0, len(s.Overrides))
			for _, o := range s.Overrides {
				overrides = append(overrides, map[string]interface{}{
					"resource":  o.Resource,
					"attribute": o.Attribute,
					"value":     o.Value,
				})
			}
			attrs["overrides"] = overrides
			r := Resource{
				Kind:       "Environment",
				Name:       s.Name,
				FQN:        fmt.Sprintf("%s/Environment/%s", pkgName, s.Name),
				Attributes: attrs,
			}
			r.Hash = ComputeHash(r.Attributes)
			doc.Resources = append(doc.Resources, r)

		case *ast.Policy:
			pol := Policy{Name: s.Name}
			for _, rule := range s.Rules {
				pol.Rules = append(pol.Rules, Rule{
					Action:   rule.Action,
					Resource: rule.Resource,
					Subject:  rule.Subject,
				})
			}
			doc.Policies = append(doc.Policies, pol)

		case *ast.Binding:
			b := Binding{
				Name:    s.Name,
				Adapter: s.Adapter,
				Default: s.Default,
			}
			if len(s.Config) > 0 {
				b.Config = strMapToInterface(s.Config)
			}
			doc.Bindings = append(doc.Bindings, b)

		case *ast.TypeDef:
			attrs := map[string]interface{}{
				"name": s.Name,
			}
			if len(s.EnumVals) > 0 {
				vals := make([]interface{}, len(s.EnumVals))
				for i, v := range s.EnumVals {
					vals[i] = v
				}
				attrs["enum"] = vals
			}
			if s.ListOf != "" {
				attrs["list_of"] = s.ListOf
			}
			if len(s.Fields) > 0 {
				fields := make([]interface{}, 0, len(s.Fields))
				for _, f := range s.Fields {
					field := map[string]interface{}{
						"name": f.Name,
						"type": f.Type,
					}
					if f.Required {
						field["required"] = true
					}
					if f.Default != "" {
						field["default"] = f.Default
					}
					fields = append(fields, field)
				}
				attrs["fields"] = fields
			}
			r := Resource{
				Kind:       "Type",
				Name:       s.Name,
				FQN:        fmt.Sprintf("%s/Type/%s", pkgName, s.Name),
				Attributes: attrs,
			}
			r.Hash = ComputeHash(r.Attributes)
			doc.Resources = append(doc.Resources, r)

		case *ast.Pipeline:
			attrs := map[string]interface{}{
				"name": s.Name,
			}
			var refs []string
			if len(s.Steps) > 0 {
				steps := make([]interface{}, 0, len(s.Steps))
				for _, step := range s.Steps {
					sd := map[string]interface{}{
						"name": step.Name,
					}
					if step.Agent != "" {
						sd["agent"] = step.Agent
						refs = append(refs, fmt.Sprintf("%s/Agent/%s", pkgName, step.Agent))
					}
					if step.Input != "" {
						sd["input"] = step.Input
					}
					if step.Output != "" {
						sd["output"] = step.Output
					}
					if len(step.DependsOn) > 0 {
						deps := make([]interface{}, len(step.DependsOn))
						for i, d := range step.DependsOn {
							deps[i] = d
						}
						sd["depends_on"] = deps
					}
					if step.Parallel {
						sd["parallel"] = true
					}
					if step.When != "" {
						sd["when"] = step.When
					}
					steps = append(steps, sd)
				}
				attrs["steps"] = steps
			}
			r := Resource{
				Kind:       "Pipeline",
				Name:       s.Name,
				FQN:        fmt.Sprintf("%s/Pipeline/%s", pkgName, s.Name),
				Attributes: attrs,
				References: refs,
			}
			r.Hash = ComputeHash(r.Attributes)
			doc.Resources = append(doc.Resources, r)

		case *ast.DeployTarget:
			cfg := map[string]interface{}{}
			if s.Port > 0 {
				cfg["port"] = s.Port
			}
			if s.Namespace != "" {
				cfg["namespace"] = s.Namespace
			}
			if s.Replicas > 0 {
				cfg["replicas"] = s.Replicas
			}
			if s.Image != "" {
				cfg["image"] = s.Image
			}
			if s.Resources != nil {
				res := map[string]interface{}{}
				if s.Resources.CPU != "" {
					res["cpu"] = s.Resources.CPU
				}
				if s.Resources.Memory != "" {
					res["memory"] = s.Resources.Memory
				}
				cfg["resources"] = res
			}
			if s.Health != nil {
				h := map[string]interface{}{}
				if s.Health.Path != "" {
					h["path"] = s.Health.Path
				}
				if s.Health.Interval != "" {
					h["interval"] = s.Health.Interval
				}
				if s.Health.Timeout != "" {
					h["timeout"] = s.Health.Timeout
				}
				cfg["health"] = h
			}
			if s.Autoscale != nil {
				a := map[string]interface{}{}
				if s.Autoscale.MinReplicas > 0 {
					a["min_replicas"] = s.Autoscale.MinReplicas
				}
				if s.Autoscale.MaxReplicas > 0 {
					a["max_replicas"] = s.Autoscale.MaxReplicas
				}
				if s.Autoscale.Metric != "" {
					a["metric"] = s.Autoscale.Metric
				}
				if s.Autoscale.Target > 0 {
					a["target"] = s.Autoscale.Target
				}
				cfg["autoscale"] = a
			}
			if len(s.Env) > 0 {
				cfg["env"] = strMapToInterface(s.Env)
			}
			if len(s.Secrets) > 0 {
				cfg["secrets"] = strMapToInterface(s.Secrets)
			}
			dt := DeployTarget{
				Name:    s.Name,
				Target:  s.Target,
				Default: s.Default,
			}
			if len(cfg) > 0 {
				dt.Config = cfg
			}
			doc.DeployTargets = append(doc.DeployTargets, dt)
		}
	}

	SortResources(doc.Resources)
	return doc, nil
}

func strMapToInterface(m map[string]string) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
