"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.activate = activate;
exports.deactivate = deactivate;
const vscode = require("vscode");
const cp = require("child_process");
let diagnosticCollection;
function activate(context) {
    diagnosticCollection = vscode.languages.createDiagnosticCollection('human');
    context.subscriptions.push(diagnosticCollection);
    // Run compiler check on save
    context.subscriptions.push(vscode.workspace.onDidSaveTextDocument((document) => {
        if (document.languageId === 'human') {
            runHumanCheck(document);
        }
    }));
    // Initial check if active document is .human
    if (vscode.window.activeTextEditor && vscode.window.activeTextEditor.document.languageId === 'human') {
        runHumanCheck(vscode.window.activeTextEditor.document);
    }
    // Basic Completion Provider
    const provider = vscode.languages.registerCompletionItemProvider('human', {
        provideCompletionItems(document, position) {
            const text = document.getText();
            // Extract model names
            const modelRegex = /^data\s+([a-zA-Z0-9_]+):/gm;
            const models = [];
            let match;
            while ((match = modelRegex.exec(text)) !== null) {
                models.push(match[1]);
            }
            // Extract page names
            const pageRegex = /^page\s+([a-zA-Z0-9_]+):/gm;
            const pages = [];
            while ((match = pageRegex.exec(text)) !== null) {
                pages.push(match[1]);
            }
            const linePrefix = document.lineAt(position).text.substr(0, position.character);
            const completions = [];
            // Suggest models after 'belongs to a', 'has many'
            if (linePrefix.match(/(belongs to a|has many)\s+$/)) {
                for (const model of models) {
                    completions.push(new vscode.CompletionItem(model, vscode.CompletionItemKind.Class));
                }
            }
            // Suggest pages after 'navigates to', 'opens'
            if (linePrefix.match(/(navigates to|opens)\s+$/)) {
                for (const page of pages) {
                    completions.push(new vscode.CompletionItem(page, vscode.CompletionItemKind.Interface));
                }
            }
            return completions;
        }
    }, ' '); // Trigger on space
    context.subscriptions.push(provider);
}
function runHumanCheck(document) {
    const workspaceFolder = vscode.workspace.getWorkspaceFolder(document.uri);
    const cwd = workspaceFolder ? workspaceFolder.uri.fsPath : undefined;
    // Use absolute path or assume 'human' is in PATH.
    // For simplicity, we try running 'human check'
    cp.exec(`human check "${document.fileName}"`, { cwd }, (error, stdout, stderr) => {
        const diagnostics = [];
        // Very basic error parsing. Expects "Error in <file>, line <num>:"
        const output = stdout + stderr;
        if (output.includes('Error in')) {
            const lines = output.split('\n');
            let currentLineNum = 0;
            let currentMessage = '';
            for (const line of lines) {
                const match = line.match(/line (\d+):/);
                if (match) {
                    currentLineNum = parseInt(match[1], 10) - 1; // 0-indexed in vscode
                }
                else if (line.trim().length > 0 && currentLineNum > 0) {
                    currentMessage += line.trim() + ' ';
                }
            }
            if (currentLineNum >= 0 && currentMessage) {
                // Determine range for the line (default to whole line)
                const lineText = document.lineAt(currentLineNum).text;
                const range = new vscode.Range(currentLineNum, document.lineAt(currentLineNum).firstNonWhitespaceCharacterIndex, currentLineNum, lineText.length);
                const diagnostic = new vscode.Diagnostic(range, currentMessage.trim(), vscode.DiagnosticSeverity.Error);
                diagnostics.push(diagnostic);
            }
        }
        diagnosticCollection.set(document.uri, diagnostics);
    });
}
function deactivate() { }
//# sourceMappingURL=extension.js.map