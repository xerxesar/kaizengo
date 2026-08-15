<script lang="ts">
  import ModelWorkspace from '../lib/ModelWorkspace.svelte'
  import { t } from '@kaizengo/sdk-svelte/ui'
</script>

<ModelWorkspace
  model="inventory.iot_device"
  title={t('inventory.hardware.title')}
  subtitle={t('inventory.hardware.subtitle')}
  emptyKey="inventory.empty.hardware"
/>
