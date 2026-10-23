import type { AppIndex } from "./catalog";
import type { YamlSymbol } from "./parse";

export interface FieldInfo {
  name: string;
  type: string;
  required: boolean;
  relation?: string;
  values: string[];
  builtin?: boolean;
}

export interface ModelInfo {
  app: string;
  name: string;
  /** `app.model` */
  ref: string;
  internal: boolean;
  virtual: boolean;
  fields: FieldInfo[];
}

export interface CqrsInfo {
  app: string;
  name: string;
  /** `app.name` */
  ref: string;
  /** list, get, create, update, or delete */
  op?: string;
  /** Model name as written in the spec (`greeting` or `identity.user`). */
  model?: string;
}

export interface ModelIndex {
  models: ModelInfo[];
  queries: CqrsInfo[];
  commands: CqrsInfo[];
}

export const BUILTIN_FIELDS: FieldInfo[] = [
  { name: "id", type: "id", required: false, values: [], builtin: true },
  { name: "orgId", type: "id", required: false, values: [], builtin: true },
  { name: "authorId", type: "id", required: false, values: [], builtin: true },
  { name: "createdAt", type: "datetime", required: false, values: [], builtin: true },
  { name: "updatedAt", type: "datetime", required: false, values: [], builtin: true },
  { name: "deleted", type: "bool", required: false, values: [], builtin: true },
];

export const FIELD_TYPES = [
  "string",
  "text",
  "int",
  "number",
  "bool",
  "enum",
  "date",
  "datetime",
  "json",
  "many2one",
  "one2many",
  "many2many",
] as const;

const FIELD_TYPE_DETAIL: Record<string, string> = {
  string: "short text",
  text: "long text",
  int: "integer",
  number: "decimal",
  bool: "boolean",
  enum: "one of values:",
  date: "calendar date",
  datetime: "timestamp",
  json: "JSON value",
  many2one: "relation to one record",
  one2many: "inverse collection (needs inverse:)",
  many2many: "relation to many records",
};

export function fieldTypeDetail(type: string): string {
  return FIELD_TYPE_DETAIL[type] ?? type;
}

function sym(symbols: YamlSymbol[], path: string): string | undefined {
  return symbols.find((s) => s.path === path)?.value;
}

function flag(symbols: YamlSymbol[], path: string): boolean {
  return sym(symbols, path) === "true";
}

function modelsInDoc(appName: string, symbols: YamlSymbol[]): ModelInfo[] {
  const out: ModelInfo[] = [];
  for (const nameSym of symbols) {
    if (!/^models\[\d+\]\.name$/.test(nameSym.path)) continue;
    const prefix = nameSym.path.slice(0, -".name".length);
    const fields: FieldInfo[] = [];
    for (const fieldSym of symbols) {
      if (!new RegExp(`^${escapeRe(prefix)}\\.fields\\[\\d+\\]\\.name$`).test(fieldSym.path)) continue;
      const fp = fieldSym.path.slice(0, -".name".length);
      const values = symbols
        .filter((s) => new RegExp(`^${escapeRe(fp)}\\.values\\[\\d+\\]$`).test(s.path))
        .map((s) => s.value);
      fields.push({
        name: fieldSym.value,
        type: sym(symbols, `${fp}.type`) ?? "string",
        required: flag(symbols, `${fp}.required`),
        relation: sym(symbols, `${fp}.relation`),
        values,
      });
    }
    out.push({
      app: appName,
      name: nameSym.value,
      ref: `${appName}.${nameSym.value}`,
      internal: flag(symbols, `${prefix}.internal`),
      virtual: flag(symbols, `${prefix}.virtual`),
      fields,
    });
  }
  return out;
}

function cqrsInDoc(appName: string, symbols: YamlSymbol[], key: "queries" | "commands"): CqrsInfo[] {
  const ops = key === "queries" ? (["list", "get"] as const) : (["create", "update", "delete"] as const);
  const out: CqrsInfo[] = [];
  for (const nameSym of symbols) {
    if (!new RegExp(`^${key}\\[\\d+\\]\\.name$`).test(nameSym.path)) continue;
    const prefix = nameSym.path.slice(0, -".name".length);
    let op: string | undefined;
    let model: string | undefined;
    for (const candidate of ops) {
      const value = sym(symbols, `${prefix}.${candidate}`);
      if (value) {
        op = candidate;
        model = value;
        break;
      }
    }
    out.push({
      app: appName,
      name: nameSym.value,
      ref: `${appName}.${nameSym.value}`,
      op,
      model,
    });
  }
  return out;
}

function escapeRe(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

export function buildIndex(apps: Iterable<AppIndex>): ModelIndex {
  const models = new Map<string, ModelInfo>();
  const queries = new Map<string, CqrsInfo>();
  const commands = new Map<string, CqrsInfo>();
  for (const app of apps) {
    const docs = app.docs?.length ? app.docs : [{ path: app.yamlPath, text: app.text, symbols: app.symbols }];
    for (const doc of docs) {
      for (const model of modelsInDoc(app.name, doc.symbols)) {
        const prev = models.get(model.ref);
        if (!prev || model.fields.length >= prev.fields.length) models.set(model.ref, model);
      }
      for (const query of cqrsInDoc(app.name, doc.symbols, "queries")) queries.set(query.ref, query);
      for (const command of cqrsInDoc(app.name, doc.symbols, "commands")) commands.set(command.ref, command);
    }
  }
  return {
    models: [...models.values()],
    queries: [...queries.values()],
    commands: [...commands.values()],
  };
}

export function findModel(index: ModelIndex, app: string, name: string): ModelInfo | undefined {
  return index.models.find((m) => m.app === app && m.name === name);
}

export function fieldDetail(field: FieldInfo): string {
  const bits: string[] = [field.builtin ? "builtin" : field.type];
  if (field.required) bits.push("required");
  if (field.relation) bits.push(`→ ${field.relation}`);
  if (field.values.length) bits.push(field.values.join(" | "));
  return bits.join(" · ");
}

export function modelDetail(model: ModelInfo): string {
  const bits = [model.ref];
  if (model.internal) bits.push("internal");
  if (model.virtual) bits.push("virtual");
  const n = model.fields.length;
  bits.push(`${n} field${n === 1 ? "" : "s"}`);
  return bits.join(" · ");
}

export function modelDoc(model: ModelInfo): string {
  const names = model.fields.map((f) => f.name).join(", ");
  return names ? `${model.ref}\n${names}` : model.ref;
}
