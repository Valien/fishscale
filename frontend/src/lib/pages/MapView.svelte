<script lang="ts">
  import { onMount } from 'svelte';
  import { catches, loadCatches } from '../stores/catches';
  import maplibregl from 'maplibre-gl';
  import 'maplibre-gl/dist/maplibre-gl.css';

  let { visible = true }: { visible?: boolean } = $props();

  let mapContainer: HTMLDivElement;
  let map: maplibregl.Map | null = null;
  let activeMarkers: maplibregl.Marker[] = [];
  let hasFittedBounds = false;
  let prevVisible = true;

  function clearMarkers() {
    for (const m of activeMarkers) {
      m.remove();
    }
    activeMarkers = [];
  }

  function syncMarkers(catchList: any[]) {
    if (!map) return;

    clearMarkers();

    const bounds = new maplibregl.LngLatBounds();
    let hasBounds = false;

    for (const c of catchList) {
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

      const popupEl = document.createElement('div');
      popupEl.style.cssText = 'padding:4px;font-family:sans-serif;font-size:0.85rem;';

      const name = document.createElement('strong');
      name.textContent = c.species_name || 'Unknown';
      popupEl.appendChild(name);

      popupEl.appendChild(document.createElement('br'));
      popupEl.appendChild(document.createTextNode(c.location_name || ''));
      popupEl.appendChild(document.createElement('br'));

      const date = document.createElement('small');
      date.textContent = new Date(c.caught_at).toLocaleDateString();
      popupEl.appendChild(date);

      if (c.weight_lb) {
        popupEl.appendChild(document.createElement('br'));
        const weight = document.createElement('small');
        weight.textContent = `${c.weight_lb} lb`;
        popupEl.appendChild(weight);
      }

      if (c.bait_or_lure) {
        popupEl.appendChild(document.createElement('br'));
        const bait = document.createElement('small');
        bait.textContent = c.bait_or_lure;
        popupEl.appendChild(bait);
      }

      const popup = new maplibregl.Popup({ offset: 10 }).setDOMContent(popupEl);

      const marker = new maplibregl.Marker({ element: el })
        .setLngLat([c.longitude, c.latitude])
        .setPopup(popup)
        .addTo(map);

      activeMarkers.push(marker);
      bounds.extend([c.longitude, c.latitude]);
      hasBounds = true;
    }

    if (hasBounds && !hasFittedBounds) {
      hasFittedBounds = true;
      map.fitBounds(bounds, { padding: 50, maxZoom: 12, animate: false });
    }
  }

  onMount(() => {
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
      center: [-98.5, 39.8],
      zoom: 3,
    });

    map.addControl(new maplibregl.NavigationControl(), 'top-right');

    map.addControl(
      new maplibregl.GeolocateControl({
        positionOptions: { enableHighAccuracy: true },
        trackUserLocation: false,
        showUserLocation: true,
      }),
      'top-right',
    );

    // Load catches after map is ready
    map.on('load', () => {
      loadCatches();
    });

    return () => {
      clearMarkers();
      map?.remove();
      map = null;
    };
  });

  // Handle tab visibility: resize map when shown after being hidden
  $effect(() => {
    const isVisible = visible;
    if (isVisible && !prevVisible && map) {
      requestAnimationFrame(() => {
        map?.resize();
      });
    }
    prevVisible = isVisible;
  });

  // Sync markers when catches change
  $effect(() => {
    const catchList = $catches;
    if (catchList.length > 0) {
      syncMarkers(catchList);
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
    overflow: hidden;
    overscroll-behavior: none;
  }

  .map-container {
    width: 100%;
    height: 100%;
    overflow: hidden;
  }
</style>
