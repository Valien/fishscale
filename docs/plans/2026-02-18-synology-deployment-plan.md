# Synology NAS Deployment — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a beginner-friendly Synology NAS deployment guide, publish a pre-built Docker image to GHCR, and tag the first release as v1.0.0.

**Architecture:** Four deliverables: (1) GitHub Actions workflow to build and push Docker image to GHCR on version tags, (2) a Synology-specific docker-compose file that pulls the pre-built image, (3) a step-by-step deployment guide covering both GUI and SSH paths, (4) a README link to the guide. First release tagged as v1.0.0.

**Tech Stack:** GitHub Actions, GHCR (ghcr.io), Docker, Docker Compose, Synology DSM 7.2+ Container Manager

---

### Task 1: Create GitHub Actions workflow for GHCR publishing

**Files:**
- Create: `.github/workflows/publish-image.yml`

**Step 1: Create the workflow file**

```yaml
name: Publish Docker Image

on:
  push:
    tags:
      - 'v*'

env:
  REGISTRY: ghcr.io
  IMAGE_NAME: ${{ github.repository }}

jobs:
  build-and-push:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Log in to GHCR
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_NAME }}
          tags: |
            type=semver,pattern={{version}}
            type=semver,pattern={{major}}.{{minor}}
            type=raw,value=latest

      - name: Build and push
        uses: docker/build-push-action@v6
        with:
          context: .
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
```

**Step 2: Verify the YAML is valid**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/publish-image.yml'))"`
Expected: no errors (exits silently)

If python3 yaml module not available, verify manually that indentation is consistent.

**Step 3: Commit**

```bash
git add .github/workflows/publish-image.yml
git commit -m "ci: add GitHub Actions workflow for GHCR image publishing

Triggers on version tags (v*). Builds the multi-stage Docker image
and pushes to ghcr.io/valien/fishscale with semver + latest tags.
No tests — CI stays local per project convention.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 2: Create Synology-specific docker-compose file

**Files:**
- Create: `docker-compose.synology.yml`

**Step 1: Create the file**

This is identical to the existing `docker-compose.yml` but replaces `build: .` with `image: ghcr.io/valien/fishscale:latest`. Do NOT modify the existing `docker-compose.yml`.

```yaml
# Synology NAS deployment — uses pre-built image from GHCR.
# See docs/guides/synology-nas-deployment.md for setup instructions.
#
# Usage:
#   1. Copy this file to your Synology NAS
#   2. Create a .env file next to it with: TS_AUTHKEY=tskey-auth-...
#   3. In Container Manager → Project → Create, select this file
#
services:
  fishscale:
    image: ghcr.io/valien/fishscale:latest
    container_name: fishscale
    restart: unless-stopped
    mem_limit: 256m
    cpus: 1.0
    volumes:
      - fishscale-data:/data
      - /dev/net/tun:/dev/net/tun
    cap_add:
      - NET_ADMIN
    environment:
      - TS_AUTHKEY=${TS_AUTHKEY}
      - TS_HOSTNAME=${TS_HOSTNAME:-fishscale}
      - TS_STATE_DIR=/data/tsnet-state
      - FISHSCALE_DB_PATH=/data/fish.db
      - FISHSCALE_PHOTO_DIR=/data/photos
      - FISHSCALE_LOG_LEVEL=${FISHSCALE_LOG_LEVEL:-info}
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"

volumes:
  fishscale-data:
```

**Step 2: Verify YAML syntax**

Run: `python3 -c "import yaml; yaml.safe_load(open('docker-compose.synology.yml'))"`
Expected: no errors

**Step 3: Commit**

```bash
git add docker-compose.synology.yml
git commit -m "feat: add Synology-specific docker-compose file

Uses pre-built GHCR image instead of build-from-source.
Existing docker-compose.yml is unchanged.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 3: Write the Synology deployment guide

**Files:**
- Create: `docs/guides/synology-nas-deployment.md`

**Step 1: Write the guide**

The guide must cover these sections in this order. Write for a non-technical audience — no jargon without explanation, every step explicit, copy-pasteable commands where applicable.

```markdown
# Deploying Fishscale on a Synology NAS

This guide walks you through running Fishscale on a Synology NAS using DSM 7.2+ and the Container Manager package. Two options are covered:

- **Option A (Easy Install)** — Pull a pre-built image using the Container Manager GUI. No command line needed.
- **Option B (Build from Source)** — For technical users who prefer to clone the repo and build locally via SSH.

Both options require the same one-time setup steps (Sections 1-3).

---

## 1. What You'll Need

- A Synology NAS running **DSM 7.2 or later**
- **Container Manager** installed from Package Center (search "Container Manager" and click Install)
- A free [Tailscale account](https://tailscale.com) (this is how Fishscale handles secure access — no ports to open, no VPN to configure)

---

## 2. Get Your Tailscale Auth Key

Fishscale connects to your Tailscale network automatically. It needs an **auth key** to do this.

1. Go to the [Tailscale admin console](https://login.tailscale.com/admin/settings/keys)
2. Click **Generate auth key**
3. Settings to choose:
   - **Reusable**: Yes (so the container can restart without a new key)
   - **Ephemeral**: No (so the device stays in your tailnet when the container stops)
   - **Tags**: Optional. If you use ACL tags, add one like `tag:fishscale`
4. Click **Generate key**
5. **Copy the key now** — it starts with `tskey-auth-` and you won't be able to see it again

Save this key somewhere safe. You'll need it in a moment.

**Note:** Tailscale auth keys expire after 90 days. If Fishscale stops connecting after that, generate a new key and update your configuration (see Section 7).

---

## 3. Set Up the TUN Device (One-Time)

Tailscale needs a network tunnel device (`/dev/net/tun`) to work. Most Synology NAS devices don't have this enabled by default, so you need to create a small startup task.

1. Open **Control Panel** → **Task Scheduler**
2. Click **Create** → **Triggered Task** → **User-defined script**
3. **General tab:**
   - Task name: `Create TUN device`
   - User: `root`
   - Event: `Boot-up`
   - Enabled: checked
4. **Task Settings tab** — paste this script:

```bash
#!/bin/bash
# Load the TUN kernel module if not already loaded
if ! lsmod | grep -q "^tun "; then
  insmod /lib/modules/tun.ko
fi

# Create the device node if it doesn't exist
mkdir -p /dev/net
if [ ! -c /dev/net/tun ]; then
  mknod /dev/net/tun c 10 200
fi
chmod 0666 /dev/net/tun
```

5. Click **OK**, then enter your DSM password to confirm
6. **Run it now:** Right-click the task → **Run**

This only needs to be done once. The task will run automatically every time your NAS reboots.

---

## 4. Option A — Easy Install (Pre-Built Image)

This is the simplest path. You'll use Container Manager's Project feature to pull and run Fishscale.

### 4a. Create the project folder

1. Open **File Station**
2. Navigate to the `docker` shared folder (create it if it doesn't exist: right-click in the sidebar → Create new shared folder → name it `docker`)
3. Inside `docker`, create a new folder called `fishscale`

### 4b. Create the environment file

You need a small text file to hold your Tailscale auth key.

1. On your **computer** (not the NAS), open a text editor
2. Create a file called `.env` with this content:

```
TS_AUTHKEY=tskey-auth-PASTE-YOUR-KEY-HERE
TS_HOSTNAME=fishscale
```

Replace `tskey-auth-PASTE-YOUR-KEY-HERE` with the auth key you copied in Section 2.

3. Upload this `.env` file to the `docker/fishscale` folder on your NAS using File Station (drag and drop, or use the Upload button)

**Tip:** If File Station doesn't show the `.env` file after uploading, click the gear icon and enable "Show hidden files."

### 4c. Create the compose file

1. On your **computer**, save the following as `docker-compose.yml`:

```yaml
services:
  fishscale:
    image: ghcr.io/valien/fishscale:latest
    container_name: fishscale
    restart: unless-stopped
    mem_limit: 256m
    cpus: 1.0
    volumes:
      - fishscale-data:/data
      - /dev/net/tun:/dev/net/tun
    cap_add:
      - NET_ADMIN
    environment:
      - TS_AUTHKEY=${TS_AUTHKEY}
      - TS_HOSTNAME=${TS_HOSTNAME:-fishscale}
      - TS_STATE_DIR=/data/tsnet-state
      - FISHSCALE_DB_PATH=/data/fish.db
      - FISHSCALE_PHOTO_DIR=/data/photos
      - FISHSCALE_LOG_LEVEL=${FISHSCALE_LOG_LEVEL:-info}
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"

volumes:
  fishscale-data:
```

2. Upload this file to the same `docker/fishscale` folder on your NAS

### 4d. Create the project in Container Manager

1. Open **Container Manager** from the DSM main menu
2. Go to the **Project** tab
3. Click **Create**
4. **Project name:** `fishscale`
5. **Path:** Select `/docker/fishscale` (the folder you created)
6. Container Manager will detect the `docker-compose.yml` automatically
7. Click **Next**, review the settings, then click **Done**

Container Manager will pull the image and start the container. This may take a minute or two on first run.

### 4e. Verify it's working

1. In Container Manager → **Project** → `fishscale`, check that the status is **Running**
2. Open your browser and go to:

```
https://fishscale.<your-tailnet>.ts.net
```

Replace `<your-tailnet>` with your Tailscale network name (visible at [Tailscale admin](https://login.tailscale.com/admin/machines)).

You should see the Fishscale app. Go catch some fish!

---

## 5. Option B — Build from Source (SSH)

For users comfortable with the command line.

### 5a. Enable SSH on your NAS

1. Open **Control Panel** → **Terminal & SNMP**
2. Check **Enable SSH service**
3. Click **Apply**

### 5b. Connect via SSH

From your computer's terminal:

```bash
ssh your-username@your-nas-ip
```

Replace `your-username` with your DSM admin username and `your-nas-ip` with your NAS's local IP address (e.g., `192.168.1.100`).

### 5c. Clone and configure

```bash
# Navigate to the docker shared folder
cd /volume1/docker

# Clone the repository
sudo git clone https://github.com/Valien/fishscale.git
cd fishscale

# Create the environment file
sudo tee .env > /dev/null <<'EOF'
TS_AUTHKEY=tskey-auth-PASTE-YOUR-KEY-HERE
TS_HOSTNAME=fishscale
EOF

# Restrict permissions on the env file
sudo chmod 600 .env
```

### 5d. Build and start

```bash
sudo docker compose up -d --build
```

The first build takes several minutes (it compiles the Go backend and Svelte frontend inside Docker). Subsequent rebuilds are faster due to layer caching.

### 5e. Verify

```bash
sudo docker compose ps
```

You should see the `fishscale` container with status `Up`. Access it at `https://fishscale.<your-tailnet>.ts.net`.

---

## 6. Keeping Your Auth Key Secure

Your Tailscale auth key is sensitive — anyone with it could join a device to your tailnet.

**Recommended:** Restrict file permissions on your `.env` file so only root can read it. Via SSH:

```bash
sudo chmod 600 /volume1/docker/fishscale/.env
```

**Be aware:** Container Manager's GUI displays environment variable values in plain text when you inspect a container. This is a Synology limitation. The values are only visible to DSM admin users, which on a personal NAS is typically just you.

**For extra security:** After Fishscale has connected to your tailnet for the first time, the Tailscale identity is persisted in the data volume. You can remove the `TS_AUTHKEY` line from your `.env` file — the container will reconnect using its stored identity on restart. You'd only need the auth key again if you delete the data volume.

---

## 7. Updating Fishscale

### Option A users (pre-built image)

1. Open **Container Manager** → **Project** → `fishscale`
2. Click **Action** → **Stop**
3. Click **Action** → **Build** (this pulls the latest image)
4. The project will restart with the new version

Your data is stored in a separate volume and is preserved across updates.

### Option B users (build from source)

Via SSH:

```bash
cd /volume1/docker/fishscale
sudo git pull
sudo docker compose up -d --build
```

---

## 8. Backup & Restore

### Where your data lives

All Fishscale data is in a Docker volume called `fishscale-data`:

| Path | Contents |
|------|----------|
| `/data/fish.db` | Your catches, trips, and settings |
| `/data/photos/` | Uploaded catch photos |
| `/data/tsnet-state/` | Tailscale device identity |

### Backup

**Using Hyper Backup (recommended):**

You can back up Docker volumes using Synology's Hyper Backup. Add the `fishscale-data` volume to your backup task.

**Manual backup via SSH:**

```bash
sudo docker cp fishscale:/data/fish.db ~/fish.db.backup
sudo docker cp fishscale:/data/photos ~/photos-backup
```

### Restore

```bash
# Stop the container
cd /volume1/docker/fishscale
sudo docker compose down

# Copy files back
sudo docker cp ~/fish.db.backup fishscale:/data/fish.db
sudo docker cp ~/photos-backup/. fishscale:/data/photos/

# Restart
sudo docker compose up -d
```

---

## 9. Troubleshooting

### Checking logs

In Container Manager → **Container** → `fishscale` → **Log** tab.

Or via SSH:

```bash
cd /volume1/docker/fishscale
sudo docker compose logs -f
```

### Common issues

| Problem | Likely Cause | Fix |
|---------|-------------|-----|
| Container won't start, log shows "TUN" error | `/dev/net/tun` doesn't exist | Run the TUN startup task from Section 3 (right-click → Run), then restart the container |
| Container starts but Fishscale isn't reachable | Auth key is invalid or expired | Generate a new auth key (Section 2), update `.env`, restart the project |
| "Permission denied" errors in logs | Volume ownership issue | Via SSH: `sudo docker exec fishscale ls -la /data` to check. The container runs as a non-root user |
| Container runs but not on tailnet | Firewall or ACL issue | Check [Tailscale admin](https://login.tailscale.com/admin/machines) for a `fishscale` node. If missing, check the auth key |
| Slow performance | Resource limits too low | Edit `docker-compose.yml`: increase `mem_limit` (e.g., `512m`) and `cpus` (e.g., `2.0`) |
```

**Step 2: Commit**

```bash
git add docs/guides/synology-nas-deployment.md
git commit -m "docs: add Synology NAS deployment guide

Step-by-step guide for deploying Fishscale on Synology DSM 7.2+
with Container Manager. Covers both pre-built image (GUI) and
build-from-source (SSH) paths. Includes TUN device setup,
auth key security, updating, backup, and troubleshooting.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 4: Add link to Synology guide in README

**Files:**
- Modify: `README.md`

**Step 1: Add link after the Deployment > Prerequisites section**

After the "### Prerequisites" section (around line 68), add a note pointing to the Synology guide. Insert after the prerequisites list and before "### Auth Key Setup":

```markdown
> **Synology NAS?** See the dedicated [Synology NAS Deployment Guide](docs/guides/synology-nas-deployment.md) for step-by-step instructions using Container Manager.
```

**Step 2: Commit**

```bash
git add README.md
git commit -m "docs: link to Synology deployment guide from README

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

### Task 5: Tag v1.0.0 and push

This is the final task. It tags the release and pushes to trigger the GHCR workflow.

**Step 1: Verify everything is committed**

Run: `git status`
Expected: clean working tree

**Step 2: Create the tag**

```bash
git tag -a v1.0.0 -m "v1.0.0: First release

Features:
- Catch logging with GPS, species, bait, weight, length, photos
- Interactive map with all catch locations
- Automatic weather from Open-Meteo
- Statistics dashboard with species breakdown and personal bests
- Trip tracking
- JSON and CSV export
- Light/dark/system theme with imperial/metric units
- Tailscale authentication (accessible only on your tailnet)
- Single binary with embedded Svelte frontend
- SQLite database with zero configuration
- Docker deployment with pre-built image on GHCR
- Synology NAS deployment guide"
```

**Step 3: Push commits and tag**

```bash
git push origin main
git push origin v1.0.0
```

The `v1.0.0` tag push triggers the GitHub Actions workflow, which builds and pushes `ghcr.io/valien/fishscale:1.0.0`, `ghcr.io/valien/fishscale:1.0`, and `ghcr.io/valien/fishscale:latest`.

**Step 4: Verify the workflow**

Run: `gh run list --workflow=publish-image.yml --limit 1`
Expected: a run triggered by the `v1.0.0` tag, status should be "in_progress" or "completed"

After it completes (~2-3 minutes), verify the image exists:

Run: `gh api /user/packages/container/fishscale/versions --jq '.[0].metadata.container.tags'`
Expected: `["1.0.0", "1.0", "latest"]`

**Step 5: Update todo.md**

Add to the Completed section in `docs/plans/todo.md`:

```markdown
- [x] ~~Synology NAS deployment guide and GHCR image publishing~~ (Completed 2026-02-18: deployment guide, Synology compose file, GitHub Actions workflow, v1.0.0 tagged, see synology-nas-deployment.md)
```

**Step 6: Commit and push the docs update**

```bash
git add docs/plans/todo.md
git commit -m "docs: mark Synology deployment guide as complete

Co-Authored-By: Claude <noreply@anthropic.com>"
git push origin main
```
