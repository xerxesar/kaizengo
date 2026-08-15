<script lang="ts">
  type Props = {
    size?: 'sm' | 'md' | 'lg'
    label?: string
  }

  let { size = 'md', label = 'Loading' }: Props = $props()
</script>

<div class="kg-spinner-wrap" role="status" aria-label={label}>
  <div class="kg-spinner {size}"></div>
</div>

<style>
  .kg-spinner-wrap {
    display: flex;
    justify-content: center;
    padding: var(--kg-space-09);
  }

  .kg-spinner {
    border-radius: 50%;
    border: 3px solid var(--kg-border);
    border-top-color: var(--kg-primary);
    animation: kg-spin 0.75s linear infinite;
  }

  .kg-spinner.sm { width: 1.25rem; height: 1.25rem; }
  .kg-spinner.md { width: 2.5rem; height: 2.5rem; }
  .kg-spinner.lg { width: 3.5rem; height: 3.5rem; }

  @keyframes kg-spin {
    to { transform: rotate(360deg); }
  }
</style>
