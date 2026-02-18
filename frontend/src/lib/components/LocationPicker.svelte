<script lang="ts">
  import { onMount } from 'svelte';
  import maplibregl from 'maplibre-gl';
  import 'maplibre-gl/dist/maplibre-gl.css';

  let {
    initialLat = null,
    initialLng = null,
    onSelect,
    onCancel,
  }: {
    initialLat?: number | null;
    initialLng?: number | null;
    onSelect: (coords: { latitude: number; longitude: number }) => void;
    onCancel: () => void;
  } = $props();

  let mapContainer: HTMLDivElement;
  let map: maplibregl.Map | null = null;
  let marker: maplibregl.Marker | null = null;
  let pinLat = $state<number | null>(null);
  let pinLng = $state<number | null>(null);

  onMount(() => {
    const center: [number, number] =
      initialLat && initialLng ? [initialLng, initialLat] : [-98.5, 39.8];
    const zoom = initialLat && initialLng ? 12 : 3;

    map = new maplibregl.Map({
      container: mapContainer,
      style: {
        version: 8,
        sources: {
          osm: {
            type: 'raster',
            tiles: ['https://tile.openstreetmap.org/{z}/{x}/{y}.png'],
            tileSize: 256,
            attribution: '&copy; OpenStreetMap contributors',
          },
        },
        layers: [{ id: 'osm', type: 'raster', source: 'osm' }],
      },
      center,
      zoom,
    });

    map.addControl(new maplibregl.NavigationControl(), 'top-right');

    if (initialLat && initialLng) {
      marker = new maplibregl.Marker({ color: '#dc3545' })
        .setLngLat([initialLng, initialLat])
        .addTo(map);
      pinLat = initialLat;
      pinLng = initialLng;
    }

    map.on('click', (e) => {
      const { lng, lat } = e.lngLat;
      pinLat = lat;
      pinLng = lng;

      if (marker) {
        marker.setLngLat([lng, lat]);
      } else if (map) {
        marker = new maplibregl.Marker({ color: '#dc3545' })
          .setLngLat([lng, lat])
          .addTo(map);
      }
    });

    return () => {
      marker?.remove();
      map?.remove();
      map = null;
    };
  });

  function confirm() {
    if (pinLat !== null && pinLng !== null) {
      onSelect({ latitude: pinLat, longitude: pinLng });
    }
  }
</script>

<div class="picker-overlay">
  <div class="picker-header">
    <button class="picker-btn" type="button" onclick={onCancel}>Cancel</button>
    <span class="picker-coords">
      {#if pinLat !== null && pinLng !== null}
        {pinLat.toFixed(4)}, {pinLng.toFixed(4)}
      {:else}
        Tap to place pin
      {/if}
    </span>
  </div>

  <div class="picker-map" bind:this={mapContainer}></div>

  <div class="picker-footer">
    <button
      class="btn btn-primary btn-block"
      type="button"
      onclick={confirm}
      disabled={pinLat === null}
    >
      Confirm Location
    </button>
  </div>
</div>

<style>
  .picker-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 200;
    display: flex;
    flex-direction: column;
    background: var(--bg);
  }

  .picker-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 12px 16px;
    background: var(--card-bg);
    border-bottom: 1px solid var(--card-border);
    z-index: 1;
  }

  .picker-btn {
    background: none;
    border: none;
    color: var(--primary);
    font-size: 1rem;
    font-weight: 600;
    cursor: pointer;
    padding: 4px 0;
  }

  .picker-coords {
    font-size: 0.85rem;
    color: var(--text-secondary);
  }

  .picker-map {
    flex: 1;
    width: 100%;
  }

  .picker-footer {
    padding: 12px 16px calc(12px + env(safe-area-inset-bottom, 0px)) 16px;
    background: var(--card-bg);
    border-top: 1px solid var(--card-border);
  }
</style>
