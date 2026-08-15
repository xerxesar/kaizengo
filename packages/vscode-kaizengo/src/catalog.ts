import fs from "node:fs";
import path from "node:path";
import { collectSymbols, type YamlSymbol } from "./parse";

export interface AppIndex {
  name: string;
  dir: string;
  yamlPath: string;
  text: string;
  symbols: YamlSymbol[];
}

export interface Catalog {
  root: string;
  apps: Map<string, AppIndex>;
}

export function appNameFromFile(file: string): string | undefined {
  const parts = path.normalize(file).split(path.sep);
  const i = parts.lastIndexOf("apps");
  if (i >= 0 && parts[i + 2] === "app.yaml") return parts[i + 1];
  return undefined;
}

export function findRepoRoot(startFile: string): string | undefined {
  const normalized = path.resolve(startFile);
  const parts = normalized.split(path.sep);
  const appsIdx = parts.lastIndexOf("apps");
  if (appsIdx > 0 && parts[parts.length - 1] === "app.yaml") {
    const root = parts.slice(0, appsIdx).join(path.sep);
    if (fs.existsSync(path.join(root, "apps"))) return root;
  }
  let dir = path.dirname(normalized);
  for (let i = 0; i < 16; i++) {
    if (fs.existsSync(path.join(dir, "apps")) && fs.existsSync(path.join(dir, "go.mod"))) {
      return dir;
    }
    const parent = path.dirname(dir);
    if (parent === dir) break;
    dir = parent;
  }
  return undefined;
}

export function loadCatalog(root: string): Catalog {
  const appsDir = path.join(root, "apps");
  const apps = new Map<string, AppIndex>();
  if (!fs.existsSync(appsDir)) return { root, apps };
  for (const name of fs.readdirSync(appsDir)) {
    const dir = path.join(appsDir, name);
    let stat: fs.Stats;
    try {
      stat = fs.statSync(dir);
    } catch {
      continue;
    }
    if (!stat.isDirectory()) continue;
    const yamlPath = path.join(dir, "app.yaml");
    if (!fs.existsSync(yamlPath)) continue;
    const text = fs.readFileSync(yamlPath, "utf8");
    const symbols = collectSymbols(text);
    const declared = symbols.find((s) => s.path === "name")?.value ?? name;
    apps.set(declared, { name: declared, dir, yamlPath, text, symbols });
  }
  return { root, apps };
}
