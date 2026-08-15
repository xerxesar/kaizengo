<script lang="ts">
  type Props = {
    value?: string
    placeholder?: string
    type?: string
    disabled?: boolean
    readonly?: boolean
    oninput?: (e: Event) => void
    onchange?: (e: Event) => void
  }

  let {
    value = $bindable(''),
    placeholder = '',
    type = 'text',
    disabled = false,
    readonly = false,
    oninput,
    onchange,
  }: Props = $props()
</script>

<input
  class="kg-input"
  {type}
  {placeholder}
  {disabled}
  {readonly}
  bind:value
  {oninput}
  {onchange}
/>

<style>
  .kg-input {
    width: 100%;
    height: var(--kg-control-height);
    padding: 0 var(--kg-space-05);
    border: none;
    border-bottom: 1px solid var(--kg-border-strong);
    border-radius: var(--kg-radius);
    background: var(--kg-field);
    color: var(--kg-text);
    font-size: 0.875rem;
    line-height: 1.29;
    letter-spacing: 0.16px;
    transition: background 70ms cubic-bezier(0.2, 0, 0.38, 0.9),
      border-color 70ms cubic-bezier(0.2, 0, 0.38, 0.9);
  }

  .kg-input::placeholder {
    color: var(--kg-text-placeholder);
  }

  .kg-input:hover:not(:disabled):not(:read-only) {
    background: var(--kg-field-hover);
  }

  .kg-input:focus {
    outline: var(--kg-focus-ring);
    outline-offset: var(--kg-focus-offset);
    border-bottom-color: var(--kg-primary);
  }

  .kg-input:disabled,
  .kg-input:read-only {
    background: var(--kg-surface-muted);
    border-bottom-color: transparent;
    color: var(--kg-text-muted);
    cursor: not-allowed;
  }
</style>
