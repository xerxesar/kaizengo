<script lang="ts">
  import { KTable, KAppStatus, t } from '@kaizengo/sdk-svelte/ui'
</script>

<KTable model="hellospec.greeting" emptyMessage={t('hellospec.empty')} />

<KAppStatus />
