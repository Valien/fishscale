<script lang="ts">
  import { settings, updateSettings } from '../stores/settings';
  import { api } from '../api';

  interface TailscaleInfo {
    login_name: string;
    display_name: string;
    tailscale_id: string;
    node_name: string;
    profile_pic_url?: string;
  }

  interface MeResponse {
    id: number;
    tailscale_id: string;
    display_name: string;
    created_at: string;
    tailscale_info?: TailscaleInfo;
  }

  let accountInfo: MeResponse | null = $state(null);

  $effect(() => {
    api.me.get().then((data: MeResponse) => {
      accountInfo = data;
    });
  });

  let currentTheme = $state('system');
  let currentUnits = $state('imperial');
  let saved = $state(false);

  $effect(() => {
    const s = $settings;
    currentTheme = s.theme;
    currentUnits = s.units;
  });

  async function save() {
    await updateSettings({
      theme: currentTheme,
      units: currentUnits,
    });
    saved = true;
    setTimeout(() => (saved = false), 2000);
  }
</script>

<div class="page">
  <h1 class="page-title">Settings</h1>

  {#if accountInfo?.tailscale_info}
    <div class="card">
      <h2 class="section-title">Account</h2>
      <div class="info-grid">
        <span class="info-label">Display Name</span>
        <span class="info-value">{accountInfo.tailscale_info.display_name}</span>

        <span class="info-label">Login</span>
        <span class="info-value">{accountInfo.tailscale_info.login_name}</span>

        <span class="info-label">Device</span>
        <span class="info-value">{accountInfo.tailscale_info.node_name.split('.')[0]}</span>

        <span class="info-label">Tailnet URL</span>
        <span class="info-value">{accountInfo.tailscale_info.node_name}</span>
      </div>
    </div>
  {/if}

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

  .radio-label input[type='radio'] {
    width: auto;
  }

  .info-grid {
    display: grid;
    grid-template-columns: auto 1fr;
    gap: 8px 16px;
    font-size: 0.9rem;
  }

  .info-label {
    font-weight: 600;
    color: var(--text-secondary);
  }

  .info-value {
    word-break: break-all;
  }
</style>
