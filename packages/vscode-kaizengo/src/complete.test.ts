import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { describe, it } from "node:test";
import { loadCatalog, type Catalog } from "./catalog";
import { complete, type Completion } from "./complete";

const repoRoot = path.resolve(__dirname, "../../..");
const catalog: Catalog = loadCatalog(repoRoot);

function labels(items: Completion[]): string[] {
  return items.map((item) => item.label);
}

function at(text: string, needle: string, which = 0): number {
  let from = 0;
  for (let n = 0; n <= which; n++) {
    const found = text.indexOf(needle, from);
    if (found < 0) throw new Error(`missing ${JSON.stringify(needle)}`);
    if (n === which) return found + needle.length;
    from = found + needle.length;
  }
  return 0;
}

function completeAt(file: string, text: string, needle: string, which = 0): Completion[] {
  return complete({ catalog, file, text, offset: at(text, needle, which) });
}

describe("yaml model completion", () => {
  const hello = catalog.apps.get("hellospec");
  const perms = catalog.apps.get("permissions");
  assert.ok(hello && perms);

  it("completes query and command model names", () => {
    const items = completeAt(hello.yamlPath, hello.text, "list: gre");
    assert.ok(labels(items).includes("greeting"));
    assert.equal(items.find((item) => item.label === "user"), undefined);
    const command = completeAt(hello.yamlPath, hello.text, "create: gre");
    assert.ok(labels(command).includes("greeting"));
  });

  it("completes search fields and field types", () => {
    const fields = completeAt(hello.yamlPath, hello.text, "fields: [mes");
    assert.ok(labels(fields).includes("message"));
    assert.ok(labels(fields).includes("mood"));
    assert.ok(labels(fields).includes("internalNote"));
    assert.equal(labels(fields).includes("email"), false);

    const types = completeAt(hello.yamlPath, hello.text, "type: str");
    assert.ok(labels(types).includes("string"));
    assert.ok(labels(types).includes("many2one"));
    assert.equal(types.find((item) => item.label === "enum")?.kind, "type");
  });

  it("completes relations, inverse fields, and enum defaults", () => {
    const relation = completeAt(perms.yamlPath, perms.text, "relation: ro");
    assert.ok(labels(relation).includes("role"));
    assert.ok(labels(relation).includes("identity.user"));
    assert.ok(labels(relation).includes("hellospec.greeting"));

    const text = `name: hellospec
title: HelloSpec
summary: demo
models:
  - name: greeting
    fields:
      - name: mood
        type: enum
        values: [happy, neutral]
        default: ha
      - name: lines
        type: one2many
        relation: identity.user
        inverse: na
`;
    const defaults = completeAt(hello.yamlPath, text, "default: ha");
    assert.deepEqual(labels(defaults), ["happy", "neutral"]);
    const inverse = completeAt(hello.yamlPath, text, "inverse: na");
    assert.ok(labels(inverse).includes("name"));
    assert.ok(labels(inverse).includes("email"));
    assert.equal(labels(inverse).includes("mood"), false);
  });

  it("completes security model resources, fields, and domain keys", () => {
    const file = path.join(path.dirname(hello.yamlPath), "security.yaml");
    const text = fs.readFileSync(file, "utf8");
    const resource = completeAt(file, text, "resource: hellospec.gre");
    assert.ok(labels(resource).includes("hellospec.greeting"));
    assert.equal(
      resource.find((item) => item.label.startsWith("hellospec.query")),
      undefined,
    );

    const fields = completeAt(file, text, "fields: [mes");
    assert.ok(labels(fields).includes("message"));
    assert.ok(labels(fields).includes("internalNote"));
    assert.ok(labels(fields).includes("authorId"));

    const domain = completeAt(file, text, '[[aut');
    assert.ok(labels(domain).includes("authorId"));
    assert.ok(labels(domain).includes("message"));
  });
});

describe("go model completion", () => {
  const seedPath = path.join(repoRoot, "apps/identity/seed.go");
  const seed = fs.readFileSync(seedPath, "utf8");

  it("completes model names on ModelRegistry calls", () => {
    const items = completeAt(seedPath, seed, 'ListAll(ctx, "org');
    assert.ok(labels(items).includes("organization"));
    assert.ok(labels(items).includes("user"));
    assert.equal(labels(items).includes("greeting"), false);
    const created = completeAt(seedPath, seed, 'seedAuthor, "org');
    assert.ok(labels(created).includes("membership"));
  });

  it("completes FindBy fields and Create map keys", () => {
    const email = completeAt(seedPath, seed, 'FindBy(ctx, "user", "em');
    assert.ok(labels(email).includes("email"));
    assert.ok(labels(email).includes("name"));
    assert.ok(labels(email).includes("id"));
    assert.equal(labels(email).includes("slug"), false);

    const key = completeAt(seedPath, seed, '"slu');
    assert.ok(labels(key).includes("slug"));
    assert.ok(labels(key).includes("name"));
    assert.equal(labels(key).includes("email"), false);
  });

  it("uses ModelsFromHost and separate field maps", () => {
    const file = path.join(repoRoot, "apps/auth/service.go");
    const text = `package auth

func lookup(host *module.Host) {
	users, err := engine.ModelsFromHost(host, "identity")
	rec, err := users.FindBy(ctx, "us", "email", email)
	_ = rec
}
`;
    const models = completeAt(file, text, 'FindBy(ctx, "us');
    assert.ok(labels(models).includes("user"));
    assert.equal(labels(models).includes("greeting"), false);

    const fields = `package identity

func create(m *engine.ModelRegistry) {
	fields := map[string]any{
		"na": "",
	}
	_, err := m.Create(ctx, orgID, author, "org_unit", fields)
	_ = err
}
`;
    const keys = completeAt(path.join(repoRoot, "apps/identity/seed.go"), fields, '"na');
    assert.ok(labels(keys).includes("name"));
    assert.ok(labels(keys).includes("parentId"));
    assert.equal(labels(keys).includes("email"), false);
  });

  it("completes record indexes and ignores unrelated strings", () => {
    const text = `package identity

func load(m *engine.ModelRegistry) {
	rec, err := m.FindBy(ctx, "user", "email", email)
	_ = rec["na"]
	fmt.Println("hello")
}
`;
    const file = path.join(repoRoot, "apps/identity/seed.go");
    const index = completeAt(file, text, 'rec["na');
    assert.ok(labels(index).includes("name"));
    assert.ok(labels(index).includes("email"));
    assert.deepEqual(completeAt(file, text, 'Println("hel'), []);
  });

  it("completes enum values in field maps", () => {
    const text = `package hellospec

func post(m *engine.ModelRegistry) {
	_, err := m.Create(ctx, org, author, "greeting", map[string]any{
		"mood": "ha",
	})
	_ = err
}
`;
    const items = completeAt(path.join(repoRoot, "apps/hellospec/module.go"), text, '"mood": "ha');
    assert.deepEqual(labels(items), ["happy", "neutral", "formal"]);
  });

  it("completes Hooks model names", () => {
    const file = path.join(repoRoot, "apps/hellospec/module.go");
    const text = fs.readFileSync(file, "utf8");
    const items = completeAt(file, text, 'Hooks("gre');
    assert.ok(labels(items).includes("greeting"));
  });
});

describe("tsx model completion", () => {
  const form = path.join(repoRoot, "apps/hellospec/views/GreetingForm.page.tsx");
  const formText = fs.readFileSync(form, "utf8");
  const list = path.join(repoRoot, "apps/hellospec/views/GreetingList.page.tsx");
  const listText = fs.readFileSync(list, "utf8");

  it("completes query and command refs", () => {
    const queries = completeAt(list, listText, 'query="hellospec.gre');
    assert.ok(labels(queries).includes("hellospec.greetings"));
    assert.ok(labels(queries).includes("identity.users"));
    assert.equal(queries.find((item) => item.label === "hellospec.greetings")?.detail, "list greeting");

    const commands = completeAt(form, formText, 'command="hellospec.post');
    assert.ok(labels(commands).includes("hellospec.postGreeting"));
    assert.equal(commands.find((item) => item.label === "hellospec.postGreeting")?.detail, "create greeting");
  });

  it("completes fields from the enclosing command or query", () => {
    const fields = completeAt(form, formText, 'field="mes');
    assert.ok(labels(fields).includes("message"));
    assert.ok(labels(fields).includes("mood"));
    assert.ok(labels(fields).includes("internalNote"));
    assert.equal(labels(fields).includes("email"), false);

    const group = completeAt(list, listText, 'groupByField="mo');
    assert.ok(labels(group).includes("mood"));
    assert.equal(labels(group).includes("slug"), false);

    const i18n = completeAt(form, formText, "t('hellospec.cre");
    assert.deepEqual(i18n, []);
  });

  it("reads a query that follows the field attribute", () => {
    const text = `export default function List() {
  return (
    <KCollection
      groupByField="mo"
      query="hellospec.greetings"
    />
  )
}
`;
    const items = completeAt(list, text, 'groupByField="mo');
    assert.ok(labels(items).includes("mood"));
  });

  it("completes record fields on card and keyOf callbacks", () => {
    const text = `export default function GreetingList() {
  return (
    <KCollection
      query="hellospec.greetings"
      keyOf={(greeting) => String(greeting.name ?? greeting.id)}
      card={(model) => {
        return (
          <h4>{model.message}</h4>
        )
      }}
    />
  )
}
`;
    const fields = completeAt(list, text, "model.mes");
    assert.ok(labels(fields).includes("message"));
    assert.ok(labels(fields).includes("mood"));
    assert.ok(labels(fields).includes("internalNote"));
    assert.ok(labels(fields).includes("id"));
    assert.equal(labels(fields).includes("email"), false);
    assert.equal(fields[0]?.replace.start, text.indexOf("model.mes") + "model.".length);

    const key = completeAt(list, text, "greeting.na");
    assert.ok(labels(key).includes("message"));
    assert.equal(labels(key).includes("email"), false);

    const dot = completeAt(list, text, "model.");
    assert.ok(labels(dot).includes("message"));

    assert.deepEqual(completeAt(list, text + '\nconst label = "model.mes"\n', 'const label = "model.mes'), []);
  });
});
