<script lang="ts">
  import type { ThemeId } from './theme'

  type Props = {
    theme?: ThemeId
    children: import('svelte').Snippet
  }

  let { theme, children }: Props = $props()
</script>

<div class="kg-page kg-app" data-kg-theme={theme}>
  {@render children()}
</div>

<style>
  .kg-page {
    display: flex;
    flex-direction: column;
    width: 100%;
    min-height: 100vh;
    background: var(--kg-bg);
  }
</style>
