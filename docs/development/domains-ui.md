# Domains UI — scenarios & how to test

Hands-on checklist for **Settings → Domains** (tip-vs-code conflicts + data migrate). Use a disposable DB or `hellospec` tip you can revert.

## Setup

1. `make dev` (or restart after pulling these changes).
2. Sign in: `admin@kaizengo.local` / `changeme`.
3. Open the SPA (`http://localhost:5173/app/` or your usual URL).
4. Confirm **Settings → Domains** appears in the settings menu (`/app/settings/domains`).

Native CLI (optional):

```bash
bin/kaizen login --email admin@kaizengo.local   # password: changeme
bin/kaizen domain list
bin/kaizen domain show hellospec
```

---

## Scenario A — Data migrate (guided + advanced)

**Goal:** Tip change that touches stored fields must go through migrate UI (or `--migrate-pre` / `--migrate` / `--no-migrate`).

### A1. Trigger migrate from the web terminal

1. Open the terminal (`Alt+T`).
2. Ensure hellospec has a string field you can rename later, e.g.:

   ```text
   domain add-field hellospec.greeting scratch string --no-migrate
   ```

3. Rename it (default migrate-pre RENAME):

   ```text
   domain set-field hellospec.greeting scratch string --name scratch2
   ```

   - If the column already exists and rename succeeds, you’re done for this step.
   - To **force** the migrate gate without default pre-SQL, apply a tip that adds a new stored field **without** migrate flags (edit tip YAML and `domain apply`, or `add-field` without `--no-migrate` on a fresh name).

4. When migrate is required, the terminal should:
   - Print `migration required — open Settings → Domains…`
   - Navigate to `/app/settings/domains` (migrate tab)
   - Server stashes the proposal (`platformDomainMigratePending`); sessionStorage also carries it for SPA handoff

5. On **Domains → Data migrate**:
   - Confirm proposed tip YAML is filled automatically (from session or server pending). HelloSpec should show `(migrate pending)` in the app picker.
   - Guided hints should appear without pasting YAML. Click **Analyze migrate needs** only if you edited the tip.
   - For each hint, try guided actions: **Rename (pre)**, **Backfill (post)**, **Skip**, **Custom SQL**.
   - Open **Show advanced SQL** and check pre/post textareas update.
   - **Apply with migrate**, or **Apply with --no-migrate** for a dry acceptance of empty columns.

6. Verify: `domain show hellospec` has the new field names; list/form still load.

### A2. Migrate without terminal navigation

1. After a CLI/terminal migrate gate (even from native `kaizen`), open Settings → Domains.
2. Select the app — if `migratePending`, the **Data migrate** tab loads the stashed proposed tip + hints.
3. Or paste a full tip YAML that adds/renames a field vs current tip, then Analyze → pick guided/custom → Apply.

### A3. CLI-only migrate (control case)

```text
domain add-field hellospec.greeting demo string --migrate="UPDATE greetings SET demo = 'x' WHERE demo IS NULL OR demo = '';"
# or
domain set-field hellospec.greeting demo string --name demo2
# default migrate-pre: ALTER TABLE greetings RENAME COLUMN "demo" TO "demo2";
```

---

## Scenario B — Domain conflicts (three-way rebase)

**Goal:** Tip and compiled `app.yaml` both change the same path → `rebaseStatus=conflict` → resolve in UI.

Conflicts appear after an upgrade when tip diverged from code. To **simulate** without a full platform upgrade:

### B1. Create a tip-only edit, then diverge code (dev)

1. Note current tip: `domain show hellospec` (or Settings → Domains → Overview).
2. Change something **only on the tip** that also exists in `apps/hellospec/app.yaml`, e.g. a view label via CLI:

   ```text
   domain set-view hellospec GreetingList -ted
   # or paste a heredoc / use :wq in the web vim editor
   domain show-view hellospec GreetingList
   ```

3. Change the **same view** (or title/summary) in `apps/hellospec/app.yaml` differently (e.g. change `defaultView` or a column).
4. Restart the server (`make dev` / rebuild) so boot three-way-merges tip vs new code.
5. Check:

   ```text
   domain list
   domain conflicts hellospec
   ```

   Expect `rebaseStatus=conflict` and one or more paths.

6. Open **Settings → Domains** (or run a failing `domain rebase hellospec` from the terminal and let it navigate).
7. On **Conflicts**:
   - Each path shows **Ancestor / Tip (ours) / Code (theirs)**.
   - Click Tip or Code per path.
   - **Apply resolutions** — should clear conflicts and append a `source=rebase` tip.
8. Alternate: **Take code on all** (`--force` equivalent).

9. Confirm: `domain list` shows no conflict; app menus/views match your choices.

### B2. CLI conflict control case

```text
domain conflicts hellospec
domain rebase hellospec --dry-run
domain rebase hellospec --force
```

---

## Scenario C — Terminal → Domains handoff

| Trigger | Expected |
|---------|----------|
| Domain apply/helper returns migrate required | Navigate to Domains, tab **Data migrate**; YAML + hints filled from session and/or `platformDomainMigratePending` |
| Rebase / conflict message | Navigate to Domains, tab **Conflicts** |
| Happy path apply | Stay in terminal; Domains unchanged; pending migrate cleared |

SPA handoff: `sessionStorage` key `kaizengo:domains.pending` (cleared when Domains loads it). Server stash: in-memory until a successful apply/`--no-migrate` (survives opening Domains from the menu after native CLI).

---

## Scenario D — Smoke checklist

- [ ] Settings menu shows **Domains**
- [ ] Overview lists apps, revision, rebase status
- [ ] Conflicts empty state when no pending conflicts
- [ ] Migrate empty state until Analyze / proposed tip
- [ ] `domain history hellospec` shows new revisions after UI apply/rebase
- [ ] Escape hatch: `domain revert hellospec <N>` if a test tip is bad

---

## Reset / cleanup tips

```text
domain history hellospec
domain show hellospec --rev N
domain revert hellospec N --dry-run
domain revert hellospec N
```

Soft-clear tip (compiled remounts from code): `domain delete hellospec --dry-run` then without `--dry-run` if you intend that.

For DB columns left behind after soft removes: `domain remove-field … --force` or manual SQL in migrate-pre/post.

---

## Related docs

- [CLI — domain](../cli.md) (heredoc, `--migrate-pre` / `--migrate`, rebase)
- [Architecture — domain layers](../architecture.md)
