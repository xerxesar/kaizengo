import { isMap, isScalar, isSeq, parseDocument } from "yaml";

export interface Range {
  start: number;
  end: number;
}

export interface YamlSymbol {
  path: string;
  value: string;
  range: Range;
}

export type NavKind =
  | "app-name"
  | "depends"
  | "uses"
  | "provides"
  | "view-name"
  | "menu-view"
  | "model-name"
  | "relation"
  | "inverse"
  | "search-field"
  | "search-collection"
  | "label-key"
  | "locale-id"
  | "extend-handler"
  | "component-id"
  | "component-module"
  | "export-app"
  | "export-component-ref"
  | "export-view-ref"
  | "none";

export function classify(path: string): NavKind {
  if (path === "name") return "app-name";
  if (/^depends\[\d+\]$/.test(path)) return "depends";
  if (/^uses\[\d+\]$/.test(path)) return "uses";
  if (/^provides\[\d+\]$/.test(path)) return "provides";
  if (/^views\[\d+\]\.name$/.test(path)) return "view-name";
  if (/^models\[\d+\]\.name$/.test(path)) return "model-name";
  if (/^models\[\d+\]\.fields\[\d+\]\.relation$/.test(path)) return "relation";
  if (/^models\[\d+\]\.fields\[\d+\]\.inverse$/.test(path)) return "inverse";
  if (/^models\[\d+\]\.search\.fields\[\d+\]$/.test(path)) return "search-field";
  if (/^models\[\d+\]\.search\.collection$/.test(path)) return "search-collection";
  if (/^locales\[\d+\]\.id$/.test(path)) return "locale-id";
  if (/^extends\[\d+\]\.handler$/.test(path)) return "extend-handler";
  if (/^exports\.components\[\d+\]\.id$/.test(path)) return "component-id";
  if (/^exports\.components\[\d+\]\.module$/.test(path)) return "component-module";
  if (/^exports\.(views|menus)\[\d+\]\.app$/.test(path)) return "export-app";
  if (/^exports\.(views|menus)\[\d+\]\.component$/.test(path)) return "export-component-ref";
  if (/^exports\.menus\[\d+\]\.view$/.test(path)) return "export-view-ref";
  if (/(^|\.)(labelKey|emptyKey)$/.test(path)) return "label-key";
  if (/^menus(\[\d+\]\.children)*\[\d+\]\.view$/.test(path)) return "menu-view";
  return "none";
}

function nodeRange(node: { range?: [number, number, number] | null }): Range | undefined {
  if (!node.range) return undefined;
  return { start: node.range[0], end: node.range[1] };
}

function walk(node: unknown, path: string, out: YamlSymbol[]): void {
  if (node == null) return;
  if (isScalar(node)) {
    if (node.value == null || path === "") return;
    const range = nodeRange(node);
    if (!range) return;
    out.push({ path, value: String(node.value), range });
    return;
  }
  if (isMap(node)) {
    for (const pair of node.items) {
      const key = isScalar(pair.key) ? String(pair.key.value ?? "") : "";
      const childPath = path ? `${path}.${key}` : key;
      walk(pair.value, childPath, out);
    }
    return;
  }
  if (isSeq(node)) {
    node.items.forEach((item, i) => {
      walk(item, `${path}[${i}]`, out);
    });
  }
}

export function collectSymbols(text: string): YamlSymbol[] {
  const doc = parseDocument(text);
  const out: YamlSymbol[] = [];
  walk(doc.contents, "", out);
  return out;
}

export function symbolAt(symbols: YamlSymbol[], offset: number): YamlSymbol | undefined {
  let best: YamlSymbol | undefined;
  for (const s of symbols) {
    if (offset < s.range.start || offset > s.range.end) continue;
    if (!best || s.range.end - s.range.start < best.range.end - best.range.start) {
      best = s;
    }
  }
  return best;
}

export function sibling(symbols: YamlSymbol[], path: string, key: string): YamlSymbol | undefined {
  const prefix = path.replace(/\.[^.[\]]+$/, "");
  return symbols.find((s) => s.path === `${prefix}.${key}`);
}
