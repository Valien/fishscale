<script lang="ts">
  import { api } from '../api';
  import { loadCatches } from '../stores/catches';

  let {
    catchId = undefined,
    mode = 'create',
    onDone,
  }: {
    catchId?: number;
    mode?: 'create' | 'edit';
    onDone: () => void;
  } = $props();

  let speciesSuggestions = $state<string[]>([]);
  let showMoreDetail = $state(false);
  let saving = $state(false);
  let error = $state('');
  let loadingCatch = $state(false);

  let photoFiles = $state<File[]>([]);
  let photoInput: HTMLInputElement;

  let form = $state({
    caught_at: new Date(Date.now() - new Date().getTimezoneOffset() * 60000)
      .toISOString()
      .slice(0, 16),
    latitude: null as number | null,
    longitude: null as number | null,
    location_name: '',
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

  // Load species autocomplete suggestions from user's catch history
  $effect(() => {
    api.autocomplete.species().then((s) => {
      speciesSuggestions = s;
    });
  });

  // Get GPS location on mount (only in create mode)
  $effect(() => {
    if (mode === 'create' && 'geolocation' in navigator) {
      navigator.geolocation.getCurrentPosition(
        (pos) => {
          form.latitude = pos.coords.latitude;
          form.longitude = pos.coords.longitude;
          // Fetch weather
          if (form.latitude && form.longitude) {
            api.weather
              .get(form.latitude, form.longitude)
              .then((w) => {
                form.air_temp_f = w.air_temp_f;
                form.wind_mph = w.wind_mph;
                form.wind_dir = w.wind_dir;
                form.conditions = w.conditions;
                form.pressure_mb = w.pressure_mb;
                form.humidity_pct = w.humidity_pct;
              })
              .catch(() => {});
          }
        },
        () => {},
        { enableHighAccuracy: true },
      );
    }
  });

  // Load existing catch data in edit mode
  $effect(() => {
    if (mode === 'edit' && catchId) {
      loadingCatch = true;
      api.catches.get(catchId).then((data) => {
        form.caught_at = new Date(new Date(data.caught_at).getTime() - new Date().getTimezoneOffset() * 60000)
          .toISOString()
          .slice(0, 16);
        form.latitude = data.latitude;
        form.longitude = data.longitude;
        form.location_name = data.location_name || '';
        form.species_name = data.species_name || '';
        form.bait_or_lure = data.bait_or_lure || '';
        form.kept = data.kept || false;
        form.length_in = data.length_in;
        form.weight_lb = data.weight_lb;
        form.rod_setup = data.rod_setup || '';
        form.line_info = data.line_info || '';
        form.hook_size = data.hook_size || '';
        form.water_temp_f = data.water_temp_f;
        form.water_clarity = data.water_clarity || '';
        form.notes = data.notes || '';
        form.air_temp_f = data.air_temp_f;
        form.wind_mph = data.wind_mph;
        form.wind_dir = data.wind_dir || '';
        form.conditions = data.conditions || '';
        form.pressure_mb = data.pressure_mb;
        form.humidity_pct = data.humidity_pct;
        loadingCatch = false;
      }).catch((e: any) => {
        error = e.message || 'Failed to load catch data';
        loadingCatch = false;
      });
    }
  });

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
      if (mode === 'edit' && catchId) {
        // Update existing catch
        await api.catches.update(catchId, {
          caught_at: new Date(form.caught_at).toISOString(),
          latitude: form.latitude,
          longitude: form.longitude,
          location_name: form.location_name,
          species_name: form.species_name,
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
        await loadCatches();
        onDone();
      } else {
        // Create new catch (existing code)
        const created = await api.catches.create({
          caught_at: new Date(form.caught_at).toISOString(),
          latitude: form.latitude,
          longitude: form.longitude,
          location_name: form.location_name,
          species_name: form.species_name,
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
      }
    } catch (e: any) {
      error = e.message || 'Failed to save catch';
    } finally {
      saving = false;
    }
  }
</script>

<div class="page">
  <h1 class="page-title">{mode === 'edit' ? 'Edit Catch' : 'Log Catch'}</h1>

  {#if error}
    <div class="error-banner">{error}</div>
  {/if}

  {#if loadingCatch}
    <div class="empty-state"><p>Loading catch data...</p></div>
  {:else}
    <div class="card">
    <div class="form-group">
      <label>Date & Time</label>
      <input type="datetime-local" bind:value={form.caught_at} />
    </div>

    <div class="form-group">
      <label>Location</label>
      <input
        type="text"
        placeholder="e.g. Lake Fork, boat ramp cove"
        bind:value={form.location_name}
      />
      {#if form.latitude}
        <small class="coords">{form.latitude.toFixed(4)}, {form.longitude?.toFixed(4)}</small>
      {:else}
        <small class="coords">Getting GPS location...</small>
      {/if}
    </div>

    <div class="form-group">
      <label>Species</label>
      <input
        type="text"
        list="species-datalist"
        placeholder="e.g. Largemouth Bass"
        bind:value={form.species_name}
      />
      <datalist id="species-datalist">
        {#each speciesSuggestions as species}
          <option value={species}></option>
        {/each}
      </datalist>
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
        multiple
        bind:this={photoInput}
        onchange={handlePhotoSelect}
        style="display:none"
      />
      <button
        class="btn btn-outline btn-block"
        type="button"
        onpointerup={() => photoInput.click()}
      >
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

  <button class="btn btn-outline btn-block" onclick={() => (showMoreDetail = !showMoreDetail)}>
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
        <input
          type="text"
          placeholder="e.g. 15lb braid + 12lb fluoro"
          bind:value={form.line_info}
        />
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
      {saving ? 'Saving...' : mode === 'edit' ? 'Update Catch' : 'Save Catch'}
    </button>
  </div>
  {/if}
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
