<script lang="ts">
  import ModelWorkspace from '../lib/ModelWorkspace.svelte'
  import { t } from '@kaizengo/sdk-svelte/ui'
</script>

<ModelWorkspace
  model="inventory.stock_ledger"
  title={t('inventory.ledger.title')}
  subtitle={t('inventory.ledger.subtitle')}
  emptyKey="inventory.empty.ledger"
  deletable={false}
  showForm={false}
/>
