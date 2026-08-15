import fs from "node:fs";
import path from "node:path";
import type { AppIndex, Catalog, SpecDoc } from "./catalog";
import { classify, sibling, type YamlSymbol } from "./parse";

export interface NavTarget {
  file: string;
  start: number;
  end: number;
  tooltip: string;
}

export interface ResolveContext {
  catalog: Catalog;
  appName: string;
  file: string;
  text: string;
  symbols: YamlSymbol[];
}

const NAME_RE = /^[a-z][a-z0-9_]*$/;

const PLATFORM_CAPS: Record<string, string> = {
  "platform.i18n": "internal/platform/i18n/i18n.go",
  "platform.time": "internal/platform/time/calendar.go",
  "platform.config": "internal/platform/config/config.go",
};

const SKIP_DIRS = new Set([
  "node_modules",
  "dist",
  "__types__",
  "locale",
  "migrations",
  "views",
  "lib",
  "spa",
]);

export function parseRelation(
  rel: string,
  fromApp: string,
): { app: string; model: string } | undefined {
  const t = rel.trim();
  if (!t) return undefined;
  const dot = t.indexOf(".");
  if (dot < 0) {
    if (!NAME_RE.test(t)) return undefined;
    return { app: fromApp, model: t };
  }
  const app = t.slice(0, dot);
  const model = t.slice(dot + 1);
  if (!NAME_RE.test(app) || !NAME_RE.test(model)) return undefined;
  return { app, model };
}

function liveApp(ctx: ResolveContext): AppIndex {
  const disk = ctx.catalog.apps.get(ctx.appName);
  const liveDoc: SpecDoc = { path: ctx.file, text: ctx.text, symbols: ctx.symbols };
  const docs = (disk?.docs ?? []).map((d) => (d.path === ctx.file ? liveDoc : d));
  if (!docs.some((d) => d.path === ctx.file)) docs.push(liveDoc);
  return {
    name: ctx.appName,
    dir: disk?.dir ?? appDirFromFile(ctx.file),
    yamlPath: disk?.yamlPath ?? ctx.file,
    text: disk?.text ?? ctx.text,
    symbols: disk?.symbols ?? ctx.symbols,
    docs,
  };
}

function appDirFromFile(file: string): string {
  const parts = path.normalize(file).split(path.sep);
  const i = parts.lastIndexOf("apps");
  if (i >= 0 && parts[i + 1]) return parts.slice(0, i + 2).join(path.sep);
  return path.dirname(file);
}

function appDocs(app: AppIndex): SpecDoc[] {
  if (app.docs?.length) return app.docs;
  return [{ path: app.yamlPath, text: app.text, symbols: app.symbols }];
}

function appOf(ctx: ResolveContext, name: string): AppIndex | undefined {
  if (name === ctx.appName) return liveApp(ctx);
  return ctx.catalog.apps.get(name);
}

function allApps(ctx: ResolveContext): AppIndex[] {
  const live = liveApp(ctx);
  const out: AppIndex[] = [live];
  for (const [name, app] of ctx.catalog.apps) {
    if (name !== live.name) out.push(app);
  }
  return out;
}

function existing(file: string): string | undefined {
  return fs.existsSync(file) ? file : undefined;
}

function needleTarget(file: string, tooltip: string, needle?: string): NavTarget | undefined {
  if (!existing(file)) return undefined;
  if (!needle) return { file, start: 0, end: 0, tooltip };
  const text = fs.readFileSync(file, "utf8");
  const idx = text.indexOf(needle);
  if (idx < 0) return { file, start: 0, end: 0, tooltip };
  return { file, start: idx, end: idx + needle.length, tooltip };
}

function yamlHit(app: AppIndex, pred: (s: YamlSymbol) => boolean, tooltip: string): NavTarget | undefined {
  for (const doc of appDocs(app)) {
    const s = doc.symbols.find(pred);
    if (s) return { file: doc.path, start: s.range.start, end: s.range.end, tooltip };
  }
  return needleTarget(app.yamlPath, tooltip);
}

function appNameHit(app: AppIndex): NavTarget {
  return (
    yamlHit(app, (s) => s.path === "name", `${app.name}/app.yaml`) ?? {
      file: app.yamlPath,
      start: 0,
      end: 0,
      tooltip: `${app.name}/app.yaml`,
    }
  );
}

function viewSvelte(app: AppIndex, viewName: string): NavTarget | undefined {
  return (
    needleTarget(path.join(app.dir, "views", `${viewName}.page.svelte`), `views/${viewName}.page.svelte`) ??
    needleTarget(path.join(app.dir, "views", `${viewName}.svelte`), `views/${viewName}.svelte`)
  );
}

function viewYaml(app: AppIndex, viewName: string): NavTarget | undefined {
  return yamlHit(
    app,
    (s) => classify(s.path) === "view-name" && s.value === viewName,
    `view ${viewName}`,
  );
}

function pascal(name: string): string {
  return name
    .split("_")
    .filter(Boolean)
    .map((p) => p[0].toUpperCase() + p.slice(1))
    .join("");
}

function modelYaml(app: AppIndex, model: string): NavTarget | undefined {
  return yamlHit(
    app,
    (s) => classify(s.path) === "model-name" && s.value === model,
    `model ${model}`,
  );
}

function modelGo(app: AppIndex, model: string): NavTarget | undefined {
  const file = path.join(app.dir, "__types__", `${model}.go`);
  return needleTarget(file, `__types__/${model}.go`, `type ${pascal(model)} struct`);
}

function fieldName(app: AppIndex, model: string, field: string): NavTarget | undefined {
  for (const doc of appDocs(app)) {
    const modelSym = doc.symbols.find((s) => classify(s.path) === "model-name" && s.value === model);
    if (!modelSym) continue;
    const prefix = modelSym.path.replace(/\.name$/, "");
    const s = doc.symbols.find(
      (sym) => sym.path.startsWith(`${prefix}.fields[`) && sym.path.endsWith(".name") && sym.value === field,
    );
    if (s) return { file: doc.path, start: s.range.start, end: s.range.end, tooltip: `${model}.${field}` };
  }
  return undefined;
}

function providesHit(ctx: ResolveContext, cap: string): NavTarget[] {
  const platform = PLATFORM_CAPS[cap];
  if (platform) {
    const hit = needleTarget(path.join(ctx.catalog.root, platform), cap);
    return hit ? [hit] : [];
  }
  const out: NavTarget[] = [];
  for (const app of allApps(ctx)) {
    const hit = yamlHit(
      app,
      (s) => classify(s.path) === "provides" && s.value === cap,
      `${app.name} provides ${cap}`,
    );
    if (hit && app.symbols.some((s) => classify(s.path) === "provides" && s.value === cap)) {
      out.push(hit);
    }
  }
  return out;
}

function localeFiles(ctx: ResolveContext, app: AppIndex): string[] {
  const files: string[] = [];
  const dirs = [
    path.join(app.dir, "locale"),
    path.join(ctx.catalog.root, "internal", "platform", "i18n", "locale"),
  ];
  for (const dir of dirs) {
    if (!fs.existsSync(dir)) continue;
    const names = fs.readdirSync(dir).filter((n) => n.endsWith(".po")).sort((a, b) => {
      if (a === "en.po") return -1;
      if (b === "en.po") return 1;
      return a.localeCompare(b);
    });
    for (const n of names) files.push(path.join(dir, n));
  }
  return files;
}

function findMsgid(ctx: ResolveContext, key: string): NavTarget[] {
  const needle = `msgid "${key}"`;
  const out: NavTarget[] = [];
  for (const file of localeFiles(ctx, liveApp(ctx))) {
    const text = fs.readFileSync(file, "utf8");
    const idx = text.indexOf(needle);
    if (idx < 0) continue;
    out.push({
      file,
      start: idx,
      end: idx + needle.length,
      tooltip: `${key} (${path.basename(file)})`,
    });
  }
  return out;
}

function walkGoFiles(dir: string): string[] {
  const out: string[] = [];
  const rec = (d: string) => {
    let entries: fs.Dirent[];
    try {
      entries = fs.readdirSync(d, { withFileTypes: true });
    } catch {
      return;
    }
    for (const e of entries) {
      if (e.isDirectory()) {
        if (SKIP_DIRS.has(e.name) || e.name.startsWith(".")) continue;
        rec(path.join(d, e.name));
      } else if (e.name.endsWith(".go")) {
        out.push(path.join(d, e.name));
      }
    }
  };
  rec(dir);
  return out;
}

function findHandler(appDir: string, handler: string): NavTarget[] {
  const named = `RegisterNamed("${handler}"`;
  const fn = `func ${handler}(`;
  const namedHits: NavTarget[] = [];
  const fnHits: NavTarget[] = [];
  for (const file of walkGoFiles(appDir)) {
    const text = fs.readFileSync(file, "utf8");
    const ni = text.indexOf(named);
    if (ni >= 0) {
      namedHits.push({
        file,
        start: ni,
        end: ni + named.length,
        tooltip: `RegisterNamed("${handler}")`,
      });
    }
    const fi = text.indexOf(fn);
    if (fi >= 0) {
      fnHits.push({ file, start: fi, end: fi + fn.length - 1, tooltip: `func ${handler}` });
    }
  }
  return namedHits.length ? namedHits : fnHits;
}

function resolveModule(root: string, module: string): NavTarget | undefined {
  let file: string | undefined;
  if (module.startsWith("@kaizengo/sdk-svelte/")) {
    file = existing(path.join(root, "packages/sdk-svelte", module.slice("@kaizengo/sdk-svelte/".length)));
  } else if (module.startsWith("@apps/")) {
    file = existing(path.join(root, "apps", module.slice("@apps/".length)));
  } else {
    file = existing(path.join(root, module));
  }
  return file ? needleTarget(file, module) : undefined;
}

function findComponent(ctx: ResolveContext, id: string): NavTarget | undefined {
  for (const app of allApps(ctx)) {
    const hit = yamlHit(
      app,
      (s) => classify(s.path) === "component-id" && s.value === id,
      `component ${id}`,
    );
    if (hit && app.symbols.some((s) => classify(s.path) === "component-id" && s.value === id)) {
      return hit;
    }
  }
  return undefined;
}

function unique(targets: Array<NavTarget | undefined>): NavTarget[] {
  const out: NavTarget[] = [];
  const seen = new Set<string>();
  for (const t of targets) {
    if (!t) continue;
    const key = `${t.file}:${t.start}:${t.end}`;
    if (seen.has(key)) continue;
    seen.add(key);
    out.push(t);
  }
  return out;
}

export function resolveDefinition(ctx: ResolveContext, symbol: YamlSymbol): NavTarget[] {
  const kind = classify(symbol.path);
  const value = symbol.value;
  const app = liveApp(ctx);

  switch (kind) {
    case "app-name":
      return unique([needleTarget(path.join(app.dir, "module.go"), "module.go", "func init()")]);
    case "depends": {
      const dep = appOf(ctx, value);
      return dep ? [appNameHit(dep)] : [];
    }
    case "uses":
      return providesHit(ctx, value);
    case "provides":
      return [];
    case "view-name":
    case "export-view-ref":
      return unique([viewSvelte(app, value)]);
    case "spec-include": {
      const target = path.resolve(path.dirname(ctx.file), value);
      const file = fs.existsSync(target) && fs.statSync(target).isDirectory()
        ? ["spec.yaml", "spec.yml", `${path.basename(target)}.yaml`]
            .map((n) => path.join(target, n))
            .find((p) => fs.existsSync(p))
        : existing(target);
      return unique([file ? needleTarget(file, value) : undefined]);
    }
    case "menu-view":
      return unique([viewSvelte(app, value), viewYaml(app, value)]);
    case "model-name":
      return unique([modelGo(app, value)]);
    case "relation": {
      const ref = parseRelation(value, ctx.appName);
      if (!ref) return [];
      const target = appOf(ctx, ref.app);
      if (!target) return [];
      return unique([modelYaml(target, ref.model), modelGo(target, ref.model)]);
    }
    case "inverse": {
      const rel = sibling(ctx.symbols, symbol.path, "relation");
      if (!rel) return [];
      const ref = parseRelation(rel.value, ctx.appName);
      if (!ref) return [];
      const target = appOf(ctx, ref.app);
      if (!target) return [];
      return unique([fieldName(target, ref.model, value)]);
    }
    case "search-field": {
      const prefix = symbol.path.match(/^(models\[\d+\])/)?.[1];
      if (!prefix) return [];
      const model = ctx.symbols.find((s) => s.path === `${prefix}.name`)?.value;
      if (!model) return [];
      return unique([fieldName(app, model, value)]);
    }
    case "search-collection": {
      const ref = parseRelation(value, ctx.appName);
      if (!ref) return [];
      const target = appOf(ctx, ref.app);
      if (!target) return [];
      return unique([modelYaml(target, ref.model)]);
    }
    case "label-key":
      return findMsgid(ctx, value);
    case "locale-id":
      return unique([needleTarget(path.join(app.dir, "locale", `${value}.po`), `locale/${value}.po`)]);
    case "extend-handler":
      return findHandler(app.dir, value);
    case "component-id": {
      const mod = sibling(ctx.symbols, symbol.path, "module");
      return unique([mod ? resolveModule(ctx.catalog.root, mod.value) : undefined]);
    }
    case "component-module":
      return unique([resolveModule(ctx.catalog.root, value)]);
    case "export-app": {
      const target = appOf(ctx, value);
      return target ? [appNameHit(target)] : [];
    }
    case "export-component-ref":
      return unique([findComponent(ctx, value)]);
    case "none":
      return [];
  }
}

function hitsMatching(
  ctx: ResolveContext,
  pred: (s: YamlSymbol, app: AppIndex) => boolean,
): NavTarget[] {
  const out: NavTarget[] = [];
  for (const app of allApps(ctx)) {
    for (const doc of app.docs ?? [{ path: app.yamlPath, text: app.text, symbols: app.symbols }]) {
      for (const s of doc.symbols) {
        if (!pred(s, app)) continue;
        out.push({
          file: doc.path,
          start: s.range.start,
          end: s.range.end,
          tooltip: `${app.name} ${s.path}`,
        });
      }
    }
  }
  return unique(out);
}

export function resolveReferences(ctx: ResolveContext, symbol: YamlSymbol): NavTarget[] {
  const kind = classify(symbol.path);
  const value = symbol.value;

  switch (kind) {
    case "provides":
    case "uses":
      return hitsMatching(
        ctx,
        (s) => (classify(s.path) === "uses" || classify(s.path) === "provides") && s.value === value,
      );
    case "view-name":
    case "menu-view":
    case "export-view-ref":
      return hitsMatching(ctx, (s, app) => {
        if (app.name !== ctx.appName) return false;
        const k = classify(s.path);
        return (k === "view-name" || k === "menu-view" || k === "export-view-ref") && s.value === value;
      });
    case "model-name":
    case "relation":
    case "search-collection": {
      const ref =
        kind === "model-name" ? { app: ctx.appName, model: value } : parseRelation(value, ctx.appName);
      if (!ref) return [];
      return hitsMatching(ctx, (s, app) => {
        const k = classify(s.path);
        if (k === "model-name") return app.name === ref.app && s.value === ref.model;
        if (k === "relation" || k === "search-collection") {
          const r = parseRelation(s.value, app.name);
          return !!r && r.app === ref.app && r.model === ref.model;
        }
        return false;
      });
    }
    case "component-id":
    case "export-component-ref":
      return hitsMatching(ctx, (s) => {
        const k = classify(s.path);
        return (k === "component-id" || k === "export-component-ref") && s.value === value;
      });
    case "label-key":
      return hitsMatching(ctx, (s) => classify(s.path) === "label-key" && s.value === value);
    case "depends":
    case "export-app":
    case "app-name":
      return hitsMatching(
        ctx,
        (s) =>
          (classify(s.path) === "depends" || classify(s.path) === "export-app" || s.path === "name") &&
          s.value === value,
      );
    default:
      return [];
  }
}
