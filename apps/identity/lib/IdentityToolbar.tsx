import { Show } from 'solid-js'
import { Select } from '@kaizengo/sdk-solid/ui'
import { identityState, selectOrg } from './state'

export function IdentityToolbar() {
  const identity = identityState()

  return (
    <Show
      when={identity.orgs.length > 1}
      fallback={
        <Show when={identity.selectedOrg}>
          <div class="mb-5 flex justify-end">
            <span class="inline-flex items-center bg-[var(--kg-primary-subtle)] px-5 py-2 text-sm font-semibold text-[var(--kg-primary)]">
              {identity.selectedOrg!.name}
            </span>
          </div>
        </Show>
      }
    >
      <div class="mb-5 flex justify-end">
        <Select
          value={identity.selectedOrg?.id ?? ''}
          options={identity.orgs.map((o) => ({ value: o.id, label: o.name }))}
          onChange={(value) => selectOrg(value)}
        />
      </div>
    </Show>
  )
}
