<script lang="ts">
  import { Select } from '@kaizengo/sdk-svelte/ui'
  import { identityState, selectOrg } from './state.svelte'

  const identity = identityState()
</script>

{#if identity.orgs.length > 1}
  <div class="identity-toolbar">
    <Select
      value={identity.selectedOrg?.id ?? ''}
      options={identity.orgs.map((o) => ({ value: o.id, label: o.name }))}
      onchange={(e) => selectOrg((e.currentTarget as HTMLSelectElement).value)}
    />
  </div>
{:else if identity.selectedOrg}
  <div class="identity-toolbar">
    <span class="org-badge">{identity.selectedOrg.name}</span>
  </div>
{/if}

<style>
  .identity-toolbar {
    display: flex;
    justify-content: flex-end;
    margin-bottom: var(--kg-space-05);
  }
  .org-badge {
    display: inline-flex;
    align-items: center;
    padding: var(--kg-space-03) var(--kg-space-05);
    background: var(--kg-primary-subtle);
    color: var(--kg-primary);
    font-size: 0.875rem;
    font-weight: 600;
  }
</style>
