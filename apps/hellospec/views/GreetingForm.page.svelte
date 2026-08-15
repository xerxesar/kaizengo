<script lang="ts">
  import {
    KAppStatus,
    KTable,
    KForm,
    KFormField,
    t,
  } from '@kaizengo/sdk-svelte/ui'

  let table = $state<{ refresh: () => Promise<void> }>()

  function onFormSuccess() {
    void table?.refresh()
  }
</script>

<KForm model="hellospec.greeting" onsuccess={onFormSuccess}>
  <KFormField
    field="message"
    label={t('hellospec.create')}
    placeholder={t('hellospec.new_placeholder')}
  />
</KForm>

<KTable
  bind:this={table}
  model="hellospec.greeting"
  emptyMessage={t('hellospec.empty')}
  class="mt-4"
/>


<KAppStatus />
