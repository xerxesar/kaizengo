import fs from "node:fs";
import * as vscode from "vscode";
import { appNameFromFile, findRepoRoot, loadCatalog, type Catalog } from "./catalog";
import { offsetToPosition, positionToOffset } from "./offsets";
import { classify, collectSymbols, symbolAt } from "./parse";
import { resolveDefinition, resolveReferences, type NavTarget, type ResolveContext } from "./resolve";

const selector: vscode.DocumentSelector = { scheme: "file", pattern: "**/app.yaml" };

let catalog: Catalog | undefined;

function catalogFor(doc: vscode.TextDocument): Catalog | undefined {
  const root = findRepoRoot(doc.uri.fsPath);
  if (!root) return undefined;
  if (!catalog || catalog.root !== root) {
    catalog = loadCatalog(root);
  }
  return catalog;
}

function reindex(root?: string): void {
  if (root) {
    catalog = loadCatalog(root);
    return;
  }
  if (catalog) catalog = loadCatalog(catalog.root);
}

function resolveCtx(doc: vscode.TextDocument): ResolveContext | undefined {
  const cat = catalogFor(doc);
  if (!cat) return undefined;
  const text = doc.getText();
  const symbols = collectSymbols(text);
  const appName =
    appNameFromFile(doc.uri.fsPath) ?? symbols.find((s) => s.path === "name")?.value;
  if (!appName) return undefined;
  return { catalog: cat, appName, file: doc.uri.fsPath, text, symbols };
}

function symbolAtPosition(ctx: ResolveContext, pos: vscode.Position) {
  const offset = positionToOffset(ctx.text, pos.line, pos.character);
  return symbolAt(ctx.symbols, offset);
}

function toPosition(text: string, offset: number): vscode.Position {
  const pos = offsetToPosition(text, offset);
  return new vscode.Position(pos.line, pos.character);
}

function toLocation(target: NavTarget): vscode.Location {
  let text = "";
  try {
    text = fs.readFileSync(target.file, "utf8");
  } catch {
    text = "";
  }
  const range = new vscode.Range(toPosition(text, target.start), toPosition(text, target.end));
  return new vscode.Location(vscode.Uri.file(target.file), range);
}

export function activate(context: vscode.ExtensionContext): void {
  const watcher = vscode.workspace.createFileSystemWatcher("**/app.yaml");
  watcher.onDidChange((uri) => {
    const root = findRepoRoot(uri.fsPath);
    if (root) reindex(root);
  });
  watcher.onDidCreate((uri) => {
    const root = findRepoRoot(uri.fsPath);
    if (root) reindex(root);
  });
  watcher.onDidDelete(() => {
    catalog = undefined;
  });

  context.subscriptions.push(
    watcher,
    vscode.commands.registerCommand("kaizengo.reindex", () => {
      const editor = vscode.window.activeTextEditor;
      const root = editor ? findRepoRoot(editor.document.uri.fsPath) : catalog?.root;
      if (root) reindex(root);
      vscode.window.showInformationMessage("KaizenGo: app.yaml catalog reindexed");
    }),
    vscode.languages.registerDefinitionProvider(selector, {
      provideDefinition(doc, pos) {
        const ctx = resolveCtx(doc);
        if (!ctx) return;
        const sym = symbolAtPosition(ctx, pos);
        if (!sym) return;
        return resolveDefinition(ctx, sym).map(toLocation);
      },
    }),
    vscode.languages.registerReferenceProvider(selector, {
      provideReferences(doc, pos) {
        const ctx = resolveCtx(doc);
        if (!ctx) return;
        const sym = symbolAtPosition(ctx, pos);
        if (!sym) return;
        return resolveReferences(ctx, sym).map(toLocation);
      },
    }),
    vscode.languages.registerHoverProvider(selector, {
      provideHover(doc, pos) {
        const ctx = resolveCtx(doc);
        if (!ctx) return;
        const sym = symbolAtPosition(ctx, pos);
        if (!sym || classify(sym.path) === "none") return;
        const defs = resolveDefinition(ctx, sym);
        const refs =
          classify(sym.path) === "provides" ? resolveReferences(ctx, sym).filter((t) => t.file !== ctx.file || t.start !== sym.range.start) : [];
        const md = new vscode.MarkdownString(undefined, true);
        md.isTrusted = false;
        if (defs.length) {
          md.appendMarkdown(defs.map((d) => `\`${d.tooltip}\``).join(" · "));
          md.appendMarkdown("\n\nGo to Definition (**F12** / Ctrl+click)");
        } else if (refs.length) {
          md.appendMarkdown(`${refs.length} reference${refs.length === 1 ? "" : "s"}`);
          md.appendMarkdown("\n\nFind All References (**Shift+F12**)");
        } else {
          return;
        }
        const range = new vscode.Range(
          toPosition(ctx.text, sym.range.start),
          toPosition(ctx.text, sym.range.end),
        );
        return new vscode.Hover(md, range);
      },
    }),
  );
}

export function deactivate(): void {}
