<script lang="ts">
  import type { ButtonSize, ButtonVariant } from './types'

  type Props = {
    variant?: ButtonVariant
    size?: ButtonSize
    type?: 'button' | 'submit' | 'reset'
    disabled?: boolean
    loading?: boolean
    onclick?: (e: MouseEvent) => void
    children: import('svelte').Snippet
  }

  let {
    variant = 'primary',
    size = 'md',
    type = 'button',
    disabled = false,
    loading = false,
    onclick,
    children,
  }: Props = $props()
</script>

<button {type} class="kg-btn {variant} {size}" disabled={disabled || loading} {onclick}>
  {#if loading}<span class="kg-btn-spinner" aria-hidden="true"></span>{/if}
  {@render children()}
</button>

<style>
  .kg-btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--kg-space-03);
    border: 1px solid transparent;
    border-radius: var(--kg-radius);
    font-family: var(--kg-font);
    font-weight: 400;
    letter-spacing: 0.16px;
    cursor: pointer;
    white-space: nowrap;
    line-height: 1.28571;
    transition: background 70ms cubic-bezier(0.2, 0, 0.38, 0.9),
      border-color 70ms cubic-bezier(0.2, 0, 0.38, 0.9),
      color 70ms cubic-bezier(0.2, 0, 0.38, 0.9);
  }

  .kg-btn:focus {
    outline: none;
  }

  .kg-btn:focus-visible {
    outline: var(--kg-focus-ring);
    outline-offset: var(--kg-focus-offset);
  }

  .kg-btn:disabled {
    opacity: 1;
    color: var(--kg-text-muted);
    cursor: not-allowed;
  }

  .kg-btn.sm {
    height: var(--kg-control-height-sm);
    padding: 0 var(--kg-space-05);
    font-size: 0.875rem;
  }

  .kg-btn.md {
    height: var(--kg-control-height);
    padding: 0 var(--kg-space-05);
    font-size: 0.875rem;
  }

  .kg-btn.primary {
    background: var(--kg-primary);
    color: var(--kg-text-on-color);
    border-color: var(--kg-primary);
  }

  .kg-btn.primary:hover:not(:disabled) {
    background: var(--kg-primary-hover);
    border-color: var(--kg-primary-hover);
  }

  .kg-btn.primary:active:not(:disabled) {
    background: var(--kg-primary-active);
    border-color: var(--kg-primary-active);
  }

  .kg-btn.primary:disabled {
    background: var(--kg-border);
    border-color: var(--kg-border);
    color: var(--kg-text-muted);
  }

  .kg-btn.secondary {
    background: transparent;
    color: var(--kg-primary);
    border-color: var(--kg-primary);
  }

  .kg-btn.secondary:hover:not(:disabled) {
    background: var(--kg-primary);
    color: var(--kg-text-on-color);
  }

  .kg-btn.secondary:disabled {
    background: transparent;
    border-color: var(--kg-border);
    color: var(--kg-text-muted);
  }

  .kg-btn.ghost {
    background: transparent;
    color: var(--kg-primary);
    border-color: transparent;
  }

  .kg-btn.ghost:hover:not(:disabled) {
    background: var(--kg-surface-hover);
  }

  .kg-btn.ghost:disabled {
    color: var(--kg-text-muted);
  }

  .kg-btn.danger {
    background: var(--kg-danger);
    color: var(--kg-text-on-color);
    border-color: var(--kg-danger);
  }

  .kg-btn.danger:hover:not(:disabled) {
    background: #b81922;
    border-color: #b81922;
  }

  .kg-btn-spinner {
    width: 1rem;
    height: 1rem;
    border: 2px solid rgba(255, 255, 255, 0.35);
    border-top-color: currentColor;
    border-radius: 50%;
    animation: kg-spin 0.6s linear infinite;
  }

  @keyframes kg-spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>
