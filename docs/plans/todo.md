This is the source of truth for all future enhancements, or feature requests for now.

## In Progress

Active work currently being implemented:

- _(none)_

## Ideas & Enhancements

Quick capture list for future work. Move items to "Future Considerations" once scoped, or into an iteration plan when ready to build.

- [ ] Redo bottom nav icons (map, log, stats, settings) — current icons need improvement
- [ ] Fish-log page should remember user's location and auto-update without prompting

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
- [x] ~~Bring in the tailscale users identity (name) and display it on the app~~ (Completed 2026-02-17: added personalized fishing greetings to Log Catch page, see user-greeting-implementation.md)
- [x] ~~Investigate cancel button on fish-log page~~ (Confirmed working 2026-02-18: no issue found)
- [x] ~~Make species required on log entry~~ (Completed 2026-02-18: frontend validation in save() rejects empty species, see species-required-and-map-detail-link.md)
- [x] ~~Map popup link to catch detail~~ (Completed 2026-02-18: "View Details" link in map pin popups navigates to Log tab detail view, see species-required-and-map-detail-link.md)
- [x] ~~Map-based location picker for logging catches~~ ([GH#1](https://github.com/Valien/fishscale/issues/1), Completed 2026-02-18: full-screen map overlay with tap-to-place pin in LogCatch, see 2026-02-18-map-location-picker-design.md)
- [x] ~~Add Tailscale info/identity to Settings page~~ ([GH#2](https://github.com/Valien/fishscale/issues/2), Completed 2026-02-18: Account section showing display name, login, device, and tailnet URL, see 2026-02-18-tailscale-settings-design.md)
- [x] ~~Synology NAS deployment guide and GHCR image publishing~~ (Completed 2026-02-18: deployment guide, Synology compose file, GitHub Actions workflow, v1.0.0 tagged, see synology-nas-deployment.md)
