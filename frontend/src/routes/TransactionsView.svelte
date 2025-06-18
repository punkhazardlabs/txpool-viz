<script lang="ts">
  import { onMount } from "svelte";
  import { fetchTransactions, fetchTxDetails } from "../lib/api";

  let transactions: string[] = [];
  let txDetails: Record<string, any> = {};
  let selectedTx: string | null = null;
  let error: string | null = null;

  onMount(async () => {
    try {
      transactions = await fetchTransactions();
    } catch (err: any) {
      error = err.message;
    }
  });

  async function showDetails(txHash: string) {
    selectedTx = txHash;
    if (!txDetails[txHash]) {
      try {
        const details = await fetchTxDetails(txHash);
        txDetails = { ...txDetails, [txHash]: details };
      } catch (err: any) {
        alert(`Failed to fetch details for ${txHash}`);
      }
    }
  }
</script>

<h2>Transaction Hashes</h2>

{#if error}
  <p style="color: red">{error}</p>
{:else if transactions.length === 0}
  <p>Loading transactions…</p>
{:else}
  <table>
    <thead>
      <tr><th>#</th><th>Hash</th></tr>
    </thead>
    <tbody>
      {#each transactions as tx, i}
        <tr>
          <td>{i + 1}</td>
          <td class="clickable" on:click={() => showDetails(tx)}>{tx}</td>
        </tr>
      {/each}
    </tbody>
  </table>
{/if}

{#if selectedTx}
  <div class="pane">
    <button on:click={() => (selectedTx = null)}>Close</button>
    <h3>Details for {selectedTx}</h3>

    {#if txDetails[selectedTx]}
      {#each Object.entries(txDetails[selectedTx]) as [endpoint, detail]}
        <section class="details-pane">
          <strong>{endpoint}</strong>
          <pre>{JSON.stringify(detail, null, 2)}</pre>
        </section>
      {/each}
    {:else}
      <p>Loading details…</p>
    {/if}
  </div>
{/if}

<style>
  .clickable {
    cursor: pointer;
    color: #1e40af;
    text-decoration: underline;
  }
  .pane {
    position: fixed;
    top: 0;
    right: 0;
    width: 30%;
    height: 100%;
    background: white;
    border-left: 1px solid #ddd;
    padding: 1rem;
    overflow-y: auto;
  }
  .details-pane{
    background: var(--background-light);
    @media (prefers-color-scheme: dark) {
    background: var(--background-dark);
  }
  }
</style>
