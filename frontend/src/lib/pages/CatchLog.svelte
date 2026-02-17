<script lang="ts">
  import { catches, loading, loadCatches, deleteCatch } from '../stores/catches';

  let { onEdit }: { onEdit: (id: number) => void } = $props();
  let search = $state('');

  $effect(() => {
    loadCatches();
  });

  let filtered = $derived(
    $catches.filter((c) => {
      if (!search) return true;
      const q = search.toLowerCase();
      return (
        (c.species_name || '').toLowerCase().includes(q) ||
        (c.location_name || '').toLowerCase().includes(q) ||
        (c.bait_or_lure || '').toLowerCase().includes(q)
      );
    }),
  );

  async function handleDelete(id: number) {
    if (confirm('Delete this catch?')) {
      await deleteCatch(id);
    }
  }
</script>

<div class="page">
  <h1 class="page-title">Catch Log</h1>

  <div class="form-group">
    <input type="text" placeholder="Search catches..." bind:value={search} />
  </div>

  {#if $loading}
    <div class="empty-state"><p>Loading...</p></div>
  {:else if filtered.length === 0}
    <div class="empty-state">
      <p>No catches yet</p>
      <p>Hit the + button to log your first catch!</p>
    </div>
  {:else}
    {#each filtered as c (c.id)}
      <div class="card catch-card" onclick={() => onEdit(c.id)}>
        <div class="catch-header">
          <span class="catch-species">{c.species_name || 'Unknown Species'}</span>
          <span class="catch-date">{new Date(c.caught_at).toLocaleDateString()}</span>
        </div>
        <div class="catch-details">
          {#if c.location_name}
            <span>{c.location_name}</span>
          {/if}
          {#if c.weight_lb}
            <span>{c.weight_lb} lb</span>
          {/if}
          {#if c.length_in}
            <span>{c.length_in}"</span>
          {/if}
          {#if c.bait_or_lure}
            <span class="chip chip-primary">{c.bait_or_lure}</span>
          {/if}
          {#if c.kept}
            <span class="chip">Kept</span>
          {/if}
        </div>
        {#if c.conditions}
          <div class="catch-weather">
            {c.conditions}
            {#if c.air_temp_f}{c.air_temp_f.toFixed(0)}°F{/if}
          </div>
        {/if}
        <button
          class="delete-btn"
          onclick={(e: MouseEvent) => {
            e.stopPropagation();
            handleDelete(c.id);
          }}>Delete</button
        >
      </div>
    {/each}
  {/if}
</div>

<style>
  .catch-card {
    cursor: pointer;
    transition: box-shadow 0.2s;
  }

  .catch-card:hover {
    box-shadow: 0 2px 8px var(--shadow);
  }

  .catch-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 6px;
  }

  .catch-species {
    font-weight: 700;
    font-size: 1rem;
  }

  .catch-date {
    font-size: 0.8rem;
    color: var(--text-secondary);
  }

  .catch-details {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    font-size: 0.85rem;
    color: var(--text-secondary);
  }

  .catch-weather {
    font-size: 0.8rem;
    color: var(--text-secondary);
    margin-top: 6px;
  }

  .delete-btn {
    margin-top: 8px;
    padding: 4px 12px;
    background: transparent;
    border: 1px solid var(--danger);
    color: var(--danger);
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.8rem;
  }

  .delete-btn:hover {
    background: var(--danger);
    color: white;
  }
</style>
