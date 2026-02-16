<script lang="ts">
  import { settings, updateSettings } from '../stores/settings';
  import { api } from '../api';

  let currentTheme = $state('system');
  let currentUnits = $state('imperial');
  let currentSpeciesFilter = $state('all');
  let saved = $state(false);

  $effect(() => {
    const s = $settings;
    currentTheme = s.theme;
    currentUnits = s.units;
    currentSpeciesFilter = s.species_filter;
  });

  async function save() {
    await updateSettings({ theme: currentTheme, units: currentUnits, species_filter: currentSpeciesFilter });
    saved = true;
    setTimeout(() => saved = false, 2000);
  }
</script>

<div class="page">
  <h1 class="page-title">Settings</h1>

  <div class="card">
    <h2 class="section-title">Theme</h2>
    <div class="radio-group">
      {#each ['light', 'dark', 'system'] as theme}
        <label class="radio-label">
          <input type="radio" name="theme" value={theme} bind:group={currentTheme} />
          <span>{theme.charAt(0).toUpperCase() + theme.slice(1)}</span>
        </label>
      {/each}
    </div>
  </div>

  <div class="card">
    <h2 class="section-title">Units</h2>
    <div class="radio-group">
      {#each ['imperial', 'metric'] as unit}
        <label class="radio-label">
          <input type="radio" name="units" value={unit} bind:group={currentUnits} />
          <span>{unit.charAt(0).toUpperCase() + unit.slice(1)}</span>
        </label>
      {/each}
    </div>
  </div>

  <div class="card">
    <h2 class="section-title">Species Filter</h2>
    <p class="section-desc">Choose which species appear in the dropdown when logging a catch.</p>
    <div class="radio-group">
      {#each [['all', 'All'], ['freshwater', 'Freshwater'], ['saltwater', 'Saltwater']] as [value, label]}
        <label class="radio-label">
          <input type="radio" name="species_filter" value={value} bind:group={currentSpeciesFilter} />
          <span>{label}</span>
        </label>
      {/each}
    </div>
  </div>

  <button class="btn btn-primary btn-block" onclick={save}>
    {saved ? 'Saved!' : 'Save Settings'}
  </button>

  <div class="card" style="margin-top: 16px;">
    <h2 class="section-title">Data</h2>
    <div style="display: flex; gap: 8px;">
      <a href={api.export.csv()} class="btn btn-outline" download>Export CSV</a>
      <a href={api.export.json()} class="btn btn-outline" download>Export JSON</a>
    </div>
  </div>
</div>

<style>
  .section-title {
    font-size: 1rem;
    font-weight: 700;
    margin-bottom: 12px;
  }

  .radio-group {
    display: flex;
    gap: 16px;
  }

  .radio-label {
    display: flex;
    align-items: center;
    gap: 6px;
    cursor: pointer;
    font-size: 0.9rem;
  }

  .radio-label input[type="radio"] {
    width: auto;
  }

  .section-desc {
    font-size: 0.8rem;
    color: var(--text-secondary);
    margin-bottom: 12px;
  }
</style>
