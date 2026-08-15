<script lang="ts">
  import OperationsBoard from '../lib/OperationsBoard.svelte'
  import { t } from '@kaizengo/sdk-svelte/ui'
</script>

<OperationsBoard
  pickingType="outgoing"
  title={t('inventory.fulfillment.title')}
  subtitle={t('inventory.fulfillment.subtitle')}
/>
