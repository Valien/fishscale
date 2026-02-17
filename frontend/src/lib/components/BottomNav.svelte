<script lang="ts">
  let {
    activePage = $bindable('map'),
    onNavigate,
  }: { activePage: string; onNavigate: (page: string) => void } = $props();

  const tabs = [
    { id: 'map', label: 'Map', icon: '🗺' },
    { id: 'log', label: 'Log', icon: '📋' },
    { id: 'add', label: '', icon: '🐟' },
    { id: 'stats', label: 'Stats', icon: '📊' },
    { id: 'settings', label: 'Settings', icon: '⚙' },
  ];
</script>

<nav class="bottom-nav">
  {#each tabs as tab}
    {#if tab.id === 'add'}
      <button class="nav-fab" onclick={() => onNavigate('add')}>
        <span class="fab-icon">{tab.icon}</span>
      </button>
    {:else}
      <button
        class="nav-tab"
        class:active={activePage === tab.id}
        onclick={() => onNavigate(tab.id)}
      >
        <span class="nav-icon">{tab.icon}</span>
        <span class="nav-label">{tab.label}</span>
      </button>
    {/if}
  {/each}
</nav>

<style>
  .bottom-nav {
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    display: flex;
    align-items: center;
    justify-content: space-around;
    background: var(--nav-bg);
    border-top: 1px solid var(--nav-border);
    padding: 4px 0 env(safe-area-inset-bottom, 8px);
    z-index: 100;
    max-width: 768px;
    margin: 0 auto;
  }

  .nav-tab {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2px;
    padding: 8px 16px;
    background: none;
    border: none;
    cursor: pointer;
    color: var(--text-secondary);
    font-size: 0.7rem;
    transition: color 0.2s;
  }

  .nav-tab.active {
    color: var(--primary);
  }

  .nav-icon {
    font-size: 1.2rem;
  }

  .nav-label {
    font-weight: 500;
  }

  .nav-fab {
    width: 52px;
    height: 52px;
    border-radius: 50%;
    background: var(--primary);
    color: white;
    border: none;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-top: -20px;
    box-shadow: 0 2px 8px var(--shadow);
    transition: background 0.2s;
  }

  .nav-fab:hover {
    background: var(--primary-hover);
  }

  .fab-icon {
    font-size: 1.5rem;
    font-weight: 700;
  }
</style>
