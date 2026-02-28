import * as vscode from "vscode";

/**
 * Register the IntentLang completion provider for keyword, resource type,
 * and cross-reference completion.
 */
export function registerCompletionProvider(context: vscode.ExtensionContext) {
  const provider = vscode.languages.registerCompletionItemProvider(
    { language: "intentlang", scheme: "file" },
    new IntentLangCompletionProvider(),
    '"', // Trigger on quote for name references
    " " // Trigger on space for keyword completion
  );

  context.subscriptions.push(provider);
}

class IntentLangCompletionProvider implements vscode.CompletionItemProvider {
  provideCompletionItems(
    document: vscode.TextDocument,
    position: vscode.Position,
    _token: vscode.CancellationToken,
    _context: vscode.CompletionContext
  ): vscode.CompletionItem[] {
    const lineText = document.lineAt(position).text;
    const linePrefix = lineText.substring(0, position.character);

    // After "uses prompt " or "prompt " (inside agent) — suggest prompt names
    if (linePrefix.match(/uses\s+prompt\s+"$/) || linePrefix.match(/^\s+prompt\s+"$/)) {
      return this.getResourceNames(document, "prompt");
    }

    // After "uses skill " or "use skill " — suggest skill names
    if (linePrefix.match(/uses\s+skill\s+"$/) || linePrefix.match(/use\s+skill\s+"$/)) {
      return this.getResourceNames(document, "skill");
    }

    // After "delegate to " — suggest agent names
    if (linePrefix.match(/delegate\s+to\s+"$/)) {
      return this.getResourceNames(document, "agent");
    }

    // After "agent " in pipeline step — suggest agent names
    if (linePrefix.match(/^\s+agent\s+"$/)) {
      return this.getResourceNames(document, "agent");
    }

    // After "depends_on [" — suggest step names
    if (linePrefix.match(/depends_on\s*\[.*"$/)) {
      return this.getStepNames(document);
    }

    // After "strategy " — suggest strategy values
    if (linePrefix.match(/strategy\s+"$/)) {
      return this.getStrategyCompletions();
    }

    // After "scoring " — suggest scoring methods
    if (linePrefix.match(/scoring\s+$/)) {
      return this.getScoringCompletions();
    }

    // After "target " — suggest target types
    if (linePrefix.match(/target\s+"$/)) {
      return this.getTargetCompletions();
    }

    // After "model " — suggest model strings
    if (linePrefix.match(/model\s+"$/)) {
      return this.getModelCompletions();
    }

    // Block-level keywords (empty line or start of line)
    if (linePrefix.match(/^\s*$/)) {
      return this.getTopLevelCompletions();
    }

    // Attribute-level keywords inside a block
    if (linePrefix.match(/^\s+\S*$/)) {
      return this.getAttributeCompletions(document, position);
    }

    return [];
  }

  private getResourceNames(
    document: vscode.TextDocument,
    kind: string
  ): vscode.CompletionItem[] {
    const items: vscode.CompletionItem[] = [];
    const regex = new RegExp(`${kind}\\s+"([^"]+)"`, "g");
    const text = document.getText();
    let match;

    while ((match = regex.exec(text)) !== null) {
      const item = new vscode.CompletionItem(
        match[1],
        vscode.CompletionItemKind.Reference
      );
      item.detail = `${kind}: ${match[1]}`;
      items.push(item);
    }

    return items;
  }

  private getStepNames(
    document: vscode.TextDocument
  ): vscode.CompletionItem[] {
    const items: vscode.CompletionItem[] = [];
    const regex = /step\s+"([^"]+)"/g;
    const text = document.getText();
    let match;

    while ((match = regex.exec(text)) !== null) {
      const item = new vscode.CompletionItem(
        match[1],
        vscode.CompletionItemKind.Reference
      );
      item.detail = `step: ${match[1]}`;
      items.push(item);
    }

    return items;
  }

  private getStrategyCompletions(): vscode.CompletionItem[] {
    const strategies = [
      { name: "react", detail: "ReAct — Reason+Act loop (default)" },
      { name: "plan_execute", detail: "Plan first, then execute steps" },
      { name: "reflexion", detail: "Execute, self-critique, iterate" },
      { name: "router", detail: "Classify input, dispatch to sub-agent" },
      { name: "map_reduce", detail: "Split input, fan-out, merge results" },
    ];

    return strategies.map((s) => {
      const item = new vscode.CompletionItem(
        s.name,
        vscode.CompletionItemKind.EnumMember
      );
      item.detail = s.detail;
      return item;
    });
  }

  private getScoringCompletions(): vscode.CompletionItem[] {
    const methods = [
      { name: "contains", detail: "Check if output contains expected string" },
      { name: "semantic", detail: "Semantic similarity scoring" },
      { name: "exact", detail: "Exact string match" },
      { name: "regex", detail: "Regular expression match" },
    ];

    return methods.map((m) => {
      const item = new vscode.CompletionItem(
        m.name,
        vscode.CompletionItemKind.EnumMember
      );
      item.detail = m.detail;
      return item;
    });
  }

  private getTargetCompletions(): vscode.CompletionItem[] {
    const targets = [
      { name: "process", detail: "Local process deployment" },
      { name: "docker", detail: "Docker container deployment" },
      { name: "kubernetes", detail: "Kubernetes deployment" },
      { name: "compose", detail: "Docker Compose deployment" },
    ];

    return targets.map((t) => {
      const item = new vscode.CompletionItem(
        t.name,
        vscode.CompletionItemKind.EnumMember
      );
      item.detail = t.detail;
      return item;
    });
  }

  private getModelCompletions(): vscode.CompletionItem[] {
    const models = [
      { name: "ollama/llama3.2", detail: "Ollama — Llama 3.2 (local)" },
      { name: "ollama/mistral", detail: "Ollama — Mistral (local)" },
      { name: "ollama/codellama:7b", detail: "Ollama — CodeLlama 7B (local)" },
      { name: "ollama/qwen2.5-coder:7b", detail: "Ollama — Qwen 2.5 Coder (local)" },
      { name: "ollama/deepseek-coder:6.7b", detail: "Ollama — DeepSeek Coder (local)" },
      { name: "claude-sonnet-4-20250514", detail: "Anthropic — Claude Sonnet 4" },
      { name: "claude-haiku-3-5-20241022", detail: "Anthropic — Claude Haiku 3.5" },
      { name: "openai/gpt-4o", detail: "OpenAI — GPT-4o" },
      { name: "openai/gpt-4o-mini", detail: "OpenAI — GPT-4o Mini" },
    ];

    return models.map((m) => {
      const item = new vscode.CompletionItem(
        m.name,
        vscode.CompletionItemKind.Value
      );
      item.detail = m.detail;
      return item;
    });
  }

  private getTopLevelCompletions(): vscode.CompletionItem[] {
    const blocks = [
      { name: "package", detail: "Package declaration", kind: vscode.CompletionItemKind.Module },
      { name: "import", detail: "Import a package or file", kind: vscode.CompletionItemKind.Module },
      { name: "agent", detail: "Define an agent", kind: vscode.CompletionItemKind.Class },
      { name: "prompt", detail: "Define a prompt template", kind: vscode.CompletionItemKind.Text },
      { name: "skill", detail: "Define a skill with tool binding", kind: vscode.CompletionItemKind.Method },
      { name: "deploy", detail: "Define a deployment target", kind: vscode.CompletionItemKind.Module },
      { name: "pipeline", detail: "Define a multi-agent pipeline", kind: vscode.CompletionItemKind.Interface },
      { name: "environment", detail: "Define environment overrides", kind: vscode.CompletionItemKind.Module },
      { name: "secret", detail: "Define a secret reference", kind: vscode.CompletionItemKind.Key },
      { name: "type", detail: "Define a custom type", kind: vscode.CompletionItemKind.Struct },
    ];

    return blocks.map((b) => {
      const item = new vscode.CompletionItem(b.name, b.kind);
      item.detail = b.detail;
      return item;
    });
  }

  private getAttributeCompletions(
    document: vscode.TextDocument,
    position: vscode.Position
  ): vscode.CompletionItem[] {
    // Determine what block we're in by scanning up for the block type
    const blockType = this.findEnclosingBlock(document, position);

    switch (blockType) {
      case "agent":
        return this.makeAttrItems([
          "model", "prompt", "strategy", "max_turns", "timeout", "token_budget",
          "temperature", "stream", "on_error", "max_retries", "fallback",
          "uses prompt", "uses skill",
          "config", "validate", "eval", "on input",
        ]);
      case "prompt":
        return this.makeAttrItems(["content"]);
      case "skill":
        return this.makeAttrItems(["description", "input", "output", "tool"]);
      case "config":
        return this.makeAttrItems(["string", "int", "float", "bool"]);
      case "validate":
        return this.makeAttrItems(["rule"]);
      case "eval":
        return this.makeAttrItems(["case"]);
      case "on":
        return this.makeAttrItems(["if", "else", "for each", "use skill", "delegate to", "respond"]);
      case "if":
        return this.makeAttrItems(["use skill", "delegate to", "respond"]);
      case "deploy":
        return this.makeAttrItems([
          "port", "replicas", "cpu", "memory", "health", "autoscale", "default",
        ]);
      case "pipeline":
        return this.makeAttrItems(["step"]);
      case "step":
        return this.makeAttrItems(["agent", "input", "output", "depends_on"]);
      case "tool":
        return this.makeAttrItems([
          "server", "method", "url", "binary", "command", "args",
          "language", "code", "body_template",
        ]);
      case "memory":
        return this.makeAttrItems(["strategy", "max_messages"]);
      case "input":
      case "output":
        return this.makeAttrItems(["string", "int", "float", "bool", "required"]);
      default:
        return [];
    }
  }

  private findEnclosingBlock(
    document: vscode.TextDocument,
    position: vscode.Position
  ): string {
    for (let i = position.line; i >= 0; i--) {
      const line = document.lineAt(i).text;
      const match = line.match(
        /^\s*(agent|prompt|skill|deploy|pipeline|step|tool|type|delegate|memory|health|autoscale|resources|variables|config|validate|eval|on|if|else|for|input|output|environment|secret)\s/
      );
      if (match) {
        return match[1];
      }
    }
    return "";
  }

  private makeAttrItems(attrs: string[]): vscode.CompletionItem[] {
    return attrs.map((a) => {
      const item = new vscode.CompletionItem(
        a,
        vscode.CompletionItemKind.Property
      );
      return item;
    });
  }
}
