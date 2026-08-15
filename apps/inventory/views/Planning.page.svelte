<script lang="ts">
  import ModelWorkspace from '../lib/ModelWorkspace.svelte'
  import { t } from '@kaizengo/sdk-svelte/ui'
</script>

<ModelWorkspace
  model="inventory.reorder_rule"
  title={t('inventory.planning.title')}
  subtitle={t('inventory.planning.subtitle')}
  emptyKey="inventory.empty.planning"
/>
