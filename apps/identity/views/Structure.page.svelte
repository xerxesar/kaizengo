<script lang="ts">
  import { onMount } from 'svelte'
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
  } from '@kaizengo/sdk-svelte/ui'
  import IdentityToolbar from '../lib/IdentityToolbar.svelte'
  import {
    ORG_UNIT_TYPES,
    createOrgUnit,
    fetchOrgTree,
    flattenTree,
    formatUnitType,
    type OrgUnitNode,
  } from '../lib/graphql'
  import { identityState, initIdentity } from '../lib/state.svelte'

  const identity = identityState()

  let tree = $state<OrgUnitNode[]>([])
  let loading = $state(true)
  let error = $state('')
  let selectedId = $state<string | null>(null)
  let showModal = $state(false)
  let newName = $state('')
  let newType = $state('department')
  let newParentId = $state('')

  const treeNodes = $derived(toTreeNodes(tree))
  const flatUnits = $derived(flattenTree(tree))
  const selectedNode = $derived(flatUnits.find((u) => u.id === selectedId) ?? null)
  const org = $derived(identity.selectedOrg)

  function toTreeNodes(nodes: OrgUnitNode[]): TreeNode<OrgUnitNode>[] {
    return nodes.map((n) => ({
      id: n.id,
      label: n.name,
      meta: formatUnitType(n.type, t),
      data: n,
      children: n.children?.length ? toTreeNodes(n.children) : undefined,
    }))
  }

  function countUnits(nodes: OrgUnitNode[]): number {
    let n = 0
    for (const node of nodes) {
      n += 1
      n += countUnits(node.children ?? [])
    }
    return n
  }

  async function load() {
    if (!org) return
    loading = true
    error = ''
    try {
      const data = await fetchOrgTree(org.id)
      tree = data.orgTree
      identity.onStats({ units: flatUnits.length || countUnits(tree) })
    } catch (e) {
      error = e instanceof Error ? e.message : String(e)
    } finally {
      loading = false
    }
  }

  async function submitUnit() {
    if (!org || !newName.trim()) return
    loading = true
    error = ''
    try {
      await createOrgUnit(org.id, newType, newName.trim(), newParentId || undefined)
      newName = ''
      showModal = false
      await load()
    } catch (e) {
      error = e instanceof Error ? e.message : String(e)
      loading = false
    }
  }

  $effect(() => {
    org?.id
    if (org && identity.ready) void load()
  })

  onMount(async () => {
    await initIdentity()
  })
</script>

{#if identity.loading}
  <Spinner />
{:else}
  <IdentityToolbar />

  {#if identity.error}
    <Alert variant="danger">{identity.error}</Alert>
  {:else if !org}
    <Alert variant="warning">{t('identity.no_org')}</Alert>
  {:else}
    {#if error}
      <Alert variant="danger" dismissible ondismiss={() => (error = '')}>{error}</Alert>
    {/if}

    <Toolbar>
      {#snippet start()}
        <span class="toolbar-label">{t('identity.structure.count', flatUnits.length)}</span>
      {/snippet}
      {#snippet end()}
        <Button onclick={() => (showModal = true)}>{t('identity.structure.add')}</Button>
      {/snippet}
    </Toolbar>

    <div class="layout">
      <div class="tree-col">
        {#if loading && tree.length === 0}
          <Spinner size="sm" />
        {:else}
          <TreeView
            nodes={treeNodes}
            selectedId={selectedId}
            onselect={(node) => (selectedId = node.id)}
          />
        {/if}
      </div>

      <div class="detail-col">
        {#if selectedNode}
          <Card title={selectedNode.name}>
            <dl class="detail-list">
              <div><dt>{t('identity.structure.field.type')}</dt><dd>{formatUnitType(selectedNode.type, t)}</dd></div>
              <div><dt>{t('identity.structure.field.id')}</dt><dd class="mono">{selectedNode.id}</dd></div>
              <div>
                <dt>{t('identity.structure.field.parent')}</dt>
                <dd>
                  {selectedNode.parentId
                    ? flatUnits.find((u) => u.id === selectedNode.parentId)?.name ?? selectedNode.parentId
                    : t('identity.structure.root')}
                </dd>
              </div>
              <div>
                <dt>{t('identity.structure.field.created')}</dt>
                <dd>{selectedNode.createdAt ? new Date(selectedNode.createdAt).toLocaleDateString() : '—'}</dd>
              </div>
            </dl>
          </Card>
        {:else}
          <Card title={t('identity.structure.details')}>
            <p class="hint">{t('identity.structure.select_hint')}</p>
          </Card>
        {/if}
      </div>
    </div>

    <Modal bind:open={showModal} title={t('identity.structure.add_title')} size="md">
      <form
        class="unit-form"
        onsubmit={(e) => {
          e.preventDefault()
          void submitUnit()
        }}
      >
        <FormField label={t('identity.structure.name')} required>
          {#snippet children()}
            <Input bind:value={newName} placeholder={t('identity.structure.name_placeholder')} />
          {/snippet}
        </FormField>
        <FormField label={t('identity.structure.field.type')} required>
          {#snippet children()}
            <Select
              bind:value={newType}
              options={ORG_UNIT_TYPES.map((u) => ({ value: u.value, label: t(u.key) }))}
            />
          {/snippet}
        </FormField>
        <FormField label={t('identity.structure.parent')} hint={t('identity.structure.parent_hint')}>
          {#snippet children()}
            <Select
              bind:value={newParentId}
              placeholder={t('identity.structure.root')}
              options={flatUnits.map((u) => ({
                value: u.id,
                label: `${'  '.repeat(u.depth)}${u.name} (${formatUnitType(u.type, t)})`,
              }))}
            />
          {/snippet}
        </FormField>
      </form>

      {#snippet footer()}
        <FormActions>
          <Button variant="ghost" onclick={() => (showModal = false)}>{t('identity.common.cancel')}</Button>
          <Button type="submit" loading={loading} onclick={() => void submitUnit()}>{t('identity.structure.create')}</Button>
        </FormActions>
      {/snippet}
    </Modal>
  {/if}
{/if}

<KAppStatus />

<style>
  .toolbar-label {
    font-size: 0.875rem;
    color: var(--kg-text-secondary);
    font-weight: 500;
  }
  .layout {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--kg-space-05);
    width: 100%;
  }
  @media (max-width: 800px) {
    .layout { grid-template-columns: 1fr; }
  }
  .detail-list {
    display: grid;
    gap: 0.75rem;
    margin: 0;
  }
  .detail-list div {
    display: grid;
    grid-template-columns: 6rem 1fr;
    gap: 0.5rem;
    font-size: 0.875rem;
  }
  .detail-list dt {
    font-weight: 600;
    color: var(--kg-text-muted);
  }
  .detail-list dd {
    margin: 0;
    color: var(--kg-text);
  }
  .detail-list .mono {
    font-family: var(--kg-font-mono);
    font-size: 0.8125rem;
    word-break: break-all;
  }
  .hint {
    color: var(--kg-text-muted);
    font-size: 0.875rem;
  }
  .unit-form {
    display: flex;
    flex-direction: column;
    gap: 0.875rem;
  }
</style>
