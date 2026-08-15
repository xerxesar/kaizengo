<script lang="ts">
  import ModelWorkspace from '../lib/ModelWorkspace.svelte'
  import { t } from '@kaizengo/sdk-svelte/ui'
</script>

<ModelWorkspace
  model="inventory.attribute"
  title={t('inventory.attributes.title')}
  subtitle={t('inventory.attributes.subtitle')}
  emptyKey="inventory.empty.attributes"
/>

<ModelWorkspace
  model="inventory.attribute_value"
  title={t('inventory.attribute_values.title')}
  subtitle={t('inventory.attribute_values.subtitle')}
  emptyKey="inventory.empty.attribute_values"
/>
