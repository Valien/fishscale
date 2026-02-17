This is the source of truth for all future enhancements, or feature requests for now.

## In Progress

Active work currently being implemented:

- _(none)_

## Ideas & Enhancements

Quick capture list for future work. Move items to "Future Considerations" once scoped, or into an iteration plan when ready to build.

- [ ] Redo bottom nav icons (map, log, stats, settings) — current icons need improvement
- [ ] Fish-log page should remember user's location and auto-update without prompting
- [ ] Investigate cancel button on fish-log page — may not be working
- [ ] Bring in the tailscale users identity (name) and display it on the app
- [ ] Make the log entry for Species required. It shouldn't save if it is empty. But a user can Cancel out of the entry to go back to the log page.

## Future Considerations (v2+)

These are explicitly out of scope for v1 but the architecture is designed to accommodate them:

- **Fish species auto-ID from photos:** Hook point reserved in the photo upload handler. Most practical path is an optional ONNX model running server-side behind a feature flag.
- **Sharing via Tailscale Funnel:** tsnet already supports Funnel. Sharing a single catch page publicly is a flag flip when ready.
- **Social media sharing:** Generate a shareable image/card for a catch record.
- **S3-compatible photo storage:** Storage interface is abstracted. Add an S3 backend implementation.
- **Alternative map providers:** Map provider is abstracted behind an interface. Swap MapLibre for Google Maps or others.
- **Rich analytics:** Weather correlation, heatmaps, moon phase tracking, seasonal pattern analysis. The data model already captures enough to support this.
- **Import from CSV/JSON:** v1 supports export only. Import can be added using the same format.

---

## Completed

- [x] ~~Catch log entries should be clickable/editable~~ (Completed 2026-02-17: added CatchDetail view with photo navigation, edit mode with photo upload, see catch-detail-view-implementation.md)
- [x] ~~Photo picker should open device photo album by default instead of camera~~ (Completed 2026-02-17: removed `capture="environment"` attribute)
- [x] ~~Investigate species dropdown~~ (Completed 2026-02-17: replaced with native datalist, removed species table, see species-field-redesign.md)
- [x] ~~Run security iteration plan 3~~ (Iteration 3 completed: photo ownership, Docker hardening, Vitest, ESLint/Prettier, code splitting, Makefile, slog)
- [x] ~~Determine whether CI/CD with GitHub Actions is needed~~ (Not needed - deployments/integration/testing handled locally)
