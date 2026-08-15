<script lang="ts">
  import ModelWorkspace from '../lib/ModelWorkspace.svelte'
  import { t } from '@kaizengo/sdk-svelte/ui'
</script>

<ModelWorkspace
  model="inventory.rma"
  title={t('inventory.returns.title')}
  subtitle={t('inventory.returns.subtitle')}
  emptyKey="inventory.empty.returns"
/>
