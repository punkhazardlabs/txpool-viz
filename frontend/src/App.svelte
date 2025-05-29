<script lang="ts">
  import phlLogo from "./assets/phl.png";
  import Router from "svelte-spa-router";
  import routes from "./routes";
</script>

<main>
  <div>
    <a href="https://vite.dev" target="_blank" rel="noreferrer">
      <img src={phlLogo} class="logo" alt="Vite Logo" />
    </a>
  </div>
  <h1>txpool-viz</h1>

  <nav>
    <a href="#/">Transactions</a>
    <a href="#/inclusion-lists">Inclusion List</a>
  </nav>

  <div class="card">
    <Router {routes} />
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

  nav {
    display: flex;
    gap: 1rem;
    margin-bottom: 1rem;
  }
  a {
    text-decoration: none;
    color: #580132;
    font-weight: bold;
  }
</style>
