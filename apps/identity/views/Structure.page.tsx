import { useEffect, useMemo, useState } from 'react'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Progress } from '@/components/ui/progress'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { TreeView, t, type TreeNode, Card, Toolbar, FormActions } from '@/lib'
import { KAppStatus } from '@/k'
import { IdentityToolbar } from '../lib/IdentityToolbar'
import {
  ORG_UNIT_TYPES,
  createOrgUnit,
  fetchOrgTree,
  flattenTree,
  formatUnitType,
  type OrgUnitNode,
} from '../lib/graphql'
import { initIdentity, useIdentityState } from '../lib/state'

export default function Structure() {
  const identity = useIdentityState()

  const [tree, setTree] = useState<OrgUnitNode[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [showModal, setShowModal] = useState(false)
  const [newName, setNewName] = useState('')
  const [newType, setNewType] = useState('department')
  const [newParentId, setNewParentId] = useState('')

  const org = identity.selectedOrg

  function toTreeNodes(nodes: OrgUnitNode[]): TreeNode<OrgUnitNode>[] {
    return nodes.map((n) => ({
      id: n.id,
      label: n.name,
      meta: formatUnitType(n.type, t),
      data: n,
      children: n.children?.length ? toTreeNodes(n.children) : undefined,
    }))
  }

  const flatUnits = useMemo(() => flattenTree(tree), [tree])
  const selectedNode = useMemo(
    () => flatUnits.find((u) => u.id === selectedId) ?? null,
    [flatUnits, selectedId],
  )
  const treeNodes = useMemo(() => toTreeNodes(tree), [tree])

  async function load() {
    if (!org) return
    setLoading(true)
    setError('')
    try {
      const data = await fetchOrgTree(org.id)
      setTree(data.orgTree)
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }

  async function submitUnit() {
    if (!org || !newName.trim()) return
    setLoading(true)
    setError('')
    try {
      await createOrgUnit(org.id, newType, newName.trim(), newParentId || undefined)
      setNewName('')
      setShowModal(false)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      setLoading(false)
    }
  }

  useEffect(() => {
    void initIdentity()
  }, [])

  useEffect(() => {
    if (identity.ready && org?.id) void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [identity.ready, org?.id])

  if (identity.loading) {
    return (
      <div className="flex items-center justify-center py-8" role="status" aria-label="Loading">
        <Progress indeterminate className="w-48 max-w-full" />
      </div>
    )
  }

  return (
    <>
      <IdentityToolbar />

      {identity.error ? (
        <Alert variant="danger">{identity.error}</Alert>
      ) : !org ? (
        <Alert variant="warning">{t('identity.no_org')}</Alert>
      ) : (
        <>
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

          <Toolbar
            start={
              <span className="text-sm font-medium text-[var(--kg-text-secondary)]">
                {t('identity.structure.count', flatUnits.length)}
              </span>
            }
            end={<Button onClick={() => setShowModal(true)}>{t('identity.structure.add')}</Button>}
          />

          <div className="grid grid-cols-1 gap-5 md:grid-cols-2">
            <div>
              {loading && tree.length === 0 ? (
                <Progress indeterminate className="w-48 max-w-full" />
              ) : (
                <TreeView
                  nodes={treeNodes}
                  selectedId={selectedId}
                  onSelect={(node) => setSelectedId(node.id)}
                />
              )}
            </div>

            <div>
              {!selectedNode ? (
                <Card title={t('identity.structure.details')}>
                  <p className="text-sm text-[var(--kg-text-muted)]">
                    {t('identity.structure.select_hint')}
                  </p>
                </Card>
              ) : (
                <Card title={selectedNode.name}>
                  <dl className="detail-list">
                    <div>
                      <dt>{t('identity.structure.field.type')}</dt>
                      <dd>{formatUnitType(selectedNode.type, t)}</dd>
                    </div>
                    <div>
                      <dt>{t('identity.structure.field.id')}</dt>
                      <dd className="mono">{selectedNode.id}</dd>
                    </div>
                    <div>
                      <dt>{t('identity.structure.field.parent')}</dt>
                      <dd>
                        {selectedNode.parentId
                          ? (flatUnits.find((u) => u.id === selectedNode.parentId)?.name ??
                            selectedNode.parentId)
                          : t('identity.structure.root')}
                      </dd>
                    </div>
                    <div>
                      <dt>{t('identity.structure.field.created')}</dt>
                      <dd>
                        {selectedNode.createdAt
                          ? new Date(selectedNode.createdAt).toLocaleDateString()
                          : '—'}
                      </dd>
                    </div>
                  </dl>
                </Card>
              )}
            </div>
          </div>

          {showModal ? (
            <div className="fixed inset-0 z-[9000] flex items-center justify-center bg-black/50 p-4">
              <div className="flex max-h-[90vh] w-full max-w-xl flex-col bg-[var(--kg-surface)] shadow-lg">
                <header className="flex items-center justify-between border-b border-[var(--kg-border)] px-5 py-4">
                  <h2 className="text-xl font-normal text-[var(--kg-text)]">
                    {t('identity.structure.add_title')}
                  </h2>
                  <button
                    type="button"
                    className="border-0 bg-transparent text-2xl leading-none text-[var(--kg-text-secondary)]"
                    onClick={() => setShowModal(false)}
                  >
                    ×
                  </button>
                </header>
                <div className="p-5">
                  <form
                    className="flex flex-col gap-3.5"
                    onSubmit={(e) => {
                      e.preventDefault()
                      void submitUnit()
                    }}
                  >
                    <div className="flex flex-col gap-1">
                      <Label>
                        {t('identity.structure.name')}
                        <span className="text-[var(--kg-danger)]"> *</span>
                      </Label>
                      <Input
                        value={newName}
                        onChange={(e) => setNewName(e.target.value)}
                        placeholder={t('identity.structure.name_placeholder')}
                      />
                    </div>
                    <div className="flex flex-col gap-1">
                      <Label>
                        {t('identity.structure.field.type')}
                        <span className="text-[var(--kg-danger)]"> *</span>
                      </Label>
                      <Select value={newType} onValueChange={setNewType}>
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {ORG_UNIT_TYPES.map((u) => (
                            <SelectItem key={u.value} value={u.value}>
                              {t(u.key)}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="flex flex-col gap-1">
                      <Label>{t('identity.structure.parent')}</Label>
                      <Select
                        value={newParentId || '__root__'}
                        onValueChange={(v) => setNewParentId(v === '__root__' ? '' : v)}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder={t('identity.structure.root')} />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="__root__">{t('identity.structure.root')}</SelectItem>
                          {flatUnits.map((u) => (
                            <SelectItem key={u.id} value={u.id}>
                              {`${'  '.repeat(u.depth)}${u.name} (${formatUnitType(u.type, t)})`}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      <p className="text-xxs font-light text-[var(--kg-text-muted)]">
                        {t('identity.structure.parent_hint')}
                      </p>
                    </div>
                    <FormActions>
                      <Button variant="ghost" type="button" onClick={() => setShowModal(false)}>
                        {t('identity.common.cancel')}
                      </Button>
                      <Button type="submit" disabled={loading}>
                        {loading ? '…' : t('identity.structure.create')}
                      </Button>
                    </FormActions>
                  </form>
                </div>
              </div>
            </div>
          ) : null}
        </>
      )}

      <KAppStatus />

      <style>{`
        .detail-list { display: grid; gap: 0.75rem; margin: 0; }
        .detail-list div { display: grid; grid-template-columns: 6rem 1fr; gap: 0.5rem; font-size: 0.875rem; }
        .detail-list dt { font-weight: 600; color: var(--kg-text-muted); }
        .detail-list dd { margin: 0; color: var(--kg-text); }
        .detail-list .mono { font-family: var(--kg-font-mono); font-size: 0.8125rem; word-break: break-all; }
      `}</style>
    </>
  )
}
