<script lang="ts">
  import {
    KAppStatus,
    KForm,
    KFormField,
    KTable,
    t,
  } from '@kaizengo/sdk-svelte/ui'

  let table = $state<{ refresh: () => Promise<void> }>()

  function onFormSuccess() {
    void table?.refresh()
  }
</script>

<KForm
  model="identity.user"
  submitLabel={t('identity.users.create')}
  successMessage={t('identity.created')}
  onsuccess={onFormSuccess}
>
  <KFormField
    field="name"
    label={t('identity.users.name')}
    placeholder={t('identity.users.placeholder.name')}
  />
  <KFormField
    field="email"
    label={t('identity.users.email')}
    placeholder={t('identity.users.placeholder.email')}
  />
</KForm>

<KTable
  bind:this={table}
  model="identity.user"
  emptyMessage={t('identity.users.empty')}
  deletable={false}
  class="mt-4"
/>

<KAppStatus />
