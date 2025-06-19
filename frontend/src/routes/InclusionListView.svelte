<script lang="ts">
  import { onMount } from "svelte";
  import { writable } from "svelte/store";

  // store for fetched reports
  const inclusionReports = writable([]);

  let focilEnabled: boolean | null = null;
  let interval: ReturnType<typeof setInterval>;

  // load feature flag
  async function loadFocilFlag() {
    try {
      const res = await fetch("/api/feature/focil");
      if (!res.ok) throw new Error();
      const { status } = (await res.json()) as { status: boolean };
      focilEnabled = status;
    } catch {
      focilEnabled = false;
    }
  }

  // fetch inclusion list reports
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

  onMount(async () => {
    await loadFocilFlag();

    if (focilEnabled) {
      // initial load + polling
      fetchReports();
      interval = setInterval(fetchReports, 6000);
    }

    return () => {
      if (interval) clearInterval(interval);
    };
  });
</script>

{#if focilEnabled === null}
  <div>Loading feature flag…</div>
{:else if focilEnabled === false}
  <div class="not-enabled">FOCIL monitoring not enabled</div>
{:else}
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
          Included: {report.report.summary.included} / {report.report.summary.total}
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
          {#if report.report.missing?.length}
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
    <div>Loading inclusion reports…</div>
  {/if}
{/if}

<style>
  /* center the “not enabled” box and reports */
  .not-enabled,
  .report {
    max-width: 50%;
    margin: 1.5rem auto;
  }

  .not-enabled {
    padding: 2rem;
    text-align: center;
    font-size: 1.2rem;
    color: #6b7280;
    background: #f3f4f6;
    border-radius: 8px;
  }

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
    background: #fff;
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

  /* Dark mode */
  @media (prefers-color-scheme: dark) {
    .not-enabled {
      background: #1f2937;
      color: #9ca3af;
    }
    .report {
      background: #1f2937;
      border-color: #4b5563;
    }
    .included,
    .missing {
      background: #374151;
      border-color: #4b5563;
    }
    .not-enabled,
    .report,
    .included,
    .missing {
      color: #e5e7eb;
    }
  }
</style>