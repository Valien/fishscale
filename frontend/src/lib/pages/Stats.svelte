<script lang="ts">
  import { api } from '../api';

  let stats = $state<any>(null);
  let loading = $state(true);

  $effect(() => {
    api.stats.get().then(s => {
      stats = s;
      loading = false;
    }).catch(() => {
      loading = false;
    });
  });
</script>

<div class="page">
  <h1 class="page-title">Stats</h1>

  {#if loading}
    <div class="empty-state"><p>Loading...</p></div>
  {:else if !stats}
    <div class="empty-state"><p>Could not load stats</p></div>
  {:else}
    <div class="stats-grid">
      <div class="stat-card card">
        <div class="stat-value">{stats.total_catches}</div>
        <div class="stat-label">Total Catches</div>
      </div>
      <div class="stat-card card">
        <div class="stat-value">{stats.total_species}</div>
        <div class="stat-label">Species Caught</div>
      </div>
      <div class="stat-card card">
        <div class="stat-value">{stats.total_trips}</div>
        <div class="stat-label">Trips</div>
      </div>
    </div>

    {#if stats.species_counts?.length > 0}
      <div class="card">
        <h2 class="section-title">Top Species</h2>
        {#each stats.species_counts as s}
          <div class="stat-row">
            <span class="stat-row-label">{s.species_name}</span>
            <div class="stat-bar-wrapper">
              <div
                class="stat-bar"
                style="width: {(s.count / stats.species_counts[0].count) * 100}%"
              ></div>
            </div>
            <span class="stat-row-value">{s.count}</span>
          </div>
        {/each}
      </div>
    {/if}

    {#if stats.personal_bests?.length > 0}
      <div class="card">
        <h2 class="section-title">Personal Bests</h2>
        {#each stats.personal_bests as pb}
          <div class="stat-row">
            <span class="stat-row-label">{pb.species_name}</span>
            <span class="stat-row-value">
              {#if pb.max_weight_lb > 0}{pb.max_weight_lb.toFixed(1)} lb{/if}
              {#if pb.max_length_in > 0} {pb.max_length_in.toFixed(1)}"{/if}
            </span>
          </div>
        {/each}
      </div>
    {/if}

    {#if stats.bait_counts?.length > 0}
      <div class="card">
        <h2 class="section-title">Top Baits / Lures</h2>
        {#each stats.bait_counts as b}
          <div class="stat-row">
            <span class="stat-row-label">{b.bait_or_lure}</span>
            <span class="stat-row-value">{b.count}</span>
          </div>
        {/each}
      </div>
    {/if}

    {#if stats.monthly_counts?.length > 0}
      <div class="card">
        <h2 class="section-title">Catches by Month</h2>
        <div class="chart">
          {#each stats.monthly_counts as m}
            <div class="chart-bar-group">
              <div
                class="chart-bar"
                style="height: {Math.max(4, (m.count / Math.max(...stats.monthly_counts.map((mc: any) => mc.count))) * 100)}%"
              ></div>
              <div class="chart-label">{m.month.slice(5)}</div>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <div style="text-align: center; margin-top: 16px;">
      <a href={api.export.csv()} class="btn btn-outline" download>Export CSV</a>
      <a href={api.export.json()} class="btn btn-outline" download style="margin-left: 8px;">Export JSON</a>
    </div>
  {/if}
</div>

<style>
  .stats-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 12px;
    margin-bottom: 16px;
  }

  .stat-card {
    text-align: center;
    padding: 12px;
  }

  .stat-value {
    font-size: 1.8rem;
    font-weight: 800;
    color: var(--primary);
  }

  .stat-label {
    font-size: 0.75rem;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .section-title {
    font-size: 1rem;
    font-weight: 700;
    margin-bottom: 12px;
  }

  .stat-row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 6px 0;
    border-bottom: 1px solid var(--card-border);
  }

  .stat-row:last-child {
    border-bottom: none;
  }

  .stat-row-label {
    flex: 1;
    font-size: 0.9rem;
  }

  .stat-bar-wrapper {
    flex: 1;
    height: 8px;
    background: var(--bg-secondary);
    border-radius: 4px;
    overflow: hidden;
  }

  .stat-bar {
    height: 100%;
    background: var(--primary);
    border-radius: 4px;
    transition: width 0.5s ease;
  }

  .stat-row-value {
    font-weight: 600;
    font-size: 0.9rem;
    min-width: 30px;
    text-align: right;
  }

  .chart {
    display: flex;
    align-items: flex-end;
    gap: 4px;
    height: 120px;
    padding-top: 8px;
  }

  .chart-bar-group {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    height: 100%;
    justify-content: flex-end;
  }

  .chart-bar {
    width: 100%;
    max-width: 32px;
    background: var(--primary);
    border-radius: 4px 4px 0 0;
    transition: height 0.5s ease;
  }

  .chart-label {
    font-size: 0.65rem;
    color: var(--text-secondary);
    margin-top: 4px;
  }
</style>
