import * as vscode from "vscode";

/**
 * Register the IntentLang go-to-definition provider.
 *
 * Resolves references like:
 *   uses prompt "name"  -> prompt "name" { ... } block
 *   uses skill "name"   -> skill "name" { ... } block
 *   agent = "name"      -> agent "name" { ... } block (in pipeline steps)
 *   depends_on = ["x"]  -> step "x" { ... } block (in pipeline steps)
 */
export function registerDefinitionProvider(context: vscode.ExtensionContext) {
  const provider = vscode.languages.registerDefinitionProvider(
    { language: "intentlang", scheme: "file" },
    new IntentLangDefinitionProvider()
  );

  context.subscriptions.push(provider);
}

class IntentLangDefinitionProvider implements vscode.DefinitionProvider {
  provideDefinition(
    document: vscode.TextDocument,
    position: vscode.Position,
    _token: vscode.CancellationToken
  ): vscode.Definition | undefined {
    const lineText = document.lineAt(position).text;

    // Check if cursor is on a quoted name
    const nameAtCursor = this.getQuotedNameAtPosition(lineText, position.character);
    if (!nameAtCursor) return undefined;

    // Determine what kind of reference this is
    const ref = this.classifyReference(lineText, nameAtCursor);
    if (!ref) return undefined;

    // Find the definition
    return this.findDefinition(document, ref.kind, ref.name);
  }

  private getQuotedNameAtPosition(
    line: string,
    character: number
  ): string | undefined {
    // Find the quoted string that the cursor is within
    const regex = /"([^"]+)"/g;
    let match;

    while ((match = regex.exec(line)) !== null) {
      const start = match.index + 1; // after opening quote
      const end = start + match[1].length;
      if (character >= start && character <= end) {
        return match[1];
      }
    }

    return undefined;
  }

  private classifyReference(
    line: string,
    name: string
  ): { kind: string; name: string } | undefined {
    // uses prompt "name"
    if (line.match(/uses\s+prompt\s+"/)) {
      return { kind: "prompt", name };
    }

    // uses skill "name"
    if (line.match(/uses\s+skill\s+"/)) {
      return { kind: "skill", name };
    }

    // agent = "name" (in pipeline step)
    if (line.match(/agent\s*=\s*"/)) {
      return { kind: "agent", name };
    }

    // depends_on = ["name"]
    if (line.match(/depends_on\s*=\s*\[/)) {
      return { kind: "step", name };
    }

    // delegate to agent "name"
    if (line.match(/delegate\s+to\s+agent\s+"/)) {
      return { kind: "agent", name };
    }

    // fallback = "name"
    if (line.match(/fallback\s*=\s*"/)) {
      return { kind: "agent", name };
    }

    return undefined;
  }

  private findDefinition(
    document: vscode.TextDocument,
    kind: string,
    name: string
  ): vscode.Location | undefined {
    const text = document.getText();
    const lines = text.split("\n");

    // Build regex for block definition
    const regex = new RegExp(`^\\s*${kind}\\s+"${escapeRegex(name)}"\\s*\\{?`);

    for (let i = 0; i < lines.length; i++) {
      if (regex.test(lines[i])) {
        // Find the column where the name starts
        const nameIndex = lines[i].indexOf(`"${name}"`);
        const col = nameIndex >= 0 ? nameIndex + 1 : 0;
        return new vscode.Location(
          document.uri,
          new vscode.Position(i, col)
        );
      }
    }

    return undefined;
  }
}

function escapeRegex(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
