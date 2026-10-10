import React, { useState, useEffect, useRef } from 'react';
import {
  Tv,
  Save,
  Plus,
  Trash2,
  Copy,
  AlertOctagon,
  Radio,
  Shield,
  Key,
  Sliders,
  Check,
  ArrowLeft,
  Search,
  X,
  Play,
  Square,
  Sparkles,
  Layout,
  Layers,
  Palette,
  Eye,
  AlertTriangle,
  ExternalLink,
  ChevronDown,
  ChevronUp,
  ChevronRight,
  RefreshCw,
  Activity,
  Terminal,
  FileText
} from 'lucide-react';
import { VideoPlayer } from './VideoPlayer';
import { api } from '../api';

export function ChannelScreen({
  channels = [],
  activeChannelId,
  onSwitchChannel,
  onSaveChannel,
  onCreateChannel,
  onDeleteChannel,
  resolutions = [],
  adTemplates = [],
  agents = [],
  onBackToDashboard,
  onNavigateToAdStudio,
  onShowToast,
  t
}) {
  // Navigation: "list" (Channels Directory) vs "detail" (Channel Master Control)
  const [viewMode, setViewMode] = useState(activeChannelId ? "detail" : "list");
  const [channelSearch, setChannelSearch] = useState("");

  const activeChannel = channels.find((c) => c.id === activeChannelId) || channels[0] || {};

  const [copiedHls, setCopiedHls] = useState(false);
  const [copiedEpg, setCopiedEpg] = useState(false);
  const [isSlateActive, setIsSlateActive] = useState(false);
  const [playoutStatus, setPlayoutStatus] = useState(null);
  const [isCallSignManual, setIsCallSignManual] = useState(false);
  const [isAdvancedOpen, setIsAdvancedOpen] = useState(false);

  // Playout & FFmpeg Diagnostics Modal State
  const [isPlayoutBusy, setIsPlayoutBusy] = useState(false);
  const [isLogModalOpen, setIsLogModalOpen] = useState(false);
  const [logChannelId, setLogChannelId] = useState("");
  const [ffmpegCommand, setFfmpegCommand] = useState("");
  const [ffmpegLogs, setFfmpegLogs] = useState("");
  const [isLogLoading, setIsLogLoading] = useState(false);
  const [copiedCmd, setCopiedCmd] = useState(false);
  const [autoRefreshLogs, setAutoRefreshLogs] = useState(true);

  // Live Browser Preview Modal State
  const [isPreviewModalOpen, setIsPreviewModalOpen] = useState(false);
  const [previewChannelId, setPreviewChannelId] = useState("");
  const [previewHlsUrl, setPreviewHlsUrl] = useState("");
  const [isPreviewLoading, setIsPreviewLoading] = useState(false);

  const toCallSignSlug = (name) => {
    if (!name) return "";
    return name
      .trim()
      .toUpperCase()
      .replace(/[^A-Z0-9]+/g, '-')
      .replace(/^-+|-+$/g, '')
      .slice(0, 16);
  };

  const [formData, setFormData] = useState({
    name: "",
    call_sign: "",
    lcn: 101,
    resolution_id: "res-in-1080i50",
    video_codec: "libx264",
    audio_codec: "aac",
    ad_template_id: "",
    hls_web_token: "",
    epg_web_token: "",
    udp_enabled: true,
    udp_url: "udp://239.255.10.1:5000?pkt_size=1316",
    rtmp_enabled: false,
    rtmp_url: "",
    rtmp_key: "",
    hls_enabled: true
  });

  // Sync formData when activeChannel changes
  useEffect(() => {
    if (activeChannel && activeChannel.id) {
      const udpDest = activeChannel.destinations?.find((d) => d.protocol === "UDP_MULTICAST" || d.protocol === "udp" || d.type === "udp");
      const rtmpDest = activeChannel.destinations?.find((d) => d.protocol === "RTMP" || d.protocol === "rtmp" || d.type === "rtmp");
      const hlsDest = activeChannel.destinations?.find((d) => d.protocol === "HLS" || d.protocol === "hls" || d.type === "hls");

      const rawCallSign = activeChannel.call_sign || "";
      setFormData({
        name: activeChannel.name || "",
        call_sign: rawCallSign || toCallSignSlug(activeChannel.name || ""),
        lcn: activeChannel.lcn || 101,
        resolution_id: activeChannel.resolution_id || "res-in-1080i50",
        video_codec: activeChannel.video_codec || "libx264",
        audio_codec: activeChannel.audio_codec || "aac",
        ad_template_id: activeChannel.ad_template_id || "",
        hls_web_token: activeChannel.hls_web_token || "",
        epg_web_token: activeChannel.epg_web_token || "",
        udp_enabled: udpDest ? udpDest.enabled : true,
        udp_url: udpDest?.endpoint_url || udpDest?.url || "udp://239.255.10.1:5000?pkt_size=1316",
        rtmp_enabled: rtmpDest ? rtmpDest.enabled : false,
        rtmp_url: rtmpDest?.endpoint_url || rtmpDest?.url || "",
        rtmp_key: rtmpDest?.stream_key || "",
        hls_enabled: hlsDest ? hlsDest.enabled : true
      });

      setIsCallSignManual(!!rawCallSign && rawCallSign !== toCallSignSlug(activeChannel.name || ""));

      // Refresh playout status
      api.getPlayoutStatus(activeChannel.id).then((st) => {
        if (st) setPlayoutStatus(st);
      }).catch(() => {});
    }
  }, [activeChannel]);

  // Periodic playout status polling
  useEffect(() => {
    if (!activeChannel?.id || viewMode !== "detail") return;
    const interval = setInterval(() => {
      api.getPlayoutStatus(activeChannel.id).then((st) => {
        if (st) setPlayoutStatus(st);
      }).catch(() => {});
    }, 4000);
    return () => clearInterval(interval);
  }, [activeChannel?.id, viewMode]);

  const handleInputChange = (field, value) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
  };

  const handleNameChange = (val) => {
    setFormData((prev) => {
      const prevAutoSlug = toCallSignSlug(prev.name);
      const shouldAutoSlug = !isCallSignManual || !prev.call_sign || prev.call_sign === prevAutoSlug;
      return {
        ...prev,
        name: val,
        call_sign: shouldAutoSlug ? toCallSignSlug(val) : prev.call_sign
      };
    });
  };

  const handleCallSignChange = (val) => {
    setIsCallSignManual(true);
    setFormData((prev) => ({ ...prev, call_sign: val }));
  };

  const handleRegenerateCallSign = () => {
    const slug = toCallSignSlug(formData.name) || "CHANNEL";
    setFormData((prev) => ({ ...prev, call_sign: slug }));
    setIsCallSignManual(false);
    onShowToast(`Auto-generated Call Sign: ${slug}`, "info");
  };

  // Helper to open a channel in detail mode
  const handleOpenChannelDetail = (chId) => {
    onSwitchChannel(chId);
    setViewMode("detail");
  };

  const handleCreateChannelAndOpen = () => {
    onCreateChannel();
    setViewMode("detail");
  };

  // Detect layout collisions on the assigned template
  const assignedTemplate = adTemplates.find((t) => t.id === formData.ad_template_id);
  const detectedCollisions = [];
  if (assignedTemplate && Array.isArray(assignedTemplate.overlay_elements)) {
    const activeElements = assignedTemplate.overlay_elements.filter((el) => el.is_active);
    for (let i = 0; i < activeElements.length; i++) {
      for (let j = i + 1; j < activeElements.length; j++) {
        const a = activeElements[i];
        const b = activeElements[j];
        const aW = a.width || 300;
        const aH = a.height || 60;
        const bW = b.width || 300;
        const bH = b.height || 60;
        const isColliding = !(a.x + aW <= b.x || b.x + bW <= a.x || a.y + aH <= b.y || b.y + bH <= a.y);
        if (isColliding) {
          detectedCollisions.push({ a, b });
        }
      }
    }
  }

  // Handle Save Channel
  const handleSave = () => {
    if (!formData.name.trim()) {
      onShowToast("Channel name is required", "error");
      return;
    }

    // Construct destinations: UDP, RTMP, HLS only (No SRT!)
    const destinations = [
      {
        type: "udp",
        protocol: "UDP_MULTICAST",
        enabled: formData.udp_enabled,
        url: formData.udp_url,
        endpoint_url: formData.udp_url
      },
      {
        type: "rtmp",
        protocol: "RTMP",
        enabled: formData.rtmp_enabled && !!formData.rtmp_url,
        url: formData.rtmp_url,
        endpoint_url: formData.rtmp_url,
        stream_key: formData.rtmp_key
      },
      {
        type: "hls",
        protocol: "HLS",
        enabled: formData.hls_enabled,
        url: `/hls/${activeChannel.id}/master.m3u8`,
        endpoint_url: `/hls/${activeChannel.id}/master.m3u8`
      }
    ];

    const updated = {
      ...activeChannel,
      name: formData.name.trim(),
      call_sign: formData.call_sign.trim().toUpperCase(),
      lcn: Number(formData.lcn) || 101,
      resolution_id: formData.resolution_id,
      video_codec: formData.video_codec,
      audio_codec: formData.audio_codec,
      ad_template_id: formData.ad_template_id,
      hls_web_token: formData.hls_web_token.trim(),
      epg_web_token: formData.epg_web_token.trim(),
      destinations
    };

    onSaveChannel(activeChannel.id, updated);
  };

  // Toggle Emergency Slate
  const handleToggleSlate = async () => {
    try {
      const nextState = !isSlateActive;
      await api.toggleChannelSlate(activeChannel.id, nextState);
      setIsSlateActive(nextState);
      onShowToast(nextState ? "Emergency slate ENGAGED on-air" : "Emergency slate RELEASED", nextState ? "error" : "info");
      const st = await api.getPlayoutStatus(activeChannel.id);
      if (st) setPlayoutStatus(st);
    } catch (e) {
      onShowToast("Failed to toggle emergency slate: " + e.message, "error");
    }
  };

  // Dynamic Host & Stream Resolution (Control Plane vs Edge Agent)
  const resolveAgentHost = (channel = activeChannel) => {
    if (channel?.primary_agent_id && Array.isArray(agents)) {
      const ag = agents.find((a) => a.id === channel.primary_agent_id);
      if (ag) {
        return {
          host: ag.ip_address || ag.hostname || window.location.hostname,
          port: ag.port || 3082,
          isEdge: true,
          agentName: ag.name || ag.id
        };
      }
    }
    return {
      host: window.location.hostname || "localhost",
      port: window.location.port || (window.location.protocol === "https:" ? "443" : "80"),
      isEdge: false,
      agentName: "Control Plane Host"
    };
  };

  const getResolvedHlsUrl = (channel = activeChannel, token = formData.hls_web_token) => {
    if (!channel?.id) return "";
    const { host, port } = resolveAgentHost(channel);
    const portPart = port && port !== "80" && port !== "443" ? `:${port}` : "";
    const proto = window.location.protocol || "http:";
    let url = `${proto}//${host}${portPart}/hls/${channel.id}/master.m3u8`;
    const tok = token || channel.hls_web_token;
    if (tok) {
      url += `?token=${encodeURIComponent(tok)}`;
    }
    return url;
  };

  const getResolvedRtmpUrl = (channel = activeChannel) => {
    if (!channel?.id) return "";
    const rtmpDest = channel.destinations?.find((d) => (d.protocol === "RTMP" || d.type === "rtmp") && d.enabled);
    if (rtmpDest?.url && !rtmpDest.url.startsWith("rtmp://localhost") && !rtmpDest.url.startsWith("rtmp://127.0.0.1")) {
      return rtmpDest.url;
    }
    const { host } = resolveAgentHost(channel);
    return `rtmp://${host}:1935/live/${channel.call_sign || channel.id}`;
  };

  const getResolvedEpgUrl = (channel = activeChannel, token = formData.epg_web_token) => {
    if (!channel?.id) return "";
    const { host, port } = resolveAgentHost(channel);
    const portPart = port && port !== "80" && port !== "443" ? `:${port}` : "";
    const proto = window.location.protocol || "http:";
    let url = `${proto}//${host}${portPart}/api/v1/epg/${channel.id}.xml`;
    const tok = token || channel.epg_web_token;
    if (tok) {
      url += `?token=${encodeURIComponent(tok)}`;
    }
    return url;
  };

  // Start / Stop Playout Handlers
  const handleStartPlayout = async (chId = activeChannel.id) => {
    if (!chId) return;
    setIsPlayoutBusy(true);
    try {
      await api.startPlayout(chId);
      onShowToast("Playout broadcast pipeline launched on-air!", "success");
      const st = await api.getPlayoutStatus(chId);
      if (st) setPlayoutStatus(st);
    } catch (err) {
      onShowToast("Failed to start playout: " + err.message, "error");
    } finally {
      setIsPlayoutBusy(false);
    }
  };

  const handleStopPlayout = async (chId = activeChannel.id) => {
    if (!chId) return;
    setIsPlayoutBusy(true);
    try {
      await api.stopPlayout(chId);
      onShowToast("Playout pipeline stopped", "info");
      const st = await api.getPlayoutStatus(chId);
      if (st) setPlayoutStatus(st);
    } catch (err) {
      onShowToast("Failed to stop playout: " + err.message, "error");
    } finally {
      setIsPlayoutBusy(false);
    }
  };

  // FFmpeg Diagnostics Log Loader
  const loadPlayoutLogs = async (chId = activeChannel.id) => {
    if (!chId) return;
    setIsLogLoading(true);
    try {
      const data = await api.getFFmpegLogs(chId, 300);
      setFfmpegCommand(data.command || "ffmpeg (idle / standby - not active)");
      setFfmpegLogs(data.logs || "No active log trace available for this channel.");
    } catch (err) {
      setFfmpegLogs("Error fetching logs: " + err.message);
    } finally {
      setIsLogLoading(false);
    }
  };

  const handleOpenLogModal = (chId = activeChannel.id) => {
    setLogChannelId(chId);
    setIsLogModalOpen(true);
    loadPlayoutLogs(chId);
  };

  // Periodic Auto-refresh for FFmpeg Logs Modal
  useEffect(() => {
    if (!isLogModalOpen || !autoRefreshLogs) return;
    const targetId = logChannelId || activeChannel.id;
    if (!targetId) return;
    const interval = setInterval(() => {
      loadPlayoutLogs(targetId);
    }, 3000);
    return () => clearInterval(interval);
  }, [isLogModalOpen, autoRefreshLogs, logChannelId, activeChannel?.id]);

  // Live Browser Preview Handling
  const handleOpenPreview = async (chId = activeChannel.id) => {
    if (!chId) return;
    setPreviewChannelId(chId);
    setIsPreviewLoading(true);
    setIsPreviewModalOpen(true);
    const targetCh = channels.find((c) => c.id === chId) || activeChannel;
    try {
      const res = await api.startPreview(chId);
      let streamUrl = res?.hls_url;
      if (!streamUrl || streamUrl.startsWith("/")) {
        streamUrl = getResolvedHlsUrl(targetCh);
      }
      setPreviewHlsUrl(streamUrl);
    } catch (err) {
      onShowToast("Preview stream connecting: " + err.message, "info");
      const streamUrl = getResolvedHlsUrl(targetCh);
      setPreviewHlsUrl(streamUrl);
    } finally {
      setIsPreviewLoading(false);
    }
  };

  const handleClosePreview = async () => {
    setIsPreviewModalOpen(false);
    const targetId = previewChannelId || activeChannel.id;
    setPreviewHlsUrl("");
    setPreviewChannelId("");
    try {
      // Auto-terminates temporary preview if HLS is not permanent destination
      if (targetId) await api.stopPreview(targetId);
    } catch (e) {}
  };

  const copyToClipboard = (text, type = "hls") => {
    navigator.clipboard.writeText(text);
    if (type === "hls") {
      setCopiedHls(true);
      setTimeout(() => setCopiedHls(false), 2000);
    } else if (type === "epg") {
      setCopiedEpg(true);
      setTimeout(() => setCopiedEpg(false), 2000);
    } else if (type === "cmd") {
      setCopiedCmd(true);
      setTimeout(() => setCopiedCmd(false), 2000);
    }
    onShowToast(`Copied ${type.toUpperCase()} endpoint to clipboard`, "info");
  };

  const fullHlsUrl = getResolvedHlsUrl(activeChannel, formData.hls_web_token);
  const fullEpgUrl = getResolvedEpgUrl(activeChannel, formData.epg_web_token);
  const fullRtmpUrl = getResolvedRtmpUrl(activeChannel);

  // Filter channels for List view
  const filteredChannels = channels.filter((ch) => {
    if (!channelSearch.trim()) return true;
    const q = channelSearch.toLowerCase();
    return (
      ch.name?.toLowerCase().includes(q) ||
      ch.call_sign?.toLowerCase().includes(q) ||
      String(ch.lcn).includes(q)
    );
  });

  // FFmpeg Playout Diagnostics Modal
  const renderFFmpegLogModal = () => {
    if (!isLogModalOpen) return null;
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/85 backdrop-blur-sm p-4 animate-in fade-in duration-200">
        <div className="bg-[#111622] border border-gray-800 rounded-xl max-w-5xl w-full max-h-[90vh] overflow-hidden shadow-2xl flex flex-col">
          {/* Modal Header */}
          <div className="flex items-center justify-between px-6 py-4 border-b border-gray-800 bg-[#0d121c]">
            <div className="flex items-center gap-3">
              <div className="p-2 bg-blue-950/60 border border-blue-800/50 rounded-lg text-blue-400">
                <Terminal className="w-5 h-5" />
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <h3 className="text-base font-bold text-white">
                    FFmpeg Playout Diagnostics & Log Trace
                  </h3>
                  <span className="px-2 py-0.5 rounded text-[10px] font-mono font-bold bg-blue-950 border border-blue-700 text-blue-300">
                    {logChannelId || activeChannel.id}
                  </span>
                </div>
                <p className="text-xs text-gray-400 mt-0.5">
                  Real-time broadcast process execution commands, parameters, filter graphs, and stderr telemetry
                </p>
              </div>
            </div>
            <button
              onClick={() => setIsLogModalOpen(false)}
              className="p-1.5 text-gray-400 hover:text-white rounded-lg transition-colors"
            >
              <X className="w-5 h-5" />
            </button>
          </div>

          {/* Modal Body */}
          <div className="p-6 space-y-5 overflow-y-auto max-h-[calc(90vh-140px)] bg-[#0B0F17]">
            {/* Section 1: Active Executed FFmpeg Command */}
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-xs font-bold text-gray-300 uppercase tracking-wider flex items-center gap-1.5">
                  <FileText className="w-3.5 h-3.5 text-blue-400" />
                  Triggered Command Line
                </span>
                <button
                  type="button"
                  onClick={() => copyToClipboard(ffmpegCommand, "cmd")}
                  className="px-2.5 py-1 bg-gray-800 hover:bg-gray-700 text-gray-200 rounded text-xs font-medium flex items-center gap-1 transition-colors"
                >
                  {copiedCmd ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                  {copiedCmd ? "Copied Command!" : "Copy Command"}
                </button>
              </div>
              <div className="bg-[#070A0F] border border-gray-800 rounded-lg p-3 text-xs font-mono text-blue-300 break-all select-all max-h-36 overflow-y-auto leading-relaxed">
                {ffmpegCommand || "ffmpeg (idle / standby - not active)"}
              </div>
            </div>

            {/* Section 2: Real-time Process Log Trace */}
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <span className="text-xs font-bold text-gray-300 uppercase tracking-wider flex items-center gap-1.5">
                    <Terminal className="w-3.5 h-3.5 text-emerald-400" />
                    Process stdout / stderr Trace
                  </span>
                  <span className="text-[11px] text-gray-500 font-mono">
                    (Target Channel: {logChannelId || activeChannel.id})
                  </span>
                </div>

                <div className="flex items-center gap-3">
                  <label className="flex items-center gap-1.5 text-xs text-gray-300 cursor-pointer select-none">
                    <input
                      type="checkbox"
                      checked={autoRefreshLogs}
                      onChange={(e) => setAutoRefreshLogs(e.target.checked)}
                      className="rounded accent-blue-600"
                    />
                    <span>Auto-refresh (3s)</span>
                  </label>

                  <button
                    type="button"
                    onClick={() => loadPlayoutLogs(logChannelId || activeChannel.id)}
                    disabled={isLogLoading}
                    className="px-2.5 py-1 bg-blue-600 hover:bg-blue-500 text-white rounded text-xs font-medium flex items-center gap-1 transition-colors disabled:opacity-50"
                  >
                    <RefreshCw className={`w-3.5 h-3.5 ${isLogLoading ? "animate-spin" : ""}`} />
                    Refresh
                  </button>
                </div>
              </div>

              <div className="bg-[#05080E] border border-gray-800 rounded-lg p-4 font-mono text-xs text-gray-300 h-80 overflow-y-auto whitespace-pre-wrap leading-relaxed shadow-inner">
                {isLogLoading && !ffmpegLogs ? (
                  <div className="h-full flex items-center justify-center text-gray-500">
                    <RefreshCw className="w-5 h-5 animate-spin mr-2 text-blue-400" />
                    Loading process log trace...
                  </div>
                ) : ffmpegLogs ? (
                  ffmpegLogs
                ) : (
                  <div className="text-gray-500 italic">No output logs recorded yet for this channel session.</div>
                )}
              </div>
            </div>
          </div>

          {/* Modal Footer */}
          <div className="px-6 py-3 border-t border-gray-800 bg-[#0d121c] flex items-center justify-between">
            <span className="text-xs text-gray-500">
              Playout Status: <strong className="text-white">{playoutStatus?.state || "ON-AIR"}</strong>
            </span>
            <button
              onClick={() => setIsLogModalOpen(false)}
              className="px-4 py-1.5 bg-gray-800 hover:bg-gray-700 text-white rounded-lg text-xs font-semibold"
            >
              Close Diagnostics
            </button>
          </div>
        </div>
      </div>
    );
  };

  // ==========================================
  // VIEW MODE 1: CHANNELS DIRECTORY (LIST FIRST)
  // ==========================================
  if (viewMode === "list") {
    return (
      <div className="space-y-6 pb-12 animate-in fade-in duration-300">
        {/* Top Header & Search Bar */}
        <div className="bg-[#111622] border border-gray-800 rounded-xl p-5 shadow-lg flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-2">
              <Tv className="w-5 h-5 text-blue-400" />
              <h1 className="text-xl font-bold text-white tracking-wide">Broadcast Channels</h1>
              <span className="px-2 py-0.5 rounded text-[11px] font-mono bg-blue-950/70 border border-blue-700/50 text-blue-300">
                Directory
              </span>
            </div>
            <p className="text-xs text-gray-400 mt-1">
              Linear television stations, raster profiles, assigned Ad & Layout templates, and playout statuses
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            <div className="relative min-w-[220px]">
              <Search className="w-4 h-4 text-gray-400 absolute left-3 top-2.5" />
              <input
                type="text"
                value={channelSearch}
                onChange={(e) => setChannelSearch(e.target.value)}
                placeholder="Search channels or LCN..."
                className="w-full bg-[#182030] border border-gray-700 rounded-lg pl-9 pr-3 py-2 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-blue-500"
              />
            </div>

            <button
              onClick={handleCreateChannelAndOpen}
              className="px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white rounded-lg text-xs font-semibold flex items-center gap-1.5 shadow-md shadow-blue-900/30 transition-all shrink-0"
            >
              <Plus className="w-4 h-4" />
              Create Channel
            </button>
          </div>
        </div>

        {/* Metrics Overview Row */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
          <div className="bg-[#111622] border border-gray-800 rounded-xl p-4 shadow-sm flex items-center gap-3">
            <div className="p-2.5 bg-blue-950/60 border border-blue-800/40 rounded-lg text-blue-400">
              <Tv className="w-5 h-5" />
            </div>
            <div>
              <div className="text-lg font-bold text-white font-mono">{channels.length}</div>
              <div className="text-[11px] text-gray-400">Total Channels</div>
            </div>
          </div>

          <div className="bg-[#111622] border border-gray-800 rounded-xl p-4 shadow-sm flex items-center gap-3">
            <div className="p-2.5 bg-red-950/60 border border-red-800/40 rounded-lg text-red-400">
              <Activity className="w-5 h-5" />
            </div>
            <div>
              <div className="text-lg font-bold text-red-400 font-mono">
                {channels.filter((c) => c.status === "ON-AIR" || c.status === "ACTIVE").length || 1}
              </div>
              <div className="text-[11px] text-gray-400">On-Air Stations</div>
            </div>
          </div>

          <div className="bg-[#111622] border border-gray-800 rounded-xl p-4 shadow-sm flex items-center gap-3">
            <div className="p-2.5 bg-purple-950/60 border border-purple-800/40 rounded-lg text-purple-400">
              <Layout className="w-5 h-5" />
            </div>
            <div>
              <div className="text-lg font-bold text-white font-mono">{adTemplates.length}</div>
              <div className="text-[11px] text-gray-400">Layout Templates</div>
            </div>
          </div>

          <div className="bg-[#111622] border border-gray-800 rounded-xl p-4 shadow-sm flex items-center gap-3">
            <div className="p-2.5 bg-emerald-950/60 border border-emerald-800/40 rounded-lg text-emerald-400">
              <Radio className="w-5 h-5" />
            </div>
            <div>
              <div className="text-lg font-bold text-emerald-400 font-mono">UDP • RTMP • HLS</div>
              <div className="text-[11px] text-gray-400">Active Destinations</div>
            </div>
          </div>
        </div>

        {/* Channels Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
          {filteredChannels.map((ch) => {
            const resObj = resolutions.find((r) => r.id === ch.resolution_id);
            const tmplObj = adTemplates.find((t) => t.id === ch.ad_template_id);
            const hasUdp = ch.destinations?.some((d) => (d.protocol === "UDP_MULTICAST" || d.type === "udp") && d.enabled);
            const hasRtmp = ch.destinations?.some((d) => (d.protocol === "RTMP" || d.type === "rtmp") && d.enabled);
            const hasHls = ch.destinations?.some((d) => (d.protocol === "HLS" || d.type === "hls") && d.enabled);

            return (
              <div
                key={ch.id}
                className="bg-[#111622] border border-gray-800 hover:border-blue-600/70 rounded-xl p-5 shadow-lg flex flex-col justify-between transition-all group"
              >
                <div>
                  {/* Channel Header & LCN */}
                  <div className="flex items-start justify-between gap-3 mb-3">
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="px-2 py-0.5 rounded text-[10px] font-mono font-bold bg-blue-950 border border-blue-800 text-blue-300">
                          LCN {ch.lcn}
                        </span>
                        <span className="text-xs font-mono text-gray-400 uppercase tracking-wider font-semibold">
                          {ch.call_sign || `CH-${ch.lcn}`}
                        </span>
                      </div>
                      <h3 className="text-base font-bold text-white group-hover:text-blue-300 transition-colors mt-1">
                        {ch.name}
                      </h3>
                    </div>

                    {/* Status Pill */}
                    <span className={`px-2.5 py-1 rounded-full text-[10px] font-bold uppercase tracking-wider flex items-center gap-1.5 shrink-0 ${
                      ch.status === "STOPPED"
                        ? "bg-gray-800 text-gray-400 border border-gray-700"
                        : "bg-red-950/80 border border-red-700/80 text-red-300"
                    }`}>
                      <span className={`w-1.5 h-1.5 rounded-full ${ch.status === "STOPPED" ? "bg-gray-500" : "bg-red-500 animate-pulse"}`} />
                      {ch.status || "ON-AIR"}
                    </span>
                  </div>

                  {/* Metadata Specs */}
                  <div className="space-y-2 mb-4 bg-[#141b2b] p-3 rounded-lg border border-gray-800/80 text-xs">
                    <div className="flex items-center justify-between">
                      <span className="text-gray-400">Resolution:</span>
                      <span className="text-white font-mono font-medium">
                        {resObj ? `${resObj.name}` : ch.resolution_id || '1080i50 HD'}
                      </span>
                    </div>

                    <div className="flex items-center justify-between">
                      <span className="text-gray-400">Assigned Layout:</span>
                      <span className="text-purple-300 font-medium truncate max-w-[170px]" title={tmplObj?.name}>
                        {tmplObj?.name || 'Default News Template'}
                      </span>
                    </div>

                    <div className="flex items-center justify-between pt-1 border-t border-gray-800/60">
                      <span className="text-gray-400">Egress Targets:</span>
                      <div className="flex items-center gap-1">
                        <span className={`px-1.5 py-0.5 rounded text-[9px] font-mono font-bold ${
                          hasUdp ? "bg-emerald-950 text-emerald-300 border border-emerald-800" : "bg-gray-800 text-gray-500"
                        }`}>
                          UDP
                        </span>
                        <span className={`px-1.5 py-0.5 rounded text-[9px] font-mono font-bold ${
                          hasRtmp ? "bg-emerald-950 text-emerald-300 border border-emerald-800" : "bg-gray-800 text-gray-500"
                        }`}>
                          RTMP
                        </span>
                        <span className={`px-1.5 py-0.5 rounded text-[9px] font-mono font-bold ${
                          hasHls ? "bg-emerald-950 text-emerald-300 border border-emerald-800" : "bg-gray-800 text-gray-500"
                        }`}>
                          HLS
                        </span>
                      </div>
                    </div>
                  </div>
                </div>

                {/* Card Actions */}
                <div className="pt-3 border-t border-gray-800 flex items-center justify-between gap-2">
                  <div className="flex items-center gap-1.5">
                    {/* Playout Start / Stop Quick Toggle */}
                    <button
                      onClick={() => (ch.status === "STOPPED" ? handleStartPlayout(ch.id) : handleStopPlayout(ch.id))}
                      disabled={isPlayoutBusy}
                      className={`p-1.5 rounded-lg border transition-colors ${
                        ch.status === "STOPPED"
                          ? "bg-emerald-950/60 hover:bg-emerald-900 border-emerald-800/60 text-emerald-400"
                          : "bg-red-950/50 hover:bg-red-900 border-red-800/60 text-red-400"
                      }`}
                      title={ch.status === "STOPPED" ? "Start Playout Pipeline" : "Stop Playout Pipeline"}
                    >
                      {ch.status === "STOPPED" ? (
                        <Play className="w-3.5 h-3.5 fill-current" />
                      ) : (
                        <Square className="w-3.5 h-3.5 fill-current" />
                      )}
                    </button>

                    {/* FFmpeg Process Logs */}
                    <button
                      onClick={() => handleOpenLogModal(ch.id)}
                      className="p-1.5 bg-gray-800 hover:bg-gray-700 text-blue-400 hover:text-blue-300 rounded-lg transition-colors border border-gray-700"
                      title="Inspect FFmpeg Command & Process Logs"
                    >
                      <Terminal className="w-3.5 h-3.5" />
                    </button>

                    {/* Live Preview */}
                    <button
                      onClick={() => handleOpenPreview(ch.id)}
                      className="p-1.5 bg-gray-800 hover:bg-gray-700 text-indigo-400 hover:text-indigo-300 rounded-lg transition-colors border border-gray-700"
                      title="Live Preview"
                    >
                      <Eye className="w-3.5 h-3.5" />
                    </button>

                    {channels.length > 1 && (
                      <button
                        onClick={() => onDeleteChannel(ch.id)}
                        className="p-1.5 bg-red-950/40 hover:bg-red-900/60 text-red-400 rounded-lg transition-colors border border-red-800/40"
                        title="Delete Channel"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    )}
                  </div>

                  <button
                    onClick={() => handleOpenChannelDetail(ch.id)}
                    className="px-3 py-1.5 bg-blue-600 hover:bg-blue-500 text-white rounded-lg text-xs font-semibold flex items-center gap-1 transition-all shadow-sm"
                  >
                    Master Control
                    <ChevronRight className="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>
            );
          })}
        </div>

        {/* Live Browser Preview Modal */}
        {isPreviewModalOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm p-4 animate-in fade-in duration-200">
            <div className="bg-[#111622] border border-gray-800 rounded-xl max-w-4xl w-full overflow-hidden shadow-2xl flex flex-col">
              <div className="flex items-center justify-between px-5 py-3 border-b border-gray-800 bg-[#0d121c]">
                <div className="flex items-center gap-2">
                  <Eye className="w-4 h-4 text-indigo-400" />
                  <h3 className="text-sm font-bold text-white">
                    Live Channel Playout Preview
                  </h3>
                  <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-indigo-950 border border-indigo-700 text-indigo-300">
                    Auto-Terminating Session
                  </span>
                </div>
                <button
                  onClick={handleClosePreview}
                  className="p-1 text-gray-400 hover:text-white rounded-lg transition-colors"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              <div className="p-4 bg-black">
                {isPreviewLoading ? (
                  <div className="w-full aspect-video flex flex-col items-center justify-center text-gray-400 space-y-2">
                    <RefreshCw className="w-6 h-6 animate-spin text-indigo-400" />
                    <span className="text-xs font-mono">Initializing HLS Playout Stream...</span>
                  </div>
                ) : previewHlsUrl ? (
                  <VideoPlayer streamUrl={previewHlsUrl} src={previewHlsUrl} autoPlay={true} />
                ) : (
                  <div className="w-full aspect-video flex items-center justify-center text-gray-500 text-xs">
                    Stream unavailable
                  </div>
                )}
              </div>

              <div className="px-5 py-3 border-t border-gray-800 flex items-center justify-between bg-[#0d121c] text-xs text-gray-400">
                <span className="font-mono truncate max-w-md">Source: {previewHlsUrl}</span>
                <button
                  onClick={handleClosePreview}
                  className="px-4 py-1.5 bg-gray-800 hover:bg-gray-700 text-white rounded-lg text-xs font-semibold"
                >
                  Close Preview
                </button>
              </div>
            </div>
          </div>
        )}

        {/* FFmpeg Process Diagnostics Modal */}
        {renderFFmpegLogModal()}
      </div>
    );
  }

  // ==========================================
  // VIEW MODE 2: CHANNEL MASTER CONTROL (DETAIL)
  // ==========================================
  return (
    <div className="space-y-6 pb-12 animate-in fade-in duration-300">
      {/* Top Header & Navigation Bar */}
      <div className="bg-[#111622] border border-gray-800 rounded-xl p-4 shadow-lg flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <button
            onClick={() => setViewMode("list")}
            className="p-2 bg-gray-800 hover:bg-gray-700 text-gray-200 border border-gray-700 rounded-lg transition-colors flex items-center gap-1.5 text-xs font-semibold"
          >
            <ArrowLeft className="w-4 h-4" />
            Channels Directory
          </button>

          <div className="h-6 w-px bg-gray-700" />

          <div>
            <div className="flex items-center gap-2">
              <Tv className="w-4 h-4 text-blue-400" />
              <h1 className="text-base font-bold text-white tracking-wide">Channel Master Control</h1>
              <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-blue-950/70 border border-blue-700/50 text-blue-300">
                LCN {formData.lcn}
              </span>
            </div>
            <p className="text-[11px] text-gray-400">
              Managing: <span className="text-blue-300 font-semibold">{formData.name}</span> ({formData.call_sign || `CH-${formData.lcn}`})
            </p>
          </div>
        </div>

        {/* Channel Selector & Playout Actions */}
        <div className="flex flex-wrap items-center gap-2">
          <select
            value={activeChannel.id || ""}
            onChange={(e) => onSwitchChannel(e.target.value)}
            className="bg-[#182030] border border-gray-700 text-white rounded-lg px-3 py-1.5 text-xs font-medium focus:ring-2 focus:ring-blue-500 focus:outline-none"
          >
            {channels.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name} ({c.call_sign || `CH-${c.lcn}`})
              </option>
            ))}
          </select>

          {/* Playout Status Badge */}
          <div className={`px-2.5 py-1.5 rounded-lg border text-xs font-semibold flex items-center gap-1.5 ${
            playoutStatus?.state === "ON-AIR"
              ? "bg-red-950/70 border-red-700 text-red-300 animate-pulse"
              : playoutStatus?.state === "EMERGENCY_SLATE"
              ? "bg-amber-950/70 border-amber-700 text-amber-300"
              : "bg-gray-800/80 border-gray-700 text-gray-400"
          }`}>
            <span className={`w-2 h-2 rounded-full ${
              playoutStatus?.state === "ON-AIR" ? "bg-red-500" : playoutStatus?.state === "EMERGENCY_SLATE" ? "bg-amber-500" : "bg-gray-500"
            }`} />
            {playoutStatus?.state || "ON-AIR"}
            {playoutStatus?.smpt_timecode && (
              <span className="font-mono text-gray-300 font-normal ml-1 text-[11px]">
                {playoutStatus.smpt_timecode}
              </span>
            )}
          </div>

          {/* Playout Start / Stop Control */}
          {playoutStatus?.state === "ON-AIR" || playoutStatus?.state === "EMERGENCY_SLATE" ? (
            <button
              onClick={() => handleStopPlayout(activeChannel.id)}
              disabled={isPlayoutBusy}
              className="px-3 py-1.5 bg-red-950/80 hover:bg-red-900 border border-red-700/80 text-red-200 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-all shadow-sm disabled:opacity-50"
              title="Stop master playout pipeline"
            >
              <Square className="w-3.5 h-3.5 fill-current" />
              Stop Playout
            </button>
          ) : (
            <button
              onClick={() => handleStartPlayout(activeChannel.id)}
              disabled={isPlayoutBusy}
              className="px-3 py-1.5 bg-emerald-700 hover:bg-emerald-600 border border-emerald-600 text-white rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-all shadow-sm shadow-emerald-900/40 disabled:opacity-50"
              title="Start master broadcast playout pipeline"
            >
              <Play className="w-3.5 h-3.5 fill-current" />
              Start Playout
            </button>
          )}

          {/* Inspect FFmpeg Command & Process Logs */}
          <button
            onClick={() => handleOpenLogModal(activeChannel.id)}
            className="px-3 py-1.5 bg-gray-800 hover:bg-gray-700 text-gray-200 border border-gray-700 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-colors"
            title="Inspect active FFmpeg command and live process trace logs"
          >
            <Terminal className="w-3.5 h-3.5 text-blue-400" />
            FFmpeg Logs
          </button>

          {/* Emergency Slate Toggle */}
          <button
            onClick={handleToggleSlate}
            className={`px-3 py-1.5 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-colors border ${
              isSlateActive
                ? "bg-amber-600 hover:bg-amber-500 text-white border-amber-500"
                : "bg-gray-800 hover:bg-gray-700 text-amber-400 border-gray-700"
            }`}
            title="Toggle Technical Difficulties Slate"
          >
            <AlertOctagon className="w-3.5 h-3.5" />
            {isSlateActive ? "Release Slate" : "Emergency Slate"}
          </button>

          {/* Live Browser Preview Button */}
          <button
            onClick={() => handleOpenPreview(activeChannel.id)}
            className="px-3 py-1.5 bg-indigo-900/40 hover:bg-indigo-900/70 text-indigo-300 border border-indigo-700/60 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-colors"
          >
            <Eye className="w-3.5 h-3.5 text-indigo-400" />
            Live Preview
          </button>

          {/* Save Channel Button */}
          <button
            onClick={handleSave}
            className="px-3.5 py-1.5 bg-blue-600 hover:bg-blue-500 text-white rounded-lg text-xs font-semibold flex items-center gap-1.5 shadow-md shadow-blue-900/30 transition-all"
          >
            <Save className="w-3.5 h-3.5" />
            Save Changes
          </button>

          {/* Delete Channel Button */}
          {channels.length > 1 && (
            <button
              onClick={() => onDeleteChannel(activeChannel.id)}
              className="p-1.5 bg-red-950/40 hover:bg-red-900/60 text-red-400 border border-red-800/40 rounded-lg transition-colors"
              title="Delete Channel"
            >
              <Trash2 className="w-3.5 h-3.5" />
            </button>
          )}
        </div>
      </div>

      {/* Main Form Cards Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Card 1: Channel Identity & Broadcast Standards */}
        <div className="bg-[#111622] border border-gray-800 rounded-xl p-5 shadow-lg space-y-4">
          <div className="flex items-center gap-2 pb-3 border-b border-gray-800">
            <Radio className="w-4 h-4 text-blue-400" />
            <h2 className="text-sm font-bold text-white uppercase tracking-wider">Channel Identification & Raster</h2>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-medium text-gray-400 mb-1">Channel Name</label>
              <input
                type="text"
                value={formData.name}
                onChange={(e) => handleNameChange(e.target.value)}
                placeholder="e.g. DD National HD"
                className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 focus:ring-2 focus:ring-blue-500 focus:outline-none"
              />
            </div>

            <div>
              <div className="flex items-center justify-between mb-1">
                <label className="block text-xs font-medium text-gray-400">DVB Call Sign</label>
                <button
                  type="button"
                  onClick={handleRegenerateCallSign}
                  className="text-[11px] text-blue-400 hover:text-blue-300 underline"
                >
                  Auto Slug
                </button>
              </div>
              <input
                type="text"
                value={formData.call_sign}
                onChange={(e) => handleCallSignChange(e.target.value)}
                placeholder="e.g. DD-NATIONAL"
                className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-sm text-white placeholder-gray-500 font-mono uppercase focus:ring-2 focus:ring-blue-500 focus:outline-none"
              />
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-medium text-gray-400 mb-1">Logical Channel Number (LCN)</label>
              <input
                type="number"
                value={formData.lcn}
                onChange={(e) => handleInputChange("lcn", parseInt(e.target.value) || 100)}
                className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-sm text-white font-mono focus:ring-2 focus:ring-blue-500 focus:outline-none"
              />
            </div>

            <div>
              <label className="block text-xs font-medium text-gray-400 mb-1">Broadcast Resolution Standard</label>
              <select
                value={formData.resolution_id}
                onChange={(e) => handleInputChange("resolution_id", e.target.value)}
                className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-sm text-white focus:ring-2 focus:ring-blue-500 focus:outline-none"
              >
                {resolutions.map((res) => (
                  <option key={res.id} value={res.id}>
                    {res.name} ({res.width}x{res.height}{res.interlaced ? 'i' : 'p'}@{res.frame_rate}fps)
                  </option>
                ))}
              </select>
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-medium text-gray-400 mb-1">Video Codec</label>
              <select
                value={formData.video_codec}
                onChange={(e) => handleInputChange("video_codec", e.target.value)}
                className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-sm text-white focus:ring-2 focus:ring-blue-500 focus:outline-none"
              >
                <option value="libx264">H.264 / AVC (libx264 broadcast profile)</option>
                <option value="libx265">H.265 / HEVC (libx265 UHD)</option>
              </select>
            </div>

            <div>
              <label className="block text-xs font-medium text-gray-400 mb-1">Audio Codec</label>
              <select
                value={formData.audio_codec}
                onChange={(e) => handleInputChange("audio_codec", e.target.value)}
                className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-sm text-white focus:ring-2 focus:ring-blue-500 focus:outline-none"
              >
                <option value="aac">AAC-LC (192kbps stereo / 5.1)</option>
                <option value="ac3">Dolby Digital AC-3 (384kbps)</option>
                <option value="mp2">MPEG-1 Audio Layer II (MP2 256k)</option>
              </select>
            </div>
          </div>
        </div>

        {/* Card 2: Ad & Layout Management Assignment & Collision Detection */}
        <div className="bg-[#111622] border border-gray-800 rounded-xl p-5 shadow-lg space-y-4 flex flex-col justify-between">
          <div className="space-y-4">
            <div className="flex items-center justify-between pb-3 border-b border-gray-800">
              <div className="flex items-center gap-2">
                <Layout className="w-4 h-4 text-purple-400" />
                <h2 className="text-sm font-bold text-white uppercase tracking-wider">Ad & Layout Template Assignment</h2>
              </div>
              {onNavigateToAdStudio && (
                <button
                  type="button"
                  onClick={onNavigateToAdStudio}
                  className="text-xs text-purple-400 hover:text-purple-300 flex items-center gap-1 font-medium"
                >
                  Manage Templates
                  <ExternalLink className="w-3.5 h-3.5" />
                </button>
              )}
            </div>

            <div>
              <label className="block text-xs font-medium text-gray-400 mb-1">Assigned Layout Template</label>
              <select
                value={formData.ad_template_id}
                onChange={(e) => handleInputChange("ad_template_id", e.target.value)}
                className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-sm text-white font-semibold focus:ring-2 focus:ring-purple-500 focus:outline-none"
              >
                <option value="">None (Standard Clean Bug Playout)</option>
                {adTemplates.map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.name}
                  </option>
                ))}
              </select>
              <p className="text-[11px] text-gray-400 mt-1">
                Branding, station watermark logo, dynamic Now Playing, and ad banners are centralized in the template.
              </p>
            </div>

            {/* Template Conflict Warning Banner */}
            {assignedTemplate && detectedCollisions.length > 0 && (
              <div className="p-3 bg-amber-950/60 border border-amber-700/80 rounded-xl text-xs text-amber-200 flex items-start gap-2.5 animate-in fade-in">
                <AlertTriangle className="w-4 h-4 text-amber-400 shrink-0 mt-0.5" />
                <div>
                  <span className="font-bold text-amber-300 block">
                    Layout Overlap Warning: {detectedCollisions.length} Collisions Detected
                  </span>
                  <p className="text-[11px] text-amber-200/80 mt-0.5">
                    Assigned template has overlapping graphics layers. Playout engine will resolve this using:
                    <strong className="text-white ml-1 uppercase font-mono">
                      {assignedTemplate.collision_behavior === "alternate"
                        ? `Alternate (${assignedTemplate.alternate_duration_seconds || 15}s)`
                        : "Priority Order"}
                    </strong>
                  </p>
                </div>
              </div>
            )}

            {assignedTemplate && detectedCollisions.length === 0 && (
              <div className="p-3 bg-emerald-950/40 border border-emerald-800/80 rounded-xl text-xs text-emerald-300 flex items-center gap-2">
                <Check className="w-4 h-4 text-emerald-400 shrink-0" />
                <span>Template has clean raster geometry with 0 collisions.</span>
              </div>
            )}
          </div>

          {assignedTemplate && (
            <div className="p-3 bg-[#182030] border border-gray-800 rounded-lg text-xs space-y-1 text-gray-300 font-mono">
              <div className="font-bold text-purple-300">Active Template Summary</div>
              <div>Watermark: {assignedTemplate.logo_path ? assignedTemplate.logo_position || 'top-right' : 'Default Bug'}</div>
              <div>Active Overlays: {assignedTemplate.overlay_elements?.filter(e => e.is_active).length || 0}</div>
              <div>Commercial Breaks: {assignedTemplate.commercial_breaks?.length || 0} slots</div>
            </div>
          )}
        </div>

        {/* Card 3: Streaming Output Destinations (HLS, UDP, RTMP - No SRT) */}
        <div className="bg-[#111622] border border-gray-800 rounded-xl p-5 shadow-lg space-y-4 lg:col-span-2">
          <div className="flex items-center justify-between pb-3 border-b border-gray-800">
            <div className="flex items-center gap-2">
              <Radio className="w-4 h-4 text-emerald-400" />
              <h2 className="text-sm font-bold text-white uppercase tracking-wider">
                Streaming Output Destinations (HLS, UDP, RTMP)
              </h2>
            </div>
            <span className="text-xs text-gray-400 font-mono">Selectively enable output pipelines</span>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
            {/* Destination 1: UDP Multicast / Unicast */}
            <div className={`p-4 rounded-xl border transition-all ${
              formData.udp_enabled ? "bg-[#141b2b] border-blue-600/70 shadow-md" : "bg-[#121622] border-gray-800 opacity-70"
            }`}>
              <div className="flex items-center justify-between mb-3">
                <span className="text-xs font-bold text-white uppercase tracking-wider flex items-center gap-1.5">
                  <Radio className="w-3.5 h-3.5 text-blue-400" />
                  UDP MPEG-TS
                </span>
                <input
                  type="checkbox"
                  checked={formData.udp_enabled}
                  onChange={(e) => handleInputChange("udp_enabled", e.target.checked)}
                  className="rounded accent-blue-600 w-4 h-4 cursor-pointer"
                />
              </div>
              <div className="space-y-2">
                <label className="block text-[11px] text-gray-400">Multicast / Unicast URL</label>
                <input
                  type="text"
                  disabled={!formData.udp_enabled}
                  value={formData.udp_url}
                  onChange={(e) => handleInputChange("udp_url", e.target.value)}
                  placeholder="udp://239.255.10.1:5000?pkt_size=1316"
                  className="w-full bg-[#182030] border border-gray-700 rounded-lg px-2.5 py-1.5 text-xs text-white font-mono placeholder-gray-500 focus:outline-none focus:border-blue-500 disabled:opacity-50"
                />
                <span className="text-[10px] text-gray-500 block">DVB-ASI / Edge Transcoder Delivery</span>
              </div>
            </div>

            {/* Destination 2: RTMP Streaming Egress */}
            <div className={`p-4 rounded-xl border transition-all ${
              formData.rtmp_enabled ? "bg-[#141b2b] border-red-600/70 shadow-md" : "bg-[#121622] border-gray-800 opacity-70"
            }`}>
              <div className="flex items-center justify-between mb-3">
                <span className="text-xs font-bold text-white uppercase tracking-wider flex items-center gap-1.5">
                  <Play className="w-3.5 h-3.5 text-red-400" />
                  RTMP Egress
                </span>
                <input
                  type="checkbox"
                  checked={formData.rtmp_enabled}
                  onChange={(e) => handleInputChange("rtmp_enabled", e.target.checked)}
                  className="rounded accent-red-600 w-4 h-4 cursor-pointer"
                />
              </div>
              <div className="space-y-2">
                <div>
                  <label className="block text-[11px] text-gray-400">Server Endpoint URL</label>
                  <input
                    type="text"
                    disabled={!formData.rtmp_enabled}
                    value={formData.rtmp_url}
                    onChange={(e) => handleInputChange("rtmp_url", e.target.value)}
                    placeholder="rtmp://live.youtube.com/live2"
                    className="w-full bg-[#182030] border border-gray-700 rounded-lg px-2.5 py-1.5 text-xs text-white font-mono placeholder-gray-500 focus:outline-none focus:border-red-500 disabled:opacity-50"
                  />
                </div>
                <div>
                  <label className="block text-[11px] text-gray-400">Stream Key</label>
                  <input
                    type="password"
                    disabled={!formData.rtmp_enabled}
                    value={formData.rtmp_key}
                    onChange={(e) => handleInputChange("rtmp_key", e.target.value)}
                    placeholder="••••••••••••"
                    className="w-full bg-[#182030] border border-gray-700 rounded-lg px-2.5 py-1.5 text-xs text-white font-mono placeholder-gray-500 focus:outline-none focus:border-red-500 disabled:opacity-50"
                  />
                </div>
                <div className="pt-1 text-[10px] text-gray-400">
                  Host Egress Target: <span className="text-emerald-300 font-mono select-all">{fullRtmpUrl}</span>
                </div>
              </div>
            </div>

            {/* Destination 3: HLS Live Packaging */}
            <div className={`p-4 rounded-xl border transition-all ${
              formData.hls_enabled ? "bg-[#141b2b] border-emerald-600/70 shadow-md" : "bg-[#121622] border-gray-800 opacity-70"
            }`}>
              <div className="flex items-center justify-between mb-3">
                <span className="text-xs font-bold text-white uppercase tracking-wider flex items-center gap-1.5">
                  <Sparkles className="w-3.5 h-3.5 text-emerald-400" />
                  HLS Streaming
                </span>
                <input
                  type="checkbox"
                  checked={formData.hls_enabled}
                  onChange={(e) => handleInputChange("hls_enabled", e.target.checked)}
                  className="rounded accent-emerald-600 w-4 h-4 cursor-pointer"
                />
              </div>
              <div className="space-y-2">
                <label className="block text-[11px] text-gray-400">Master HLS Endpoint</label>
                <div className="flex items-center gap-1">
                  <input
                    type="text"
                    readOnly
                    value={fullHlsUrl}
                    className="w-full bg-[#182030] border border-gray-700 rounded-lg px-2.5 py-1.5 text-xs text-emerald-300 font-mono truncate select-all"
                  />
                  <button
                    type="button"
                    onClick={() => copyToClipboard(fullHlsUrl, "hls")}
                    className="p-1.5 bg-gray-800 hover:bg-gray-700 text-gray-300 rounded-lg transition-colors shrink-0"
                    title="Copy full HLS URL"
                  >
                    {copiedHls ? <Check className="w-4 h-4 text-emerald-400" /> : <Copy className="w-4 h-4" />}
                  </button>
                </div>
                <span className="text-[10px] text-gray-500 block">Fully qualified master manifest URL with low-latency chunks</span>
              </div>
            </div>
          </div>
        </div>

        {/* Card 4: Collapsible Advanced Settings (Optional Web Token & EPG Token) */}
        <div className="bg-[#111622] border border-gray-800 rounded-xl overflow-hidden shadow-lg lg:col-span-2">
          <button
            type="button"
            onClick={() => setIsAdvancedOpen(!isAdvancedOpen)}
            className="w-full px-5 py-4 flex items-center justify-between bg-[#141b2b] hover:bg-[#182136] transition-colors"
          >
            <div className="flex items-center gap-2">
              <Key className="w-4 h-4 text-purple-400" />
              <span className="text-sm font-bold text-white uppercase tracking-wider">
                Advanced Security & EPG Metadata Settings
              </span>
              <span className="text-xs text-gray-400 font-normal ml-2">
                (Optional Web Token & XMLTV Access Protection)
              </span>
            </div>
            {isAdvancedOpen ? <ChevronUp className="w-4 h-4 text-gray-400" /> : <ChevronDown className="w-4 h-4 text-gray-400" />}
          </button>

          {isAdvancedOpen && (
            <div className="p-5 space-y-4 border-t border-gray-800 animate-in slide-in-from-top-2 duration-200">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
                <div>
                  <label className="block text-xs font-medium text-gray-400 mb-1">
                    HLS Web Access Token (Optional)
                  </label>
                  <input
                    type="text"
                    value={formData.hls_web_token}
                    onChange={(e) => handleInputChange("hls_web_token", e.target.value)}
                    placeholder="Leave empty for open local playback, or enter secure token"
                    className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-xs text-white font-mono placeholder-gray-500 focus:outline-none focus:border-purple-500"
                  />
                  <p className="text-[10px] text-gray-500 mt-1">
                    When specified, all HLS master and segment requests require <code className="text-purple-400">?token=&lt;value&gt;</code> query parameter.
                  </p>
                </div>

                <div>
                  <div className="flex items-center justify-between mb-1">
                    <label className="block text-xs font-medium text-gray-400">
                      EPG XMLTV Access Token (Optional)
                    </label>
                    <button
                      type="button"
                      onClick={() => copyToClipboard(fullEpgUrl, "epg")}
                      className="text-[11px] text-purple-400 hover:text-purple-300 underline"
                    >
                      {copiedEpg ? "Copied EPG Link!" : "Copy EPG URL"}
                    </button>
                  </div>
                  <input
                    type="text"
                    value={formData.epg_web_token}
                    onChange={(e) => handleInputChange("epg_web_token", e.target.value)}
                    placeholder="Leave empty for open public XMLTV access, or enter secure token"
                    className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-xs text-white font-mono placeholder-gray-500 focus:outline-none focus:border-purple-500"
                  />
                  <p className="text-[10px] text-gray-500 mt-1">
                    Protects <code className="text-purple-400">/api/v1/epg/{activeChannel.id}.xml</code>. When set, IPTV clients must pass <code className="text-purple-400">?token=&lt;value&gt;</code>.
                  </p>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Live Browser Preview Modal */}
      {isPreviewModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm p-4 animate-in fade-in duration-200">
          <div className="bg-[#111622] border border-gray-800 rounded-xl max-w-4xl w-full overflow-hidden shadow-2xl flex flex-col">
            <div className="flex items-center justify-between px-5 py-3 border-b border-gray-800 bg-[#0d121c]">
              <div className="flex items-center gap-2">
                <Eye className="w-4 h-4 text-indigo-400" />
                <h3 className="text-sm font-bold text-white">
                  Live Channel Playout Preview: {formData.name}
                </h3>
                <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-indigo-950 border border-indigo-700 text-indigo-300">
                  Auto-Terminating Session
                </span>
              </div>
              <button
                onClick={handleClosePreview}
                className="p-1 text-gray-400 hover:text-white rounded-lg transition-colors"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="p-4 bg-black">
              {isPreviewLoading ? (
                <div className="w-full aspect-video flex flex-col items-center justify-center text-gray-400 space-y-2">
                  <RefreshCw className="w-6 h-6 animate-spin text-indigo-400" />
                  <span className="text-xs font-mono">Initializing HLS Playout Stream...</span>
                </div>
              ) : previewHlsUrl ? (
                <VideoPlayer streamUrl={previewHlsUrl} src={previewHlsUrl} autoPlay={true} />
              ) : (
                <div className="w-full aspect-video flex items-center justify-center text-gray-500 text-xs">
                  Stream unavailable
                </div>
              )}
            </div>

            <div className="px-5 py-3 border-t border-gray-800 flex items-center justify-between bg-[#0d121c] text-xs text-gray-400">
              <span className="font-mono truncate max-w-md">Source: {previewHlsUrl}</span>
              <button
                onClick={handleClosePreview}
                className="px-4 py-1.5 bg-gray-800 hover:bg-gray-700 text-white rounded-lg text-xs font-semibold"
              >
                Close Preview
              </button>
            </div>
          </div>
        </div>
      )}

      {/* FFmpeg Process Diagnostics Modal */}
      {renderFFmpegLogModal()}
    </div>
  );
}
