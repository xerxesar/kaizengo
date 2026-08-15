<script lang="ts">
  type Props = {
    label: string
    hint?: string
    error?: string
    required?: boolean
    /** Snippet — field control; may be implicit markup or `{#snippet children()}`. */
    children?: import('svelte').Snippet
  }

  let { label, hint, error, required = false, children }: Props = $props()
  const fieldId = `kg-field-${Math.random().toString(36).slice(2, 9)}`
</script>

<div class="kg-field" class:has-error={!!error}>
  <label class="kg-field-label" for={fieldId}>
    {label}
    {#if required}<span class="kg-required" aria-hidden="true">*</span>{/if}
  </label>
  <div class="kg-field-control" id={fieldId}>
    {@render children?.()}
  </div>
  {#if error}
    <p class="kg-field-error">{error}</p>
  {:else if hint}
    <p class="kg-field-hint">{hint}</p>
  {/if}
</div>

<style>
  .kg-field {
    display: flex;
    flex-direction: column;
    gap: var(--kg-space-03);
  }

  .kg-field-label {
    font-size: 0.75rem;
    font-weight: 400;
    letter-spacing: 0.32px;
    color: var(--kg-text-secondary);
  }

  .kg-required {
    color: var(--kg-danger);
    margin-left: var(--kg-space-02);
  }

  .kg-field-hint {
    font-size: 0.75rem;
    letter-spacing: 0.32px;
    color: var(--kg-text-muted);
  }

  .kg-field-error {
    font-size: 0.75rem;
    letter-spacing: 0.32px;
    color: var(--kg-danger);
  }

  .kg-field-control :global(input),
  .kg-field-control :global(select),
  .kg-field-control :global(textarea) {
    width: 100%;
  }

  .kg-field.has-error :global(input),
  .kg-field.has-error :global(select),
  .kg-field.has-error :global(textarea) {
    border-bottom-color: var(--kg-danger);
  }
</style>
