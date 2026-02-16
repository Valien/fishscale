<script lang="ts">
  import { api } from '../api';
  import { loadCatches } from '../stores/catches';

  let { onDone }: { onDone: () => void } = $props();

  let speciesList = $state<any[]>([]);
  let speciesQuery = $state('');
  let filteredSpecies = $state<any[]>([]);
  let showSpeciesDropdown = $state(false);
  let justSelected = $state(false);

  let showMoreDetail = $state(false);
  let saving = $state(false);
  let error = $state('');

  let photoFiles = $state<File[]>([]);
  let photoInput: HTMLInputElement;

  let form = $state({
    caught_at: new Date().toISOString().slice(0, 16),
    latitude: null as number | null,
    longitude: null as number | null,
    location_name: '',
    species_id: null as number | null,
    species_name: '',
    bait_or_lure: '',
    kept: false,
    length_in: null as number | null,
    weight_lb: null as number | null,
    rod_setup: '',
    line_info: '',
    hook_size: '',
    water_temp_f: null as number | null,
    water_clarity: '',
    notes: '',
    air_temp_f: null as number | null,
    wind_mph: null as number | null,
    wind_dir: '',
    conditions: '',
    pressure_mb: null as number | null,
    humidity_pct: null as number | null,
  });

  // Load species list
  $effect(() => {
    api.species.list().then(s => { speciesList = s; });
  });

  // Filter species on query change
  $effect(() => {
    if (justSelected) {
      justSelected = false;
      return;
    }
    if (speciesQuery.length > 0) {
      filteredSpecies = speciesList.filter(s =>
        s.name.toLowerCase().includes(speciesQuery.toLowerCase())
      ).slice(0, 8);
      showSpeciesDropdown = filteredSpecies.length > 0;
    } else {
      showSpeciesDropdown = false;
      form.species_id = null;
      form.species_name = '';
    }
  });

  // Get GPS location on mount
  $effect(() => {
    if ('geolocation' in navigator) {
      navigator.geolocation.getCurrentPosition(
        (pos) => {
          form.latitude = pos.coords.latitude;
          form.longitude = pos.coords.longitude;
          // Fetch weather
          if (form.latitude && form.longitude) {
            api.weather.get(form.latitude, form.longitude).then(w => {
              form.air_temp_f = w.air_temp_f;
              form.wind_mph = w.wind_mph;
              form.wind_dir = w.wind_dir;
              form.conditions = w.conditions;
              form.pressure_mb = w.pressure_mb;
              form.humidity_pct = w.humidity_pct;
            }).catch(() => {});
          }
        },
        () => {},
        { enableHighAccuracy: true }
      );
    }
  });

  function selectSpecies(s: any) {
    justSelected = true;
    form.species_id = s.id;
    form.species_name = s.name;
    speciesQuery = s.name;
    showSpeciesDropdown = false;
    if (document.activeElement instanceof HTMLElement) {
      document.activeElement.blur();
    }
  }

  function handlePhotoSelect(e: Event) {
    const input = e.target as HTMLInputElement;
    if (input.files) {
      photoFiles = [...photoFiles, ...Array.from(input.files)];
    }
    input.value = '';
  }

  function removePhoto(index: number) {
    photoFiles = photoFiles.filter((_, i) => i !== index);
  }

  async function save() {
    saving = true;
    error = '';
    try {
      const created = await api.catches.create({
        caught_at: new Date(form.caught_at).toISOString(),
        latitude: form.latitude,
        longitude: form.longitude,
        location_name: form.location_name,
        species_id: form.species_id,
        bait_or_lure: form.bait_or_lure,
        kept: form.kept,
        length_in: form.length_in,
        weight_lb: form.weight_lb,
        rod_setup: form.rod_setup,
        line_info: form.line_info,
        hook_size: form.hook_size,
        water_temp_f: form.water_temp_f,
        water_clarity: form.water_clarity,
        notes: form.notes,
        air_temp_f: form.air_temp_f,
        wind_mph: form.wind_mph,
        wind_dir: form.wind_dir,
        conditions: form.conditions,
        pressure_mb: form.pressure_mb,
        humidity_pct: form.humidity_pct,
      });

      if (photoFiles.length > 0 && created?.id) {
        const formData = new FormData();
        for (const file of photoFiles) {
          formData.append('photos', file);
        }
        await api.catches.addPhotos(created.id, formData);
      }

      await loadCatches();
      onDone();
    } catch (e: any) {
      error = e.message || 'Failed to save catch';
    } finally {
      saving = false;
    }
  }
</script>

<div class="page">
  <h1 class="page-title">Log Catch</h1>

  {#if error}
    <div class="error-banner">{error}</div>
  {/if}

  <div class="card">
    <div class="form-group">
      <label>Date & Time</label>
      <input type="datetime-local" bind:value={form.caught_at} />
    </div>

    <div class="form-group">
      <label>Location</label>
      <input type="text" placeholder="e.g. Lake Fork, boat ramp cove" bind:value={form.location_name} />
      {#if form.latitude}
        <small class="coords">{form.latitude.toFixed(4)}, {form.longitude?.toFixed(4)}</small>
      {:else}
        <small class="coords">Getting GPS location...</small>
      {/if}
    </div>

    <div class="form-group species-field">
      <label>Species</label>
      <input
        type="text"
        placeholder="Search species..."
        bind:value={speciesQuery}
        onfocus={() => { if (speciesQuery.length > 0 && !justSelected) showSpeciesDropdown = true; }}
      />
      {#if showSpeciesDropdown}
        <div class="dropdown-backdrop" onpointerup={() => { showSpeciesDropdown = false; }}></div>
        <div class="dropdown">
          {#each filteredSpecies as s}
            <button class="dropdown-item" onpointerup={() => selectSpecies(s)}>
              {s.name}
              <span class="chip">{s.category}</span>
            </button>
          {/each}
        </div>
      {/if}
    </div>

    <div class="form-group">
      <label>Bait / Lure</label>
      <input type="text" placeholder="e.g. Texas Rig, Senko" bind:value={form.bait_or_lure} />
    </div>

    <div class="form-group">
      <label>Photo</label>
      <input
        type="file"
        accept="image/*"
        capture="environment"
        multiple
        bind:this={photoInput}
        onchange={handlePhotoSelect}
        style="display:none"
      />
      <button class="btn btn-outline btn-block" type="button" onpointerup={() => photoInput.click()}>
        {photoFiles.length > 0 ? `${photoFiles.length} photo(s) selected` : 'Add Photo'}
      </button>
      {#if photoFiles.length > 0}
        <div class="photo-previews">
          {#each photoFiles as file, i}
            <div class="photo-thumb">
              <img src={URL.createObjectURL(file)} alt="Preview" />
              <button class="photo-remove" onpointerup={() => removePhoto(i)}>x</button>
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <div class="form-group">
      <label class="toggle">
        <input type="checkbox" bind:checked={form.kept} />
        <span>Kept</span>
      </label>
    </div>

    {#if form.conditions}
      <div class="weather-preview">
        <span>{form.conditions}</span>
        {#if form.air_temp_f}<span>{form.air_temp_f.toFixed(0)}°F</span>{/if}
        {#if form.wind_mph}<span>Wind {form.wind_mph.toFixed(0)} mph {form.wind_dir}</span>{/if}
      </div>
    {/if}
  </div>

  <button class="btn btn-outline btn-block" onclick={() => showMoreDetail = !showMoreDetail}>
    {showMoreDetail ? '- Less Detail' : '+ More Detail'}
  </button>

  {#if showMoreDetail}
    <div class="card" style="margin-top: 12px;">
      <div class="form-row">
        <div class="form-group">
          <label>Length (in)</label>
          <input type="number" step="0.1" bind:value={form.length_in} />
        </div>
        <div class="form-group">
          <label>Weight (lb)</label>
          <input type="number" step="0.01" bind:value={form.weight_lb} />
        </div>
      </div>

      <div class="form-group">
        <label>Rod Setup</label>
        <input type="text" placeholder="e.g. 7' MH spinning" bind:value={form.rod_setup} />
      </div>

      <div class="form-group">
        <label>Line Info</label>
        <input type="text" placeholder="e.g. 15lb braid + 12lb fluoro" bind:value={form.line_info} />
      </div>

      <div class="form-group">
        <label>Hook Size</label>
        <input type="text" placeholder="e.g. 3/0 EWG" bind:value={form.hook_size} />
      </div>

      <div class="form-row">
        <div class="form-group">
          <label>Water Temp (°F)</label>
          <input type="number" step="0.1" bind:value={form.water_temp_f} />
        </div>
        <div class="form-group">
          <label>Water Clarity</label>
          <select bind:value={form.water_clarity}>
            <option value="">--</option>
            <option value="Clear">Clear</option>
            <option value="Stained">Stained</option>
            <option value="Muddy">Muddy</option>
          </select>
        </div>
      </div>

      <div class="form-group">
        <label>Notes</label>
        <textarea rows="3" bind:value={form.notes}></textarea>
      </div>
    </div>
  {/if}

  <div style="margin-top: 16px; display: flex; gap: 12px;">
    <button class="btn btn-outline" style="flex:1;" onclick={onDone}>Cancel</button>
    <button class="btn btn-primary" style="flex:2;" onclick={save} disabled={saving}>
      {saving ? 'Saving...' : 'Save Catch'}
    </button>
  </div>
</div>

<style>
  .error-banner {
    background: var(--danger);
    color: white;
    padding: 8px 12px;
    border-radius: 8px;
    margin-bottom: 12px;
    font-size: 0.9rem;
  }

  .coords {
    color: var(--text-secondary);
    font-size: 0.8rem;
  }

  .species-field {
    position: relative;
  }

  .dropdown-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 9;
  }

  .dropdown {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    background: var(--card-bg);
    border: 1px solid var(--card-border);
    border-radius: 8px;
    max-height: 200px;
    overflow-y: auto;
    z-index: 10;
    box-shadow: 0 4px 12px var(--shadow);
  }

  .dropdown-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
    padding: 10px 12px;
    border: none;
    background: none;
    color: var(--text);
    cursor: pointer;
    text-align: left;
    font-size: 0.9rem;
  }

  .dropdown-item:hover {
    background: var(--bg-secondary);
  }

  .photo-previews {
    display: flex;
    gap: 8px;
    margin-top: 8px;
    overflow-x: auto;
  }

  .photo-thumb {
    position: relative;
    flex-shrink: 0;
  }

  .photo-thumb img {
    width: 64px;
    height: 64px;
    object-fit: cover;
    border-radius: 8px;
  }

  .photo-remove {
    position: absolute;
    top: -6px;
    right: -6px;
    width: 20px;
    height: 20px;
    border-radius: 50%;
    background: var(--danger);
    color: white;
    border: none;
    font-size: 0.7rem;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .weather-preview {
    display: flex;
    gap: 12px;
    padding: 8px 0;
    font-size: 0.85rem;
    color: var(--text-secondary);
  }

  .form-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }
</style>
