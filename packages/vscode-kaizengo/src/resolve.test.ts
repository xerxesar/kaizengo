import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { describe, it } from "node:test";
import { loadCatalog } from "./catalog";
import { offsetToPosition, positionToOffset } from "./offsets";
import { classify, collectSymbols, symbolAt } from "./parse";
import { parseRelation, resolveDefinition, resolveReferences, type ResolveContext } from "./resolve";

const repoRoot = path.resolve(__dirname, "../../..");

function ctxFor(appName: string): ResolveContext {
  const catalog = loadCatalog(repoRoot);
  const app = catalog.apps.get(appName);
  assert.ok(app, `app ${appName} not found`);
  return {
    catalog,
    appName,
    file: app.yamlPath,
    text: app.text,
    symbols: app.symbols,
  };
}

function atValue(ctx: ResolveContext, path: string, value?: string) {
  const sym = ctx.symbols.find((s) => s.path === path && (value === undefined || s.value === value));
  assert.ok(sym, `missing symbol ${path}${value ? `=${value}` : ""}`);
  return sym;
}

function filesOf(ctx: ResolveContext, path: string, value?: string): string[] {
  return resolveDefinition(ctx, atValue(ctx, path, value)).map((t) => t.file);
}

describe("offsets", () => {
  it("round-trips line/character", () => {
    const text = "a\nbc\n";
    const off = positionToOffset(text, 1, 1);
    assert.equal(off, 3);
    assert.deepEqual(offsetToPosition(text, off), { line: 1, character: 1 });
  });
});

describe("classify", () => {
  it("maps app.yaml paths", () => {
    assert.equal(classify("depends[0]"), "depends");
    assert.equal(classify("uses[1]"), "uses");
    assert.equal(classify("models[0]"), "spec-include");
    assert.equal(classify("views[0].name"), "view-name");
    assert.equal(classify("views[2].path"), "view-path");
    assert.equal(classify("views[0].components[0].model"), "view-model");
    assert.equal(classify("views[0].components[0].id"), "screen-component");
    assert.equal(classify("views[1].layout.items[0].items[0].component"), "layout-component");
    assert.equal(classify("views[0].components[0].columns[0].field"), "view-field");
    assert.equal(classify("views[1].components[1].groups[0].fields[0].field"), "view-field");
    assert.equal(classify("views[0].components[0].groupBy[0]"), "view-field");
    assert.equal(classify("menus[1].children[0].view"), "menu-view");
    assert.equal(classify("models[0].fields[2].relation"), "relation");
    assert.equal(classify("nav.labelKey"), "label-key");
    assert.equal(classify("exports.menus[0].app"), "export-app");
    assert.equal(classify("exports.menus[0].view"), "export-view-ref");
    assert.equal(classify("title"), "none");
  });
});

describe("parse permissions specs", () => {
  it("collects depends, models, and relations", () => {
    const text = fs.readFileSync(path.join(repoRoot, "apps/permissions/app.yaml"), "utf8");
    const symbols = collectSymbols(text);
    assert.equal(symbols.find((s) => s.path === "name")?.value, "permissions");
    assert.ok(symbols.some((s) => s.path === "depends[0]" && s.value === "core"));
    assert.ok(symbols.some((s) => classify(s.path) === "model-name" && s.value === "role"));
    const rel = symbols.find((s) => classify(s.path) === "relation" && s.value === "role");
    assert.ok(rel);
    assert.equal(symbolAt(symbols, rel.range.start)?.value, "role");
  });

  it("indexes each app.yaml", () => {
    const catalog = loadCatalog(repoRoot);
    const app = catalog.apps.get("permissions");
    assert.ok(app);
    assert.ok(app.docs.some((d) => d.path.endsWith("app.yaml")));
    assert.ok(app.symbols.some((s) => classify(s.path) === "model-name" && s.value === "acl_entry"));
  });
});

describe("parseRelation", () => {
  it("defaults to the declaring app", () => {
    assert.deepEqual(parseRelation("uom", "inventory"), { app: "inventory", model: "uom" });
    assert.deepEqual(parseRelation("identity.user", "inventory"), { app: "identity", model: "user" });
  });
});

describe("resolveDefinition", () => {
  it("jumps depends to the target app.yaml", () => {
    const ctx = ctxFor("permissions");
    const files = filesOf(ctx, "depends[0]", "core");
    assert.ok(files.some((f) => f.endsWith(path.join("apps", "core", "app.yaml"))));
  });

  it("jumps uses to provides and platform sources", () => {
    const ctx = ctxFor("hellospec");
    const rbac = filesOf(ctx, "uses[1]", "permissions.rbac");
    assert.ok(rbac.some((f) => f.endsWith(path.join("apps", "permissions", "app.yaml"))));

    const core = ctxFor("core");
    const i18n = filesOf(core, "uses[0]", "platform.i18n");
    assert.ok(i18n.some((f) => f.endsWith(path.join("internal", "platform", "i18n", "i18n.go"))));
  });

  it("jumps menus to their pages", () => {
    const ctx = ctxFor("identity");
    const menu = filesOf(ctx, "menus[0].view", "Overview");
    assert.ok(menu.some((f) => f.endsWith(path.join("apps", "identity", "views", "Overview.page.tsx"))));
  });

  it("jumps relation to the related model", () => {
    const ctx = ctxFor("permissions");
    const rel = ctx.symbols.find((s) => classify(s.path) === "relation" && s.value === "role");
    assert.ok(rel);
    const hits = resolveDefinition(ctx, rel);
    assert.ok(hits.some((t) => t.file === ctx.file && ctx.text.slice(t.start, t.end) === "role" && t.start !== rel.range.start));
    assert.ok(hits.some((f) => f.file.endsWith(path.join("__types__", "role.go"))));
  });

  it("jumps model names to generated types", () => {
    const ctx = ctxFor("hellospec");
    const model = ctx.symbols.find((s) => classify(s.path) === "model-name" && s.value === "greeting");
    assert.ok(model);
    const files = resolveDefinition(ctx, model).map((t) => t.file);
    assert.ok(files.some((f) => f.endsWith(path.join("apps", "hellospec", "__types__", "greeting.go"))));
  });

  it("jumps labelKey and locale id into .po catalogs", () => {
    const ctx = ctxFor("hellospec");
    const label = filesOf(ctx, "nav.labelKey", "nav.hellospec");
    assert.ok(label.some((f) => f.includes(path.join("platform", "i18n", "locale"))));

    const locale = filesOf(ctx, "locales[0].id", "en");
    assert.ok(locale.some((f) => f.endsWith(path.join("apps", "hellospec", "locale", "en.po"))));
  });

  it("jumps extend handlers and exported modules", () => {
    const ctx = ctxFor("typesense");
    const handler = filesOf(ctx, "extends[0].handler", "indexDocument");
    assert.ok(handler.some((f) => f.endsWith(path.join("apps", "typesense", "module.go"))));

    const mod = filesOf(ctx, "exports.components[0].module");
    assert.ok(mod.some((f) => f.endsWith(path.join("apps", "core", "spa", "src", "components", "shell", "SearchBar.tsx"))));

    const settings = filesOf(ctx, "exports.components[1].module");
    assert.ok(settings.some((f) => f.endsWith(path.join("apps", "typesense", "views", "SearchSettings.page.tsx"))));

    const app = filesOf(ctx, "exports.views[0].app", "hellospec");
    assert.ok(app.some((f) => f.endsWith(path.join("apps", "hellospec", "app.yaml"))));
  });

  it("jumps app name to module.go", () => {
    const ctx = ctxFor("hellospec");
    const files = filesOf(ctx, "name", "hellospec");
    assert.ok(files.some((f) => f.endsWith(path.join("apps", "hellospec", "module.go"))));
  });

  it("jumps view components, fields, and page paths", () => {
    const ctx = ctxFor("hellospec");
    const model = filesOf(ctx, "views[0].components[0].model", "greeting");
    assert.ok(model.some((f) => f.endsWith(path.join("__types__", "greeting.go"))));
    assert.ok(model.some((f) => f.endsWith(path.join("apps", "hellospec", "app.yaml"))));

    const field = ctx.symbols.find((s) => s.path === "views[0].components[0].columns[0].field");
    assert.ok(field);
    const fieldHits = resolveDefinition(ctx, field);
    assert.equal(ctx.text.slice(fieldHits[0].start, fieldHits[0].end), "message");
    assert.notEqual(fieldHits[0].start, field.range.start);

    const layout = ctx.symbols.find((s) => classify(s.path) === "layout-component" && s.value === "form");
    assert.ok(layout);
    const id = ctx.symbols.find((s) => classify(s.path) === "screen-component" && s.value === "form");
    assert.ok(id);
    assert.equal(resolveDefinition(ctx, layout)[0].start, id.range.start);

    const page = filesOf(ctx, "views[2].path", "views/GreetingBoard.page.tsx");
    assert.ok(page.some((f) => f.endsWith(path.join("views", "GreetingBoard.page.tsx"))));
  });
});

describe("resolveReferences", () => {
  it("finds uses of a provided capability", () => {
    const ctx = ctxFor("permissions");
    const cap = ctx.symbols.find((s) => classify(s.path) === "provides" && s.value === "permissions.rbac");
    assert.ok(cap);
    const refs = resolveReferences(ctx, cap);
    assert.ok(refs.some((t) => t.file.endsWith(path.join("apps", "hellospec", "app.yaml"))));
    assert.ok(refs.some((t) => t.file.endsWith(path.join("apps", "appman", "app.yaml"))));
  });

  it("finds relation fields that point at a model", () => {
    const ctx = ctxFor("permissions");
    const model = ctx.symbols.find((s) => classify(s.path) === "model-name" && s.value === "role");
    assert.ok(model);
    const refs = resolveReferences(ctx, model);
    assert.ok(refs.length > 1);
    assert.ok(refs.some((t) => classify(ctx.symbols.find((s) => s.range.start === t.start)?.path ?? "") === "relation"));
  });

  it("finds view models and layout components", () => {
    const ctx = ctxFor("hellospec");
    const model = ctx.symbols.find((s) => classify(s.path) === "model-name" && s.value === "greeting");
    assert.ok(model);
    const refs = resolveReferences(ctx, model);
    assert.ok(refs.some((t) => ctx.text.slice(t.start, t.end) === "greeting" && t.start !== model.range.start));

    const layout = ctx.symbols.find((s) => classify(s.path) === "layout-component" && s.value === "recent");
    assert.ok(layout);
    const uses = resolveReferences(ctx, layout);
    assert.ok(uses.some((t) => classify(ctx.symbols.find((s) => s.range.start === t.start)?.path ?? "") === "screen-component"));
    assert.ok(uses.some((t) => t.start === layout.range.start));
  });
});
