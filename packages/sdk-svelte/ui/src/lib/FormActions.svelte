<script lang="ts">
  type Props = {
    children: import('svelte').Snippet
  }

  let { children }: Props = $props()
</script>

<div class="kg-form-actions">
  {@render children()}
</div>

<style>
  .kg-form-actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
    padding-top: 0.5rem;
  }
</style>
