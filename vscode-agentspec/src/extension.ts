import * as vscode from "vscode";
import { registerDiagnostics, validateDocument } from "./language-server/diagnostics";
import { registerCompletionProvider } from "./language-server/completion";
import { registerDefinitionProvider } from "./language-server/definition";

let diagnosticCollection: vscode.DiagnosticCollection;

export function activate(context: vscode.ExtensionContext) {
  diagnosticCollection = vscode.languages.createDiagnosticCollection("intentlang");
  context.subscriptions.push(diagnosticCollection);

  // Register diagnostics (validate on save)
  registerDiagnostics(context, diagnosticCollection);

  // Register completion provider
  registerCompletionProvider(context);

  // Register go-to-definition provider
  registerDefinitionProvider(context);

  // Register format command
  context.subscriptions.push(
    vscode.commands.registerCommand("agentspec.format", async () => {
      const editor = vscode.window.activeTextEditor;
      if (!editor || editor.document.languageId !== "intentlang") {
        vscode.window.showWarningMessage("No IntentLang file is open");
        return;
      }
      await formatDocument(editor.document);
    })
  );

  // Register validate command
  context.subscriptions.push(
    vscode.commands.registerCommand("agentspec.validate", async () => {
      const editor = vscode.window.activeTextEditor;
      if (!editor || editor.document.languageId !== "intentlang") {
        vscode.window.showWarningMessage("No IntentLang file is open");
        return;
      }
      await validateDocument(editor.document, diagnosticCollection);
      vscode.window.showInformationMessage("Validation complete");
    })
  );

  // Register plan command
  context.subscriptions.push(
    vscode.commands.registerCommand("agentspec.plan", async () => {
      const editor = vscode.window.activeTextEditor;
      if (!editor || editor.document.languageId !== "intentlang") {
        vscode.window.showWarningMessage("No IntentLang file is open");
        return;
      }
      await showPlan(editor.document);
    })
  );

  // Register compile command
  context.subscriptions.push(
    vscode.commands.registerCommand("agentspec.compile", async () => {
      const editor = vscode.window.activeTextEditor;
      if (!editor || editor.document.languageId !== "intentlang") {
        vscode.window.showWarningMessage("No IntentLang file is open");
        return;
      }
      await runCliCommand("compile", editor.document);
    })
  );

  // Register eval command
  context.subscriptions.push(
    vscode.commands.registerCommand("agentspec.eval", async () => {
      const editor = vscode.window.activeTextEditor;
      if (!editor || editor.document.languageId !== "intentlang") {
        vscode.window.showWarningMessage("No IntentLang file is open");
        return;
      }
      await runCliCommand("eval", editor.document);
    })
  );

  // Register package command
  context.subscriptions.push(
    vscode.commands.registerCommand("agentspec.package", async () => {
      const editor = vscode.window.activeTextEditor;
      if (!editor || editor.document.languageId !== "intentlang") {
        vscode.window.showWarningMessage("No IntentLang file is open");
        return;
      }
      await runCliCommand("package", editor.document);
    })
  );

  // Format on save
  context.subscriptions.push(
    vscode.workspace.onDidSaveTextDocument(async (document) => {
      if (document.languageId !== "intentlang") return;

      const config = vscode.workspace.getConfiguration("agentspec");
      if (config.get<boolean>("formatOnSave", true)) {
        await formatDocument(document);
      }
    })
  );
}

export function deactivate() {
  if (diagnosticCollection) {
    diagnosticCollection.dispose();
  }
}

async function formatDocument(document: vscode.TextDocument): Promise<void> {
  const config = vscode.workspace.getConfiguration("agentspec");
  const execPath = config.get<string>("executablePath", "agentspec");

  try {
    const { exec } = await import("child_process");
    const { promisify } = await import("util");
    const execAsync = promisify(exec);

    const { stdout } = await execAsync(`${execPath} fmt "${document.fileName}"`);
    if (stdout.trim()) {
      // Reload the document if formatter modified it
      const edit = new vscode.WorkspaceEdit();
      const fullRange = new vscode.Range(
        document.positionAt(0),
        document.positionAt(document.getText().length)
      );
      const formatted = await vscode.workspace.fs.readFile(document.uri);
      edit.replace(document.uri, fullRange, Buffer.from(formatted).toString("utf-8"));
      await vscode.workspace.applyEdit(edit);
    }
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : String(err);
    vscode.window.showErrorMessage(`AgentSpec format failed: ${message}`);
  }
}

async function showPlan(document: vscode.TextDocument): Promise<void> {
  const config = vscode.workspace.getConfiguration("agentspec");
  const execPath = config.get<string>("executablePath", "agentspec");

  try {
    const { exec } = await import("child_process");
    const { promisify } = await import("util");
    const execAsync = promisify(exec);

    const { stdout } = await execAsync(`${execPath} plan "${document.fileName}"`);

    const panel = vscode.window.createOutputChannel("AgentSpec Plan");
    panel.clear();
    panel.appendLine(stdout);
    panel.show();
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : String(err);
    vscode.window.showErrorMessage(`AgentSpec plan failed: ${message}`);
  }
}

async function runCliCommand(command: string, document: vscode.TextDocument): Promise<void> {
  const config = vscode.workspace.getConfiguration("agentspec");
  const execPath = config.get<string>("executablePath", "agentspec");

  try {
    const { exec } = await import("child_process");
    const { promisify } = await import("util");
    const execAsync = promisify(exec);

    const { stdout } = await execAsync(`${execPath} ${command} "${document.fileName}"`);

    const panel = vscode.window.createOutputChannel(`AgentSpec ${command}`);
    panel.clear();
    panel.appendLine(stdout);
    panel.show();
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : String(err);
    vscode.window.showErrorMessage(`AgentSpec ${command} failed: ${message}`);
  }
}
