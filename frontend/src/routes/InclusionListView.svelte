<script>
  import { onMount } from "svelte";
  import { writable } from "svelte/store";

  const inclusionReports = writable([]);

  let interval;

  async function fetchReports() {
    try {
      const res = await fetch("/api/inclusion-lists");
      if (res.ok) {
        const data = await res.json();
        inclusionReports.set(data);
      } else {
        console.error("Failed to fetch inclusion reports");
      }
    } catch (e) {
      console.error("Error fetching reports", e);
    }
  }

  onMount(() => {
    fetchReports();
    interval = setInterval(fetchReports, 6000);
    return () => clearInterval(interval);
  });
</script>

{#if $inclusionReports.length > 0}
  {#each $inclusionReports as report}
    <div class="report">
      <div class="slot">
        ⛓️ Slot {report.slot}
        {#if report.report.summary.included === report.report.summary.total}
          <span class="valid-tag">Valid</span>
        {:else}
          <span class="invalid-tag">Incomplete</span>
        {/if}
      </div>

      <div>
        Included: {report.report.summary.included} / {report.report.summary
          .total}
      </div>
      <div>Missing: {report.report.summary.missing}</div>

      <div class="included">
        <h4>✅ Included Hashes:</h4>
        {#each report.report.included as tx, i}
          <div>{i + 1}: {tx}</div>
        {/each}
      </div>

      <div class="missing">
        <h4>❌ Missing Hashes:</h4>
        {#if report.report.missing && report.report.missing.length > 0}
          {#each report.report.missing as tx, i}
            <div>{i + 1}: {tx}</div>
          {/each}
        {:else}
          <div>None</div>
        {/if}
      </div>
    </div>
  {/each}
{:else}
  <div>Loading inclusion reports...</div>
{/if}

<style>
  .report {
    padding: 1rem;
    border: 1px solid #333;
    margin-bottom: 1rem;
    border-radius: 8px;
    background: #f9f9f9;
  }
  .slot {
    font-weight: bold;
    margin-bottom: 0.5rem;
  }
  .included,
  .missing {
    margin-top: 1rem;
    padding: 0.5rem;
    border: 1px solid #ccc;
    border-radius: 6px;
  }
  .valid-tag {
    background-color: #d4edda;
    color: #155724;
    padding: 2px 6px;
    margin-left: 8px;
    border-radius: 4px;
    font-size: 0.85rem;
  }

  .invalid-tag {
    background-color: #f8d7da;
    color: #721c24;
    padding: 2px 6px;
    margin-left: 8px;
    border-radius: 4px;
    font-size: 0.85rem;
  }
</style>
