import { isMap, isScalar, isSeq, parseDocument, Scalar, type YAMLMap } from "yaml";

export interface Range {
  start: number;
  end: number;
}

export interface SpecIssue {
  message: string;
  range: Range;
  severity: "error" | "warning";
}

export interface FieldBox {
  name: string;
  widget?: string;
  span?: number;
  width?: string;
  align?: string;
  order?: number;
  range: Range;
}

export interface GroupBox {
  label: string;
  columns: number;
  fields: FieldBox[];
  range: Range;
}

export interface FilterBox {
  id: string;
  range: Range;
}

export interface ComponentBox {
  id: string;
  kind: string;
  model: string;
  defaultView: string;
  modes: string[];
  groupBy: string[];
  columns: FieldBox[];
  groups: GroupBox[];
  filters: FilterBox[];
  span?: number;
  size?: string;
  range: Range;
}

export interface LayoutBox {
  direction: "" | "column" | "row";
  componentId?: string;
  size?: string;
  span?: number;
  items: LayoutBox[];
  range: Range;
  directionRange?: Range;
}

export interface ViewBox {
  name: string;
  path?: string;
  pathRange?: Range;
  layout: LayoutBox;
  components: ComponentBox[];
  range: Range;
}

export interface TreeNode {
  label: string;
  detail?: string;
  range: Range;
  children: TreeNode[];
}

export interface SpecTree {
  name: string;
  title: string;
  range: Range;
  views: ViewBox[];
  models: TreeNode[];
  queries: TreeNode[];
  commands: TreeNode[];
  menus: TreeNode[];
  locales: TreeNode[];
  depends: TreeNode[];
  issues: SpecIssue[];
  error?: string;
}

const KINDS = new Set(["collection", "table", "kanban", "chart", "form"]);
const MODES = new Set(["table", "kanban", "chart", "pivot"]);
const ALIGNS = new Set(["left", "center", "right"]);
const SIZE_RE = /^(?:auto|\d+(?:\.\d+)?(?:fr|%|px|rem|em|vw|vh))$/;
const BUILTIN = new Set(["id", "orgId", "authorId", "createdAt", "updatedAt", "deleted"]);

interface InlineModel {
  fields: Set<string>;
}

export function parseSpec(text: string): SpecTree {
  const empty = blank();
  let doc: ReturnType<typeof parseDocument>;
  try {
    doc = parseDocument(text);
  } catch (err) {
    return { ...empty, error: err instanceof Error ? err.message : String(err) };
  }
  if (!isMap(doc.contents)) {
    return { ...empty, error: doc.errors[0]?.message };
  }
  const root = doc.contents;
  const issues: SpecIssue[] = [];
  const nameNode = mapGet(root, "name");
  const titleNode = mapGet(root, "title");
  const models = parseModels(mapGet(root, "models"));
  const views = parseViews(mapGet(root, "views"), models.inline, models.includes, issues);
  return {
    name: scalarString(nameNode) ?? "",
    title: scalarString(titleNode) ?? "",
    range: isScalar(nameNode) ? valueRange(nameNode) : rangeOf(root),
    views,
    models: models.nodes,
    queries: parseOps(mapGet(root, "queries")),
    commands: parseOps(mapGet(root, "commands")),
    menus: parseMenus(mapGet(root, "menus")),
    locales: parseLocales(mapGet(root, "locales")),
    depends: parseDepends(mapGet(root, "depends")),
    issues,
    error: doc.errors[0]?.message,
  };
}

function blank(): SpecTree {
  return {
    name: "",
    title: "",
    range: { start: 0, end: 0 },
    views: [],
    models: [],
    queries: [],
    commands: [],
    menus: [],
    locales: [],
    depends: [],
    issues: [],
  };
}

function parseModels(node: unknown): { nodes: TreeNode[]; inline: Map<string, InlineModel>; includes: number } {
  const nodes: TreeNode[] = [];
  const inline = new Map<string, InlineModel>();
  let includes = 0;
  if (!isSeq(node)) return { nodes, inline, includes };
  for (const item of node.items) {
    if (isScalar(item)) {
      includes++;
      const label = scalarString(item) ?? "";
      if (label) nodes.push({ label, detail: "include", range: valueRange(item), children: [] });
      continue;
    }
    if (!isMap(item)) continue;
    const nameNode = mapGet(item, "name");
    const name = scalarString(nameNode) ?? "";
    const fields = new Set<string>();
    const children: TreeNode[] = [];
    const fieldSeq = mapGet(item, "fields");
    if (isSeq(fieldSeq)) {
      for (const field of fieldSeq.items) {
        if (!isMap(field)) continue;
        const fieldName = mapGet(field, "name");
        const label = scalarString(fieldName);
        if (!label) continue;
        fields.add(label);
        const type = scalarString(mapGet(field, "type"));
        children.push({
          label,
          detail: type,
          range: isScalar(fieldName) ? valueRange(fieldName) : rangeOf(field),
          children: [],
        });
      }
    }
    if (name) inline.set(name, { fields });
    nodes.push({
      label: name || "(model)",
      detail: scalarString(mapGet(item, "virtual")) === "true" ? "virtual" : undefined,
      range: isScalar(nameNode) ? valueRange(nameNode) : rangeOf(item),
      children,
    });
  }
  return { nodes, inline, includes };
}

function parseViews(
  node: unknown,
  models: Map<string, InlineModel>,
  includes: number,
  issues: SpecIssue[],
): ViewBox[] {
  if (!isSeq(node)) return [];
  const views: ViewBox[] = [];
  const seen = new Set<string>();
  for (const item of node.items) {
    if (!isMap(item)) continue;
    const nameNode = mapGet(item, "name");
    const name = scalarString(nameNode) ?? "";
    const nameRange = isScalar(nameNode) ? valueRange(nameNode) : rangeOf(item);
    if (name) {
      if (seen.has(name)) issues.push({ message: `Duplicate view "${name}"`, range: nameRange, severity: "error" });
      seen.add(name);
    }
    const pathNode = mapGet(item, "path");
    const kindNode = mapGet(item, "kind");
    const components = parseComponents(mapGet(item, "components"), name, models, includes, issues);
    if (scalarString(kindNode) && components.length && isScalar(kindNode)) {
      issues.push({
        message: `View "${name}" sets kind and components`,
        range: valueRange(kindNode),
        severity: "error",
      });
    }
    const own = parseComponentBody(item, scalarString(mapGet(item, "model")) ?? "", models, includes, issues);
    const layoutNode = mapGet(item, "layout");
    const raw = layoutNode == null ? { direction: "column" as const, items: [], range: nameRange } : parseLayout(layoutNode, issues);
    if (components.length) {
      layoutIssues(raw, new Set(components.map((c) => c.id).filter(Boolean)), new Set(), issues);
    }
    const screens = components.length ? components : own.kind || own.columns.length || own.groups.length ? [synthetic(name, nameRange, own)] : [];
    views.push({
      name,
      path: scalarString(pathNode),
      pathRange: isScalar(pathNode) ? valueRange(pathNode) : undefined,
      layout: fillLayout(raw, screens),
      components: screens,
      range: nameRange,
    });
  }
  return views;
}

interface ComponentBody {
  kind: string;
  model: string;
  defaultView: string;
  modes: string[];
  groupBy: string[];
  columns: FieldBox[];
  groups: GroupBox[];
  filters: FilterBox[];
}

function parseComponents(
  node: unknown,
  view: string,
  models: Map<string, InlineModel>,
  includes: number,
  issues: SpecIssue[],
): ComponentBox[] {
  if (!isSeq(node)) return [];
  const out: ComponentBox[] = [];
  const seen = new Set<string>();
  for (const item of node.items) {
    if (!isMap(item)) continue;
    const idNode = mapGet(item, "id");
    const id = scalarString(idNode) ?? "";
    const idRange = isScalar(idNode) ? valueRange(idNode) : rangeOf(item);
    if (!id) issues.push({ message: `View "${view}" has a component without an id`, range: rangeOf(item), severity: "error" });
    else if (seen.has(id)) issues.push({ message: `Duplicate component "${id}"`, range: idRange, severity: "error" });
    if (id) seen.add(id);
    const model = scalarString(mapGet(item, "model")) ?? "";
    const body = parseComponentBody(item, model, models, includes, issues);
    const span = num(mapGet(item, "span"));
    const sizeNode = mapGet(item, "size");
    const size = scalarString(sizeNode);
    if (span != null && span < 0) issues.push({ message: "Invalid span", range: rangeOf(item), severity: "warning" });
    if (size && !SIZE_RE.test(size) && isScalar(sizeNode)) {
      issues.push({ message: `Invalid size "${size}"`, range: valueRange(sizeNode), severity: "warning" });
    }
    out.push({
      id,
      ...body,
      span,
      size,
      range: idRange,
    });
  }
  return out;
}

function parseComponentBody(
  item: YAMLMap,
  model: string,
  models: Map<string, InlineModel>,
  includes: number,
  issues: SpecIssue[],
): ComponentBody {
  const kindNode = mapGet(item, "kind");
  const kind = scalarString(kindNode) ?? "";
  if (kind && !KINDS.has(kind) && isScalar(kindNode)) {
    issues.push({ message: `Unknown kind "${kind}"`, range: valueRange(kindNode), severity: "error" });
  }
  const modelNode = mapGet(item, "model");
  noteModel(model, isScalar(modelNode) ? valueRange(modelNode) : undefined, models, includes, issues);
  const defaultNode = mapGet(item, "defaultView");
  const defaultView = scalarString(defaultNode) ?? "";
  const modes = stringList(mapGet(item, "views"));
  if (defaultView && !MODES.has(defaultView) && isScalar(defaultNode)) {
    issues.push({ message: `Unknown defaultView "${defaultView}"`, range: valueRange(defaultNode), severity: "warning" });
  } else if (defaultView && modes.length && !modes.includes(defaultView) && isScalar(defaultNode)) {
    issues.push({ message: `defaultView "${defaultView}" is not in views`, range: valueRange(defaultNode), severity: "warning" });
  }
  const modeSeq = mapGet(item, "views");
  if (isSeq(modeSeq)) {
    for (const mode of modeSeq.items) {
      const value = scalarString(mode);
      if (value && !MODES.has(value) && isScalar(mode)) {
        issues.push({ message: `Unknown view mode "${value}"`, range: valueRange(mode), severity: "warning" });
      }
    }
  }
  const known = fieldsOf(model, models);
  const groupBy = stringList(mapGet(item, "groupBy"));
  const groupSeq = mapGet(item, "groupBy");
  if (known && isSeq(groupSeq)) {
    for (const entry of groupSeq.items) {
      const value = scalarString(entry);
      if (value && !known.has(value) && isScalar(entry)) {
        issues.push({ message: `Unknown field "${value}"`, range: valueRange(entry), severity: "warning" });
      }
    }
  }
  return {
    kind,
    model,
    defaultView,
    modes,
    groupBy,
    columns: parseFields(mapGet(item, "columns"), known, issues),
    groups: parseGroups(mapGet(item, "groups"), known, issues),
    filters: parseFilters(mapGet(item, "filters")),
  };
}

function synthetic(name: string, range: Range, body: ComponentBody): ComponentBox {
  return { id: name, ...body, range };
}

function noteModel(
  model: string,
  range: Range | undefined,
  models: Map<string, InlineModel>,
  includes: number,
  issues: SpecIssue[],
): void {
  if (!model || !range || model.includes(".")) return;
  if (models.has(model) || includes > 0 || models.size === 0) return;
  issues.push({ message: `Unknown model "${model}"`, range, severity: "warning" });
}

function fieldsOf(model: string, models: Map<string, InlineModel>): Set<string> | undefined {
  if (!model || model.includes(".")) return undefined;
  const found = models.get(model);
  if (!found) return undefined;
  const fields = new Set(found.fields);
  for (const name of BUILTIN) fields.add(name);
  return fields;
}

function parseFields(node: unknown, known: Set<string> | undefined, issues: SpecIssue[]): FieldBox[] {
  if (!isSeq(node)) return [];
  const fields: FieldBox[] = [];
  for (const item of node.items) {
    if (!isMap(item)) continue;
    const nameNode = mapGet(item, "field");
    const name = scalarString(nameNode) ?? "";
    const range = isScalar(nameNode) ? valueRange(nameNode) : rangeOf(item);
    if (name && known && !known.has(name)) {
      issues.push({ message: `Unknown field "${name}"`, range, severity: "warning" });
    }
    const alignNode = mapGet(item, "align");
    const align = scalarString(alignNode);
    if (align && !ALIGNS.has(align) && isScalar(alignNode)) {
      issues.push({ message: `Invalid align "${align}"`, range: valueRange(alignNode), severity: "warning" });
    }
    const span = num(mapGet(item, "span"));
    if (span != null && span < 0) issues.push({ message: "Invalid span", range, severity: "warning" });
    fields.push({
      name,
      widget: scalarString(mapGet(item, "widget")),
      span,
      width: scalarString(mapGet(item, "width")),
      align,
      order: num(mapGet(item, "order")),
      range,
    });
  }
  return byOrder(fields);
}

function parseGroups(node: unknown, known: Set<string> | undefined, issues: SpecIssue[]): GroupBox[] {
  if (!isSeq(node)) return [];
  const groups: GroupBox[] = [];
  for (const item of node.items) {
    if (!isMap(item)) continue;
    const label = scalarString(mapGet(item, "label")) || scalarString(mapGet(item, "labelKey")) || "";
    const labelNode = mapGet(item, "label") ?? mapGet(item, "labelKey");
    if (!label) issues.push({ message: "Group requires label or labelKey", range: rangeOf(item), severity: "warning" });
    const columns = num(mapGet(item, "columns")) ?? 0;
    if (columns < 0) issues.push({ message: "Invalid columns", range: rangeOf(item), severity: "warning" });
    groups.push({
      label,
      columns: columns > 0 ? columns : 1,
      fields: parseFields(mapGet(item, "fields"), known, issues),
      range: isScalar(labelNode) ? valueRange(labelNode) : rangeOf(item),
    });
  }
  return groups;
}

function parseFilters(node: unknown): FilterBox[] {
  if (!isSeq(node)) return [];
  const filters: FilterBox[] = [];
  for (const item of node.items) {
    if (!isMap(item)) continue;
    const idNode = mapGet(item, "id");
    const id = scalarString(idNode);
    if (!id) continue;
    filters.push({ id, range: isScalar(idNode) ? valueRange(idNode) : rangeOf(item) });
  }
  return filters;
}

function parseLayout(node: unknown, issues: SpecIssue[]): LayoutBox {
  if (isScalar(node)) {
    const raw = scalarString(node) ?? "";
    const ok = raw === "row" || raw === "column";
    if (raw && !ok) issues.push({ message: `Invalid layout direction "${raw}"`, range: valueRange(node), severity: "error" });
    return {
      direction: raw === "row" ? "row" : "column",
      items: [],
      range: valueRange(node),
      directionRange: valueRange(node),
    };
  }
  if (!isMap(node)) return { direction: "column", items: [], range: { start: 0, end: 0 } };
  const directionNode = mapGet(node, "direction");
  const rawDir = scalarString(directionNode) ?? "";
  let direction: LayoutBox["direction"] = "";
  if (rawDir === "row" || rawDir === "column") direction = rawDir;
  else if (rawDir && isScalar(directionNode)) {
    issues.push({ message: `Invalid layout direction "${rawDir}"`, range: valueRange(directionNode), severity: "error" });
  }
  const componentNode = mapGet(node, "component");
  const componentId = scalarString(componentNode);
  const sizeNode = mapGet(node, "size");
  const size = scalarString(sizeNode);
  if (size && !SIZE_RE.test(size) && isScalar(sizeNode)) {
    issues.push({ message: `Invalid size "${size}"`, range: valueRange(sizeNode), severity: "warning" });
  }
  const span = num(mapGet(node, "span"));
  if (span != null && span < 0) issues.push({ message: "Invalid span", range: rangeOf(node), severity: "warning" });
  const itemsNode = mapGet(node, "items");
  const items = isSeq(itemsNode) ? itemsNode.items.map((item) => parseLayout(item, issues)) : [];
  if (componentId && items.length) {
    issues.push({ message: `Layout node "${componentId}" cannot nest items`, range: rangeOf(node), severity: "error" });
  }
  return {
    direction,
    componentId,
    size,
    span,
    items,
    range: isScalar(componentNode) ? valueRange(componentNode) : rangeOf(node),
    directionRange: isScalar(directionNode) ? valueRange(directionNode) : undefined,
  };
}

function layoutIssues(node: LayoutBox, ids: Set<string>, used: Set<string>, issues: SpecIssue[]): void {
  if (node.componentId) {
    if (!ids.has(node.componentId)) {
      issues.push({ message: `Unknown component "${node.componentId}"`, range: node.range, severity: "error" });
    } else if (used.has(node.componentId)) {
      issues.push({ message: `Component "${node.componentId}" is repeated in the layout`, range: node.range, severity: "error" });
    } else {
      used.add(node.componentId);
    }
  }
  for (const child of node.items) layoutIssues(child, ids, used, issues);
}

function fillLayout(layout: LayoutBox, components: ComponentBox[]): LayoutBox {
  if (layout.items.length > 0 || layout.componentId) return layout;
  return {
    ...layout,
    direction: layout.direction === "row" ? "row" : "column",
    items: components.map((component) => ({
      direction: "" as const,
      componentId: component.id,
      size: component.size,
      span: component.span,
      items: [],
      range: component.range,
    })),
  };
}

function parseOps(node: unknown): TreeNode[] {
  if (!isSeq(node)) return [];
  const out: TreeNode[] = [];
  for (const item of node.items) {
    if (!isMap(item)) continue;
    const nameNode = mapGet(item, "name");
    const label = scalarString(nameNode) ?? "(unnamed)";
    let detail = "";
    for (const key of ["list", "get", "create", "update", "delete"]) {
      const value = scalarString(mapGet(item, key));
      if (value) {
        detail = `${key} ${value}`;
        break;
      }
    }
    out.push({
      label,
      detail,
      range: isScalar(nameNode) ? valueRange(nameNode) : rangeOf(item),
      children: [],
    });
  }
  return out;
}

function parseMenus(node: unknown): TreeNode[] {
  if (!isSeq(node)) return [];
  const out: TreeNode[] = [];
  for (const item of node.items) {
    if (!isMap(item)) continue;
    const idNode = mapGet(item, "id");
    const view = scalarString(mapGet(item, "view"));
    const model = scalarString(mapGet(item, "model"));
    out.push({
      label: scalarString(idNode) ?? "(menu)",
      detail: view || model,
      range: isScalar(idNode) ? valueRange(idNode) : rangeOf(item),
      children: parseMenus(mapGet(item, "children")),
    });
  }
  return out;
}

function parseLocales(node: unknown): TreeNode[] {
  if (!isSeq(node)) return [];
  const out: TreeNode[] = [];
  for (const item of node.items) {
    if (!isMap(item)) continue;
    const idNode = mapGet(item, "id");
    out.push({
      label: scalarString(idNode) ?? "(locale)",
      detail: scalarString(mapGet(item, "name")),
      range: isScalar(idNode) ? valueRange(idNode) : rangeOf(item),
      children: [],
    });
  }
  return out;
}

function parseDepends(node: unknown): TreeNode[] {
  if (!isSeq(node)) return [];
  const out: TreeNode[] = [];
  for (const item of node.items) {
    if (!isScalar(item)) continue;
    const label = scalarString(item);
    if (!label) continue;
    out.push({ label, range: valueRange(item), children: [] });
  }
  return out;
}

function byOrder(fields: FieldBox[]): FieldBox[] {
  if (!fields.some((field) => (field.order ?? 0) > 0)) return fields;
  return [...fields].sort((a, b) => (a.order ?? 0) - (b.order ?? 0));
}

function mapGet(node: YAMLMap, key: string): unknown {
  for (const pair of node.items) {
    if (isScalar(pair.key) && String(pair.key.value ?? "") === key) return pair.value ?? undefined;
  }
  return undefined;
}

function scalarString(node: unknown): string | undefined {
  if (!isScalar(node) || node.value == null) return undefined;
  if (typeof node.value === "string") return node.value;
  if (typeof node.value === "number" || typeof node.value === "boolean") return String(node.value);
  return undefined;
}

function stringList(node: unknown): string[] {
  if (!isSeq(node)) return [];
  return node.items.map(scalarString).filter((value): value is string => !!value);
}

function num(node: unknown): number | undefined {
  if (!isScalar(node) || typeof node.value !== "number") return undefined;
  return node.value;
}

function rangeOf(node: { range?: [number, number, number] | null }): Range {
  if (!node.range) return { start: 0, end: 0 };
  return { start: node.range[0], end: node.range[1] };
}

function valueRange(node: Scalar): Range {
  if (!node.range) return { start: 0, end: 0 };
  const [start, valueEnd] = node.range;
  if (node.type === Scalar.QUOTE_DOUBLE || node.type === Scalar.QUOTE_SINGLE) {
    return { start: start + 1, end: Math.max(start + 1, valueEnd - 1) };
  }
  return { start, end: valueEnd };
}
