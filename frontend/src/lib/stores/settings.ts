import { writable } from 'svelte/store';
import { api } from '../api';

interface Settings {
  theme: string;
  units: string;
  species_filter: string;
}

export const settings = writable<Settings>({
  theme: 'system',
  units: 'imperial',
  species_filter: 'all',
});

export async function loadSettings() {
  try {
    const s = await api.settings.get();
    settings.set({ theme: s.theme, units: s.units, species_filter: s.species_filter || 'all' });
    applyTheme(s.theme);
  } catch {
    applyTheme('system');
  }
}

export async function updateSettings(s: Settings) {
  settings.set(s);
  applyTheme(s.theme);
  await api.settings.update(s);
}

function applyTheme(theme: string) {
  const html = document.documentElement;
  html.classList.remove('theme-light', 'theme-dark', 'theme-system');
  html.classList.add(`theme-${theme}`);
}
