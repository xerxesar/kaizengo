<script lang="ts">
  type Option = { value: string; label: string }

  type Props = {
    value?: string
    options: Option[]
    placeholder?: string
    disabled?: boolean
    onchange?: (e: Event) => void
  }

  let {
    value = $bindable(''),
    options,
    placeholder = '',
    disabled = false,
    onchange,
  }: Props = $props()
</script>

<select class="kg-select" bind:value {disabled} {onchange}>
  {#if placeholder}
    <option value="">{placeholder}</option>
  {/if}
  {#each options as opt (opt.value)}
    <option value={opt.value}>{opt.label}</option>
  {/each}
</select>

<style>
  .kg-select {
    width: 100%;
    height: var(--kg-control-height);
    padding: 0 var(--kg-space-07) 0 var(--kg-space-05);
    border: none;
    border-bottom: 1px solid var(--kg-border-strong);
    border-radius: var(--kg-radius);
    background: var(--kg-field);
    color: var(--kg-text);
    font-size: 0.875rem;
    line-height: 1.29;
    letter-spacing: 0.16px;
    cursor: pointer;
    appearance: none;
    background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='16' height='16' viewBox='0 0 16 16' fill='%23525252'%3E%3Cpath d='M8 11 3 6h10z'/%3E%3C/svg%3E");
    background-repeat: no-repeat;
    background-position: right var(--kg-space-04) center;
    transition: background-color 70ms cubic-bezier(0.2, 0, 0.38, 0.9);
  }

  :global([dir='rtl']) .kg-select {
    background-position: left var(--kg-space-04) center;
  }

  .kg-select:hover:not(:disabled) {
    background-color: var(--kg-field-hover);
  }

  .kg-select:focus {
    outline: var(--kg-focus-ring);
    outline-offset: var(--kg-focus-offset);
    border-bottom-color: var(--kg-primary);
  }

  .kg-select:disabled {
    background-color: var(--kg-surface-muted);
    border-bottom-color: transparent;
    color: var(--kg-text-muted);
    cursor: not-allowed;
  }
</style>
