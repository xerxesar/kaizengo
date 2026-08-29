import { createEffect, createMemo, createSignal, For, on, onMount, Show } from 'solid-js'
import {
  Alert,
  Button,
  Card,
  FormActions,
  FormField,
  Input,
  KAppStatus,
  Modal,
  Select,
  Spinner,
  Toolbar,
  TreeView,
  t,
  type TreeNode,
} from '@kaizengo/sdk-solid/ui'
import { IdentityToolbar } from '../lib/IdentityToolbar'
import {
  ORG_UNIT_TYPES,
  createOrgUnit,
  fetchOrgTree,
  flattenTree,
  formatUnitType,
  type OrgUnitNode,
} from '../lib/graphql'
import { identityState, initIdentity } from '../lib/state'

export default function Structure() {
  const identity = identityState()

  const [tree, setTree] = createSignal<OrgUnitNode[]>([])
  const [loading, setLoading] = createSignal(true)
  const [error, setError] = createSignal('')
  const [selectedId, setSelectedId] = createSignal<string | null>(null)
  const [showModal, setShowModal] = createSignal(false)
  const [newName, setNewName] = createSignal('')
  const [newType, setNewType] = createSignal('department')
  const [newParentId, setNewParentId] = createSignal('')

  const org = createMemo(() => identity.selectedOrg)

  function toTreeNodes(nodes: OrgUnitNode[]): TreeNode<OrgUnitNode>[] {
    return nodes.map((n) => ({
      id: n.id,
      label: n.name,
      meta: formatUnitType(n.type, t),
      data: n,
      children: n.children?.length ? toTreeNodes(n.children) : undefined,
    }))
  }

  const flatUnits = createMemo(() => flattenTree(tree()))
  const selectedNode = createMemo(() => flatUnits().find((u) => u.id === selectedId()) ?? null)
  const treeNodes = createMemo(() => toTreeNodes(tree()))

  async function load() {
    const currentOrg = org()
    if (!currentOrg) return
    setLoading(true)
    setError('')
    try {
      const data = await fetchOrgTree(currentOrg.id)
      setTree(data.orgTree)
      identity.onStats({ units: flatUnits().length })
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
    } finally {
      setLoading(false)
    }
  }

  async function submitUnit() {
    const currentOrg = org()
    if (!currentOrg || !newName().trim()) return
    setLoading(true)
    setError('')
    try {
      await createOrgUnit(currentOrg.id, newType(), newName().trim(), newParentId() || undefined)
      setNewName('')
      setShowModal(false)
      await load()
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e))
      setLoading(false)
    }
  }

  createEffect(
    on(
      () => (identity.ready ? org()?.id : undefined),
      (orgId) => {
        if (orgId) void load()
      },
    ),
  )

  onMount(async () => {
    await initIdentity()
  })

  return (
    <Show when={!identity.loading} fallback={<Spinner />}>
      <IdentityToolbar />

      <Show when={identity.error} fallback={
        <Show when={org()} fallback={<Alert variant="warning">{t('identity.no_org')}</Alert>}>
          <Show when={error()}>
            <Alert variant="danger" dismissible onDismiss={() => setError('')}>
              {error()}
            </Alert>
          </Show>

          <Toolbar
            start={<span class="text-sm font-medium text-[var(--kg-text-secondary)]">{t('identity.structure.count', flatUnits().length)}</span>}
            end={<Button onClick={() => setShowModal(true)}>{t('identity.structure.add')}</Button>}
          />

          <div class="grid grid-cols-1 gap-5 md:grid-cols-2">
            <div>
              <Show when={loading() && tree().length === 0} fallback={
                <TreeView nodes={treeNodes()} selectedId={selectedId()} onSelect={(node) => setSelectedId(node.id)} />
              }>
                <Spinner />
              </Show>
            </div>

            <div>
              <Show
                when={selectedNode()}
                fallback={
                  <Card title={t('identity.structure.details')}>
                    <p class="text-sm text-[var(--kg-text-muted)]">{t('identity.structure.select_hint')}</p>
                  </Card>
                }
              >
                {(node) => (
                  <Card title={node().name}>
                    <dl class="detail-list">
                      <div>
                        <dt>{t('identity.structure.field.type')}</dt>
                        <dd>{formatUnitType(node().type, t)}</dd>
                      </div>
                      <div>
                        <dt>{t('identity.structure.field.id')}</dt>
                        <dd class="mono">{node().id}</dd>
                      </div>
                      <div>
                        <dt>{t('identity.structure.field.parent')}</dt>
                        <dd>
                          {node().parentId
                            ? flatUnits().find((u) => u.id === node().parentId)?.name ?? node().parentId
                            : t('identity.structure.root')}
                        </dd>
                      </div>
                      <div>
                        <dt>{t('identity.structure.field.created')}</dt>
                        <dd>{node().createdAt ? new Date(node().createdAt).toLocaleDateString() : '—'}</dd>
                      </div>
                    </dl>
                  </Card>
                )}
              </Show>
            </div>
          </div>

          <Modal
            open={showModal()}
            title={t('identity.structure.add_title')}
            onOpenChange={setShowModal}
            footer={
              <FormActions>
                <Button variant="ghost" onClick={() => setShowModal(false)}>
                  {t('identity.common.cancel')}
                </Button>
                <Button loading={loading()} onClick={() => void submitUnit()}>
                  {t('identity.structure.create')}
                </Button>
              </FormActions>
            }
          >
            <form
              class="flex flex-col gap-3.5"
              onSubmit={(e) => {
                e.preventDefault()
                void submitUnit()
              }}
            >
              <FormField label={t('identity.structure.name')} required>
                <Input value={newName()} onChange={setNewName} placeholder={t('identity.structure.name_placeholder')} />
              </FormField>
              <FormField label={t('identity.structure.field.type')} required>
                <Select
                  value={newType()}
                  options={ORG_UNIT_TYPES.map((u) => ({ value: u.value, label: t(u.key) }))}
                  onChange={setNewType}
                />
              </FormField>
              <FormField label={t('identity.structure.parent')} hint={t('identity.structure.parent_hint')}>
                <Select
                  value={newParentId()}
                  placeholder={t('identity.structure.root')}
                  options={flatUnits().map((u) => ({
                    value: u.id,
                    label: `${'  '.repeat(u.depth)}${u.name} (${formatUnitType(u.type, t)})`,
                  }))}
                  onChange={setNewParentId}
                />
              </FormField>
            </form>
          </Modal>
        </Show>
      }>
        <Alert variant="danger">{identity.error}</Alert>
      </Show>

      <KAppStatus />

      <style>{`
        .detail-list { display: grid; gap: 0.75rem; margin: 0; }
        .detail-list div { display: grid; grid-template-columns: 6rem 1fr; gap: 0.5rem; font-size: 0.875rem; }
        .detail-list dt { font-weight: 600; color: var(--kg-text-muted); }
        .detail-list dd { margin: 0; color: var(--kg-text); }
        .detail-list .mono { font-family: var(--kg-font-mono); font-size: 0.8125rem; word-break: break-all; }
      `}</style>
    </Show>
  )
}
