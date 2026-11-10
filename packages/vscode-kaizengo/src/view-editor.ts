import crypto from "node:crypto";
import * as vscode from "vscode";
import { offsetToPosition } from "./offsets";
import { parseSpec } from "./view-model";

export function isAppYaml(doc: vscode.TextDocument): boolean {
  return doc.uri.scheme === "file" && doc.uri.path.endsWith("/app.yaml");
}

/** Side panel that draws an app.yaml view tree and jumps back into the spec. */
export class ViewEditor {
  private panel: vscode.WebviewPanel | undefined;
  private doc: vscode.TextDocument | undefined;
  private messageSub: vscode.Disposable | undefined;

  constructor(private readonly context: vscode.ExtensionContext) {}

  open(doc: vscode.TextDocument): void {
    this.doc = doc;
    this.ensure();
    this.panel?.reveal(vscode.ViewColumn.Beside, false);
    this.post();
  }

  follow(doc: vscode.TextDocument): void {
    if (!this.panel || this.doc?.uri.toString() === doc.uri.toString()) return;
    this.doc = doc;
    this.post();
  }

  sync(doc: vscode.TextDocument): void {
    if (!this.panel || !this.doc || doc.uri.toString() !== this.doc.uri.toString()) return;
    this.doc = doc;
    this.post();
  }

  private ensure(): void {
    if (this.panel) return;
    const panel = vscode.window.createWebviewPanel("kaizengo.vis", "Vis", vscode.ViewColumn.Beside, {
      enableScripts: true,
      retainContextWhenHidden: true,
    });
    panel.webview.html = html(crypto.randomBytes(16).toString("hex"));
    this.messageSub = panel.webview.onDidReceiveMessage((msg) => this.onMessage(msg));
    panel.onDidDispose(() => {
      this.messageSub?.dispose();
      this.messageSub = undefined;
      this.panel = undefined;
      this.doc = undefined;
    });
    this.context.subscriptions.push(panel);
    this.panel = panel;
  }

  private post(): void {
    if (!this.panel || !this.doc) return;
    const spec = parseSpec(this.doc.getText());
    void this.panel.webview.postMessage({ type: "spec", spec });
  }

  private onMessage(msg: unknown): void {
    if (!msg || typeof msg !== "object") return;
    const data = msg as { type?: string; start?: number; end?: number; text?: string };
    if (data.type === "ready") {
      this.post();
      return;
    }
    const doc = this.doc;
    if (!doc || typeof data.start !== "number" || typeof data.end !== "number") return;
    if (data.type === "reveal") {
      void this.reveal(doc, data.start, data.end);
      return;
    }
    if (data.type === "replace" && (data.text === "column" || data.text === "row")) {
      void this.replaceDirection(doc, data.start, data.end, data.text);
    }
  }

  private async reveal(doc: vscode.TextDocument, start: number, end: number): Promise<void> {
    const text = doc.getText();
    const from = offsetToPosition(text, start);
    const to = offsetToPosition(text, Math.max(start, end));
    const range = new vscode.Range(new vscode.Position(from.line, from.character), new vscode.Position(to.line, to.character));
    const visible = vscode.window.visibleTextEditors.find((editor) => editor.document.uri.toString() === doc.uri.toString());
    const column = visible?.viewColumn ?? vscode.ViewColumn.One;
    const editor = await vscode.window.showTextDocument(doc, { selection: range, viewColumn: column, preserveFocus: false });
    editor.revealRange(range, vscode.TextEditorRevealType.InCenter);
  }

  private async replaceDirection(doc: vscode.TextDocument, start: number, end: number, text: string): Promise<void> {
    const from = offsetToPosition(doc.getText(), start);
    const to = offsetToPosition(doc.getText(), end);
    const range = new vscode.Range(new vscode.Position(from.line, from.character), new vscode.Position(to.line, to.character));
    const current = doc.getText(range);
    if (!/^[A-Za-z]+$/.test(current) || current === text) return;
    const edit = new vscode.WorkspaceEdit();
    edit.replace(doc.uri, range, text);
    await vscode.workspace.applyEdit(edit);
  }
}

export function specDiagnostics(doc: vscode.TextDocument): vscode.Diagnostic[] {
  const text = doc.getText();
  const spec = parseSpec(text);
  return spec.issues.map((issue) => {
    const start = offsetToPosition(text, issue.range.start);
    const end = offsetToPosition(text, issue.range.end);
    const diagnostic = new vscode.Diagnostic(
      new vscode.Range(new vscode.Position(start.line, start.character), new vscode.Position(end.line, end.character)),
      issue.message,
      issue.severity === "error" ? vscode.DiagnosticSeverity.Error : vscode.DiagnosticSeverity.Warning,
    );
    diagnostic.source = "kaizengo";
    return diagnostic;
  });
}

function html(nonce: string): string {
  const csp = `default-src 'none'; style-src 'nonce-${nonce}'; script-src 'nonce-${nonce}';`;
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8" />
<meta http-equiv="Content-Security-Policy" content="${csp}" />
<meta name="viewport" content="width=device-width, initial-scale=1" />
<title>Vis</title>
<style nonce="${nonce}">
  html, body { height: 100%; }
  body {
    margin: 0;
    display: flex;
    flex-direction: column;
    color: var(--vscode-editor-foreground);
    background: var(--vscode-editor-background);
    font-family: var(--vscode-font-family);
    font-size: var(--vscode-font-size);
  }
  header {
    flex: none;
    padding: 7px 12px;
    font-weight: 650;
    letter-spacing: 0.02em;
    border-bottom: 1px solid var(--vscode-panel-border);
    background: var(--vscode-editor-background);
  }
  main { flex: 1; overflow: auto; padding: 10px 12px 18px; }
  footer {
    flex: none;
    padding: 6px 12px;
    border-top: 1px solid var(--vscode-panel-border);
    font-size: 12px;
    opacity: 0.75;
  }
  button, summary { font: inherit; color: inherit; }
  button { background: none; border: 0; padding: 0; text-align: start; cursor: pointer; }
  .app-line { margin-bottom: 12px; }
  .app-line .title { opacity: 0.7; margin-left: 8px; }
  .section { margin: 0 0 8px; }
  .section > summary { cursor: pointer; font-weight: 650; }
  .view-card { margin: 8px 0 12px; }
  .view-title { display: flex; gap: 8px; align-items: baseline; margin-bottom: 6px; }
  .view-name { font-weight: 650; }
  .path, .meta, .kind, .dir-label { opacity: 0.7; font-size: 12px; }
  .page-only, .empty {
    border: 1px dashed var(--vscode-panel-border);
    border-radius: 4px;
    padding: 8px;
    opacity: 0.75;
  }
  .region { min-width: 0; }
  .region-bar { display: flex; gap: 4px; align-items: center; margin-bottom: 4px; }
  .dir {
    border: 1px solid var(--vscode-panel-border);
    border-radius: 3px;
    padding: 0 6px;
    font-size: 11px;
    cursor: pointer;
  }
  .dir.active { background: var(--vscode-badge-background); color: var(--vscode-badge-foreground); border-color: transparent; }
  .layout { display: grid; gap: 8px; min-width: 0; }
  .tile {
    border: 1px solid var(--vscode-panel-border);
    border-radius: 4px;
    padding: 8px;
    min-width: 0;
    cursor: pointer;
  }
  .tile:hover, .node:hover, .chip:hover, .view-title:hover { background: var(--vscode-list-hoverBackground); }
  .tile.missing { border-color: var(--vscode-errorForeground); color: var(--vscode-errorForeground); }
  .tile-head { display: flex; justify-content: space-between; gap: 8px; }
  .id { font-weight: 650; }
  .chips, .fields { display: flex; flex-wrap: wrap; gap: 4px; margin-top: 6px; }
  .fields { display: grid; }
  .chip {
    font-size: 11px;
    padding: 1px 6px;
    border-radius: 3px;
    background: var(--vscode-badge-background);
    color: var(--vscode-badge-foreground);
    cursor: pointer;
  }
  .group { margin-top: 6px; }
  .group-label { font-size: 11px; opacity: 0.7; }
  .tree { list-style: none; margin: 4px 0 0; padding-left: 14px; }
  .node { display: flex; gap: 8px; width: 100%; padding: 1px 4px; border-radius: 3px; }
  .detail { opacity: 0.65; }
  .issues { list-style: none; margin: 0 0 12px; padding: 0; }
  .issue { display: block; width: 100%; padding: 2px 0; cursor: pointer; }
  .issue.error { color: var(--vscode-errorForeground); }
  .issue.warning { color: var(--vscode-editorWarning-foreground); }
  .parse-error { color: var(--vscode-errorForeground); margin-bottom: 8px; }
</style>
</head>
<body>
<header>Vis</header>
<main id="tree"></main>
<footer id="status"></footer>
<script nonce="${nonce}">
${clientScript()}
</script>
</body>
</html>`;
}

function clientScript(): string {
  return `
const vscode = acquireVsCodeApi();
const main = document.getElementById("tree");
const status = document.getElementById("status");

function openSections() {
  const state = vscode.getState();
  return new Set(state && Array.isArray(state.open) ? state.open : ["views", "models", "menus"]);
}

function remember() {
  const open = [];
  document.querySelectorAll("details[data-section]").forEach((node) => {
    if (node.open) open.push(node.dataset.section);
  });
  vscode.setState({ open });
}

function el(tag, className, text) {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text != null && text !== "") node.textContent = text;
  return node;
}

function setRange(node, range) {
  if (!range) return;
  node.dataset.start = String(range.start);
  node.dataset.end = String(range.end);
}

function track(node) {
  if (node.size) return node.size;
  const span = node.span && node.span > 0 ? node.span : 1;
  return "minmax(0, " + span + "fr)";
}

function chip(field) {
  const node = el("button", "chip", field.name || "(field)");
  node.type = "button";
  const bits = [field.widget, field.width, field.align].filter(Boolean);
  if (bits.length) node.title = bits.join(" · ");
  if (field.span && field.span > 1) node.style.gridColumn = "span " + field.span;
  setRange(node, field.range);
  return node;
}

function renderTile(node, components) {
  const found = components.find((item) => item.id === node.componentId);
  if (!found) {
    const missing = el("div", "tile missing", "Unknown component " + (node.componentId || ""));
    setRange(missing, node.range);
    return missing;
  }
  const tile = el("div", "tile");
  setRange(tile, found.range);
  const head = el("div", "tile-head");
  head.append(el("span", "id", found.id));
  const meta = [found.kind, found.model, node.size, !node.size && node.span ? node.span + "fr" : ""].filter(Boolean).join(" · ");
  head.append(el("span", "kind", meta));
  tile.append(head);
  if (found.kind === "form") {
    for (const group of found.groups) {
      const block = el("div", "group");
      const label = el("div", "group-label", group.label);
      setRange(label, group.range);
      block.append(label);
      const grid = el("div", "fields");
      grid.style.gridTemplateColumns = "repeat(" + (group.columns || 1) + ", minmax(0, 1fr))";
      for (const field of group.fields) grid.append(chip(field));
      block.append(grid);
      tile.append(block);
    }
  } else {
    const chips = el("div", "chips");
    for (const field of found.columns) chips.append(chip(field));
    tile.append(chips);
    const extra = [];
    if (found.defaultView) extra.push(found.defaultView);
    if (found.modes && found.modes.length) extra.push(found.modes.join(", "));
    if (found.groupBy && found.groupBy.length) extra.push("group " + found.groupBy.join(", "));
    if (extra.length) tile.append(el("div", "meta", extra.join(" · ")));
  }
  if (found.filters && found.filters.length) {
    const chips = el("div", "chips");
    for (const filter of found.filters) {
      const node = el("button", "chip", filter.id);
      node.type = "button";
      node.title = "filter";
      setRange(node, filter.range);
      chips.append(node);
    }
    tile.append(chips);
  }
  return tile;
}

function dirButton(dir, node) {
  const button = el("button", "dir" + (node.direction === dir ? " active" : ""), dir);
  button.type = "button";
  button.dataset.edit = dir;
  button.title = "Set direction to " + dir;
  if (node.directionRange) {
    button.dataset.start = String(node.directionRange.start);
    button.dataset.end = String(node.directionRange.end);
  }
  return button;
}

function renderLayout(node, components) {
  if (node.componentId) return renderTile(node, components);
  const wrap = el("div", "region");
  setRange(wrap, node.range);
  if (node.directionRange) {
    const bar = el("div", "region-bar");
    bar.append(el("span", "dir-label", node.direction || "column"));
    bar.append(dirButton("column", node));
    bar.append(dirButton("row", node));
    wrap.append(bar);
  }
  const grid = el("div", "layout");
  const row = (node.direction || "column") === "row";
  const items = node.items || [];
  grid.style[row ? "gridTemplateColumns" : "gridTemplateRows"] = (items.length ? items.map(track) : ["minmax(0, 1fr)"]).join(" ");
  if (!items.length) grid.append(el("div", "empty", "Empty layout"));
  for (const child of items) grid.append(renderLayout(child, components));
  wrap.append(grid);
  return wrap;
}

function renderView(view) {
  const card = el("section", "view-card");
  const title = el("button", "view-title");
  title.type = "button";
  setRange(title, view.range);
  title.append(el("span", "view-name", view.name || "(view)"));
  if (view.path) {
    const path = el("span", "path", view.path);
    setRange(path, view.pathRange);
    title.append(path);
  }
  card.append(title);
  const pageOnly = !(view.components || []).length && !view.layout.componentId && !(view.layout.items || []).length;
  card.append(pageOnly ? el("div", "page-only", "Compiled page") : renderLayout(view.layout, view.components || []));
  return card;
}

function renderNode(item) {
  const li = el("li");
  const button = el("button", "node");
  button.type = "button";
  setRange(button, item.range);
  button.append(el("span", "label", item.label));
  if (item.detail) button.append(el("span", "detail", item.detail));
  li.append(button);
  if (item.children && item.children.length) {
    const list = el("ul", "tree");
    for (const child of item.children) list.append(renderNode(child));
    li.append(list);
  }
  return li;
}

function section(id, label, body) {
  const details = el("details", "section");
  details.dataset.section = id;
  details.open = openSections().has(id);
  details.addEventListener("toggle", remember);
  details.append(el("summary", "", label));
  details.append(body);
  return details;
}

function treeSection(id, label, nodes) {
  if (!nodes || !nodes.length) return null;
  const list = el("ul", "tree");
  for (const node of nodes) list.append(renderNode(node));
  return section(id, label, list);
}

function render(spec) {
  const scroll = main.scrollTop;
  main.replaceChildren();
  if (spec.error) main.append(el("div", "parse-error", spec.error));
  const app = el("div", "app-line");
  const name = el("button", "view-name", spec.name || "(app)");
  name.type = "button";
  setRange(name, spec.range);
  app.append(name);
  if (spec.title) app.append(el("span", "title", spec.title));
  main.append(app);
  if (spec.issues && spec.issues.length) {
    const list = el("ul", "issues");
    for (const issue of spec.issues) {
      const item = el("li");
      const button = el("button", "issue " + issue.severity, issue.message);
      button.type = "button";
      setRange(button, issue.range);
      item.append(button);
      list.append(item);
    }
    main.append(list);
  }
  const views = el("div");
  for (const view of spec.views || []) views.append(renderView(view));
  if (!(spec.views || []).length) views.append(el("div", "empty", "No views in this spec"));
  main.append(section("views", "views", views));
  for (const block of [
    treeSection("models", "models", spec.models),
    treeSection("queries", "queries", spec.queries),
    treeSection("commands", "commands", spec.commands),
    treeSection("menus", "menus", spec.menus),
    treeSection("depends", "depends", spec.depends),
    treeSection("locales", "locales", spec.locales),
  ]) {
    if (block) main.append(block);
  }
  const problems = spec.issues ? spec.issues.length : 0;
  status.textContent = problems
    ? problems + (problems === 1 ? " problem. " : " problems. ") + "Click a block to open it in the spec."
    : "Click a block to open it in the spec. Column and row rewrite that layout direction.";
  main.scrollTop = scroll;
}

main.addEventListener("click", (event) => {
  const target = event.target;
  if (!(target instanceof Element)) return;
  const edit = target.closest("[data-edit]");
  if (edit && edit.dataset.start) {
    vscode.postMessage({
      type: "replace",
      start: Number(edit.dataset.start),
      end: Number(edit.dataset.end),
      text: edit.dataset.edit,
    });
    return;
  }
  const hit = target.closest("[data-start]");
  if (!hit) return;
  vscode.postMessage({ type: "reveal", start: Number(hit.dataset.start), end: Number(hit.dataset.end) });
});

window.addEventListener("message", (event) => {
  if (event.data && event.data.type === "spec") render(event.data.spec);
});

vscode.postMessage({ type: "ready" });
`;
}
