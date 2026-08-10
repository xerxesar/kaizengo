<script lang="ts">
  import { onMount } from 'svelte'
  import { addCounter, fetchCounter, resetCounter } from './graphql'

  let count = $state(0)
  let spins = $state(0)
  let loading = $state(true)
  let error = $state('')

  async function refresh() {
    loading = true
    error = ''
    try {
      const data = await fetchCounter()
      count = data.counter
    } catch (e) {
      error = e instanceof Error ? e.message : String(e)
    } finally {
      loading = false
    }
  }

  async function bump(by: number) {
    loading = true
    error = ''
    try {
      const data = await addCounter(by)
      count = data.addCounter
      spins += 1
    } catch (e) {
      error = e instanceof Error ? e.message : String(e)
    } finally {
      loading = false
    }
  }

  async function reset() {
    loading = true
    error = ''
    try {
      const data = await resetCounter()
      count = data.resetCounter
      spins += 1
    } catch (e) {
      error = e instanceof Error ? e.message : String(e)
    } finally {
      loading = false
    }
  }

  onMount(() => {
    void refresh()
  })
</script>

<div class="counter">
  <h1>Svelte Counter</h1>
  <p>State lives on the server via GraphQL (<code>counter</code> / <code>addCounter</code>).</p>
  <p class="value" class:dim={loading} style="transform: rotate({spins * 8}deg)">{count}</p>
  {#if error}
    <p class="err">{error}</p>
  {/if}
  <div class="row">
    <button type="button" disabled={loading} onclick={() => bump(-1)}>−1</button>
    <button type="button" disabled={loading} onclick={() => bump(1)}>+1</button>
    <button type="button" class="ghost" disabled={loading} onclick={() => reset()}>Reset</button>
  </div>
</div>

<style>
  .counter {
    text-align: center;
  }
  .value {
    font-size: 3rem;
    font-weight: 700;
    margin: 1rem 0;
    display: inline-block;
    transition: transform 0.2s ease;
  }
  .value.dim {
    opacity: 0.45;
  }
  .err {
    color: #b00020;
  }
  .row {
    display: flex;
    gap: 0.5rem;
    justify-content: center;
  }
  button {
    padding: 0.45rem 0.85rem;
    border: 1px solid #111;
    border-radius: 0.25rem;
    background: #111;
    color: #fff;
    cursor: pointer;
  }
  button:disabled {
    opacity: 0.5;
    cursor: wait;
  }
  button.ghost {
    background: #fff;
    color: #111;
  }
</style>
