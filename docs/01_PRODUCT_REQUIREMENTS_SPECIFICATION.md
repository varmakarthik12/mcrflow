# Product Requirements Specification (PRS)
## Next-Gen Cloud-Native TV Playout & Channel Automation Platform
**Codename:** `MCRFlow Playout`  
**Target Class:** Enterprise-Grade Master Control & Playout Automation

---

## 1. Executive Summary & Market Positioning

Modern television broadcasting and FAST (Free Ad-Supported Streaming TV) networks require 24/7 linear playout automation that is agile, resilient, cost-effective, and independent of proprietary hardware dongles.

Traditional linear playout systems typically suffer from:
- Reliance on monolithic servers and hardware USB license dongles.
- Fragmented management UIs across separate desktop applications (separate playout scheduler, graphics maker, capture utility, and monitoring console).
- Rigid on-premise infrastructure with complex, vendor-locked redundancy switches.
- Clunky metadata entry with manual copy-pasting from movie databases.
- Inflexible deployment without native containerization or modern API-first architectures.

**MCRFlow Playout** solves this by providing a unified, web-native **React frontend** backed by high-performance **Go microservices** and distributed **Go Edge Playout Agents**. It combines broadcast-grade master control automation, dynamic resolution & FFmpeg profile management (with out-of-the-box presets for Indian Cable & DTH operations), real-time SCTE-35 ad insertion, intuitive WYSIWYG animated graphics creation, redundant failover, inline TMDb metadata enrichment, intelligent Telegram bot ChatOps, cross-platform standalone binaries via GoReleaser, and deep multi-language localization (defaulting to English with native support for all 10 Indian constitutional languages).

---

## 2. Industry Capabilities Matrix & Feature Standards

| Feature / Capability | Traditional Playout Systems | **MCRFlow Playout (Our Platform)** |
| :--- | :--- | :--- |
| **Architecture** | Monolithic OS server / client | **Cloud-Native Linux/Docker, Go + React** |
| **Web UI Management** | Fragmented desktop tools / partial web | **Single Responsive React UI for N Channels** |
| **Edge Agent Scalability** | Fixed physical server instances | **Dockerized Go Agent: All-in-One or Edge-Only** |
| **Zero-Interruption Redundancy** | Hardware bypass / manual switchover | **1+1 & N+M Hot Standby, Virtual IP / SRT Hitless** |
| **Agent Authentication** | Hardware USB dongle / node-locked license | **Auto-Generated Cryptographic Pairing Tokens** |
| **Streaming Outputs** | Limited SDI / UDP card outputs | **Multi-dest: UDP, SRT, RTMP, HLS, NDI, DeckLink** |
| **Graphics & Ad Maker** | Separate standalone CG software suites | **Built-in WYSIWYG Drag-and-Drop Canvas with CSS/Canvas Animations** |
| **Ad Splicing & Cueing** | Basic external GPI triggers | **SCTE-35 Splice Cues + Ad Break Pre/Mid/Post-Roll** |
| **Metadata Enrichment** | Manual CSV/XML import | **Inline Seamless TMDb / IMDb / TVDb Search & Autofill** |
| **Auto EPG Generation** | External XMLTV converter utility | **Built-in DVB-SI EIT, XMLTV, JSON, SCTE-118 Export** |
| **ChatOps / Remote Bot** | None | **Telegram Bot NLP Scheduling + Fuzzy Search + Conflict Resolver** |
| **Storage Connectors** | Local drives / fixed UNC share | **NAS (NFS/SMB), Local Disk Alias, Object Storage** |
| **Internationalization** | Single-language / English only | **English Default + 10 Indian Languages Built-in** |
---

## 3. Detailed Product Capabilities & Functional Specifications

### 3.1 Screen 1: Executive Operations Dashboard (Bird’s-Eye Telemetry)
The central operational nerve center designed for Broadcast Engineers and Master Control Operators (MCOs).
- **Global Channel Overview**:
  - Live grid or card matrix of all active channels with current on-air thumbnail/live WebRTC/HLS proxy stream.
  - Channel state indicators: `ON-AIR (PRIMARY)`, `ON-AIR (FALLBACK)`, `STANDBY`, `HOLD`, `ERROR`.
  - Current item progress bar with elapsed time, remaining time, and SMPTE timecode (`01:24:18:12`).
  - Next queued item preview with start time countdown.
- **System Telemetry & Resource Gauges**:
  - Cluster-wide CPU, GPU NVENC/VAAPI utilization, RAM consumption, and network I/O throughput (Mbps ingress vs egress).
  - Edge Agent health telemetry: Ping latency, frame drops, buffer health, encoder bitrates.
- **Active Stream Destination Counters**:
  - Breakdown of active outputs across channels: UDP Multicast endpoints, SRT listeners/callers, RTMP CDN publishing points, HLS package health.
- **Incident & Alarm Log (Real-time SSE / WebSocket feed)**:
  - Severity-coded alerts: Critical (agent disconnected, transcode stalled, audio silence detected, black frame detected), Warning (packet loss on SRT > 1%, disk storage capacity > 85%), Info (playlist schedule rollover, ad break started).
  - One-click acknowledge and filter by channel.

### 3.2 Screen 2: Multi-Channel Master Management & Live Control
Enables operators to provision, configure, and monitor an arbitrary number of broadcast channels ($N$ channels).
- **Channel Configuration & Identity**:
  - Channel Name, Call Sign, Unique Channel ID, Channel Number / LCN (Logical Channel Number).
  - Default Output Resolution & Frame Rate: 1080p50/59.94, 1080i50, 720p60, 4K UHD 2160p.
  - Video Encoder Profiles: H.264 (AVC), H.265 (HEVC), AV1, MPEG-2 TS (for legacy cable/DVB).
  - Audio Format & Bitrate: AAC-LC, AC-3 (Dolby Digital), Enhanced AC-3 (E-AC-3), MPEG-1 Layer II.
- **Edge Agent Assignment & Redundancy Pairing**:
  - Primary Edge Agent selector (from pool of registered agents).
  - Secondary / Fallback Edge Agent selector (hot standby).
  - Redundancy Mode: `1+1 Mirroring` (simultaneous duplicate streams with hitless receiver merge) or `Active-Passive Auto-Failover` (standby takes over within < 100ms on primary heartbeat loss).
  - Health check ping interval (default: 500ms, timeout: 1500ms).
- **Multi-Destination Stream Egress Routing**:
  - Independent toggle and configuration for:
    - **UDP TS**: Multicast IP (`udp://239.255.0.1:5000`), Unicast IP, TTL, Interface binding.
    - **SRT**: Mode (`Caller`, `Listener`, `Rendezvous`), Port, Latency (ms), Passphrase (AES-128/256), Stream ID.
    - **RTMP / RTMPS**: Server URL, Stream Key (auto-obscured), Reconnect interval.
    - **HLS / LL-HLS**: Local HTTP packaging path, segment duration (2s-6s), playlist window, SCTE-35 signaling in HLS manifest (`#EXT-X-DATERANGE`).
    - **SDI / NDI**: Blackmagic DeckLink output card selection, NDI channel discovery name.
- **On-Air Branding & Default Channel Ad Template**:
  - Station Logo / Channel Bug upload (PNG with transparency, SVG, animated APNG/WebP).
  - Bug screen position (Top-Left, Top-Right, Bottom-Left, Bottom-Right) with customizable X/Y offsets, opacity slider, and auto-dimming during commercials.
  - Default Global Ad Template dropdown: Applies fallback graphics and commercial rolls to any scheduled media that lacks content-specific ad templates.
- **Integrated Live Preview Player**:
  - Real-time video preview of the channel's actual transmission output directly in the side pane via low-latency WebRTC (WHIP/WHEP) or ultra-low-latency HLS.
  - Audio VU meter (Left / Right stereo + 5.1 surround) with true-peak indicator.
  - Emergency manual controls: `Skip to Next`, `Hold / Freeze`, `Emergency Slate (Logo + Music Loop)`, `Restart Playout Engine`.

### 3.3 Screen 3: Precision Scheduling, Storage Browser, Metadata & Audio Routing
The core broadcast playlist creation and automation screen.
- **Interactive Broadcast Timeline & Grid**:
  - Daily 24-hour visual timeline bar with color-coded blocks: Content (Blue), Commercial Break (Green), Station ID/Promo (Purple), Emergency/Live Feed (Orange).
  - SMPTE frame-accurate timecode display (`HH:MM:SS:FF`).
  - Item status: `Played`, `On-Air Now`, `Cued Next`, `Pending`, `Missing Media Error`.
- **Integrated Storage Browser (Source Picker)**:
  - Dropdown selecting storage mount alias (configured in Settings: NAS, SMB, NFS, Local Storage Alias).
  - Interactive breadcrumb file explorer (`/nas-movies/bollywood/2025/`).
  - File details: Filename, size, extension, video codec, audio channels, duration.
  - Proactive ffprobe inspection: Auto-reads duration, container framerate, and audio track IDs upon file selection.
  - Auto-calculation: End time is automatically calculated based on Selected Start Time + Probed Video Duration.
- **Seamless Inline TMDb / IMDb / TVDb Search & Match**:
  - Directly embedded in the content selection modal/drawer without page switching.
  - Auto-search triggered using cleaned filename (e.g., stripping `.mp4`, `1080p`, `x264` tokens).
  - Fallback manual search input for custom query.
  - Results card view: Displays official movie poster, original title, localized title (based on current Indian locale or English), release year, synopsis, genres, runtime, and MPAA / CBFC rating.
  - One-click "Select & Apply": Instantly injects synopsis, poster URL, cast, and directors into the event metadata for automated EPG publication.
- **Hierarchical Ad Template Precedence**:
  - Content-Specific Ad Template selector dropdown:
    - *Precedence Rule*: If a template is chosen here, it overrides the channel's global ad template for the duration of this movie.
    - *Fallback*: If left as `Use Channel Global Template (Default)`, the channel-level template remains active.
- **Audio Track Management & Multi-PID Routing**:
  - Probes all audio streams embedded within the media file.
  - Displays track index, language tag (`hin`, `tam`, `tel`, `eng`, `und`), codec (`aac`, `ac3`, `eac3`, `pcm`), channels (`2.0 Stereo`, `5.1 Surround`).
  - Selector allows the operator to designate which audio PID to broadcast, or map multiple languages to discrete output audio streams (e.g., Audio 1: Hindi, Audio 2: English commentary).
  - Audio normalization toggle: EBU R128 / CALM Act (-23 LUFS target, ±1 LU tolerance).
- **Subtitle & Closed Caption Management**:
  - Detects embedded subtitle streams (`mov_text`, `subrip`, `dvb_subtitle`) and sidecar `.srt`/`.vtt` files in the folder.
  - Selector to enable closed captions (CEA-608/708 VANC insertion) or open captions (burn-in overlay).
- **Advanced FFmpeg Parameters (Collapsible Accordion)**:
  - Custom video filters: Deinterlacing (`yadif`), aspect ratio scaling (`scale=1920:1080:force_original_aspect_ratio=decrease,pad=1920:1080:(ow-iw)/2:(oh-ih)/2`), color matrix conversion.
  - Audio filters: Dynamic range compression, custom channel remixing matrix.
  - Custom transport stream metadata: Service Name, Provider Name, PMT PID, Video PID, Audio PID, PCR PID.

### 3.4 Screen 4: WYSIWYG Ad & CG Template Maker
A visual graphics design studio eliminating the need for expensive third-party CG tools or video editors.
- **Dual Template Architecture**:
  1. **Section A: On-Screen Persistent / Timed Banners & Bugs (Overlay CG)**:
     - Graphic elements rendered on top of active video without interrupting playback.
     - Element types:
       - Static or transparent image (PNG, WebP).
       - Animated GIF / WebP (for dynamic animated sponsor logos).
       - Lower-Third Graphic (sponsor bug + movie title + next up teaser).
       - Breaking News / Announcement Ticker Crawl (horizontal scrolling text with custom speed and background banner).
       - Video-in-Video (DVE squeeze-back: scales down main video to 75% and displays L-shaped sponsor ad).
     - **Interactive WYSIWYG Drag-and-Drop Canvas**:
       - 16:9 canvas preview representing 1920x1080 broadcast raster.
       - Visual drag handles for moving, resizing, and snapping to safe broadcast margins (Action Safe 90%, Title Safe 80%).
       - Layer management (Bring to front, Send to back, Lock layer, Hide layer).
     - **Built-in No-Code Transitions & Animations**:
       - Entrance animations: `Fade In`, `Slide In (Left/Right/Top/Bottom)`, `Zoom In`, `Bounce In`, `Wipe`.
       - Exit animations: `Fade Out`, `Slide Out`, `Zoom Out`.
       - Timing controls: Entrance start offset (seconds into movie), element display duration, repeat interval (e.g., show every 15 minutes for 10 seconds).
  2. **Section B: Commercial Ad Rolls (Linear Break Schedulers)**:
     - Dedicated ad video slots inserted into the playout stream.
     - Break types:
       - `Pre-Roll`: Plays before the main feature starts (sponsors, channel promo).
       - `Mid-Roll`: Spliced at specified timecodes (e.g., 00:30:00, 01:00:00) or automatic cue intervals.
       - `Post-Roll`: Plays immediately following feature conclusion before next program.
     - Multi-clip playlist inside ad roll: Supports sequencing multiple sponsor clips with automated cross-fade or hard cut.
     - SCTE-35 Cue Insertion: Automatically emits `splice_insert()` or `time_signal()` cues with splice duration and event IDs for downstream digital ad insertion (DAI) by cable operators and SSAI servers.

### 3.5 Screen 5: Advanced Infrastructure & System Settings
Unified configuration for backend nodes, distributed edge agents, network mesh, storage, and bot integrations.
- **Section 1: Distributed Edge Agent Management & Cryptographic Pairing**:
  - Displays registered playout agents across the infrastructure.
  - Agent cards: Agent Hostname, IP address (LAN or Tailscale IP), Status (`ONLINE`, `STREAMING`, `STANDBY`, `OFFLINE`), Assigned Channels, CPU/GPU transcode capability.
  - **Pairing Workflow**:
    - When an Edge Agent boots in agent-only mode, it generates a cryptographically secure token (SHA-256 HMAC / UUIDv4 entropy token) saved to `/var/lib/playout-agent/token` and printed to Docker logs.
    - Operator enters Agent IP/Host and the secure token in the UI.
    - Control Plane verifies handshake over gRPC via mTLS / token header, registers agent into cluster database, and initiates heartbeat monitoring.
    - Token survives restarts and can be rotated/revoked from this screen.
- **Section 2: Storage Device Management**:
  - Add and manage storage mount definitions:
    - `NAS (NFS v3/v4)`: Server address (`192.168.1.50`), Export path (`/volume1/broadcast_media`), Mount options.
    - `SMB / CIFS Share`: Hostname, Share name, Username, Password / Secret, Domain.
    - `Local Filesystem Alias`: Path on host (`/mnt/storage/movies`), Friendly alias (`Local RAID Master`).
  - Real-time mount status indicator (Mounted, Unreachable, Read-Only).
  - Storage capacity gauge (Used / Free space, auto-warning when < 10% space remains).
- **Section 3: Bot & ChatOps Configuration (Multiple Bots)**:
  - Configuration page supporting multiple communication channels:
    - **Telegram Bot**: Bot Name, API Token (`bot123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11`), Allowed User IDs / Chat IDs (whitelist for broadcast control security), Enabled Channels toggle.
    - Future expansion slots for Slack Webhook, WhatsApp Cloud API, and Discord.
  - Natural Language Playout Processor:
    - Telegram webhook listener parses natural commands (e.g., `"Schedule Avengers at 9 AM on Channel 1"`).
    - Executes fuzzy search against indexed storage mounts (e.g., matches `"Avengers: Endgame (2019) 1080p.mkv"`).
    - Responds in chat with interactive inline keyboard buttons confirming title match.
    - **Conflict Resolution Engine**: If the 9:00 AM slot is occupied, the bot detects the collision and presents 3 instant action buttons:
      1. `[Force Overwrite Slot]` (Overwrites existing program).
      2. `[Queue After Current Ends]` (Schedules immediately after current program finishes).
      3. `[Replace Conflicting Program]` (Replaces overlapping item and ripples future schedule).
- **Section 4: Intranet & Service Mesh (Tailscale / WireGuard)**:
  - Tailscale AuthKey input or Tailscale Daemon status widget.
  - Node MagicDNS name display, enabling zero-config secure edge-to-cloud mesh communication across remote data centers and cloud VPCs without public IP exposure or port forwarding.
- **Section 5: Internationalization (i18n) & Regional Configuration**:
  - Active UI Language selector with real-time preview (English + 10 Indian constitutional languages: Hindi, Tamil, Telugu, Bengali, Marathi, Gujarati, Kannada, Malayalam, Punjabi, Odia).
  - Timezone selector (defaults to `Asia/Kolkata` IST).
  - SMPTE framerate defaults (25 fps for PAL/India/Europe, 29.97/30 fps for NTSC).
  - TMDb API Key and Primary Metadata Language default.

---

## 4. Automatic EPG (Electronic Program Guide) Generation Engine

The system features an automated EPG generation microservice that runs continuously:
1. **DVB-SI Event Information Table (EIT)**:
   - Formats schedule into binary DVB-SI EIT Actual/Other tables ($P/F$ Present/Following and Schedule tables).
   - Injects multilingual Short Event Descriptors (`ISO 639-2` language tags `hin`, `tam`, `tel`, `eng`) populated directly from TMDb metadata.
   - Embeds Content Advisory & Parental Rating descriptors (`DVB-SI ETSI EN 300 468`).
2. **XMLTV Format (`xmltv.xml`)**:
   - Standards-compliant XMLTV output served over HTTP (`/api/v1/epg/{channel_id}/xmltv.xml`) for OTT, IPTV, and Plex/Jellyfin integrations.
   - Includes `<programme start="..." stop="..." channel="...">`, `<title>`, `<sub-title>`, `<desc>`, `<category>`, `<icon src="...">`.
3. **SCTE-118 / CableLabs ADI**:
   - Cable-ready scheduling metadata for linear cable operators and digital headends.
