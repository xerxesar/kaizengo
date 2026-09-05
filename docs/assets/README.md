# Docs static assets (local)

Vendored so the published site does not depend on Google Fonts or unpkg CDNs.

| Path | Source | License |
|------|--------|---------|
| `javascripts/mermaid.min.js` | Mermaid 11.12.2 UMD (`unpkg.com/mermaid@11.12.2/dist/mermaid.min.js`) | MIT |
| `assets/fonts/inter-*.woff2` | [@fontsource-variable/inter](https://www.npmjs.com/package/@fontsource-variable/inter) 5.2.5 | OFL-1.1 |
| `assets/fonts/jetbrains-mono-*.woff2` | [@fontsource-variable/jetbrains-mono](https://www.npmjs.com/package/@fontsource-variable/jetbrains-mono) 5.2.5 | OFL-1.1 |

Wired via `zensical.toml` (`extra_css`, `theme.font = false`, `theme.custom_dir` → `overrides/main.html` loads Mermaid in `<head>` before the theme bundle).
