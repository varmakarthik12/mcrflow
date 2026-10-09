# Docker & Container Deployment Architecture Guide
## Enterprise High-Availability & Edge Agent Pairing Lifecycle

---

## 1. Overview of Build Modes

MCRFlow Playout is engineered for modern microservices and container orchestration. It provides two purpose-built Docker image targets, alongside cross-platform standalone binaries:

1. **`mcrflow-all-in-one:latest` (Single-Box Turnkey Node)**
   - **Target**: Regional broadcast studios, cable headends, testing labs, or single-node deployments.
   - **Contents**: React Web Console, Go Control Plane backend (`mcrflow-control`), embedded Go Local Playout Agent (`mcrflow-agent`), FFmpeg 7.x/8.x broadcast media suite, embedded SQLite database.
   - **Startup**: Uses Supervisor to orchestrate the control plane and embedded agent locally.

2. **`mcrflow-agent-only:latest` (Headless Edge Worker Node)**
   - **Target**: Distributed playout farms, cloud worker clusters, and remote transmitters.
   - **Contents**: Ultra-lean Go daemon binary (`mcrflow-agent`), hardware-accelerated FFmpeg / GStreamer pipelines (NVIDIA NVENC, Intel QuickSync, VAAPI), OpenSSL.
   - **Footprint**: < 120MB base image, headless (no web server, no database).

---

## 2. Cryptographic Token Lifecycle & Pairing Workflow

```mermaid
sequenceDiagram
    autonumber
    participant Admin as Systems Administrator
    participant Edge as Edge Agent Container
    participant Disk as Persistent Volume (/var/lib/mcrflow-agent)
    participant CP as MCRFlow Web UI / Control Plane

    Admin->>Edge: docker run -v agent-data:/var/lib/mcrflow-agent ...
    Edge->>Disk: Check if agent_auth.json exists
    alt First Boot (Fresh Install)
        Edge->>Edge: crypto/rand generates 256-bit entropy token
        Edge->>Disk: Write agent_auth.json (chmod 0600)
        Edge->>Edge: Print prominent banner to container stdout
    else Container Restart
        Edge->>Disk: Load existing pairing_token from agent_auth.json
        Edge->>Edge: Resume without resetting token
    end
    Admin->>Edge: docker logs edge-agent
    Edge-->>Admin: Displays [agt_sec_8f43a9b2c011e749a1d2e8b4...]
    Admin->>CP: Navigate to Settings > Edge Agents > Pair New Agent
    Admin->>CP: Enter Agent IP:Port and paste Token
    CP->>Edge: gRPC Handshake over mTLS with x-agent-token
    Edge-->>CP: Constant-time Token Verification -> Success
    CP-->>Admin: Agent Paired & Active (🟢 ONLINE)
```

### 2.1 First-Boot Token Generation
When `mcrflow-agent-only` boots for the first time:
1. The container entrypoint checks `/var/lib/mcrflow-agent/agent_auth.json`.
2. If absent, it invokes `openssl rand -hex 32` (or Go's `crypto/rand`), yielding a 256-bit cryptographically secure token prefixed with `agt_sec_`.
3. The file is saved with strict `0600` permissions.
4. The token is emitted to `stdout` in an ASCII banner:
   ```text
   ================================================================================
     MCRFLOW EDGE PLAYOUT AGENT - CRYPTOGRAPHIC PAIRING REQUIRED
   ================================================================================
     Agent ID:         delhi-edge-primary-01
     Listen Port:      :9095 (gRPC / mTLS)
     Persistent Token: agt_sec_8f43a9b2c011e749a1d2e8b409c2513f87a6b4c3d2e1f0a9b8c7d6e5f4a3b2c1

     👉 Copy and paste the token above into MCRFlow Web UI:
        Settings > Edge Agents > Pair New Agent
   ================================================================================
   ```

### 2.2 Container Restarts & Persistence
- **No Token Drift**: Container restarts, node reboots, or container upgrades (`docker stop && docker rm && docker run`) read the persistent volume.
- The existing token is maintained without reset, ensuring zero pairing disruption.
- **Revocation**: The token can only be reset via an explicit authenticated command from the Control Plane UI (`Reset Agent Token`) or by manually deleting `/var/lib/mcrflow-agent/agent_auth.json`.

---

## 3. High-Availability 1+1 Redundancy Deployment

To configure a broadcast channel with 1+1 active-passive auto-failover:

1. **Deploy Control Plane Node (Master Server)**:
   ```bash
   docker run -d --name mcrflow-cp \
     -p 8080:8080 -p 9090:9090 \
     -v mcrflow-data:/var/lib/mcrflow-data \
     mcrflow-all-in-one:latest
   ```

2. **Deploy Primary Edge Agent (Site A - Delhi Data Center)**:
   ```bash
   docker run -d --name agent-delhi-primary \
     --gpus all \
     -p 9095:9095 -p 5000-5010:5000-5010/udp \
     -v agent-delhi-auth:/var/lib/mcrflow-agent \
     -v /nas/movies:/media/storage:ro \
     mcrflow-agent-only:latest
   ```

3. **Deploy Fallback Edge Agent (Site B - Mumbai Data Center)**:
   ```bash
   docker run -d --name agent-mumbai-fallback \
     --gpus all \
     -p 9095:9095 -p 5000-5010:5000-5010/udp \
     -v agent-mumbai-auth:/var/lib/mcrflow-agent \
     -v /nas/movies:/media/storage:ro \
     mcrflow-agent-only:latest
   ```

4. **Pair Agents in Management Console**:
   - In the Web UI (Screen 5: Settings > Edge Agents), pair both agents using their respective tokens.
   - In Screen 2 (Channel Management), assign `agent-delhi-primary` as Primary and `agent-mumbai-fallback` as Fallback.
   - Select Redundancy Mode: `Active-Passive Auto-Failover` or `1+1 Mirroring`.

---

## 4. Hardware GPU Acceleration & Storage Mounts

### 4.1 GPU Passthrough
- **NVIDIA GPU (NVENC/NVDEC)**:
  Add `--gpus all` and ensure NVIDIA Container Toolkit is installed on the host. The Go agent automatically invokes `-c:v h264_nvenc` or `-c:v hevc_nvenc`.
- **Intel QuickSync / VAAPI**:
  Pass device nodes `--device /dev/dri:/dev/dri`. The agent selects `-vaapi_device /dev/dri/renderD128 -vf 'format=nv12,hwupload' -c:v h264_vaapi`.

### 4.2 Network Storage Mounting (NAS / SMB / NFS)
Network storage can be mounted to the container in two ways:
1. **Host-Level Mount (Recommended for high throughput)**:
   Mount the SMB/NFS share on the Docker host OS (`/mnt/nas_movies`), then bind-mount into the container `-v /mnt/nas_movies:/media/storage:ro`.
2. **Container-Managed Mount (Configured via UI Screen 5)**:
   The Control Plane executes `mount -t cifs -o username=... //nas.local/movies /media/mounts/smb1` using the container's built-in `cifs-utils` and `nfs-common`.

---

## 5. Intranet & Service Mesh Setup (Tailscale / WireGuard)

For multi-site setups where edge playout nodes reside behind carrier-grade NAT (CGNAT) or cellular 5G modems without static public IPs:

```yaml
services:
  tailscale:
    image: tailscale/tailscale:latest
    container_name: ts-edge-delhi
    hostname: edge-delhi-playout
    environment:
      - TS_AUTHKEY=tskey-auth-k123456789-xxxxxxxx
      - TS_STATE_DIR=/var/lib/tailscale
    volumes:
      - ts-state:/var/lib/tailscale
      - /dev/net/tun:/dev/net/tun
    cap_add:
      - NET_ADMIN
    restart: unless-stopped

  edge-agent:
    image: mcrflow-agent-only:latest
    network_mode: service:tailscale
    depends_on:
      - tailscale
    volumes:
      - agent-auth:/var/lib/mcrflow-agent
      - /nas/movies:/media/storage:ro
```
The agent is immediately reachable on its secure Tailscale MagicDNS IP (e.g., `100.64.1.15:9095`) via WireGuard encryption without any open router ports.

---

## 6. Non-Docker Standalone Binaries (GoReleaser)

For broadcast environments where containerization is not permitted or desired:
- **Build Spec**: Defined in `.goreleaser.yaml`.
- **Binaries Published**:
  - `mcrflow-control`: Control Plane & UI Server
  - `mcrflow-agent`: Standalone Edge Playout Daemon
- **Supported Operating Systems & Architectures**:
  - Linux (`amd64`, `arm64`)
  - Windows (`amd64`)
  - macOS (`darwin/amd64`, `darwin/arm64`)
- **Direct Invocation Example (Windows / Linux)**:
  ```bash
  # Start Edge Agent Daemon
  ./mcrflow-agent -port 9095 -auth-dir /var/lib/mcrflow-agent

  # Start Control Plane
  ./mcrflow-control -port 8080
  ```
