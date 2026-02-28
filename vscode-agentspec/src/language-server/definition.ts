import * as vscode from "vscode";

/**
 * Register the IntentLang go-to-definition provider.
 *
 * Resolves references like:
 *   uses prompt "name"       -> prompt "name" { ... }
 *   uses skill "name"        -> skill "name" { ... }
 *   use skill "name"         -> skill "name" { ... } (in on input blocks)
 *   prompt "name"            -> prompt "name" { ... } (agent attribute)
 *   delegate to "name"       -> agent "name" { ... }
 *   agent "name"             -> agent "name" { ... } (in pipeline steps)
 *   depends_on ["name"]      -> step "name" { ... }
 *   fallback "name"          -> agent "name" { ... }
 *   import "./path.ias" as x -> opens the file
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

    // Handle import — open the referenced file
    const importMatch = lineText.match(/^\s*import\s+"([^"]+)"/);
    if (importMatch && nameAtCursor === importMatch[1]) {
      return this.resolveImportPath(document, importMatch[1]);
    }

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
    const regex = /"([^"]+)"/g;
    let match;

    while ((match = regex.exec(line)) !== null) {
      const start = match.index + 1;
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

    // prompt "name" (as agent attribute — no brace on same line)
    if (line.match(/^\s+prompt\s+"/) && !line.match(/\{/)) {
      return { kind: "prompt", name };
    }

    // uses skill "name" or use skill "name"
    if (line.match(/uses?\s+skill\s+"/)) {
      return { kind: "skill", name };
    }

    // delegate to "name"
    if (line.match(/delegate\s+to\s+"/)) {
      return { kind: "agent", name };
    }

    // agent "name" in pipeline step (indented)
    if (line.match(/^\s+agent\s+"/) && !line.match(/\{/)) {
      return { kind: "agent", name };
    }

    // depends_on ["name"]
    if (line.match(/depends_on\s*\[/)) {
      return { kind: "step", name };
    }

    // fallback "name"
    if (line.match(/fallback\s+"/)) {
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

    const regex = new RegExp(`^\\s*${kind}\\s+"${escapeRegex(name)}"\\s*\\{?`);

    for (let i = 0; i < lines.length; i++) {
      if (regex.test(lines[i])) {
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

  private resolveImportPath(
    document: vscode.TextDocument,
    importPath: string
  ): vscode.Location | undefined {
    // Resolve relative to the current file
    const dir = vscode.Uri.joinPath(document.uri, "..");
    const resolved = vscode.Uri.joinPath(dir, importPath);
    return new vscode.Location(resolved, new vscode.Position(0, 0));
  }
}

function escapeRegex(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
