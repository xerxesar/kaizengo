<script lang="ts">
  import { onMount } from 'svelte'
  import { ping } from './graphql'

  let message = $state('…')
  let error = $state('')

  onMount(async () => {
    try {
      message = await ping()
    } catch (e) {
      error = e instanceof Error ? e.message : String(e)
    }
  })
</script>

<div class="wrap">
  <h1>Notes</h1>
  <p>Bootstrapped Svelte app <code>notes</code>.</p>
  {#if error}
    <p class="err">{error}</p>
  {:else}
    <p>GraphQL: {message}</p>
  {/if}
</div>

<style>
  .wrap {
    text-align: center;
  }
  .err {
    color: #b00020;
  }
</style>
