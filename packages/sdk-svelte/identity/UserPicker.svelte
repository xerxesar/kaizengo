<script lang="ts">
  import Select from '../ui/src/lib/Select.svelte'
  import { fetchActiveUsers, type IdentityUser } from './client'

  type Props = {
    orgId: string
    value?: string
    excludeIds?: string[]
    placeholder?: string
    disabled?: boolean
  }

  let {
    orgId,
    value = $bindable(''),
    excludeIds = [],
    placeholder = 'Select user…',
    disabled = false,
  }: Props = $props()

  let users = $state<IdentityUser[]>([])
  let loading = $state(false)
  let loadError = $state('')

  const options = $derived(
    users
      .filter((u) => !excludeIds.includes(u.id))
      .map((u) => ({ value: u.id, label: `${u.name} — ${u.email}` })),
  )

  async function loadUsers(id: string) {
    if (!id) {
      users = []
      return
    }
    loading = true
    loadError = ''
    try {
      users = await fetchActiveUsers(id)
    } catch (e) {
      loadError = e instanceof Error ? e.message : String(e)
      users = []
    } finally {
      loading = false
    }
  }

  $effect(() => {
    void loadUsers(orgId)
  })
</script>

{#if loadError}
  <p class="kg-user-picker-error">{loadError}</p>
{:else}
  <Select bind:value {options} {placeholder} disabled={disabled || loading || options.length === 0} />
{/if}

<style>
  .kg-user-picker-error {
    margin: 0;
    font-size: 0.875rem;
    color: var(--kg-danger, #da1e28);
  }
</style>
