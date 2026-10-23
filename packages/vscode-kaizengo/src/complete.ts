import { isMap, isScalar, isSeq, parseDocument, Scalar } from "yaml";
import { appNameFromFile, type AppIndex, type Catalog } from "./catalog";
import {
  BUILTIN_FIELDS,
  FIELD_TYPES,
  buildIndex,
  fieldDetail,
  fieldTypeDetail,
  findModel,
  modelDetail,
  modelDoc,
  type CqrsInfo,
  type FieldInfo,
  type ModelIndex,
  type ModelInfo,
} from "./model-index";
import { collectSymbols, type YamlSymbol } from "./parse";
import { parseRelation } from "./resolve";

export interface ReplaceSpan {
  start: number;
  end: number;
}

export interface Completion {
  label: string;
  kind: "model" | "field" | "query" | "command" | "type" | "enum";
  detail: string;
  documentation: string;
  sort: string;
  replace: ReplaceSpan;
}

export interface CompleteInput {
  catalog: Catalog;
  file: string;
  text: string;
  offset: number;
}

const FIELD_ATTRS = new Set(["field", "groupByField", "xField", "yField", "seriesField", "keyField"]);
const RECORD_ATTRS = new Set(["card", "keyOf", "actions", "columnOf"]);

interface GoMethod {
  model: number;
  field?: number;
  fieldsArg?: number;
}

const GO_METHODS: Record<string, GoMethod> = {
  Create: { model: 3, fieldsArg: 4 },
  Update: { model: 2, fieldsArg: 4 },
  Delete: { model: 2 },
  List: { model: 2 },
  ListPage: { model: 2 },
  Count: { model: 2 },
  CountOpts: { model: 2 },
  Groups: { model: 2 },
  Get: { model: 2 },
  GetByID: { model: 1 },
  FindBy: { model: 1, field: 2 },
  ListBy: { model: 2, field: 3 },
  ListAll: { model: 1 },
  Hooks: { model: 0 },
};

export function complete(input: CompleteInput): Completion[] {
  if (input.file.includes("__types__") || input.file.includes("node_modules")) return [];
  const { index, appName, symbols } = indexFor(input);
  if (input.file.endsWith(".go")) return completeGo(input, index, appName);
  if (input.file.endsWith(".tsx") || input.file.endsWith(".jsx")) return completeTsx(input, index, appName);
  if (/\.ya?ml$/i.test(input.file)) return completeYaml(input, index, appName, symbols);
  return [];
}

function indexFor(input: CompleteInput): { index: ModelIndex; appName: string; symbols: YamlSymbol[] } {
  const appName = appNameFromFile(input.file) ?? "";
  const yaml = /\.ya?ml$/i.test(input.file);
  const symbols = yaml ? collectSymbols(input.text) : [];
  const apps: AppIndex[] = [];
  for (const app of input.catalog.apps.values()) {
    if (!yaml || (app.name !== appName && !app.docs.some((d) => d.path === input.file))) {
      apps.push(app);
      continue;
    }
    const docs = app.docs.map((d) => (d.path === input.file ? { path: input.file, text: input.text, symbols } : d));
    if (!docs.some((d) => d.path === input.file)) docs.push({ path: input.file, text: input.text, symbols });
    const main = docs.find((d) => d.path === app.yamlPath) ?? docs[0];
    apps.push({ ...app, docs, text: main?.text ?? app.text, symbols: main?.symbols ?? symbols });
  }
  return { index: buildIndex(apps), appName, symbols };
}

function modelItems(index: ModelIndex, appName: string, replace: ReplaceSpan, form: "bare" | "qualified" | "relation"): Completion[] {
  const out: Completion[] = [];
  const seen = new Set<string>();
  const models = [...index.models].sort((a, b) => a.ref.localeCompare(b.ref));
  const push = (model: ModelInfo, label: string, sort: string) => {
    if (seen.has(label)) return;
    seen.add(label);
    out.push({
      label,
      kind: "model",
      detail: modelDetail(model),
      documentation: modelDoc(model),
      sort,
      replace,
    });
  };
  for (const model of models) {
    const same = model.app === appName;
    if (form === "bare") {
      if (same) push(model, model.name, `0_${model.name}`);
      continue;
    }
    if (form === "relation" && same) push(model, model.name, `0_${model.name}`);
    push(model, model.ref, `${same ? "1" : "2"}_${model.ref}`);
  }
  return out;
}

function fieldItems(model: ModelInfo | undefined, replace: ReplaceSpan, builtins: boolean): Completion[] {
  if (!model) return [];
  const out: Completion[] = [];
  const seen = new Set<string>();
  const push = (field: FieldInfo, sort: string) => {
    if (seen.has(field.name)) return;
    seen.add(field.name);
    out.push({
      label: field.name,
      kind: "field",
      detail: fieldDetail(field),
      documentation: `${model.ref}.${field.name}`,
      sort,
      replace,
    });
  };
  model.fields.forEach((field, i) => push(field, `0_${String(i).padStart(3, "0")}`));
  if (builtins) {
    for (const field of BUILTIN_FIELDS) push(field, `1_${field.name}`);
  }
  return out;
}

function enumItems(model: ModelInfo | undefined, fieldName: string, replace: ReplaceSpan): Completion[] {
  const field = model?.fields.find((f) => f.name === fieldName);
  if (!field?.values.length) return [];
  return field.values.map((value, i) => ({
    label: value,
    kind: "enum" as const,
    detail: `${field.name} · ${field.type}`,
    documentation: `${model!.ref}.${field.name}`,
    sort: `0_${String(i).padStart(3, "0")}`,
    replace,
  }));
}

function typeItems(replace: ReplaceSpan): Completion[] {
  return FIELD_TYPES.map((type, i) => ({
    label: type,
    kind: "type" as const,
    detail: fieldTypeDetail(type),
    documentation: `Field type ${type}`,
    sort: `0_${String(i).padStart(2, "0")}`,
    replace,
  }));
}

function cqrsItems(rows: CqrsInfo[], appName: string, replace: ReplaceSpan, kind: "query" | "command"): Completion[] {
  return [...rows]
    .sort((a, b) => a.ref.localeCompare(b.ref))
    .map((row) => ({
      label: row.ref,
      kind,
      detail: row.op && row.model ? `${row.op} ${row.model}` : (row.op ?? kind),
      documentation: row.model ? `${row.ref} → ${row.model}` : row.ref,
      sort: `${row.app === appName ? "0" : "1"}_${row.ref}`,
      replace,
    }));
}

function resolveModelRef(index: ModelIndex, fromApp: string, ref: string | undefined): ModelInfo | undefined {
  if (!ref) return undefined;
  const parsed = parseRelation(ref, fromApp);
  if (!parsed) return undefined;
  return findModel(index, parsed.app, parsed.model);
}

interface YamlHit {
  path: string;
  replace: ReplaceSpan;
}

function completeYaml(input: CompleteInput, index: ModelIndex, appName: string, symbols: YamlSymbol[]): Completion[] {
  let hit: YamlHit | undefined;
  try {
    hit = yamlHit(input.text, input.offset);
  } catch {
    return [];
  }
  if (!hit) return [];
  const { path, replace } = hit;

  const modelPrefix = path.match(/^(models\[\d+\])/)?.[1];
  const fieldPrefix = path.match(/^(models\[\d+\]\.fields\[\d+\])/)?.[1];

  if (fieldPrefix && path === `${fieldPrefix}.type`) return typeItems(replace);
  if (fieldPrefix && path === `${fieldPrefix}.relation`) return modelItems(index, appName, replace, "relation");
  if (fieldPrefix && path === `${fieldPrefix}.inverse`) {
    const related = resolveModelRef(index, appName, sym(symbols, `${fieldPrefix}.relation`));
    return fieldItems(related, replace, false);
  }
  if (fieldPrefix && path === `${fieldPrefix}.default`) {
    const name = sym(symbols, `${fieldPrefix}.name`);
    const modelName = modelPrefix ? sym(symbols, `${modelPrefix}.name`) : undefined;
    return enumItems(modelName ? findModel(index, appName, modelName) : undefined, name ?? "", replace);
  }
  if (modelPrefix && /^models\[\d+\]\.search\.fields(\[\d+\])?$/.test(path)) {
    return fieldItems(findModel(index, appName, sym(symbols, `${modelPrefix}.name`) ?? ""), replace, false);
  }
  if (modelPrefix && path === `${modelPrefix}.search.collection`) return modelItems(index, appName, replace, "qualified");
  if (/^queries\[\d+\]\.(list|get)$/.test(path) || /^commands\[\d+\]\.(create|update|delete)$/.test(path)) {
    return modelItems(index, appName, replace, "bare");
  }

  const entry = path.match(/^(entries\[\d+\])/)?.[1];
  if (!entry) return [];
  if (path === `${entry}.resource`) {
    if (sym(symbols, `${entry}.kind`) !== "model") return [];
    return modelItems(index, appName, replace, "qualified");
  }
  const resourceModel = resolveModelRef(index, appName, sym(symbols, `${entry}.resource`));
  if (/^entries\[\d+\]\.fields(\[\d+\])?$/.test(path)) return fieldItems(resourceModel, replace, true);
  if (/^entries\[\d+\]\.domain\[\d+\]\[0\]$/.test(path)) return fieldItems(resourceModel, replace, true);
  return [];
}

function sym(symbols: YamlSymbol[], path: string): string | undefined {
  return symbols.find((s) => s.path === path)?.value;
}

function yamlHit(text: string, offset: number): YamlHit | undefined {
  const doc = parseDocument(text);
  const hits: YamlHit[] = [];
  locate(doc.contents, "", offset, hits);
  if (!hits.length) return undefined;
  hits.sort((a, b) => b.path.length - a.path.length);
  return hits[0];
}

function locate(node: unknown, path: string, offset: number, hits: YamlHit[]): void {
  if (!node || typeof node !== "object" || !("range" in node)) return;
  const range = (node as { range?: [number, number, number] | null }).range;
  if (!range) return;
  const [start, valueEnd] = range;
  if (offset < start || offset > valueEnd) return;

  if (isScalar(node)) {
    const content = scalarContent(node);
    if (content && offset >= content.start && offset <= content.end) {
      hits.push({ path, replace: content });
    }
    return;
  }
  if (isMap(node)) {
    for (const pair of node.items) {
      if (!isScalar(pair.key) || pair.value == null) continue;
      const key = String(pair.key.value ?? "");
      const child = path ? `${path}.${key}` : key;
      locate(pair.value, child, offset, hits);
    }
    return;
  }
  if (isSeq(node)) {
    const before = hits.length;
    node.items.forEach((item, i) => {
      if (item) locate(item, `${path}[${i}]`, offset, hits);
    });
    if (hits.length === before) hits.push({ path, replace: { start: offset, end: offset } });
  }
}

function scalarContent(node: Scalar): ReplaceSpan | undefined {
  if (!node.range) return undefined;
  const [start, valueEnd] = node.range;
  if (node.type === Scalar.QUOTE_DOUBLE || node.type === Scalar.QUOTE_SINGLE) {
    const contentStart = start + 1;
    const contentEnd = Math.max(contentStart, valueEnd - 1);
    return { start: contentStart, end: contentEnd };
  }
  return { start, end: valueEnd };
}

interface GoString {
  start: number;
  end: number;
}

function completeGo(input: CompleteInput, index: ModelIndex, fileApp: string): Completion[] {
  const quoted = goStringAt(input.text, input.offset);
  if (!quoted) return [];
  const site = inspectGo(input.text, quoted.start - 1);
  if (!site) return [];
  const bindings = registryBindings(input.text, site.funcStart, quoted.start);
  const replace = { start: quoted.start, end: quoted.end };
  const appOf = (receiver?: string) => (receiver && bindings.get(receiver)) || fileApp;

  if (site.kind === "call") {
    const spec = GO_METHODS[site.name];
    if (!spec) return [];
    const app = site.name === "Hooks" ? fileApp : appOf(site.receiver);
    if (spec.field !== undefined && site.argIndex === spec.field) {
      const model = stringArg(callArg(input.text, site.paren, spec.model, quoted.start));
      return fieldItems(findModel(index, app, model ?? ""), replace, true);
    }
    if (site.argIndex === spec.model) return modelItems(index, app, replace, "bare");
    return [];
  }

  if (site.kind === "key" || site.kind === "value" || site.kind === "index") {
    const resolved = resolveGoModel(input.text, site, fileApp, bindings, quoted.start);
    if (!resolved) return [];
    const model = findModel(index, resolved.app, resolved.model);
    if (site.kind === "value") return enumItems(model, site.key ?? "", replace);
    return fieldItems(model, replace, true);
  }
  return [];
}

interface ResolvedModel {
  app: string;
  model: string;
}

type GoSite =
  | { kind: "call"; name: string; receiver?: string; paren: number; argIndex: number; funcStart: number }
  | {
      kind: "key" | "value";
      key?: string;
      callName?: string;
      callReceiver?: string;
      callParen?: number;
      callArg?: number;
      varName?: string;
      funcStart: number;
    }
  | { kind: "index"; ident: string; funcStart: number };

function resolveGoModel(
  text: string,
  site: Extract<GoSite, { kind: "key" | "value" | "index" }>,
  fileApp: string,
  bindings: Map<string, string>,
  offset: number,
): ResolvedModel | undefined {
  const appOf = (receiver?: string): string => {
    const name = lastIdent(receiver);
    if (name && bindings.has(name)) return bindings.get(name) ?? "";
    return fileApp;
  };

  if (site.kind !== "index" && site.callName && site.callParen !== undefined && site.callArg !== undefined) {
    const spec = GO_METHODS[site.callName];
    if (spec?.fieldsArg === site.callArg) {
      const modelName = stringArg(readArgs(text, site.callParen, funcEnd(text, site.funcStart))[spec.model]);
      const app = appOf(site.callReceiver);
      if (modelName && app) return { app, model: modelName };
    }
  }

  const ident = site.kind === "index" ? site.ident : site.varName;
  if (!ident) return undefined;
  const found =
    (site.kind === "index" ? assignedModel(text, site.funcStart, offset, ident) : undefined) ??
    fieldsVarModel(text, site.funcStart, ident);
  if (!found) return undefined;
  const app = appOf(found.receiver);
  if (!app) return undefined;
  return { app, model: found.model };
}

function inspectGo(text: string, quotePos: number): GoSite | undefined {
  const funcStart = text.lastIndexOf("\nfunc ", quotePos);
  const from = funcStart >= 0 ? funcStart + 1 : 0;
  type Call = { kind: "call"; name: string; receiver?: string; paren: number; commas: number };
  type Brace = {
    kind: "brace";
    map: boolean;
    varName?: string;
    seenColon: boolean;
    pendingKey?: string;
    lastKey?: string;
    callName?: string;
    callReceiver?: string;
    callParen?: number;
    callArg?: number;
  };
  type Bracket = { kind: "bracket"; ident?: string };
  const frames: Array<Call | Brace | Bracket> = [];

  let i = from;
  while (i < quotePos) {
    const c = text[i];
    if (c === '"' || c === "'" || c === "`") {
      const end = skipGoString(text, i, quotePos);
      const top = frames[frames.length - 1];
      if (c === '"' && top?.kind === "brace" && top.map && !top.seenColon && end <= quotePos) {
        top.pendingKey = text.slice(i + 1, Math.max(i + 1, end - 1));
      }
      i = end;
      continue;
    }
    if (c === "/" && text[i + 1] === "/") {
      i += 2;
      while (i < quotePos && text[i] !== "\n") i++;
      continue;
    }
    if (c === "/" && text[i + 1] === "*") {
      i += 2;
      while (i < quotePos && !(text[i] === "*" && text[i + 1] === "/")) i++;
      i = Math.min(i + 2, quotePos);
      continue;
    }
    if (c === "(") {
      const ident = identBefore(text, i);
      frames.push({ kind: "call", name: ident.name, receiver: ident.receiver, paren: i, commas: 0 });
      i++;
      continue;
    }
    if (c === ")") {
      popFrame(frames, "call");
      i++;
      continue;
    }
    if (c === "{") {
      const mapped = mapBefore(text, i);
      const call = [...frames].reverse().find((f): f is Call => f.kind === "call");
      frames.push({
        kind: "brace",
        map: !!mapped,
        varName: mapped?.varName,
        seenColon: false,
        callName: call?.name,
        callReceiver: call?.receiver,
        callParen: call?.paren,
        callArg: call?.commas,
      });
      i++;
      continue;
    }
    if (c === "}") {
      popFrame(frames, "brace");
      i++;
      continue;
    }
    if (c === "[") {
      const ident = identBefore(text, i);
      frames.push({ kind: "bracket", ident: ident.name || undefined });
      i++;
      continue;
    }
    if (c === "]") {
      popFrame(frames, "bracket");
      i++;
      continue;
    }
    if (c === ",") {
      const top = frames[frames.length - 1];
      if (top?.kind === "call") top.commas++;
      if (top?.kind === "brace") {
        top.seenColon = false;
        top.pendingKey = undefined;
      }
      i++;
      continue;
    }
    if (c === ":" && text[i + 1] !== "=") {
      const top = frames[frames.length - 1];
      if (top?.kind === "brace") {
        top.seenColon = true;
        if (top.pendingKey) top.lastKey = top.pendingKey;
      }
      i++;
      continue;
    }
    i++;
  }

  const top = frames[frames.length - 1];
  if (!top) return undefined;
  if (top.kind === "call" && top.name) {
    return {
      kind: "call",
      name: top.name,
      receiver: top.receiver,
      paren: top.paren,
      argIndex: top.commas,
      funcStart: from,
    };
  }
  if (top.kind === "brace" && top.map) {
    return {
      kind: top.seenColon ? "value" : "key",
      key: top.lastKey,
      callName: top.callName,
      callReceiver: top.callReceiver,
      callParen: top.callParen,
      callArg: top.callArg,
      varName: top.varName,
      funcStart: from,
    };
  }
  if (top.kind === "bracket" && top.ident) {
    return { kind: "index", ident: top.ident, funcStart: from };
  }
  return undefined;
}

function popFrame(frames: Array<{ kind: string }>, kind: string): void {
  while (frames.length && frames[frames.length - 1].kind !== kind) frames.pop();
  frames.pop();
}

function goStringAt(text: string, offset: number): GoString | undefined {
  const lineStart = text.lastIndexOf("\n", Math.max(0, offset - 1)) + 1;
  const lineEndIdx = text.indexOf("\n", offset);
  const lineEnd = lineEndIdx === -1 ? text.length : lineEndIdx;
  const line = text.slice(lineStart, lineEnd);
  const col = offset - lineStart;
  let i = 0;
  while (i < line.length) {
    if (line[i] !== '"') {
      i++;
      continue;
    }
    let j = i + 1;
    while (j < line.length) {
      if (line[j] === "\\") {
        j += 2;
        continue;
      }
      if (line[j] === '"') break;
      j++;
    }
    const closed = j < line.length;
    const contentEnd = closed ? j : line.length;
    if (col > i && col <= contentEnd) {
      return {
        start: lineStart + i + 1,
        end: lineStart + (closed ? contentEnd : col),
      };
    }
    i = contentEnd + (closed ? 1 : 0);
    if (!closed) break;
  }
  return undefined;
}

function skipGoString(text: string, i: number, limit: number): number {
  const q = text[i];
  if (q === "`") {
    i++;
    while (i < limit && text[i] !== "`") i++;
    return Math.min(i + 1, limit);
  }
  if (q === "'") {
    i++;
    if (i < limit && text[i] === "\\") i += 2;
    else i++;
    if (i < limit && text[i] === "'") i++;
    return Math.min(i, limit);
  }
  i++;
  while (i < limit) {
    if (text[i] === "\\") {
      i += 2;
      continue;
    }
    if (text[i] === '"') return i + 1;
    i++;
  }
  return i;
}

function identBefore(text: string, pos: number): { name: string; receiver?: string } {
  let i = pos - 1;
  while (i >= 0 && /\s/.test(text[i])) i--;
  const end = i + 1;
  while (i >= 0 && /[\w]/.test(text[i])) i--;
  const name = text.slice(i + 1, end);
  if (!name) return { name: "" };
  let j = i;
  while (j >= 0 && /\s/.test(text[j])) j--;
  if (text[j] !== ".") return { name };
  j--;
  while (j >= 0 && /\s/.test(text[j])) j--;
  const rend = j + 1;
  while (j >= 0 && /[\w.]/.test(text[j])) j--;
  const receiver = text.slice(j + 1, rend).replace(/\.$/, "");
  return receiver ? { name, receiver } : { name };
}

function mapBefore(text: string, bracePos: number): { varName?: string } | undefined {
  let i = bracePos - 1;
  while (i >= 0 && /\s/.test(text[i])) i--;
  const end = i + 1;
  const slice = text.slice(Math.max(0, end - 80), end);
  const match = slice.match(/map\s*\[\s*string\s*\]\s*(?:any|interface\s*\{\s*\})\s*$/);
  if (!match || match.index === undefined) return undefined;
  const mapStart = Math.max(0, end - 80) + match.index;
  const before = text.slice(Math.max(0, mapStart - 80), mapStart);
  const assigned = before.match(/(\w+)\s*(?::=|=)\s*$/);
  return { varName: assigned?.[1] };
}

function registryBindings(text: string, funcStart: number, offset: number): Map<string, string> {
  const map = new Map<string, string>();
  const region = text.slice(funcStart, offset);
  const re = /(\w+)\s*,\s*\w+\s*:=\s*(?:[\w.]+\.)?ModelsFromHost\s*\(\s*[^,)\n]+\s*,\s*"([a-z][a-z0-9_]*)"\s*\)/g;
  for (const match of region.matchAll(re)) map.set(match[1], match[2]);
  return map;
}

function lastIdent(receiver?: string): string | undefined {
  if (!receiver) return undefined;
  const parts = receiver.split(".").filter(Boolean);
  return parts[parts.length - 1];
}

function readArgs(text: string, paren: number, limit: number): string[] {
  const args: string[] = [];
  let start = paren + 1;
  let parenD = 0;
  let braceD = 0;
  let bracketD = 0;
  let i = paren + 1;
  const end = Math.min(limit, text.length);
  while (i < end) {
    const c = text[i];
    if (c === '"' || c === "'" || c === "`") {
      i = skipGoString(text, i, end);
      continue;
    }
    if (c === "/" && text[i + 1] === "/") {
      while (i < end && text[i] !== "\n") i++;
      continue;
    }
    if (parenD === 0 && braceD === 0 && bracketD === 0 && c === ",") {
      args.push(text.slice(start, i));
      start = i + 1;
      i++;
      continue;
    }
    if (c === "(") parenD++;
    else if (c === ")") {
      if (parenD === 0) break;
      parenD--;
    } else if (c === "{") braceD++;
    else if (c === "}") braceD--;
    else if (c === "[") bracketD++;
    else if (c === "]") bracketD--;
    i++;
  }
  args.push(text.slice(start, Math.min(i, end)));
  return args;
}

function callArg(text: string, paren: number, index: number, limit: number): string | undefined {
  return readArgs(text, paren, limit)[index];
}

function stringArg(raw: string | undefined): string | undefined {
  if (!raw) return undefined;
  const match = raw.match(/^\s*"([^"]*)"\s*(?:,)?\s*$/);
  return match?.[1];
}

function funcEnd(text: string, funcStart: number): number {
  const idx = text.indexOf("\nfunc ", funcStart + 1);
  return idx < 0 ? text.length : idx;
}

function assignedModel(
  text: string,
  funcStart: number,
  limit: number,
  ident: string,
): { receiver?: string; model: string } | undefined {
  const region = text.slice(funcStart, limit);
  const re = new RegExp(
    `\\b${ident}\\s*,\\s*\\w+\\s*:=\\s*(?:([\\w.]+)\\.)?\\b(Create|Update|Get|GetByID|FindBy|ListAll|List)\\s*\\(`,
    "g",
  );
  let last: RegExpExecArray | undefined;
  for (const match of region.matchAll(re)) last = match;
  if (!last) return undefined;
  const paren = funcStart + last.index + last[0].length - 1;
  const spec = GO_METHODS[last[2]];
  const model = stringArg(readArgs(text, paren, funcEnd(text, funcStart))[spec.model]);
  if (!model) return undefined;
  return { receiver: last[1], model };
}

function fieldsVarModel(text: string, funcStart: number, varName: string): { receiver?: string; model: string } | undefined {
  const end = funcEnd(text, funcStart);
  const region = text.slice(funcStart, end);
  const re = /(?:([\w.]+)\.)?\b(Create|Update)\s*\(/g;
  for (const match of region.matchAll(re)) {
    if (match[2] !== "Create" && match[2] !== "Update") continue;
    const paren = funcStart + match.index + match[0].length - 1;
    const spec = GO_METHODS[match[2]];
    const args = readArgs(text, paren, end);
    if (args[spec.fieldsArg ?? -1]?.trim() !== varName) continue;
    const model = stringArg(args[spec.model]);
    if (model) return { receiver: match[1], model };
  }
  return undefined;
}

interface TagFrame {
  attrs: Record<string, string>;
}

function completeTsx(input: CompleteInput, index: ModelIndex, appName: string): Completion[] {
  const quoted = attrStringAt(input.text, input.offset);
  if (!quoted) return completeMember(input, index, appName);
  const tag = tagAt(input.text, quoted.quote, input.offset);
  if (!tag) return [];
  const replace = { start: quoted.start, end: quoted.end };
  if (tag.attr === "query") return cqrsItems(index.queries, appName, replace, "query");
  if (tag.attr === "command") return cqrsItems(index.commands, appName, replace, "command");
  if (tag.attr === "model") return modelItems(index, appName, replace, "qualified");
  if (!FIELD_ATTRS.has(tag.attr)) return [];
  return fieldItems(boundModel(index, appName, tag), replace, true);
}

function completeMember(input: CompleteInput, index: ModelIndex, appName: string): Completion[] {
  const access = memberAccessAt(input.text, input.offset);
  if (!access) return [];
  const callback = recordCallback(input.text, input.offset, access.ident);
  if (!callback) return [];
  const tag = tagContaining(input.text, callback.at);
  if (!tag) return [];
  return fieldItems(
    boundModel(index, appName, { attr: "", current: tag.attrs, ancestors: tag.ancestors }),
    access.replace,
    true,
  );
}

function memberAccessAt(text: string, offset: number): { ident: string; replace: ReplaceSpan } | undefined {
  let end = offset;
  while (end < text.length && /[\w$]/.test(text[end])) end++;
  let i = offset;
  while (i > 0 && /[\w$]/.test(text[i - 1])) i--;
  const propStart = i;
  while (i > 0 && /\s/.test(text[i - 1])) i--;
  if (text[i - 1] !== ".") return undefined;
  i--;
  while (i > 0 && /\s/.test(text[i - 1])) i--;
  const identEnd = i;
  while (i > 0 && /[\w$]/.test(text[i - 1])) i--;
  const ident = text.slice(i, identEnd);
  if (!/^[$A-Za-z_][\w$]*$/.test(ident)) return undefined;
  if (i > 0 && text[i - 1] === ".") return undefined;
  return { ident, replace: { start: propStart, end } };
}

function recordCallback(text: string, offset: number, ident: string): { at: number } | undefined {
  const re = new RegExp(
    `(\\w+)\\s*=\\s*\\{\\s*(?:\\(\\s*${ident}\\b[^)]*\\)|${ident})\\s*=>`,
    "g",
  );
  let found: { at: number } | undefined;
  for (const match of text.matchAll(re)) {
    if (!match[1] || !RECORD_ATTRS.has(match[1]) || match.index === undefined) continue;
    const arrowEnd = match.index + match[0].length;
    if (arrowEnd > offset) continue;
    if (!arrowBodyContains(text, arrowEnd, offset)) continue;
    found = { at: match.index };
  }
  return found;
}

function arrowBodyContains(text: string, arrowEnd: number, offset: number): boolean {
  let i = arrowEnd;
  while (i < text.length && /\s/.test(text[i])) i++;
  const block = text[i] === "{";
  let depth = 0;
  if (block) {
    depth = 1;
    i++;
  }
  while (i < offset && i < text.length) {
    const c = text[i];
    if (c === '"' || c === "'" || c === "`") {
      i = skipJsString(text, i);
      continue;
    }
    if (c === "/" && text[i + 1] === "/") {
      while (i < text.length && text[i] !== "\n") i++;
      continue;
    }
    if (c === "/" && text[i + 1] === "*") {
      i += 2;
      while (i < text.length && !(text[i] === "*" && text[i + 1] === "/")) i++;
      i += 2;
      continue;
    }
    if (c === "{") depth++;
    else if (c === "}") {
      if (depth === 0) return false;
      depth--;
      if (block && depth === 0) return false;
    }
    i++;
  }
  return true;
}

interface AttrString {
  start: number;
  end: number;
  quote: number;
}

function attrStringAt(text: string, offset: number): AttrString | undefined {
  const lineStart = text.lastIndexOf("\n", Math.max(0, offset - 1)) + 1;
  const lineEndIdx = text.indexOf("\n", offset);
  const lineEnd = lineEndIdx === -1 ? text.length : lineEndIdx;
  const line = text.slice(lineStart, lineEnd);
  const col = offset - lineStart;
  let i = 0;
  while (i < line.length) {
    const q = line[i];
    if (q !== '"' && q !== "'") {
      i++;
      continue;
    }
    let j = i + 1;
    while (j < line.length) {
      if (line[j] === "\\") {
        j += 2;
        continue;
      }
      if (line[j] === q) break;
      j++;
    }
    const closed = j < line.length;
    const contentEnd = closed ? j : line.length;
    if (col > i && col <= contentEnd) {
      return {
        quote: lineStart + i,
        start: lineStart + i + 1,
        end: lineStart + (closed ? contentEnd : col),
      };
    }
    i = contentEnd + (closed ? 1 : 0);
    if (!closed) break;
  }
  return undefined;
}

interface TagHit {
  attr: string;
  current: Record<string, string>;
  ancestors: TagFrame[];
}

function tagContaining(text: string, offset: number): { attrs: Record<string, string>; ancestors: TagFrame[] } | undefined {
  const stack: TagFrame[] = [];
  let i = 0;
  while (i < text.length && i <= offset) {
    const c = text[i];
    if (c === '"' || c === "'" || c === "`") {
      i = skipJsString(text, i);
      continue;
    }
    if (c === "/" && text[i + 1] === "/") {
      while (i < text.length && text[i] !== "\n") i++;
      continue;
    }
    if (c === "/" && text[i + 1] === "*") {
      i += 2;
      while (i < text.length && !(text[i] === "*" && text[i + 1] === "/")) i++;
      i += 2;
      continue;
    }
    if (c === "<" && text[i + 1] === "/") {
      const close = text.indexOf(">", i + 2);
      if (close === -1) return undefined;
      stack.pop();
      i = close + 1;
      continue;
    }
    if (c === "<" && text[i + 1] === ">" && jsxOpenAllowed(text, i)) {
      stack.push({ attrs: {} });
      i += 2;
      continue;
    }
    if (c === "<" && /[A-Za-z]/.test(text[i + 1] ?? "") && jsxOpenAllowed(text, i)) {
      const tag = parseOpenTag(text, i, -1);
      if (!tag || tag.end <= i) {
        i++;
        continue;
      }
      if (offset >= i && offset < tag.end) return { attrs: tag.attrs, ancestors: stack.map((frame) => ({ attrs: { ...frame.attrs } })) };
      if (!tag.selfClose) stack.push({ attrs: tag.attrs });
      i = tag.end;
      continue;
    }
    i++;
  }
  return undefined;
}

function tagAt(text: string, quotePos: number, offset: number): TagHit | undefined {
  const stack: TagFrame[] = [];
  let i = 0;
  while (i < text.length && i <= quotePos) {
    const c = text[i];
    if (c === '"' || c === "'" || c === "`") {
      i = skipJsString(text, i);
      continue;
    }
    if (c === "/" && text[i + 1] === "/") {
      while (i < text.length && text[i] !== "\n") i++;
      continue;
    }
    if (c === "/" && text[i + 1] === "*") {
      i += 2;
      while (i < text.length && !(text[i] === "*" && text[i + 1] === "/")) i++;
      i += 2;
      continue;
    }
    if (c === "<" && text[i + 1] === "/") {
      const close = text.indexOf(">", i + 2);
      if (close === -1) return undefined;
      stack.pop();
      i = close + 1;
      continue;
    }
    if (c === "<" && text[i + 1] === ">" && jsxOpenAllowed(text, i)) {
      stack.push({ attrs: {} });
      i += 2;
      continue;
    }
    if (c === "<" && /[A-Za-z]/.test(text[i + 1] ?? "") && jsxOpenAllowed(text, i)) {
      const tag = parseOpenTag(text, i, offset);
      if (!tag) {
        i++;
        continue;
      }
      if (tag.attr && tag.quote <= quotePos && offset <= tag.end) {
        return { attr: tag.attr, current: tag.attrs, ancestors: stack.map((f) => ({ attrs: { ...f.attrs } })) };
      }
      if (!tag.selfClose) stack.push({ attrs: tag.attrs });
      i = tag.end;
      continue;
    }
    i++;
  }
  return undefined;
}

interface ParsedTag {
  attrs: Record<string, string>;
  selfClose: boolean;
  end: number;
  attr?: string;
  quote: number;
}

function parseOpenTag(text: string, lt: number, offset: number): ParsedTag | undefined {
  let i = lt + 1;
  while (i < text.length && /[\w.]/.test(text[i])) i++;
  const attrs: Record<string, string> = {};
  let attr: string | undefined;
  let quote = -1;
  let brace = 0;
  while (i < text.length) {
    const c = text[i];
    if (brace > 0) {
      if (c === '"' || c === "'" || c === "`") {
        i = skipJsString(text, i);
        continue;
      }
      if (c === "{") brace++;
      else if (c === "}") brace--;
      i++;
      continue;
    }
    if (c === "{") {
      brace++;
      i++;
      continue;
    }
    if (text.startsWith("/>", i)) {
      return { attrs, selfClose: true, end: i + 2, attr, quote };
    }
    if (c === ">") return { attrs, selfClose: false, end: i + 1, attr, quote };
    if (/[\s]/.test(c)) {
      i++;
      continue;
    }
    if (!/[A-Za-z_]/.test(c)) {
      i++;
      continue;
    }
    const nameStart = i;
    while (i < text.length && /[\w:-]/.test(text[i])) i++;
    const name = text.slice(nameStart, i);
    while (i < text.length && /[\s]/.test(text[i])) i++;
    if (text[i] !== "=") continue;
    i++;
    while (i < text.length && /[\s]/.test(text[i])) i++;
    const q = text[i];
    if (q === '"' || q === "'") {
      const contentStart = i + 1;
      let j = contentStart;
      while (j < text.length && text[j] !== q) {
        if (text[j] === "\\") j += 2;
        else j++;
      }
      const value = text.slice(contentStart, j);
      const closed = j < text.length;
      if (offset >= contentStart && offset <= (closed ? j : offset)) {
        attr = name;
        quote = i;
      } else {
        attrs[name] = value;
      }
      i = closed ? j + 1 : j;
      continue;
    }
    if (q === "{") {
      brace = 1;
      i++;
      continue;
    }
    i++;
  }
  if (attr) return { attrs, selfClose: false, end: text.length, attr, quote };
  return undefined;
}

function jsxOpenAllowed(text: string, lt: number): boolean {
  let i = lt - 1;
  while (i >= 0 && /[\s]/.test(text[i])) i--;
  if (i < 0) return true;
  const c = text[i];
  if ("({[=,;?:&|}]".includes(c) || c === ">") return true;
  if (/[\w$]/.test(c)) {
    const end = i + 1;
    while (i >= 0 && /[\w$]/.test(text[i])) i--;
    const word = text.slice(i + 1, end);
    return word === "return" || word === "yield";
  }
  return false;
}

function skipJsString(text: string, i: number): number {
  const q = text[i];
  i++;
  if (q === "`") {
    while (i < text.length && text[i] !== "`") {
      if (text[i] === "\\") i += 2;
      else i++;
    }
    return Math.min(i + 1, text.length);
  }
  while (i < text.length && text[i] !== q) {
    if (text[i] === "\\") i += 2;
    else i++;
  }
  return Math.min(i + 1, text.length);
}

function boundModel(index: ModelIndex, appName: string, tag: TagHit): ModelInfo | undefined {
  const chain = [tag.current, ...[...tag.ancestors].reverse().map((f) => f.attrs)];
  for (const attrs of chain) {
    if (attrs.model) {
      const model = resolveModelRef(index, appName, attrs.model);
      if (model) return model;
    }
    if (attrs.command) {
      const model = modelFromCqrs(index, index.commands, attrs.command);
      if (model) return model;
    }
    if (attrs.query) {
      const model = modelFromCqrs(index, index.queries, attrs.query);
      if (model) return model;
    }
  }
  return undefined;
}

function modelFromCqrs(index: ModelIndex, rows: CqrsInfo[], ref: string): ModelInfo | undefined {
  const row = rows.find((r) => r.ref === ref);
  if (!row?.model) return undefined;
  return resolveModelRef(index, row.app, row.model);
}
