<script lang="ts">
  type Props = {
    title: string
    description?: string
    children: import('svelte').Snippet
  }

  let { title, description, children }: Props = $props()
</script>

<fieldset class="kg-form-section">
  <legend class="kg-form-section-title">{title}</legend>
  {#if description}<p class="kg-form-section-desc">{description}</p>{/if}
  <div class="kg-form-section-fields">{@render children()}</div>
</fieldset>

<style>
  .kg-form-section {
    border: 0;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: var(--kg-space-05);
  }

  .kg-form-section-title {
    font-size: 0.875rem;
    font-weight: 600;
    letter-spacing: 0.16px;
    color: var(--kg-text);
    padding: 0;
  }

  .kg-form-section-desc {
    font-size: 0.875rem;
    letter-spacing: 0.16px;
    color: var(--kg-text-secondary);
  }

  .kg-form-section-fields {
    display: flex;
    flex-direction: column;
    gap: var(--kg-space-05);
  }
</style>
