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

function ctxForDoc(appName: string, suffix: string): ResolveContext {
  const catalog = loadCatalog(repoRoot);
  const app = catalog.apps.get(appName);
  assert.ok(app, `app ${appName} not found`);
  const doc = app.docs.find((d) => d.path.endsWith(suffix));
  assert.ok(doc, `doc ${suffix} not found in ${appName}`);
  return {
    catalog,
    appName,
    file: doc.path,
    text: doc.text,
    symbols: doc.symbols,
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
    assert.equal(classify("menus[1].children[0].view"), "menu-view");
    assert.equal(classify("models[0].fields[2].relation"), "relation");
    assert.equal(classify("nav.labelKey"), "label-key");
    assert.equal(classify("exports.menus[0].app"), "export-app");
    assert.equal(classify("exports.menus[0].view"), "export-view-ref");
    assert.equal(classify("title"), "none");
  });
});

describe("parse inventory specs", () => {
  it("collects depends, model includes, and menus", () => {
    const text = fs.readFileSync(path.join(repoRoot, "apps/inventory/app.yaml"), "utf8");
    const symbols = collectSymbols(text);
    assert.equal(symbols.find((s) => s.path === "name")?.value, "inventory");
    assert.ok(symbols.some((s) => s.path === "depends[0]" && s.value === "core"));
    assert.ok(symbols.some((s) => classify(s.path) === "spec-include" && s.value === "models/product"));
    const dash = symbols.find((s) => classify(s.path) === "menu-view" && s.value === "Dashboard");
    assert.ok(dash);
    assert.equal(symbolAt(symbols, dash.range.start)?.value, "Dashboard");
  });

  it("indexes model spec.yaml files", () => {
    const catalog = loadCatalog(repoRoot);
    const app = catalog.apps.get("inventory");
    assert.ok(app);
    assert.ok(app.docs.some((d) => d.path.endsWith(path.join("models", "product", "spec.yaml"))));
    const product = app.docs.find((d) => d.path.endsWith(path.join("models", "product", "spec.yaml")));
    assert.ok(product?.symbols.some((s) => classify(s.path) === "model-name" && s.value === "product"));
    assert.ok(product?.symbols.some((s) => classify(s.path) === "relation" && s.value === "product_variant"));
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
    const ctx = ctxFor("inventory");
    const files = filesOf(ctx, "depends[0]", "core");
    assert.ok(files.some((f) => f.endsWith(path.join("apps", "core", "app.yaml"))));
  });

  it("jumps uses to provides and platform sources", () => {
    const ctx = ctxFor("inventory");
    const rbac = filesOf(ctx, "uses[1]", "permissions.rbac");
    assert.ok(rbac.some((f) => f.endsWith(path.join("apps", "permissions", "app.yaml"))));

    const core = ctxFor("core");
    const i18n = filesOf(core, "uses[0]", "platform.i18n");
    assert.ok(i18n.some((f) => f.endsWith(path.join("internal", "platform", "i18n", "i18n.go"))));
  });

  it("jumps menus and model includes to their files", () => {
    const ctx = ctxFor("inventory");
    const menu = filesOf(ctx, "menus[0].view", "Dashboard");
    assert.ok(menu.some((f) => f.endsWith(path.join("apps", "inventory", "views", "Dashboard.page.tsx"))));

    const spec = filesOf(ctx, "models[4]", "models/product");
    assert.ok(spec.some((f) => f.endsWith(path.join("apps", "inventory", "models", "product", "spec.yaml"))));
  });

  it("jumps relation and inverse to the related model field", () => {
    const ctx = ctxForDoc("inventory", path.join("models", "uom_group", "spec.yaml"));
    const rel = ctx.symbols.find((s) => classify(s.path) === "relation" && s.value === "uom");
    assert.ok(rel);
    const relFiles = resolveDefinition(ctx, rel).map((t) => t.file);
    assert.ok(relFiles.some((f) => f.endsWith(path.join("models", "uom", "spec.yaml"))));

    const inv = ctx.symbols.find((s) => classify(s.path) === "inverse" && s.value === "groupId");
    assert.ok(inv);
    const invHits = resolveDefinition(ctx, inv);
    assert.ok(invHits.length >= 1);
    assert.ok(invHits[0].file.endsWith(path.join("models", "uom", "spec.yaml")));
    const uom = fs.readFileSync(invHits[0].file, "utf8");
    assert.equal(uom.slice(invHits[0].start, invHits[0].end), "groupId");
  });

  it("jumps model names to generated types", () => {
    const ctx = ctxFor("hellospec");
    const files = filesOf(ctx, "models[0].name", "greeting");
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
    assert.ok(mod.some((f) => f.endsWith(path.join("packages", "sdk-solid", "search", "SearchBar.tsx"))));

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
});

describe("resolveReferences", () => {
  it("finds uses of a provided capability", () => {
    const ctx = ctxFor("permissions");
    const cap = ctx.symbols.find((s) => classify(s.path) === "provides" && s.value === "permissions.rbac");
    assert.ok(cap);
    const refs = resolveReferences(ctx, cap);
    assert.ok(refs.some((t) => t.file.endsWith(path.join("apps", "inventory", "app.yaml"))));
    assert.ok(refs.some((t) => t.file.endsWith(path.join("apps", "hellospec", "app.yaml"))));
  });

  it("finds relation fields that point at a model", () => {
    const ctx = ctxForDoc("inventory", path.join("models", "uom", "spec.yaml"));
    const model = ctx.symbols.find((s) => classify(s.path) === "model-name" && s.value === "uom");
    assert.ok(model);
    const refs = resolveReferences(ctx, model);
    assert.ok(refs.length > 1);
    assert.ok(refs.some((t) => t.file !== ctx.file || t.start !== model.range.start));
  });
});
