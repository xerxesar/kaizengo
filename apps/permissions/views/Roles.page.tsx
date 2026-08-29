import { createMemo, createSignal, onMount, Show, type JSX } from 'solid-js'
import {
  Alert,
  Badge,
  Button,
  Card,
  KAppStatus,
  Spinner,
  Table,
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
      const [roleRows, urRows, userRows] = await Promise.all([
        listModelRecords('permissions', 'role', ['name', 'label', 'description', 'active']),
        listModelRecords('permissions', 'user_role', ['userId', 'roleId']),
        listModelRecords('identity', 'user', ['name', 'email']),
      ])
      const nextRoles = (roleRows as Role[]).sort((a, b) => String(a.name).localeCompare(String(b.name)))
      setRoles(nextRoles)
      setUserRoles(urRows as UserRole[])
      const map: Record<string, User> = {}
      for (const u of userRows as User[]) {
        map[u.id] = u
      }
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
            <p class="m-0 text-[var(--kg-text-muted)]">{t('permissions.loading')}</p>
          </div>
        }
      >
        <Show when={error()}>
          <Alert variant="danger">{error()}</Alert>
        </Show>

        <div class="flex flex-col gap-[var(--kg-space-07)]">
          <header class="flex flex-wrap items-center gap-[var(--kg-space-04)]">
            <h2 class="m-0 text-xl">{t('permissions.roles_title')}</h2>
            <Button size="sm" variant="ghost" onClick={() => void loadAll()}>
              {t('permissions.refresh')}
            </Button>
          </header>

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

          <Show when={selectedRole()} fallback={<p class="m-0 text-[var(--kg-text-muted)]">{t('permissions.select_role')}</p>}>
            {(role) => (
              <Card title={t('permissions.users_title')}>
                <p class="mb-[var(--kg-space-04)] mt-0 text-sm text-[var(--kg-text-muted)]">
                  {role().label || role().name}
                  <Badge variant="muted" class="ms-2">
                    {String(role().name)}
                  </Badge>
                </p>
                <Show
                  when={roleUsers().length > 0}
                  fallback={<p class="m-0 text-[var(--kg-text-muted)]">{t('permissions.users_empty')}</p>}
                >
                  <Table columns={userColumns()} rows={roleUsers()} />
                </Show>
              </Card>
            )}
          </Show>
        </div>
      </Show>

      <KAppStatus />
    </>
  )
}
