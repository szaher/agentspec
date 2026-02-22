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
