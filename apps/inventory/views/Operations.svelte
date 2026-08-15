<script lang="ts">
  import OperationsBoard from '../lib/OperationsBoard.svelte'
  import { t } from '@kaizengo/sdk-svelte/ui'
</script>

<OperationsBoard title={t('inventory.operations.title')} subtitle={t('inventory.operations.subtitle')} />
