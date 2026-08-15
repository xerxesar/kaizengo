<script lang="ts">
  import ModelWorkspace from '../lib/ModelWorkspace.svelte'
  import { t } from '@kaizengo/sdk-svelte/ui'
</script>

<ModelWorkspace
  model="inventory.uom_group"
  title={t('inventory.uom_groups.title')}
  subtitle={t('inventory.uom_groups.subtitle')}
  emptyKey="inventory.empty.uom_groups"
/>

<ModelWorkspace
  model="inventory.uom"
  title={t('inventory.uoms.title')}
  subtitle={t('inventory.uoms.subtitle')}
  emptyKey="inventory.empty.uoms"
/>
