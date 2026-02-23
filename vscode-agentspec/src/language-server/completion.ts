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

    // After "uses prompt " or "uses skill " — suggest names from document
    if (linePrefix.match(/uses\s+prompt\s+"$/)) {
      return this.getResourceNames(document, "prompt");
    }
    if (linePrefix.match(/uses\s+skill\s+"$/)) {
      return this.getResourceNames(document, "skill");
    }

    // After "agent = " in pipeline step — suggest agent names
    if (linePrefix.match(/agent\s*=\s*"$/)) {
      return this.getResourceNames(document, "agent");
    }

    // After "depends_on = [" — suggest step names
    if (linePrefix.match(/depends_on\s*=\s*\[.*"$/)) {
      return this.getStepNames(document);
    }

    // After "strategy = " — suggest strategy values
    if (linePrefix.match(/strategy\s*=\s*"$/)) {
      return this.getStrategyCompletions();
    }

    // After "target " — suggest target types
    if (linePrefix.match(/target\s+"$/)) {
      return this.getTargetCompletions();
    }

    // Block-level keywords
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

  private getTopLevelCompletions(): vscode.CompletionItem[] {
    const blocks = [
      { name: "agent", detail: "Define an agent", kind: vscode.CompletionItemKind.Class },
      { name: "prompt", detail: "Define a prompt template", kind: vscode.CompletionItemKind.Text },
      { name: "skill", detail: "Define a skill with tool binding", kind: vscode.CompletionItemKind.Method },
      { name: "deploy", detail: "Define a deployment target", kind: vscode.CompletionItemKind.Module },
      { name: "pipeline", detail: "Define a multi-agent pipeline", kind: vscode.CompletionItemKind.Interface },
      { name: "type", detail: "Define a custom type", kind: vscode.CompletionItemKind.Struct },
      { name: "package", detail: "Package declaration", kind: vscode.CompletionItemKind.Module },
      { name: "import", detail: "Import another package", kind: vscode.CompletionItemKind.Module },
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
          "model", "strategy", "max_turns", "timeout", "token_budget",
          "temperature", "stream", "on_error", "max_retries", "fallback",
          "uses prompt", "uses skill",
        ]);
      case "prompt":
        return this.makeAttrItems(["content", "variables"]);
      case "skill":
        return this.makeAttrItems(["description", "tool"]);
      case "deploy":
        return this.makeAttrItems([
          "port", "replicas", "cpu", "memory", "health", "autoscale",
        ]);
      case "pipeline":
        return this.makeAttrItems(["step"]);
      case "step":
        return this.makeAttrItems(["agent", "input", "output", "depends_on"]);
      case "tool":
        return this.makeAttrItems([
          "server", "method", "url", "command", "args",
        ]);
      case "memory":
        return this.makeAttrItems(["strategy", "max_messages"]);
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
        /^\s*(agent|prompt|skill|deploy|pipeline|step|tool|type|delegate|memory|health|autoscale|resources|variables)\s/
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
