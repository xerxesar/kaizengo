<script lang="ts">
  import ModelWorkspace from '../lib/ModelWorkspace.svelte'
  import { t } from '@kaizengo/sdk-svelte/ui'
</script>

<ModelWorkspace
  model="inventory.adjustment"
  title={t('inventory.adjustments.title')}
  subtitle={t('inventory.adjustments.subtitle')}
  emptyKey="inventory.empty.adjustments"
/>
