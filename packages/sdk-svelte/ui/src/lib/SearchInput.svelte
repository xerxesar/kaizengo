<script lang="ts">
  type Props = {
    value?: string
    placeholder?: string
    oninput?: (value: string) => void
  }

  let { value = $bindable(''), placeholder = 'Search…', oninput }: Props = $props()

  function handleInput(e: Event) {
    const v = (e.currentTarget as HTMLInputElement).value
    value = v
    oninput?.(v)
  }
</script>

<div class="kg-search">
  <svg class="kg-search-icon" viewBox="0 0 16 16" fill="currentColor" aria-hidden="true">
    <path d="M15 14.3 11.2 10.5a5.5 5.5 0 1 0-1.4 1.4L14.3 15zM5.5 9.5a4 4 0 1 1 0-8 4 4 0 0 1 0 8z"/>
  </svg>
  <input
    class="kg-search-input"
    type="search"
    {placeholder}
    bind:value
    oninput={handleInput}
  />
</div>

<style>
  .kg-search {
    position: relative;
    display: flex;
    align-items: center;
    width: 100%;
    max-width: 20rem;
    height: var(--kg-control-height);
  }

  .kg-search-icon {
    position: absolute;
    left: var(--kg-space-05);
    width: 1rem;
    height: 1rem;
    color: var(--kg-text-muted);
    pointer-events: none;
  }

  .kg-search-input {
    width: 100%;
    height: 100%;
    margin: 0;
    padding: 0 var(--kg-space-05) 0 calc(var(--kg-space-05) + 1.25rem);
    border: none;
    border-bottom: 1px solid var(--kg-border-strong);
    border-radius: var(--kg-radius);
    background: var(--kg-field);
    color: var(--kg-text);
    font-size: 0.875rem;
    letter-spacing: 0.16px;
    transition: background 70ms cubic-bezier(0.2, 0, 0.38, 0.9);
  }

  .kg-search-input::placeholder {
    color: var(--kg-text-placeholder);
  }

  .kg-search-input:hover {
    background: var(--kg-field-hover);
  }

  .kg-search-input:focus {
    outline: var(--kg-focus-ring);
    outline-offset: var(--kg-focus-offset);
    border-bottom-color: var(--kg-primary);
  }
</style>
