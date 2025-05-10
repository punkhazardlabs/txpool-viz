<script lang="ts">
  import phlLogo from "./assets/phl.png";
  import TransactionsView from "./routes/TransactionsView.svelte";
</script>

<main>
  <div>
    <a href="https://vite.dev" target="_blank" rel="noreferrer">
      <img src={phlLogo} class="logo" alt="Vite Logo" />
    </a>
  </div>
  <h1>txpool-viz</h1>

  <div class="card">
    <TransactionsView />
  </div>
</main>

<style>
  .logo {
    height: 6em;
    padding: 1.5em;
    will-change: filter;
    transition: filter 300ms;
  }
  .logo:hover {
    filter: drop-shadow(0 0 2em #580132);
  }
  .logo.svelte:hover {
    filter: drop-shadow(0 0 2em #ff3e00aa);
  }
  .read-the-docs {
    color: #888;
  }
</style>
