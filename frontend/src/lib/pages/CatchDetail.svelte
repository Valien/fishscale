<script lang="ts">
  interface Catch {
    id: number;
    species_name: string;
    caught_at: string;
    location_name: string;
    latitude: number | null;
    longitude: number | null;
    length_in: number | null;
    weight_lb: number | null;
    kept: boolean;
    bait_or_lure: string;
    rod_setup: string;
    line_info: string;
    hook_size: string;
    air_temp_f: number | null;
    wind_mph: number | null;
    wind_dir: string;
    conditions: string;
    pressure_mb: number | null;
    humidity_pct: number | null;
    water_temp_f: number | null;
    water_clarity: string;
    notes: string;
    photos: Array<{ id: number; url: string; filename: string }>;
  }

  let {
    catch: catchData,
    onBack,
    onEdit,
    onDelete,
  }: {
    catch?: Catch;
    onBack: () => void;
    onEdit: () => void;
    onDelete: (id: number) => void;
  } = $props();

  let showPhotoModal = $state(false);
  let selectedPhotoUrl = $state('');

  function viewPhoto(url: string) {
    selectedPhotoUrl = url;
    showPhotoModal = true;
  }

  function closePhotoModal() {
    showPhotoModal = false;
    selectedPhotoUrl = '';
  }

  function handleDelete() {
    if (confirm('Delete this catch? This cannot be undone.')) {
      onDelete(catchData.id);
    }
  }
</script>

<div class="page">
  <h1 class="page-title">Catch Details</h1>

  {#if !catchData}
    <div class="empty-state"><p>No catch data available</p></div>
  {:else}
  <!-- Photo Gallery -->
  {#if catchData.photos && catchData.photos.length > 0}
    <div class="photo-gallery">
      {#each catchData.photos as photo}
        <button class="photo-thumb" onclick={() => viewPhoto(photo.url)} type="button" aria-label="View photo">
          <img src={photo.url} alt="Catch photo" />
        </button>
      {/each}
      {#if catchData.photos.length > 1}
        <p class="photo-count">({catchData.photos.length} photos)</p>
      {/if}
    </div>
  {/if}

  <!-- Header -->
  <div class="card header-card">
    <h2 class="species-name">{catchData.species_name || 'Unknown Species'}</h2>
    <p class="catch-date">{new Date(catchData.caught_at).toLocaleString()}</p>
    {#if catchData.kept}
      <span class="chip chip-success">Kept</span>
    {:else}
      <span class="chip">Released</span>
    {/if}
  </div>

  <!-- Location Card -->
  {#if catchData.location_name || catchData.latitude}
    <div class="card">
      <h3 class="card-label">📍 Location</h3>
      {#if catchData.location_name}
        <p>{catchData.location_name}</p>
      {/if}
      {#if catchData.latitude && catchData.longitude}
        <p class="coords">{catchData.latitude.toFixed(4)}, {catchData.longitude.toFixed(4)}</p>
      {/if}
    </div>
  {/if}

  <!-- Size Card -->
  {#if catchData.length_in || catchData.weight_lb}
    <div class="card">
      <h3 class="card-label">📏 Size</h3>
      {#if catchData.length_in}
        <p>Length: {catchData.length_in} inches</p>
      {/if}
      {#if catchData.weight_lb}
        <p>Weight: {catchData.weight_lb} lb</p>
      {/if}
    </div>
  {/if}

  <!-- Gear Card -->
  {#if catchData.bait_or_lure || catchData.rod_setup || catchData.line_info || catchData.hook_size}
    <div class="card">
      <h3 class="card-label">🎣 Gear</h3>
      {#if catchData.bait_or_lure}
        <p><strong>Bait/Lure:</strong> {catchData.bait_or_lure}</p>
      {/if}
      {#if catchData.rod_setup}
        <p><strong>Rod:</strong> {catchData.rod_setup}</p>
      {/if}
      {#if catchData.line_info}
        <p><strong>Line:</strong> {catchData.line_info}</p>
      {/if}
      {#if catchData.hook_size}
        <p><strong>Hook:</strong> {catchData.hook_size}</p>
      {/if}
    </div>
  {/if}

  <!-- Weather Card -->
  {#if catchData.conditions || catchData.air_temp_f}
    <div class="card">
      <h3 class="card-label">☁️ Weather</h3>
      {#if catchData.conditions}
        <p>{catchData.conditions}{#if catchData.air_temp_f}, {catchData.air_temp_f.toFixed(0)}°F{/if}</p>
      {/if}
      {#if catchData.wind_mph}
        <p>Wind {catchData.wind_mph.toFixed(0)} mph {catchData.wind_dir || ''}</p>
      {/if}
      {#if catchData.pressure_mb || catchData.humidity_pct}
        <p>
          {#if catchData.pressure_mb}Pressure {catchData.pressure_mb.toFixed(0)} mb{/if}
          {#if catchData.pressure_mb && catchData.humidity_pct}, {/if}
          {#if catchData.humidity_pct}Humidity {catchData.humidity_pct.toFixed(0)}%{/if}
        </p>
      {/if}
    </div>
  {/if}

  <!-- Water Card -->
  {#if catchData.water_temp_f || catchData.water_clarity}
    <div class="card">
      <h3 class="card-label">💧 Water Conditions</h3>
      {#if catchData.water_temp_f}
        <p>Temperature: {catchData.water_temp_f.toFixed(0)}°F</p>
      {/if}
      {#if catchData.water_clarity}
        <p>Clarity: {catchData.water_clarity}</p>
      {/if}
    </div>
  {/if}

  <!-- Notes Card -->
  {#if catchData.notes}
    <div class="card">
      <h3 class="card-label">📝 Notes</h3>
      <p class="notes-text">{catchData.notes}</p>
    </div>
  {/if}

  <!-- Action Buttons -->
  <div class="action-buttons">
    <button class="btn btn-outline" onclick={onBack} type="button">Back</button>
    <button class="btn btn-primary" onclick={onEdit} type="button">Edit</button>
  </div>
  <button class="delete-link" onclick={handleDelete} type="button" aria-label="Delete this catch">Delete</button>
  {/if}
</div>

<!-- Photo Modal -->
{#if showPhotoModal}
  <div class="photo-modal" onclick={closePhotoModal} role="dialog" aria-label="Photo viewer">
    <div class="photo-modal-content" onclick={(e) => e.stopPropagation()}>
      <button class="photo-modal-close" onclick={closePhotoModal} type="button">×</button>
      <img src={selectedPhotoUrl} alt="Full size catch photo" />
    </div>
  </div>
{/if}

<style>
  .photo-gallery {
    display: flex;
    gap: 8px;
    overflow-x: auto;
    padding: 8px 0;
    margin-bottom: 12px;
  }

  .photo-thumb {
    flex-shrink: 0;
    width: 80px;
    height: 80px;
    border-radius: 8px;
    overflow: hidden;
    border: 1px solid var(--card-border);
    background: none;
    padding: 0;
    cursor: pointer;
  }

  .photo-thumb img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .photo-count {
    font-size: 0.8rem;
    color: var(--text-secondary);
    align-self: center;
  }

  .header-card {
    text-align: center;
  }

  .species-name {
    font-size: 1.5rem;
    font-weight: 700;
    margin-bottom: 6px;
  }

  .catch-date {
    font-size: 0.9rem;
    color: var(--text-secondary);
    margin-bottom: 8px;
  }

  .card-label {
    font-size: 0.8rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-secondary);
    margin-bottom: 8px;
  }

  .card p {
    margin-bottom: 4px;
    font-size: 0.9rem;
  }

  .coords {
    font-size: 0.85rem;
    color: var(--text-secondary);
  }

  .notes-text {
    white-space: pre-wrap;
    line-height: 1.6;
  }

  .chip-success {
    background: rgba(25, 135, 84, 0.1);
    color: var(--success);
  }

  .action-buttons {
    display: flex;
    gap: 12px;
    margin-top: 16px;
  }

  .action-buttons button {
    flex: 1;
  }

  .delete-link {
    display: block;
    margin: 12px auto 0;
    background: none;
    border: none;
    color: var(--danger);
    cursor: pointer;
    text-align: center;
    font-size: 0.9rem;
    text-decoration: underline;
  }

  .photo-modal {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.9);
    z-index: 1000;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .photo-modal-content {
    position: relative;
    max-width: 90vw;
    max-height: 90vh;
  }

  .photo-modal-content img {
    width: 100%;
    height: 100%;
    object-fit: contain;
  }

  .photo-modal-close {
    position: absolute;
    top: -40px;
    right: 0;
    background: none;
    border: none;
    color: white;
    font-size: 2rem;
    cursor: pointer;
    width: 40px;
    height: 40px;
  }

  @media (max-width: 360px) {
    .action-buttons {
      flex-direction: column;
    }
  }
</style>
