<script lang="ts">
  import ModelWorkspace from '../lib/ModelWorkspace.svelte'
  import { t } from '@kaizengo/sdk-svelte/ui'
</script>

<ModelWorkspace
  model="inventory.stock_quant"
  title={t('inventory.stock.title')}
  subtitle={t('inventory.stock.subtitle')}
  emptyKey="inventory.empty.stock"
  deletable={false}
  showForm={false}
/>
