<script lang="ts">
  import { params } from 'svelte-spa-router';
  import { fetchTxDetails, type ApiTxResponse } from '../lib/api';

  import { onDestroy } from 'svelte';

  let data: ApiTxResponse | null = null;
  let error = '';
  let txHash: string;

  // Subscribe to the params store to get the current txHash
  const unsubscribe = params.subscribe(($params) => {
    txHash = $params?.txHash ?? '';
    if (txHash) {
      data = null;
      error = '';
      fetchTxDetails(txHash)
        .then(res => data = res)
        .catch(e => error = e.message);
    }
  });

  onDestroy(unsubscribe);

  function formatEventName(key: string): string {
    const map: Record<string,string> = {
      timeReceived: 'Time Received',
      timePending:  'Time Pending',
      timeMined:    'Time Mined',
      timeDropped:  'Time Dropped'
    };
    return map[key] ?? key;
  }

  function formatTimestamp(val: any): string {
    if (!val) return '—';
    return new Date(val * 1000).toLocaleTimeString();
  }
</script>

{#if error}
  <p class="error">Error: {error}</p>
{:else if !data}
  <p>Loading transaction details…</p>
{:else}
  <h1>Transaction {data.hash}</h1>

  <!-- Propagation Overview -->
  <section>
    <h2>Propagation Overview</h2>
    <table>
      <thead>
        <tr>
          <th>Event</th>
          {#each data.clients as client}
            <th>{client}</th>
          {/each}
        </tr>
      </thead>
      <tbody>
        {#each ['timeReceived','timePending','timeMined','timeDropped'] as key}
          <tr>
            <td>{formatEventName(key)}</td>
            {#each data.clients as c}
              <td>
                {#if data.diff.metadata[key]}
                  {formatTimestamp(data.diff.metadata[key][c])}
                {:else}
                  {formatTimestamp(data.common.metadata[key])}
                {/if}
              </td>
            {/each}
          </tr>
        {/each}
        <tr>
          <td>Block Number</td>
          <td colspan={data.clients.length}>{data.common.metadata.blockNumber}</td>
        </tr>
        <tr>
          <td>Mine Status</td>
          <td colspan={data.clients.length}>{data.common.metadata.mineStatus}</td>
        </tr>
      </tbody>
    </table>
  </section>

  <!-- Parameter Differences -->
  <section>
    <h2>Parameter Differences</h2>
    <table>
      <thead>
        <tr>
          <th>Field</th>
          {#each data.clients as client}
            <th>{client}</th>
          {/each}
        </tr>
      </thead>
      <tbody>
        {#each Object.entries(data.diff.tx) as [field, map]}
          <tr>
            <td>{field}</td>
            {#each data.clients as c}
              <td class:highlight={map[c] !== data.common.tx[field]}>{map[c]}</td>
            {/each}
          </tr>
        {/each}
        {#each Object.entries(data.diff.metadata) as [field, map]}
          <tr>
            <td>{field}</td>
            {#each data.clients as c}
              <td class:highlight={map[c] !== data.common.metadata[field]}>{map[c]}</td>
            {/each}
          </tr>
        {/each}
      </tbody>
    </table>
  </section>

  <!-- Common Parameters -->
  <details>
    <summary><h2>Common Parameters</h2></summary>
    <dl>
      {#each Object.entries(data.common.tx) as [field, val]}
        <dt>{field}</dt><dd>{val}</dd>
      {/each}
      {#each Object.entries(data.common.metadata) as [field, val]}
        <dt>{field}</dt><dd>{val}</dd>
      {/each}
    </dl>
  </details>
{/if}

<style>
  table { width: 100%; border-collapse: collapse; margin-bottom: 1em; }
  th, td { border: 1px solid #ccc; padding: 6px 8px; }
  .highlight { background: #ffecb3; font-weight: bold; }
  .error { color: red; }
  section { margin-bottom: 2rem; }
  details dl { display: grid; grid-template-columns: max-content 1fr; gap: 4px 8px; }
</style>
