import fs from "node:fs";
import path from "node:path";
import { collectSymbols, type YamlSymbol } from "./parse";

export interface SpecDoc {
  path: string;
  text: string;
  symbols: YamlSymbol[];
}

export interface AppIndex {
  name: string;
  dir: string;
  yamlPath: string;
  text: string;
  symbols: YamlSymbol[];
  docs: SpecDoc[];
}

export interface Catalog {
  root: string;
  apps: Map<string, AppIndex>;
}

export function appNameFromFile(file: string): string | undefined {
  const parts = path.normalize(file).split(path.sep);
  const i = parts.lastIndexOf("apps");
  if (i < 0 || !parts[i + 1]) return undefined;
  if (parts[i + 2] === "app.yaml") return parts[i + 1];
  if (/\.ya?ml$/i.test(parts[parts.length - 1] ?? "")) return parts[i + 1];
  return undefined;
}

export function findRepoRoot(startFile: string): string | undefined {
  const normalized = path.resolve(startFile);
  const parts = normalized.split(path.sep);
  const appsIdx = parts.lastIndexOf("apps");
  if (appsIdx > 0) {
    const root = parts.slice(0, appsIdx).join(path.sep);
    if (fs.existsSync(path.join(root, "apps")) && fs.existsSync(path.join(root, "go.mod"))) {
      return root;
    }
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

function resolveModelInclude(fromFile: string, ref: string): string | undefined {
  const target = path.resolve(path.dirname(fromFile), ref);
  try {
    const st = fs.statSync(target);
    if (st.isDirectory()) {
      for (const name of ["spec.yaml", "spec.yml", `${path.basename(target)}.yaml`]) {
        const candidate = path.join(target, name);
        if (fs.existsSync(candidate)) return candidate;
      }
      return undefined;
    }
  } catch {
    return undefined;
  }
  if (/\.ya?ml$/i.test(target)) return target;
  return undefined;
}

function loadSpecDocs(yamlPath: string): SpecDoc[] {
  const docs: SpecDoc[] = [];
  const seen = new Set<string>();
  const add = (file: string) => {
    const abs = path.resolve(file);
    if (seen.has(abs)) return;
    seen.add(abs);
    if (!fs.existsSync(abs)) return;
    const text = fs.readFileSync(abs, "utf8");
    const symbols = collectSymbols(text);
    docs.push({ path: abs, text, symbols });
    for (const s of symbols) {
      if (!/^models\[\d+\]$/.test(s.path)) continue;
      const included = resolveModelInclude(abs, s.value.trim());
      if (included) add(included);
    }
  };
  add(yamlPath);
  return docs;
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
    const docs = loadSpecDocs(yamlPath);
    const main = docs[0];
    const declared = main.symbols.find((s) => s.path === "name")?.value ?? name;
    apps.set(declared, {
      name: declared,
      dir,
      yamlPath,
      text: main.text,
      symbols: main.symbols,
      docs,
    });
  }
  return { root, apps };
}
