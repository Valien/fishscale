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

**Alternatively**, you can download the ready-made [`docker-compose.synology.yml`](https://github.com/Valien/fishscale/blob/main/docker-compose.synology.yml) from the Fishscale repository and rename it to `docker-compose.yml` before uploading.

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
