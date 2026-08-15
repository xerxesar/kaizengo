<script lang="ts">
  import ModelWorkspace from '../lib/ModelWorkspace.svelte'
  import { t } from '@kaizengo/sdk-svelte/ui'
</script>

<ModelWorkspace
  model="inventory.integration"
  title={t('inventory.integrations.title')}
  subtitle={t('inventory.integrations.subtitle')}
  emptyKey="inventory.empty.integrations"
/>
