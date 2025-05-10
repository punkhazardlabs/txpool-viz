<script lang="ts">
  import { onMount } from "svelte";
  import { fetchTransactions } from "../lib/api";

  let transactions: string[] = [];
  let error: string | null = null;

  onMount(async () => {
    try {
      transactions = await fetchTransactions();
    } catch (err: any) {
      error = err.message;
    }
  });
</script>

<h2 class="text-2xl font-bold mb-4">Transaction Hashes</h2>

{#if error}
  <p class="text-red-500">{error}</p>
{:else if transactions.length === 0}
  <p>Loading transactions...</p>
{:else}
  <table class="table-auto border-collapse border border-gray-400">
    <thead>
      <tr>
        <th class="border border-gray-400 px-4 py-2">#</th>
        <th class="border border-gray-400 px-4 py-2">Transaction Hash</th>
      </tr>
    </thead>
    <tbody>
      {#each transactions as tx, i}
        <tr>
          <td class="border border-gray-400 px-4 py-2">{i + 1}</td>
          <td class="border border-gray-400 px-4 py-2">{tx}</td>
        </tr>
      {/each}
    </tbody>
  </table>
{/if}
