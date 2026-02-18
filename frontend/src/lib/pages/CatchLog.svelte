<script lang="ts">
  import { catches, loading, loadCatches, deleteCatch } from '../stores/catches';
  import CatchDetail from './CatchDetail.svelte';
  import LogCatch from './LogCatch.svelte';
  import { api } from '../api';

  let {
    onEdit,
    viewCatchId = null,
    onViewCatchConsumed,
  }: {
    onEdit?: (id: number) => void;
    viewCatchId?: number | null;
    onViewCatchConsumed?: () => void;
  } = $props();

  let search = $state('');

  $effect(() => {
    loadCatches();
  });

  // Auto-open catch detail when navigated from map
  $effect(() => {
    if (viewCatchId) {
      handleViewDetail(viewCatchId);
      onViewCatchConsumed?.();
    }
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

  // View state management
  let view = $state<'list' | 'detail' | 'edit'>('list');
  let selectedCatchId = $state<number | null>(null);
  let selectedCatch = $state<any | null>(null);
  let loadingDetail = $state(false);

  async function fetchCatch(id: number) {
    loadingDetail = true;
    try {
      const data = await api.catches.get(id);
      selectedCatch = data;
      view = 'detail';
    } catch (err) {
      alert('Failed to load catch details');
      view = 'list';
    } finally {
      loadingDetail = false;
    }
  }

  function handleViewDetail(id: number) {
    selectedCatchId = id;
    fetchCatch(id);
  }

  function handleBackToList() {
    view = 'list';
    selectedCatchId = null;
    selectedCatch = null;
    loadCatches(); // Refresh list
  }

  function handleEditCatch() {
    view = 'edit';
  }

  function handleEditDone() {
    if (selectedCatchId) {
      fetchCatch(selectedCatchId); // Reload catch to show updated data
    }
  }

  async function handleDeleteCatch(id: number) {
    try {
      await deleteCatch(id);
      handleBackToList();
    } catch (err) {
      alert('Failed to delete catch');
    }
  }
</script>

<div class="page">
  {#if view === 'list'}
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
        <div class="card catch-card" onclick={() => handleViewDetail(c.id)}>
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
        </div>
      {/each}
    {/if}
  {:else if view === 'detail'}
    {#if loadingDetail}
      <div class="empty-state"><p>Loading...</p></div>
    {:else if selectedCatch}
      <CatchDetail
        catch={selectedCatch}
        onBack={handleBackToList}
        onEdit={handleEditCatch}
        onDelete={handleDeleteCatch}
      />
    {/if}
  {:else if view === 'edit'}
    <LogCatch catchId={selectedCatchId} mode="edit" onDone={handleEditDone} />
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
</style>
