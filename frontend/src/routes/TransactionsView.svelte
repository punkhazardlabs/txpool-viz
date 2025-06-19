<!-- App.svelte -->
<script lang="ts">
  import { onMount } from "svelte";
  import {
    fetchTransactions,
    fetchTxDetails,
    type TxSummary,
    type ApiTxResponse,
  } from "../lib/api";

  let transactions: TxSummary[] = [];
  let filtered: TxSummary[] = [];
  let txDetails: Record<string, ApiTxResponse> = {};
  let selectedTx: string | null = null;
  let error: string | null = null;

  // Filter & sort state
  let nonceExact: number | "" = "";
  let txHashFilter = "";
  let addressFilter = "";
  let typeFilter: string[] = [];

  let sortKey: keyof TxSummary = "hash";
  let sortAsc = true;

  // Define the “natural” ordering for transaction types
  const typeOrder = [
    "LegacyTxType",
    "AccessListTxType",
    "DynamicFeeTxType",
    "BlobTxType",
    "SetCodeTxType",
    "unknown",
  ];

  // load (and reload) the transactions list
  async function loadTransactions() {
    try {
      const list = await fetchTransactions();
      transactions = list || [];
    } catch (e: any) {
      error = e.message;
    }
  }

  onMount(() => {
    // initial load
    loadTransactions();
    // poll every 10 seconds
    const id = setInterval(loadTransactions, 10_000);
    // cleanup on component destroy
    return () => clearInterval(id);
  });

  $: normalizedTypeFilter = typeFilter.map((t) => t.toLowerCase());

  // Derived filtered + sorted, guard against null
  $: filtered = (transactions ?? [])
    .filter((tx) => {
      if (txHashFilter && !tx.hash.includes(txHashFilter)) return false;
      if (nonceExact !== "" && tx.nonce !== +nonceExact) return false;
      if (
        addressFilter &&
        !tx.from.toLowerCase().includes(addressFilter.toLowerCase())
      )
        return false;
      if (
        normalizedTypeFilter.length > 0 &&
        !normalizedTypeFilter.includes(tx.type.toLowerCase())
      )
        return false;
      return true;
    })
    .sort((a, b) => {
      const va = (a[sortKey] ?? "").toString();
      const vb = (b[sortKey] ?? "").toString();

      // custom sort for 'type'
      if (sortKey === "type") {
        const ia = typeOrder.indexOf(va.toLowerCase());
        const ib = typeOrder.indexOf(vb.toLowerCase());
        if (ia < ib) return sortAsc ? -1 : 1;
        if (ia > ib) return sortAsc ? 1 : -1;
        return 0;
      }

      // default numeric or string compare
      if (va < vb) return sortAsc ? -1 : 1;
      if (va > vb) return sortAsc ? 1 : -1;
      return 0;
    });

  function toggleSort(key: keyof TxSummary) {
    if (sortKey === key) sortAsc = !sortAsc;
    else {
      sortKey = key;
      sortAsc = true;
    }
  }

  function showDetails(hash: string) {
    selectedTx = hash;
    if (!txDetails[hash]) {
      fetchTxDetails(hash)
        .then((d) => (txDetails[hash] = d))
        .catch((e: any) => (error = e.message));
    }
  }

  function formatTimestamp(val: any): string {
    if (!val) return "—";
    return new Date(val * 1000).toLocaleTimeString();
  }

  function clearFilters() {
    txHashFilter = "";
    nonceExact = "";
    addressFilter = "";
    typeFilter = [];
  }
</script>

{#if error}
  <p class="error">Error: {error}</p>
{:else}
  <h2>Transactions</h2>
  <div class="filters">
    <input type="text" bind:value={txHashFilter} placeholder="Tx Hash contains…" />
    <input type="number" bind:value={nonceExact} placeholder="Nonce equals…" />
    <input type="text" bind:value={addressFilter} placeholder="From address" />
    <select multiple bind:value={typeFilter} size="4">
      <option value="LegacyTxType">LegacyTxType</option>
      <option value="AccessListTxType">AccessListTxType</option>
      <option value="DynamicFeeTxType">DynamicFeeTxType</option>
      <option value="BlobTxType">BlobTxType</option>
      <option value="SetCodeTxType">SetCodeTxType</option>
    </select>
    <button class="clear-button" on:click={clearFilters}>Clear Filters</button>
  </div>

  <div class="table-container">
    <table>
      <thead>
        <tr>
          <th on:click={() => toggleSort("hash")}>Hash</th>
          <th on:click={() => toggleSort("from")}>From</th>
          <th on:click={() => toggleSort("gasUsed")}>Gas Used</th>
          <th on:click={() => toggleSort("nonce")}>Nonce</th>
          <th on:click={() => toggleSort("type")}>Type</th>
        </tr>
      </thead>
      <tbody>
        {#each filtered as tx}
          <tr class="clickable" on:click={() => showDetails(tx.hash)}>
            <td>{tx.hash}</td>
            <td>{tx.from}</td>
            <td>{tx.gasUsed}</td>
            <td>{tx.nonce}</td>
            <td>{tx.type}</td>
          </tr>
        {:else}
          <tr class="no-data">
            <td colspan="5">No transactions match filters</td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}

<!-- Detail Pane -->
{#if selectedTx}
  <div class="pane">
    <button class="close" on:click={() => (selectedTx = null)}>✕ Close</button>
    <h3>Details: {selectedTx}</h3>

    {#if txDetails[selectedTx]}
      {#await Promise.resolve(txDetails[selectedTx]) then data}
        <!-- 1) Propagation -->
        <section>
          <h2>Propagation</h2>
          <table>
            <thead>
              <tr>
                <th>Client</th>
                <th>Time Received</th>
                <th>Time Pending</th>
                <th>Time Mined</th>
                <th>Time Dropped</th>
              </tr>
            </thead>
            <tbody>
              {#each data.clients as c}
                <tr>
                  <td>{c}</td>
                  <td>{formatTimestamp(data.common.metadata.timeReceived)}</td>
                  <td>
                    {formatTimestamp(
                      data.diff.metadata.timePending?.[c] ??
                        data.common.metadata.timePending
                    )}
                  </td>
                  <td>{formatTimestamp(data.common.metadata.timeMined)}</td>
                  <td>{formatTimestamp(data.common.metadata.timeDropped)}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </section>

        <!-- 2) Parameter Differences -->
        <section class="details-pane">
          <h2>Differences</h2>
          {#if Object.keys(data.diff.tx).length + Object.keys(data.diff.metadata).length > 0}
            <table>
              <thead>
                <tr>
                  <th>Field</th>
                  {#each data.clients as c}<th>{c}</th>{/each}
                </tr>
              </thead>
              <tbody>
                {#each Object.entries(data.diff.tx) as [f, m]}
                  <tr>
                    <td>{f}</td>
                    {#each data.clients as c}
                      <td class:highlight={m[c] !== data.common.tx[f]}>
                        {m[c]}
                      </td>
                    {/each}
                  </tr>
                {/each}
                {#each Object.entries(data.diff.metadata) as [f, m]}
                  <tr>
                    <td>{f}</td>
                    {#each data.clients as c}
                      <td class:highlight={m[c] !== data.common.metadata[f]}>
                        {m[c]}
                      </td>
                    {/each}
                  </tr>
                {/each}
              </tbody>
            </table>
          {:else}
            <p class="no-diff">No parameter differences.</p>
          {/if}
        </section>

        <!-- 3) Common Details -->
        <section>
          <h2>Common Parameters</h2>
          <table>
            <thead>
              <tr><th>Field</th><th>Value</th></tr>
            </thead>
            <tbody>
              {#each Object.entries(data.common.tx) as [field, val]}
                <tr>
                  <td class="field">{field}</td>
                  <td class="value">{val}</td>
                </tr>
              {/each}
              {#each Object.entries(data.common.metadata) as [field, val]}
                <tr>
                  <td class="field">{field}</td>
                  <td class="value">{val}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </section>
      {:catch}
        <p class="error">Failed to load details</p>
      {/await}
    {:else}
      <p>Loading details…</p>
    {/if}
  </div>
{/if}

<style>
  /* Light Mode */
  .table-container {
    min-height: 300px;
    width: 100%;
    overflow: auto;
    background: #ffffff;
    border: 1px solid #e5e7eb;
    border-radius: 4px;
  }
  .no-data td {
    height: 300px;
    vertical-align: middle;
    text-align: center;
    color: #6b7280;
    background: #f3f4f6;
  }
  .filters {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    margin-bottom: 1rem;
    align-items: center;
  }
  .filters input,
  .filters select {
    padding: 0.3rem 0.5rem;
    border: 1px solid #d1d5db;
    border-radius: 4px;
    background: #ffffff;
    color: #1f2937;
  }
  .clear-button {
    padding: 0.4rem 0.8rem;
    border: none;
    border-radius: 4px;
    background: #580132;
    color: #ffffff;
    font-weight: bold;
    cursor: pointer;
    transition: opacity 0.2s;
  }
  .clear-button:hover {
    opacity: 0.85;
  }
  table {
    width: 100%;
    border-collapse: collapse;
    margin-bottom: 1rem;
    background: #f9fafb;
  }
  th,
  td {
    border: 1px solid #d1d5db;
    padding: 0.5rem;
    text-align: left;
    color: #1f2937;
  }
  th {
    cursor: pointer;
    background: #eef2ff;
  }
  tr.clickable:hover {
    background: #e0e7ff;
  }
  .pane {
    position: fixed;
    top: 0;
    right: 0;
    width: 55%;
    height: 100%;
    background: #ffffff;
    border-left: 1px solid #d1d5db;
    padding: 1rem;
    overflow: auto;
  }
  .close {
    float: right;
    background: none;
    border: none;
    font-size: 1.2rem;
    cursor: pointer;
    color: #1f2937;
  }
  .highlight {
    background: #ffecb3;
  }
  .error {
    color: #b91c1c;
  }

  /* Dark Mode */
  @media (prefers-color-scheme: dark) {
    .table-container {
      background: #1f2937;
      border-color: #374151;
    }
    .no-data td {
      background: #111827;
      color: #9ca3af;
    }
    .filters input,
    .filters select {
      background: #374151;
      border-color: #4b5563;
      color: #e5e7eb;
    }
    .clear-button {
      background: #a72ba7;
    }
    table {
      background: #1f2937;
    }
    th,
    td {
      border-color: #4b5563;
      color: #e5e7eb;
    }
    th {
      background: #374151;
    }
    tr.clickable:hover {
      background: #4b5563;
    }
    .pane {
      background: #111827;
      border-left-color: #374151;
    }
    .close {
      color: #e5e7eb;
    }
    .error {
      color: #f87171;
    }
  }
</style>
