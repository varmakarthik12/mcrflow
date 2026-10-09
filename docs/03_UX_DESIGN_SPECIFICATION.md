# UX Design Specification & Interaction Architecture
## MCRFlow Playout - Master Control Console

---

## 1. Design System & Ergonomics

Television playout consoles are operated in high-stress, low-light Master Control Rooms (MCRs) 24 hours a day, 7 days a week.

### 1.1 Aesthetic & Visual Palette
- **Base Theme**: Deep dark slate broadcast console (`#0B0F17` background, `#111827` surface cards, `#1F2937` borders).
- **Accent Signals**:
  - `Emerald 500` (`#10B981`): Healthy, Active Transmission, On-Air.
  - `Amber 500` (`#F59E0B`): Warning, Standby, Redundancy Failover Active, Schedule Conflict.
  - `Rose 500` (`#EF4444`): Critical Alarm, Agent Offline, Stream Stalled, Silence Detected.
  - `Indigo 500` / `Sky 400` (`#6366F1` / `#38BDF8`): System Accent, Cued Item, Selection Highlight.
- **Typography**:
  - Primary UI & Timecodes: Inter font with tabular numbers (`tnum` font feature) to prevent jitter in live millisecond counters.
  - Indic Scripts: Noto Sans regional unicode font families with proper line-height (`1.5` - `1.6`) for Hindi, Tamil, Telugu, Bengali, Marathi, Gujarati, Kannada, Malayalam, Punjabi, and Odia.
- **Information Density**: High-density compact data tables with expandable drawers, instant tooltips, and keyboard shortcuts (`Space` for preview play/pause, `Esc` to close modals, `Ctrl+S` to save schedule).

---

## 2. Detailed Screen Specifications & Wireframe Layouts

### 2.1 Screen 1: Master Operations Dashboard
```text
+----------------------------------------------------------------------------------------------------+
|  [MCRFlow Logo]   Channels: 8 Active | Health: 99.98% | Time: 16:15:32 IST (UTC+5:30) | [🌐 Hindi▼]  |
+----------------------------------------------------------------------------------------------------+
|  [Cluster CPU: 24%]  [GPU NVENC: 38%]  [Memory: 14.2 GB]  [Egress: 184 Mbps]  [Active Destinations: 24] |
+----------------------------------------------------------------------------------------------------+
|  CHANNELS GRID (Bird's Eye View)                                                                   |
|  +------------------------------+ +------------------------------+ +------------------------------+|
|  | CH 01: Star Gold HD   [ON-AIR]| | CH 02: Cinema One    [ON-AIR]| | CH 03: Action Prime [STANDBY]| |
|  | [Live Video Proxy Thumbnail] | | [Live Video Proxy Thumbnail] | | [Live Video Proxy Thumbnail] | |
|  | Now: Jawan (2023) [01:42:15] | | Now: RRR (2022)   [00:54:12] | | Now: Leo (2023)   [02:10:05] | |
|  | [=============>---------] 65%| | [=======>-------------] 34%| | [====================>] 98%| |
|  | Next: 17:30 - Pathaan        | | Next: 18:00 - Pushpa 2       | | Next: 16:30 - Jailer         | |
|  | Outputs: UDP | SRT | HLS     | | Outputs: SRT | RTMP          | | Outputs: UDP Multicast       | |
|  | Agent: Primary (Delhi-DC1)   | | Agent: Primary (Mum-DC2)     | | Agent: Standby (Failover)    | |
|  +------------------------------+ +------------------------------+ +------------------------------+|
+----------------------------------------------------------------------------------------------------+
|  REAL-TIME INCIDENT & TELEMETRY LOGS (Filter: All Channels ▼ | Severity: All ▼)                     |
|  [16:14:02] [INFO]    CH 01: Ad Break Mid-Roll 2 completed. Returned to movie feed.                |
|  [16:11:18] [WARNING] CH 02: SRT output to Akamai CDN retransmitted 14 lost packets.                |
|  [16:05:40] [CRITICAL] Agent 'mum-dc-backup' reconnected after 45s heartbeat timeout.                |
+----------------------------------------------------------------------------------------------------+
```

### 2.2 Screen 2: Channel Management & Live Master Control
```text
+----------------------------------------------------------------------------------------------------+
|  < Channels / CH 01: Star Gold HD (LCN 101)                [+ Add New Channel] [Save Configuration]|
+--------------------------------------------------------+-------------------------------------------+
|  CHANNEL SPECIFICATIONS & AGENT ASSIGNMENT             |  LIVE ON-AIR TRANSMISSION MONITOR         |
|                                                        |  +-------------------------------------+  |
|  Channel Name: [Star Gold HD            ]  LCN: [101 ] |  |                                     |  |
|  Resolution:   [1080p50 (PAL / India)   ▼]             |  |      [LIVE WEBRTC PROXY VIDEO]      |  |
|  Video Codec:  [H.264 (NVENC GPU)       ▼]             |  |                                     |  |
|  Bitrate:      [8,500 kbps              ]              |  |  On-Air: Jawan (2023)               |  |
|                                                        |  |  Timecode: 01:42:15:18 / 02:49:00   |  |
|  PRIMARY EDGE AGENT:   [delhi-dc1-primary (Online)   ▼]|  +-------------------------------------+  |
|  FALLBACK AGENT (1+1): [mumbai-dc2-hotstandby (Ready)▼]|  AUDIO VU METERS:                         |
|  Redundancy Mode:      [1+1 Auto-Failover (Active/Pass)▼] |  L: [||||||||||||||||||.......] -14 dB |  |
|                                                        |  R: [||||||||||||||||||.......] -14 dB |  |
|  BRANDING & DEFAULT AD TEMPLATE:                       |  C: [|||||||||||||||||||......] -12 dB |  |
|  Channel Logo / Bug:   [stargold_hd_bug.png] [Browse]  |  LFE:[||||||||................] -20 dB |  |
|  Position: [Top-Right ▼]  Opacity: [ 85%  ]            |  Loudness: -23.1 LUFS (EBU R128 Compliant)|
|  Default Ad Template:  [Diwali Prime Sponsor Bug     ▼]|                                           |
|                                                        |  MASTER CONTROL OVERRIDES:                 |
|  STREAM DESTINATIONS:                                  |  [🛑 Emergency Slate]   [⏭️ Skip to Next]  |
|  [x] UDP Multicast: [udp://239.255.10.1:5000         ] |  [⏸️ Freeze / Hold]    [🔄 Restart Engine]|
|  [x] SRT Egress:    [srt://edge.star.in:9000 (Caller)] |                                           |
|  [x] RTMP CDN:      [rtmp://live.youtube.com/live2   ] |                                           |
|  [ ] LL-HLS Web:    [/var/www/hls/ch01/index.m3u8    ] |                                           |
+--------------------------------------------------------+-------------------------------------------+
```

### 2.3 Screen 3: Scheduling, Storage Browser & Seamless TMDb Enrichment
```text
+----------------------------------------------------------------------------------------------------+
|  [Schedule]  Channel: [CH 01: Star Gold HD ▼]  Date: [2026-10-08 ▼]  [+ Add Item] [Auto-Fill Gap]  |
+----------------------------------------------------------------------------------------------------+
|  24-HOUR VISUAL TIMELINE:                                                                          |
|  [00:00 ===== [06:00 ===== [12:00 ===== [16:00 ==NOW== [20:00 ===== [23:59]                       |
+----------------------------------------------------------------------------------------------------+
|  MODAL / DRAWER: ADD / EDIT SCHEDULE ITEM                                                          |
|                                                                                                    |
|  1. LOCAL MEDIA LIBRARY PICKER:                                                                    |
|  Directory:     [/media/movies/bollywood/2023/                                                  ]  |
|  +-----------------------------------------------------------------------------------------------+ |
|  |  📄 Jawan (2023) 1080p Atmos.mkv      [Selected]  Size: 14.2 GB  Probed Dur: 02:49:12          | |
|  |  📄 Dunki (2023) 1080p.mkv                        Size: 11.5 GB  Probed Dur: 02:40:05          | |
|  |  📄 Tiger 3 (2023) 1080p.mkv                      Size: 12.8 GB  Probed Dur: 02:35:48          | |
|  +-----------------------------------------------------------------------------------------------+ |
|                                                                                                    |
|  2. SEAMLESS INLINE TMDb / IMDb SEARCH & METADATA MATCH:                                           |
|  [Search TMDb: "Jawan 2023"                          ] [🔍 Refresh Search]                         |
|  +-----------------------------------------------------------------------------------------------+ |
|  |  [Poster]  Jawan (जवान) (2023) • Action / Thriller • 169 mins • CBFC: U/A 16+                  | |
|  |            Director: Atlee | Stars: Shah Rukh Khan, Nayanthara, Vijay Sethupathi               | |
|  |            Synopsis: A high-octane action thriller outlining the emotional journey of a man... | |
|  |            [✅ Matched & Applied for EPG]                                                      | |
|  +-----------------------------------------------------------------------------------------------+ |
|                                                                                                    |
|  3. TIMING & DURATION AUTO-CALCULATION:                                                            |
|  Start Time: [ 16:30:00 IST ]  Video Duration: [ 02:49:12 ]  Calculated End: [ 19:19:12 IST ]      |
|                                                                                                    |
|  4. HIERARCHICAL AD TEMPLATE OVERRIDE:                                                             |
|  Ad Template: [Blockbuster Movie Sponsor Template (Overrides Channel Default)                   ▼]|
|                                                                                                    |
|  5. MULTI-LANGUAGE AUDIO & SUBTITLE MANAGEMENT:                                                    |
|  Audio Stream:    [Track 1: Hindi 5.1 Dolby TrueHD (Stream #0:1)                                ▼] |
|  Loudness Auto:   [x] Enable EBU R128 (-23 LUFS Target)                                            |
|  Subtitles:       [Track 2: English Sidecar SRT -> CEA-708 CC Insertion                         ▼] |
|                                                                                                    |
|  6. ADVANCED FFmpeg PARAMETERS (COLLAPSIBLE):                                                      |
|  [+] Deinterlacing: yadif=0:-1:1 | Aspect Ratio: 16:9 Pad | Custom PMT PID: 4096 | Service: STAR   |
|                                                                                                    |
|  [Cancel]                                                             [💾 Confirm & Save Schedule] |
+----------------------------------------------------------------------------------------------------+
```

### 2.4 Screen 4: WYSIWYG Ad & CG Template Maker
```text
+----------------------------------------------------------------------------------------------------+
|  < Ad Templates / "Diwali Prime Sponsor Bug & Rolls"                            [Save Template]     |
+----------------------------------------------------------------------------------------------------+
|  [Tab: On-Screen Banners (CG Overlays)]   [Tab: Commercial Ad Rolls (Pre/Mid/Post)]                |
+----------------------------------------------------------------------------------------------------+
|  WYSIWYG CANVAS PREVIEW (1920 x 1080)                                                              |
|  +-----------------------------------------------------------------------------------------------+ |
|  | [Action Safe Area: 90% Outline] [Title Safe Area: 80% Outline]                                | |
|  |                                                        [Sponsored By: Vivo Festive Logo (PNG)] | |
|  |                                                         (Entrance: Slide Right | Exit: Fade)   | |
|  |                                                                                                | |
|  |                                                                                                | |
|  | [LOWER THIRD BANNER (X: 120px, Y: 890px)]                                                     | |
|  | "COMING UP NEXT: IPL 2026 LIVE STREAMING AT 7:30 PM" (Slide Bottom | 15s Duration)            | |
|  | [TICKER CRAWL] >>> Breaking News: Festive Mega Discount Announced >>> (Continuous Speed: 4)   | |
|  +-----------------------------------------------------------------------------------------------+ |
|  ELEMENT PALETTE & ANIMATION CONTROLS:                                                             |
|  [+ Add Transparent PNG]  [+ Add Animated GIF]  [+ Add Lower-Third]  [+ Add Ticker Crawl]          |
|  Selected: Lower-Third Banner                                                                      |
|  - Entrance Transition: [Slide Up From Bottom ▼]   - Exit Transition: [Fade Out ▼]                 |
|  - Animation Duration:  [600 ms               ]   - Display On Screen:[15 seconds]                 |
|  - Repeat Interval:     [Every 15 minutes     ]   - Opacity:          [90%]                        |
|  [▶️ Test Animation Playback]  [⏪ Reset Canvas]                                                   |
+----------------------------------------------------------------------------------------------------+
|  SECTION B: AD BREAKS & COMMERCIAL ROLLS                                                            |
|  - Pre-Roll Break:  [30s Channel Intro Teaser.mp4] -> [SCTE-35 Splice: 1001]                      |
|  - Mid-Roll Breaks: [Interval: Every 45 Minutes] (2 Commercial Clips: Vivo.mp4, Coke.mp4) [SCTE-35]|
|  - Post-Roll Break: [15s Channel Ident.mp4]                                                       |
+----------------------------------------------------------------------------------------------------+
```

### 2.5 Screen 5: Advanced Infrastructure & System Settings
```text
+----------------------------------------------------------------------------------------------------+
|  ADVANCED SETTINGS & INFRASTRUCTURE                                                                |
+----------------------------------------------------------------------------------------------------+
|  [Resolutions & Presets]  [Edge Agents & Pairing]  [User Management]  [Bot & ChatOps]  [i18n & Regional]|
+----------------------------------------------------------------------------------------------------+
|  SECTION: RESOLUTION PRESET & FFMPEG PROFILE MANAGEMENT                                            |
|  +-----------------------------------------------------------------------------------------------+ |
|  | Preset: 1080i50 PAL HD (Standard Indian Cable/DTH)  | 1920x1080 @ 25fps (50i TFF) [BUILT-IN]   | |
|  | Interlace: TFF | DAR: 16:9 | Color: BT.709         | Bitrate: 6500 kbps CBR                   | |
|  | -vf "yadif=0:-1:1,scale=1920:1080" -flags +ilme+ildct -top 1 -b:v 6500k       [Copy FFmpeg]   | |
|  +-----------------------------------------------------------------------------------------------+ |
|  | Preset: 576i50 PAL SD (Legacy Indian Cable 4:3)     | 720x576 @ 25fps (50i TFF)   [BUILT-IN]   | |
|  | Preset: 576i50 PAL SD Anamorphic (16:9 Cable)       | 720x576 @ 25fps (50i TFF)   [BUILT-IN]   | |
|  | Preset: 720p50 HD (Sports Cable)                    | 1280x720 @ 50fps            [BUILT-IN]   | |
|  | Preset: 1080p50 Full HD (IPTV / OTT)                | 1920x1080 @ 50fps           [BUILT-IN]   | |
|  | Preset: 4K UHD 2160p50 (UHD DTH Broadcast)          | 3840x2160 @ 50fps HEVC      [BUILT-IN]   | |
|  | [+ Add Custom Resolution Preset Modal]                                                          | |
|  +-----------------------------------------------------------------------------------------------+ |
|  SECTION: EDGE AGENT MANAGEMENT & CRYPTO PAIRING                                                   |
|  +-----------------------------------------------------------------------------------------------+ |
|  | Agent Hostname: delhi-dc1-primary   | IP: 100.64.1.15 (Tailscale) | Status: [🟢 ONLINE]       | |
|  | Active Channels: CH 01, CH 02       | CPU: 28% | GPU: 42%         | Paired: 2026-09-12        | |
|  | Token: agt_sec_e8b91a... [Rotate]   | Latency: 2.1 ms             | [Disconnect] [Re-pair]    | |
|  +-----------------------------------------------------------------------------------------------+ |
|  [+ Pair New Edge Agent Modal]                                                                     |
|  Agent IP / Host:  [100.64.2.88                   ]  gRPC Port: [9095]                             |
|  Pairing Token:    [agt_sec_8f43a9b2c011e749a1d2e8b409c2513f87a6b4c3d2e1f0a9b8c7d6e5f4a3b2c1     ] |
|  (Obtain this token from `docker logs <agent_container>`)            [Authenticate & Pair Agent]   |
+----------------------------------------------------------------------------------------------------+
|  SECTION: BOT & CHATOPS CONFIGURATION (MULTIPLE BOTS)                                              |
|  Bot Platform: [Telegram Messenger ▼]   Bot Name: [@MCRFlowPlayoutOpsBot]                          |
|  API Key:      [7123456789:AAHq0_k...xxxxxxxxxxxxxxxx]  [Test Bot Connection: 🟢 Connected]       |
|  Allowed Chat IDs Whitelist: [98471238, -100192847192 (MCR Admins Group)]                          |
|  Assigned Channels: [x] CH 01  [x] CH 02  [x] CH 03                                                |
|                                                                                                    |
|  NATURAL LANGUAGE SCHEDULING PREVIEW:                                                              |
|  Operator types in Telegram: "Schedule Avengers at 9 AM"                                          |
|  Bot responses: Fuzzy found "Avengers: Endgame (2019).mkv"                                         |
|  ⚠️ Conflict alert: "Morning News Live" is 08:30 - 09:30 IST                                       |
|  Interactive buttons: [Force Overwrite Slot] [Queue After Ends (9:30 AM)] [Replace Conflicting]    |
+----------------------------------------------------------------------------------------------------+
|  SECTION: INTERNATIONALIZATION (i18n) & REGIONAL                                                   |
|  UI Language: [English (Default) ▼]  Options: हिन्दी, தமிழ், తెలుగు, বাংলা, मराठी, ગુજરાતી, etc.    |
|  Timezone:    [Asia/Kolkata (IST - UTC+05:30) ▼]   Broadcast Standard: [PAL 50Hz (India/EU) ▼]     |
|  TMDb API Key:[tmdb_live_prod_991823...          ] Primary Poster Language: [en-IN / hi-IN ▼]      |
+----------------------------------------------------------------------------------------------------+
```
