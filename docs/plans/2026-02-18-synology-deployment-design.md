# Synology NAS Deployment Guide — Design

**Date:** 2026-02-18

## Goal

Create a beginner-friendly deployment guide for running Fishscale on a Synology NAS using DSM 7.2+ Container Manager. Also publish a pre-built Docker image to GHCR so non-technical users can deploy without SSH or building from source.

## Deliverables

### 1. Synology deployment guide (`docs/guides/synology-nas-deployment.md`)

Two-path guide targeting non-technical users:

- **Option A (Easy Install):** Pull pre-built image from GHCR, configure via Container Manager GUI. No SSH, no git, no building.
- **Option B (Build from Source):** SSH into NAS, git clone, docker compose up. For technical users who want to build locally.

Both paths cover: Tailscale auth key setup, TUN device prerequisite (boot-triggered task), auth key security (`.env` file permissions), updating, backup, and troubleshooting.

### 2. Synology-specific compose file (`docker-compose.synology.yml`)

Separate from the existing `docker-compose.yml` (which uses `build: .`). Uses `image: ghcr.io/valien/fishscale:latest` instead. Same runtime config (volumes, capabilities, env vars, resource limits).

### 3. GitHub Actions workflow (`.github/workflows/publish-image.yml`)

Triggers on version tags (`v*`). Builds the multi-stage Docker image and pushes to `ghcr.io/valien/fishscale`. No tests — CI stays local per project convention. First release: `v1.0.0`.

### 4. README link

Add a link to the Synology guide from the existing deployment section in README.md.

## Guide Structure

1. What You'll Need (prerequisites)
2. Get Your Tailscale Auth Key
3. Ensure TUN Device Exists (Synology Task Scheduler script)
4. Option A — Easy Install (pre-built image, GUI only)
5. Option B — Build from Source (SSH)
6. Auth Key Security (`.env` permissions, security notes)
7. Updating Fishscale
8. Backup & Restore
9. Troubleshooting

## Auth Key Security Approach

The `.env` file approach is reasonable for a single-user self-hosted app on a private Tailscale network. The guide recommends setting file permissions to `600` on the `.env` file and notes that env var values are visible in the Container Manager GUI. Docker Compose secrets (file-based) is mentioned as a more secure alternative for users who want it, but not the default path.

## Versioning

First tagged release: `v1.0.0`. The GHCR workflow triggers on `v*` tags.
