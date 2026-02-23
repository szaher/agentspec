import * as vscode from "vscode";

/**
 * Register diagnostics providers that run agentspec validate on save.
 */
export function registerDiagnostics(
  context: vscode.ExtensionContext,
  diagnosticCollection: vscode.DiagnosticCollection
) {
  // Validate on save
  context.subscriptions.push(
    vscode.workspace.onDidSaveTextDocument(async (document) => {
      if (document.languageId !== "intentlang") return;

      const config = vscode.workspace.getConfiguration("agentspec");
      if (!config.get<boolean>("validateOnSave", true)) return;

      await validateDocument(document, diagnosticCollection);
    })
  );

  // Validate on open
  context.subscriptions.push(
    vscode.workspace.onDidOpenTextDocument(async (document) => {
      if (document.languageId !== "intentlang") return;
      await validateDocument(document, diagnosticCollection);
    })
  );

  // Clear diagnostics when file is closed
  context.subscriptions.push(
    vscode.workspace.onDidCloseTextDocument((document) => {
      diagnosticCollection.delete(document.uri);
    })
  );

  // Validate already open documents
  vscode.workspace.textDocuments.forEach(async (document) => {
    if (document.languageId === "intentlang") {
      await validateDocument(document, diagnosticCollection);
    }
  });
}

/**
 * Run agentspec validate on a document and update diagnostics.
 */
export async function validateDocument(
  document: vscode.TextDocument,
  diagnosticCollection: vscode.DiagnosticCollection
): Promise<void> {
  const config = vscode.workspace.getConfiguration("agentspec");
  const execPath = config.get<string>("executablePath", "agentspec");

  try {
    const { exec } = await import("child_process");
    const { promisify } = await import("util");
    const execAsync = promisify(exec);

    try {
      await execAsync(`${execPath} validate "${document.fileName}"`);
      // No errors - clear diagnostics
      diagnosticCollection.set(document.uri, []);
    } catch (err: unknown) {
      const errObj = err as { stderr?: string; stdout?: string };
      const output = (errObj.stderr || errObj.stdout || "").toString();
      const diagnostics = parseValidationOutput(output, document);
      diagnosticCollection.set(document.uri, diagnostics);
    }
  } catch (err: unknown) {
    // agentspec binary not found or other system error
    const message = err instanceof Error ? err.message : String(err);
    const diagnostic = new vscode.Diagnostic(
      new vscode.Range(0, 0, 0, 0),
      `AgentSpec validation unavailable: ${message}`,
      vscode.DiagnosticSeverity.Warning
    );
    diagnostic.source = "agentspec";
    diagnosticCollection.set(document.uri, [diagnostic]);
  }
}

/**
 * Parse agentspec validate output into VS Code diagnostics.
 *
 * Expected format from agentspec validate:
 *   filename.ias:line:col: error: message
 *   filename.ias:line: error: message
 *   Error: message
 */
function parseValidationOutput(
  output: string,
  document: vscode.TextDocument
): vscode.Diagnostic[] {
  const diagnostics: vscode.Diagnostic[] = [];
  const lines = output.split("\n");

  for (const line of lines) {
    if (!line.trim()) continue;

    // Try pattern: file:line:col: severity: message
    const match = line.match(
      /(?:[^:]+):(\d+):(\d+):\s*(error|warning|info):\s*(.+)/
    );
    if (match) {
      const lineNum = Math.max(0, parseInt(match[1], 10) - 1);
      const col = Math.max(0, parseInt(match[2], 10) - 1);
      const severity = parseSeverity(match[3]);
      const message = match[4].trim();

      const range = new vscode.Range(lineNum, col, lineNum, col + 1);
      const diagnostic = new vscode.Diagnostic(range, message, severity);
      diagnostic.source = "agentspec";
      diagnostics.push(diagnostic);
      continue;
    }

    // Try pattern: file:line: severity: message
    const match2 = line.match(
      /(?:[^:]+):(\d+):\s*(error|warning|info):\s*(.+)/
    );
    if (match2) {
      const lineNum = Math.max(0, parseInt(match2[1], 10) - 1);
      const severity = parseSeverity(match2[2]);
      const message = match2[3].trim();

      const range = new vscode.Range(lineNum, 0, lineNum, 1000);
      const diagnostic = new vscode.Diagnostic(range, message, severity);
      diagnostic.source = "agentspec";
      diagnostics.push(diagnostic);
      continue;
    }

    // Try pattern: Error: message (no line info)
    const match3 = line.match(/^(?:Error|error):\s*(.+)/);
    if (match3) {
      const diagnostic = new vscode.Diagnostic(
        new vscode.Range(0, 0, 0, 0),
        match3[1].trim(),
        vscode.DiagnosticSeverity.Error
      );
      diagnostic.source = "agentspec";
      diagnostics.push(diagnostic);
    }
  }

  return diagnostics;
}

function parseSeverity(s: string): vscode.DiagnosticSeverity {
  switch (s.toLowerCase()) {
    case "error":
      return vscode.DiagnosticSeverity.Error;
    case "warning":
      return vscode.DiagnosticSeverity.Warning;
    case "info":
      return vscode.DiagnosticSeverity.Information;
    default:
      return vscode.DiagnosticSeverity.Error;
  }
}
