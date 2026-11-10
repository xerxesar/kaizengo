import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { describe, it } from "node:test";
import { loadCatalog } from "./catalog";
import { parseSpec } from "./view-model";

const repoRoot = path.resolve(__dirname, "../../..");

describe("parseSpec", () => {
  const helloPath = path.join(repoRoot, "apps/hellospec/app.yaml");
  const hello = fs.readFileSync(helloPath, "utf8");

  it("builds the hellospec view tree", () => {
    const spec = parseSpec(hello);
    assert.equal(spec.error, undefined);
    assert.deepEqual(spec.issues, []);
    assert.equal(spec.name, "hellospec");
    assert.deepEqual(
      spec.views.slice(0, 3).map((view) => view.name),
      ["GreetingList", "GreetingForm", "GreetingBoard"],
    );

    const list = spec.views[0];
    assert.equal(list.layout.direction, "column");
    assert.equal(list.layout.items[0]?.componentId, "list");
    const columns = list.components[0]?.columns.map((column) => column.name) ?? [];
    assert.equal(columns[0], "message");
    assert.ok(columns.includes("mood"));
    assert.ok(columns.includes("updatedAt"));
    assert.ok(list.layout.directionRange);
    assert.equal(hello.slice(list.layout.directionRange.start, list.layout.directionRange.end), "column");

    const form = spec.views[1];
    assert.equal(form.layout.direction, "column");
    const row = form.layout.items[0];
    assert.equal(row?.direction, "row");
    assert.equal(row?.items[0]?.componentId, "form");
    assert.equal(row?.items[0]?.span, 2);
    assert.equal(row?.items[1]?.componentId, "recent");
    assert.equal(row?.items[1]?.size, "18rem");
    assert.equal(form.components.find((c) => c.id === "form")?.groups[0]?.fields[0]?.span, 2);

    const board = spec.views[2];
    assert.equal(board.path, "views/GreetingBoard.page.tsx");
    assert.equal(board.layout.items[0]?.size, "auto");
    assert.equal(board.layout.items[1]?.size, "1fr");
    assert.equal(board.components.find((c) => c.id === "board")?.filters[0]?.id, "happy");

    assert.equal(spec.queries[0]?.detail, "list greeting");
    assert.equal(spec.commands[0]?.detail, "create greeting");
    assert.equal(spec.menus[0]?.children[0]?.children[0]?.detail, "GreetingList");
    assert.deepEqual(
      spec.depends.map((item) => item.label),
      ["core", "identity", "auth", "permissions"],
    );
  });

  it("accepts every app spec", () => {
    const catalog = loadCatalog(repoRoot);
    for (const app of catalog.apps.values()) {
      const spec = parseSpec(app.text);
      assert.equal(spec.error, undefined, app.name);
      assert.deepEqual(
        spec.issues,
        [],
        `${app.name}: ${spec.issues.map((issue) => issue.message).join("; ")}`,
      );
    }
  });

  it("reports broken layouts", () => {
    const spec = parseSpec(`name: demo
title: Demo
summary: demo
models:
  - name: greeting
    fields:
      - name: message
        type: string
views:
  - name: Board
    layout:
      direction: sideways
      items:
        - component: missing
        - component: list
        - component: list
    components:
      - id: list
        kind: grid
        model: other
        columns:
          - field: nope
            align: top
  - name: Board
    layout: column
    components:
      - id: list
        kind: collection
        model: greeting
        columns:
          - field: nope
`);
    const messages = spec.issues.map((issue) => issue.message);
    assert.ok(messages.includes('Invalid layout direction "sideways"'));
    assert.ok(messages.includes('Unknown component "missing"'));
    assert.ok(messages.includes('Component "list" is repeated in the layout'));
    assert.ok(messages.includes('Unknown kind "grid"'));
    assert.ok(messages.includes('Unknown model "other"'));
    assert.ok(messages.includes('Invalid align "top"'));
    assert.ok(messages.includes('Duplicate view "Board"'));
    assert.ok(messages.includes('Unknown field "nope"'));
  });

  it("keeps a shorthand direction editable", () => {
    const text = `name: demo
title: Demo
summary: demo
models:
  - name: greeting
    fields:
      - name: message
        type: string
views:
  - name: List
    layout: row
    components:
      - id: list
        kind: table
        model: greeting
        span: 2
        columns:
          - field: message
`;
    const spec = parseSpec(text);
    assert.deepEqual(spec.issues, []);
    const layout = spec.views[0]?.layout;
    assert.equal(layout?.direction, "row");
    assert.equal(layout?.items[0]?.componentId, "list");
    assert.equal(layout?.items[0]?.span, 2);
    assert.ok(layout?.directionRange);
    assert.equal(text.slice(layout.directionRange.start, layout.directionRange.end), "row");
  });
});
