import { describe, it, expect, vi, beforeEach } from 'vitest';

// Test that the API module constructs correct URLs and handles responses.
// We use vi.resetModules() + dynamic import to get a fresh module per test,
// since the api module captures `fetch` at import time.
describe('API client', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    vi.resetModules();
  });

  it('constructs correct catch list URL', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify([]), { status: 200 })
    );

    const { api } = await import('../api');
    await api.catches.list();

    expect(fetchSpy).toHaveBeenCalledWith('/api/v1/catches', expect.objectContaining({
      headers: expect.objectContaining({ 'Content-Type': 'application/json' }),
    }));
  });

  it('throws on non-OK response', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(JSON.stringify({ error: 'not found' }), { status: 404 })
    );

    const { api } = await import('../api');
    await expect(api.catches.get(999)).rejects.toThrow('not found');
  });

  it('returns undefined for 204 responses', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(null, { status: 204 })
    );

    const { api } = await import('../api');
    const result = await api.catches.delete(1);
    expect(result).toBeUndefined();
  });
});
