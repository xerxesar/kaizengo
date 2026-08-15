<script lang="ts">
  import OperationsBoard from '../lib/OperationsBoard.svelte'
  import { t } from '@kaizengo/sdk-svelte/ui'
</script>

<OperationsBoard
  pickingType="internal"
  title={t('inventory.transfers.title')}
  subtitle={t('inventory.transfers.subtitle')}
/>
