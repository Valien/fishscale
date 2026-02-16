<script lang="ts">
  import { onMount } from 'svelte';
  import BottomNav from './lib/components/BottomNav.svelte';
  import MapView from './lib/pages/MapView.svelte';
  import CatchLog from './lib/pages/CatchLog.svelte';
  import LogCatch from './lib/pages/LogCatch.svelte';
  import Stats from './lib/pages/Stats.svelte';
  import Settings from './lib/pages/Settings.svelte';
  import { loadSettings } from './lib/stores/settings';

  let activePage = $state('map');

  onMount(() => {
    loadSettings();
  });

  function navigate(page: string) {
    activePage = page;
  }

  function handleCatchDone() {
    activePage = 'log';
  }

  function handleEditCatch(id: number) {
    activePage = 'log';
  }
</script>

<div class="app">
  {#if activePage === 'map'}
    <MapView />
  {:else if activePage === 'log'}
    <CatchLog onEdit={handleEditCatch} />
  {:else if activePage === 'add'}
    <LogCatch onDone={handleCatchDone} />
  {:else if activePage === 'stats'}
    <Stats />
  {:else if activePage === 'settings'}
    <Settings />
  {/if}

  <BottomNav bind:activePage onNavigate={navigate} />
</div>
