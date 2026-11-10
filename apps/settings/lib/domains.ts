/** GraphQL client for Settings → Domains (conflicts + data migrate). */

export type DomainRow = {
  app: string
  kind: string
  title: string
  revision: number
  source: string
  loaded: boolean
  error?: string | null
  rebaseStatus?: string | null
  conflictCount?: number | null
  migratePending?: boolean | null
}

export type DomainConflict = {
  path: string
  ancestor?: string | null
  ours: string
  theirs: string
}

export type MigrateHint = {
  path: string
  hint: string
  stub: string
}

export type ApplyResult = {
  dryRun: boolean
  changes: string[]
  migrateRequired: boolean
  proposedSpec?: string | null
  migrateHints: MigrateHint[]
  domain: { app: string; revision: number; rebaseStatus?: string | null; conflictCount?: number | null }
}

export type { DomainsPending } from '@/lib/platform-shell'
export {
  DOMAINS_PENDING_KEY,
  DOMAINS_PENDING_EVENT,
  readDomainsPending,
  clearDomainsPending,
} from '@/lib/platform-shell'

async function gql<T>(query: string, variables?: Record<string, unknown>): Promise<T> {
  const res = await fetch('/graphql', {
    method: 'POST',
    credentials: 'include',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ query, variables }),
  })
  if (!res.ok) throw new Error(`GraphQL HTTP ${res.status}`)
  const body = await res.json()
  if (body.errors?.length) {
    throw new Error(body.errors.map((e: { message: string }) => e.message).join(', '))
  }
  return body.data as T
}

export function fetchDomains() {
  return gql<{ platformDomains: DomainRow[] }>(
    `{ platformDomains { app kind title revision source loaded error rebaseStatus conflictCount migratePending } }`,
  )
}

export function fetchMigratePending(app: string) {
  return gql<{
    platformDomainMigratePending: {
      app: string
      proposedSpec: string
      migrateHints: MigrateHint[]
    } | null
  }>(
    `query($app:String!){ platformDomainMigratePending(app:$app){ app proposedSpec migrateHints { path hint stub } } }`,
    { app },
  )
}

export function fetchConflicts(app: string) {
  return gql<{ platformDomainConflicts: DomainConflict[] }>(
    `query($app:String!){ platformDomainConflicts(app:$app){ path ancestor ours theirs } }`,
    { app },
  )
}

export function fetchDomainSpec(app: string) {
  return gql<{ platformDomain: { app: string; spec: string; rebaseStatus?: string; conflictCount?: number } }>(
    `query($app:String!){ platformDomain(app:$app){ app spec rebaseStatus conflictCount } }`,
    { app },
  )
}

export function rebaseDomain(
  app: string,
  opts: { force?: boolean; dryRun?: boolean; resolutions?: { path: string; side: string }[] },
) {
  return gql<{ platformRebaseDomain: ApplyResult }>(
    `mutation($app:String!,$f:Boolean,$d:Boolean,$r:[PlatformDomainConflictResolution!]){
      platformRebaseDomain(app:$app, force:$f, dryRun:$d, resolutions:$r) {
        dryRun changes migrateRequired proposedSpec migrateHints { path hint stub }
        domain { app revision rebaseStatus conflictCount }
      }
    }`,
    { app, f: opts.force ?? false, d: opts.dryRun ?? false, r: opts.resolutions ?? [] },
  )
}

export function applyDomain(
  spec: string,
  opts: {
    dryRun?: boolean
    force?: boolean
    migratePre?: string
    migratePost?: string
    noMigrate?: boolean
  },
) {
  return gql<{ platformApplyDomain: ApplyResult }>(
    `mutation($s:String!,$d:Boolean,$f:Boolean,$pre:String,$post:String,$n:Boolean){
      platformApplyDomain(spec:$s, dryRun:$d, force:$f, migratePre:$pre, migratePost:$post, noMigrate:$n) {
        dryRun changes migrateRequired proposedSpec migrateHints { path hint stub }
        domain { app revision rebaseStatus conflictCount }
      }
    }`,
    {
      s: spec,
      d: opts.dryRun ?? false,
      f: opts.force ?? false,
      pre: opts.migratePre ?? null,
      post: opts.migratePost ?? null,
      n: opts.noMigrate ?? false,
    },
  )
}

export function domainDiff(spec: string) {
  return gql<{ platformDomainDiff: { app: string; changes: string[]; migrateHints: MigrateHint[] } }>(
    `query($s:String!){ platformDomainDiff(spec:$s){ app changes migrateHints { path hint stub } } }`,
    { s: spec },
  )
}

/** Infer a guided migrate action from a hint path/stub. */
export type GuidedAction = 'rename' | 'backfill' | 'skip' | 'custom'

export function suggestGuided(hint: MigrateHint): GuidedAction {
  const stub = hint.stub.toLowerCase()
  if (stub.includes('rename column')) return 'rename'
  if (hint.hint.includes('new column') || stub.includes('update ')) return 'backfill'
  if (hint.hint.includes('removed')) return 'skip'
  return 'custom'
}

export function sqlForGuided(hint: MigrateHint, action: GuidedAction, customSQL: string): { phase: 'pre' | 'post' | 'none'; sql: string } {
  if (action === 'skip') return { phase: 'none', sql: '' }
  if (action === 'custom') return { phase: 'post', sql: customSQL.trim() || hint.stub.replace(/^--\s*/gm, '').trim() }
  if (action === 'rename') {
    // Prefer first RENAME-like line from stub comments
    const m = hint.stub.match(/ALTER TABLE\s+\S+\s+RENAME COLUMN\s+\S+\s+TO\s+\S+;?/i)
    const sql = m?.[0] ?? customSQL.trim()
    return { phase: 'pre', sql }
  }
  // backfill
  const lines = hint.stub
    .split('\n')
    .map((l) => l.replace(/^\s*--\s?/, '').trim())
    .filter((l) => l && !l.startsWith('--'))
  const sql = customSQL.trim() || lines.find((l) => /^UPDATE\b/i.test(l)) || lines.join('\n')
  return { phase: 'post', sql }
}
