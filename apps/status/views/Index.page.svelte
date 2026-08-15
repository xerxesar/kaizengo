<script lang="ts">
  import { KAppStatus, t } from '@kaizengo/sdk-svelte/ui'
</script>

<p class="lead">{t('status.lead')}</p>

<KAppStatus />

<style>
  .lead {
    margin: 0 0 var(--kg-space-05);
    color: var(--kg-text-secondary);
  }
</style>
