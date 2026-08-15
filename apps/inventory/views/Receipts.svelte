<script lang="ts">
  import OperationsBoard from '../lib/OperationsBoard.svelte'
  import { t } from '@kaizengo/sdk-svelte/ui'
</script>

<OperationsBoard
  pickingType="incoming"
  title={t('inventory.receipts.title')}
  subtitle={t('inventory.receipts.subtitle')}
/>
