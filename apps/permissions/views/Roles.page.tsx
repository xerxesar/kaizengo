import { createMemo, createSignal, onMount, Show, type JSX } from 'solid-js'
import {
  Alert,
  Badge,
  Button,
  Card,
  KAppStatus,
  Spinner,
  StatCard,
  Table,
  Toolbar,
  listModelRecords,
  t,
  type Column,
} from '@kaizengo/sdk-solid/ui'
import type { Role, User, UserRole } from '../lib/types'

type RoleRow = Role & { memberCount: number }

export default function Roles(): JSX.Element {
  const [loading, setLoading] = createSignal(true)
  const [error, setError] = createSignal('')
  const [roles, setRoles] = createSignal<Role[]>([])
  const [userRoles, setUserRoles] = createSignal<UserRole[]>([])
  const [usersById, setUsersById] = createSignal<Record<string, User>>({})
  const [selectedRoleId, setSelectedRoleId] = createSignal('')

  const roleRows = createMemo<RoleRow[]>(() => {
    const counts = new Map<string, number>()
    for (const ur of userRoles()) {
      const rid = String(ur.roleId ?? '')
      if (!rid) continue
      counts.set(rid, (counts.get(rid) ?? 0) + 1)
    }
    return roles()
      .map((role) => ({ ...role, memberCount: counts.get(role.id) ?? 0 }))
      .sort((a, b) => String(a.name).localeCompare(String(b.name)))
  })

  const selectedRole = createMemo(() => roles().find((r) => r.id === selectedRoleId()) ?? null)

  const summary = createMemo(() => {
    const activeRoles = roles().filter((r) => r.active !== false).length
    const assigned = new Set(userRoles().map((ur) => String(ur.userId ?? '')).filter(Boolean)).size
    return {
      roles: roles().length,
      active: activeRoles,
      assignments: userRoles().length,
      users: assigned,
    }
  })

  const roleUsers = createMemo(() => {
    const rid = selectedRoleId()
    if (!rid) return [] as { id: string; name: string; email: string }[]
    const out: { id: string; name: string; email: string }[] = []
    const seen = new Set<string>()
    for (const ur of userRoles()) {
      if (String(ur.roleId) !== rid) continue
      const uid = String(ur.userId ?? '')
      if (!uid || seen.has(uid)) continue
      seen.add(uid)
      const u = usersById()[uid]
      out.push({
        id: uid,
        name: String(u?.name ?? uid),
        email: String(u?.email ?? ''),
      })
    }
    out.sort((a, b) => a.name.localeCompare(b.name))
    return out
  })

  const roleColumns = createMemo<Column<RoleRow>[]>(() => [
    { key: 'name', label: t('permissions.col.slug'), mono: true, render: (r) => String(r.name ?? '') },
    { key: 'label', label: t('permissions.col.role'), render: (r) => String(r.label || r.name || '') },
    {
      key: 'active',
      label: t('permissions.filter.status'),
      cell: (r) => (
        <Badge variant={r.active === false ? 'muted' : 'success'}>
          {r.active === false ? t('permissions.filter.status_inactive') : t('permissions.filter.status_active')}
        </Badge>
      ),
    },
    { key: 'memberCount', label: t('permissions.col.members'), align: 'right', render: (r) => String(r.memberCount) },
  ])

  const userColumns = createMemo<Column<{ id: string; name: string; email: string }>[]>(() => [
    { key: 'name', label: t('permissions.col.user'), render: (r) => r.name },
    { key: 'email', label: t('permissions.col.email'), mono: true, render: (r) => r.email },
  ])

  onMount(() => {
    void loadAll()
  })

  let loadingAll = false

  async function loadAll() {
    if (loadingAll) return
    loadingAll = true
    setLoading(true)
    setError('')
    try {
      const [roleRowsRaw, urRows, userRows] = await Promise.all([
        listModelRecords('permissions', 'role', ['name', 'label', 'description', 'active'], 'permissionsRoles'),
        listModelRecords('permissions', 'user_role', ['userId', 'roleId'], 'permissionsUserRoles'),
        listModelRecords('identity', 'user', ['name', 'email'], 'identityUsers'),
      ])
      const nextRoles = (roleRowsRaw as Role[]).sort((a, b) => String(a.name).localeCompare(String(b.name)))
      setRoles(nextRoles)
      setUserRoles(urRows as UserRole[])
      const map: Record<string, User> = {}
      for (const u of userRows as User[]) map[u.id] = u
      setUsersById(map)
      const current = selectedRoleId()
      if (!current && nextRoles.length > 0) {
        setSelectedRoleId(nextRoles[0].id)
      } else if (current && !nextRoles.some((r) => r.id === current)) {
        setSelectedRoleId(nextRoles[0]?.id ?? '')
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
      loadingAll = false
    }
  }

  return (
    <>
      <Show
        when={!loading()}
        fallback={
          <div class="flex items-center gap-[var(--kg-space-05)]">
            <Spinner />
            <p class="m-0 text-[var(--kg-text-muted)]">{t('permissions.loading_roles')}</p>
          </div>
        }
      >
        <div class="flex flex-col gap-[var(--kg-space-06)]">
          <Toolbar
            start={
              <div class="flex min-w-0 flex-col gap-1">
                <h1 class="m-0 text-2xl font-light tracking-tight text-[var(--kg-text)]">{t('permissions.roles_title')}</h1>
                <p class="m-0 text-sm text-[var(--kg-text-muted)]">{t('permissions.roles_subtitle')}</p>
              </div>
            }
            end={
              <Button size="sm" variant="ghost" onClick={() => void loadAll()}>
                {t('permissions.refresh')}
              </Button>
            }
          />

          <Show when={error()}>
            <Alert variant="danger" dismissible onDismiss={() => setError('')}>
              {error()}
            </Alert>
          </Show>

          <div class="grid grid-cols-[repeat(auto-fit,minmax(9.5rem,1fr))] gap-3">
            <StatCard label={t('permissions.stat.roles')} value={summary().roles} />
            <StatCard label={t('permissions.stat.active_roles')} value={summary().active} />
            <StatCard label={t('permissions.stat.assignments')} value={summary().assignments} />
            <StatCard label={t('permissions.stat.assigned_users')} value={summary().users} />
          </div>

          <div class="grid min-w-0 grid-cols-1 gap-[var(--kg-space-05)] xl:grid-cols-2">
            <Card title={t('permissions.roles_title')}>
              <Show
                when={roleRows().length > 0}
                fallback={<p class="m-0 text-[var(--kg-text-muted)]">{t('permissions.roles_empty')}</p>}
              >
                <Table
                  columns={roleColumns()}
                  rows={roleRows()}
                  onRowClick={(row) => setSelectedRoleId(row.id)}
                />
              </Show>
            </Card>

            <Card title={t('permissions.users_title')}>
              <Show when={selectedRole()} fallback={<p class="m-0 text-[var(--kg-text-muted)]">{t('permissions.select_role')}</p>}>
                {(role) => (
                  <>
                    <p class="mb-4 mt-0 text-sm text-[var(--kg-text-muted)]">
                      <span class="font-medium text-[var(--kg-text)]">{role().label || role().name}</span>
                      <Badge variant="muted" class="ms-2">
                        {String(role().name)}
                      </Badge>
                      <span class="ms-2 tabular-nums">
                        {roleUsers().length} {t('permissions.col.members').toLowerCase()}
                      </span>
                    </p>
                    <Show
                      when={roleUsers().length > 0}
                      fallback={<p class="m-0 text-[var(--kg-text-muted)]">{t('permissions.users_empty')}</p>}
                    >
                      <Table columns={userColumns()} rows={roleUsers()} />
                    </Show>
                  </>
                )}
              </Show>
            </Card>
          </div>
        </div>
      </Show>

      <KAppStatus />
    </>
  )
}
