<script lang="ts">
  import { onMount } from 'svelte';
  import { catches, loadCatches } from '../stores/catches';
  import maplibregl from 'maplibre-gl';
  import 'maplibre-gl/dist/maplibre-gl.css';

  let { visible = true }: { visible?: boolean } = $props();

  let mapContainer: HTMLDivElement;
  let map: maplibregl.Map | null = null;
  let positioned = false;

  function positionMap(m: maplibregl.Map, center: [number, number], zoom: number) {
    if (positioned) return;
    positioned = true;
    m.jumpTo({ center, zoom });
  }

  onMount(() => {
    loadCatches();

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
        layers: [
          {
            id: 'osm',
            type: 'raster',
            source: 'osm',
          },
        ],
      },
      center: [0, 0],
      zoom: 2,
    });

    map.addControl(new maplibregl.NavigationControl(), 'top-right');

    // GPS fallback: only used if catches haven't positioned the map first
    if ('geolocation' in navigator) {
      navigator.geolocation.getCurrentPosition(
        (pos) => {
          if (map && !positioned) {
            positionMap(map, [pos.coords.longitude, pos.coords.latitude], 10);
          }
        },
        () => {},
        { enableHighAccuracy: true }
      );
    }

    return () => {
      map?.remove();
    };
  });

  // Resize map when tab becomes visible
  $effect(() => {
    if (visible && map) {
      setTimeout(() => map?.resize(), 0);
    }
  });

  // Update markers when catches change
  $effect(() => {
    const currentCatches = $catches;
    if (!map || !currentCatches.length) return;

    // Remove existing markers
    const markers = document.querySelectorAll('.catch-marker');
    markers.forEach(m => m.remove());

    const bounds = new maplibregl.LngLatBounds();
    let hasBounds = false;

    for (const c of currentCatches) {
      if (!c.latitude || !c.longitude) continue;

      const el = document.createElement('div');
      el.className = 'catch-marker';
      el.style.cssText = `
        width: 14px; height: 14px;
        background: var(--primary, #0d6efd);
        border: 2px solid white;
        border-radius: 50%;
        cursor: pointer;
        box-shadow: 0 1px 4px rgba(0,0,0,0.3);
      `;

      const popup = new maplibregl.Popup({ offset: 10 }).setHTML(`
        <div style="padding:4px;font-family:sans-serif;font-size:0.85rem;">
          <strong>${c.species_name || 'Unknown'}</strong><br/>
          ${c.location_name || ''}<br/>
          <small>${new Date(c.caught_at).toLocaleDateString()}</small>
          ${c.weight_lb ? `<br/><small>${c.weight_lb} lb</small>` : ''}
          ${c.bait_or_lure ? `<br/><small>${c.bait_or_lure}</small>` : ''}
        </div>
      `);

      new maplibregl.Marker({ element: el })
        .setLngLat([c.longitude, c.latitude])
        .setPopup(popup)
        .addTo(map!);

      bounds.extend([c.longitude, c.latitude]);
      hasBounds = true;
    }

    if (hasBounds && !positioned) {
      positioned = true;
      map!.fitBounds(bounds, { padding: 50, maxZoom: 12, animate: false });
    }
  });
</script>

<div class="map-page">
  <div class="map-container" bind:this={mapContainer}></div>
</div>

<style>
  .map-page {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 64px;
  }

  .map-container {
    width: 100%;
    height: 100%;
  }
</style>
