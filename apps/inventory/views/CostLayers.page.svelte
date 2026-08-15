<script lang="ts">
  import ModelWorkspace from '../lib/ModelWorkspace.svelte'
  import { t } from '@kaizengo/sdk-svelte/ui'
</script>

<ModelWorkspace
  model="inventory.cost_layer"
  title={t('inventory.cost_layers.title')}
  subtitle={t('inventory.cost_layers.subtitle')}
  emptyKey="inventory.empty.cost_layers"
  deletable={false}
  showForm={false}
/>
