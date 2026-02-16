import { writable } from 'svelte/store';
import { api } from '../api';

export const catches = writable<any[]>([]);
export const loading = writable(false);

export async function loadCatches() {
  loading.set(true);
  try {
    const data = await api.catches.list();
    catches.set(data);
  } catch (e) {
    console.error('Failed to load catches:', e);
  } finally {
    loading.set(false);
  }
}

export async function deleteCatch(id: number) {
  await api.catches.delete(id);
  catches.update(list => list.filter(c => c.id !== id));
}
