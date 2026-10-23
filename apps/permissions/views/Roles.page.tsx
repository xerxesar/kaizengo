import { useEffect, useMemo, useRef, useState } from 'react'
import { Alert } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { badgeVariants, cn, listModelRecords, t, type Column, Card, Toolbar, StatCard, DataTable } from '@/lib'
import { KAppStatus } from '@/k'
import type { Role, User, UserRole } from '../lib/types'

type RoleRow = Role & { memberCount: number }

export default function Roles() {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [roles, setRoles] = useState<Role[]>([])
  const [userRoles, setUserRoles] = useState<UserRole[]>([])
  const [usersById, setUsersById] = useState<Record<string, User>>({})
  const [selectedRoleId, setSelectedRoleId] = useState('')
  const loadingAll = useRef(false)

  const roleRows = useMemo<RoleRow[]>(() => {
    const counts = new Map<string, number>()
    for (const ur of userRoles) {
      const rid = String(ur.roleId ?? '')
      if (!rid) continue
      counts.set(rid, (counts.get(rid) ?? 0) + 1)
    }
    return roles
      .map((role) => ({ ...role, memberCount: counts.get(role.id) ?? 0 }))
      .sort((a, b) => String(a.name).localeCompare(String(b.name)))
  }, [roles, userRoles])

  const selectedRole = useMemo(
    () => roles.find((r) => r.id === selectedRoleId) ?? null,
    [roles, selectedRoleId],
  )

  const summary = useMemo(() => {
    const activeRoles = roles.filter((r) => r.active !== false).length
    const assigned = new Set(userRoles.map((ur) => String(ur.userId ?? '')).filter(Boolean)).size
    return {
      roles: roles.length,
      active: activeRoles,
      assignments: userRoles.length,
      users: assigned,
    }
  }, [roles, userRoles])

  const roleUsers = useMemo(() => {
    if (!selectedRoleId) return [] as { id: string; name: string; email: string }[]
    const out: { id: string; name: string; email: string }[] = []
    const seen = new Set<string>()
    for (const ur of userRoles) {
      if (String(ur.roleId) !== selectedRoleId) continue
      const uid = String(ur.userId ?? '')
      if (!uid || seen.has(uid)) continue
      seen.add(uid)
      const u = usersById[uid]
      out.push({
        id: uid,
        name: String(u?.name ?? uid),
        email: String(u?.email ?? ''),
      })
    }
    out.sort((a, b) => a.name.localeCompare(b.name))
    return out
  }, [selectedRoleId, userRoles, usersById])

  const roleColumns = useMemo<Column<RoleRow>[]>(
    () => [
      { key: 'name', label: t('permissions.col.slug'), mono: true, render: (r) => String(r.name ?? '') },
      { key: 'label', label: t('permissions.col.role'), render: (r) => String(r.label || r.name || '') },
      {
        key: 'active',
        label: t('permissions.filter.status'),
        cell: (r) => (
          <Badge
            className={cn(
              'inline-flex items-center rounded px-2 py-0.5 text-xs font-medium',
              r.active === false ? badgeVariants.muted : badgeVariants.success,
            )}
          >
            {r.active === false
              ? t('permissions.filter.status_inactive')
              : t('permissions.filter.status_active')}
          </Badge>
        ),
      },
      {
        key: 'memberCount',
        label: t('permissions.col.members'),
        align: 'right',
        render: (r) => String(r.memberCount),
      },
    ],
    [],
  )

  const userColumns = useMemo<Column<{ id: string; name: string; email: string }>[]>(
    () => [
      { key: 'name', label: t('permissions.col.user'), render: (r) => r.name },
      { key: 'email', label: t('permissions.col.email'), mono: true, render: (r) => r.email },
    ],
    [],
  )

  async function loadAll() {
    if (loadingAll.current) return
    loadingAll.current = true
    setLoading(true)
    setError('')
    try {
      const [roleRowsRaw, urRows, userRows] = await Promise.all([
        listModelRecords('permissions', 'role', ['name', 'label', 'description', 'active']),
        listModelRecords('permissions', 'user_role', ['userId', 'roleId']),
        listModelRecords('identity', 'user', ['name', 'email']),
      ])
      const nextRoles = (roleRowsRaw as Role[]).sort((a, b) =>
        String(a.name).localeCompare(String(b.name)),
      )
      setRoles(nextRoles)
      setUserRoles(urRows as UserRole[])
      const map: Record<string, User> = {}
      for (const u of userRows as User[]) map[u.id] = u
      setUsersById(map)
      setSelectedRoleId((current) => {
        if (!current && nextRoles.length > 0) return nextRoles[0].id
        if (current && !nextRoles.some((r) => r.id === current)) return nextRoles[0]?.id ?? ''
        return current
      })
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
      loadingAll.current = false
    }
  }

  useEffect(() => {
    void loadAll()
  }, [])

  if (loading) {
    return (
      <>
        <div className="flex items-center gap-[var(--kg-space-05)]">
          <div className="w-32" role="status" aria-label="Loading">
            <Progress indeterminate className="w-full" />
          </div>
          <p className="m-0 text-[var(--kg-text-muted)]">{t('permissions.loading_roles')}</p>
        </div>
        <KAppStatus />
      </>
    )
  }

  return (
    <>
      <div className="flex flex-col gap-[var(--kg-space-06)]">
        <Toolbar
          start={
            <div className="flex min-w-0 flex-col gap-1">
              <h1 className="m-0 text-2xl font-light tracking-tight text-[var(--kg-text)]">
                {t('permissions.roles_title')}
              </h1>
              <p className="m-0 text-sm text-[var(--kg-text-muted)]">
                {t('permissions.roles_subtitle')}
              </p>
            </div>
          }
          end={
            <Button size="sm" variant="ghost" onClick={() => void loadAll()}>
              {t('permissions.refresh')}
            </Button>
          }
        />

        {error ? (
          <Alert variant="danger">
            <div className="min-w-0 flex-1">{error}</div>
            <button
              type="button"
              className="shrink-0 text-current opacity-70 hover:opacity-100"
              aria-label="Dismiss"
              onClick={() => setError('')}
            >
              ×
            </button>
          </Alert>
        ) : null}

        <div className="grid grid-cols-[repeat(auto-fit,minmax(9.5rem,1fr))] gap-3">
          <StatCard label={t('permissions.stat.roles')} value={summary.roles} />
          <StatCard label={t('permissions.stat.active_roles')} value={summary.active} />
          <StatCard label={t('permissions.stat.assignments')} value={summary.assignments} />
          <StatCard label={t('permissions.stat.assigned_users')} value={summary.users} />
        </div>

        <div className="grid min-w-0 grid-cols-1 gap-[var(--kg-space-05)] xl:grid-cols-2">
          <Card title={t('permissions.roles_title')}>
            {roleRows.length > 0 ? (
              <DataTable
                columns={roleColumns}
                rows={roleRows}
                onRowClick={(row) => setSelectedRoleId(row.id)}
              />
            ) : (
              <p className="m-0 text-[var(--kg-text-muted)]">{t('permissions.roles_empty')}</p>
            )}
          </Card>

          <Card title={t('permissions.users_title')}>
            {selectedRole ? (
              <>
                <p className="mb-4 mt-0 text-sm text-[var(--kg-text-muted)]">
                  <span className="font-medium text-[var(--kg-text)]">
                    {selectedRole.label || selectedRole.name}
                  </span>
                  <Badge
                    className={cn(
                      'ms-2 inline-flex items-center rounded px-2 py-0.5 text-xs font-medium',
                      badgeVariants.muted,
                    )}
                  >
                    {String(selectedRole.name)}
                  </Badge>
                  <span className="ms-2 tabular-nums">
                    {roleUsers.length} {t('permissions.col.members').toLowerCase()}
                  </span>
                </p>
                {roleUsers.length > 0 ? (
                  <DataTable columns={userColumns} rows={roleUsers} />
                ) : (
                  <p className="m-0 text-[var(--kg-text-muted)]">{t('permissions.users_empty')}</p>
                )}
              </>
            ) : (
              <p className="m-0 text-[var(--kg-text-muted)]">{t('permissions.select_role')}</p>
            )}
          </Card>
        </div>
      </div>

      <KAppStatus />
    </>
  )
}
