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
  ChevronUp
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
  onBackToDashboard,
  onNavigateToAdStudio,
  onShowToast,
  t
}) {
  const activeChannel = channels.find((c) => c.id === activeChannelId) || channels[0] || {};

  const [copiedHls, setCopiedHls] = useState(false);
  const [copiedEpg, setCopiedEpg] = useState(false);
  const [isSlateActive, setIsSlateActive] = useState(false);
  const [playoutStatus, setPlayoutStatus] = useState(null);
  const [isCallSignManual, setIsCallSignManual] = useState(false);
  const [isAdvancedOpen, setIsAdvancedOpen] = useState(false);

  // Live Browser Preview Modal State
  const [isPreviewModalOpen, setIsPreviewModalOpen] = useState(false);
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
    if (!activeChannel?.id) return;
    const interval = setInterval(() => {
      api.getPlayoutStatus(activeChannel.id).then((st) => {
        if (st) setPlayoutStatus(st);
      }).catch(() => {});
    }, 4000);
    return () => clearInterval(interval);
  }, [activeChannel?.id]);

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

  // Live Browser Preview Handling
  const handleOpenPreview = async () => {
    setIsPreviewLoading(true);
    setIsPreviewModalOpen(true);
    try {
      const res = await api.startPreview(activeChannel.id);
      let streamUrl = res.hls_url || `/hls/${activeChannel.id}/master.m3u8`;
      if (formData.hls_web_token && !streamUrl.includes("token=")) {
        streamUrl += (streamUrl.includes("?") ? "&" : "?") + "token=" + encodeURIComponent(formData.hls_web_token);
      }
      setPreviewHlsUrl(streamUrl);
    } catch (err) {
      onShowToast("Failed to initiate live preview: " + err.message, "error");
      let fallback = `/hls/${activeChannel.id}/master.m3u8`;
      if (formData.hls_web_token) fallback += "?token=" + encodeURIComponent(formData.hls_web_token);
      setPreviewHlsUrl(fallback);
    } finally {
      setIsPreviewLoading(false);
    }
  };

  const handleClosePreview = async () => {
    setIsPreviewModalOpen(false);
    setPreviewHlsUrl("");
    try {
      // Auto-terminates temporary preview if HLS is not permanent destination
      await api.stopPreview(activeChannel.id);
    } catch (e) {}
  };

  const copyToClipboard = (text, type = "hls") => {
    navigator.clipboard.writeText(text);
    if (type === "hls") {
      setCopiedHls(true);
      setTimeout(() => setCopiedHls(false), 2000);
    } else {
      setCopiedEpg(true);
      setTimeout(() => setCopiedEpg(false), 2000);
    }
    onShowToast(`Copied ${type.toUpperCase()} endpoint URL to clipboard`, "info");
  };

  const fullHlsUrl = `${window.location.origin}/hls/${activeChannel.id}/master.m3u8${formData.hls_web_token ? `?token=${encodeURIComponent(formData.hls_web_token)}` : ''}`;
  const fullEpgUrl = `${window.location.origin}/api/v1/epg/${activeChannel.id}.xml${formData.epg_web_token ? `?token=${encodeURIComponent(formData.epg_web_token)}` : ''}`;

  return (
    <div className="space-y-6 pb-12 animate-in fade-in duration-300">
      {/* Top Header & Controls */}
      <div className="bg-[#111622] border border-gray-800 rounded-xl p-5 shadow-lg flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div className="flex items-center gap-4">
          <button
            onClick={onBackToDashboard}
            className="p-2 bg-gray-800/80 hover:bg-gray-700 text-gray-300 hover:text-white rounded-lg transition-colors"
            title="Back to Bird's Eye Dashboard"
          >
            <ArrowLeft className="w-5 h-5" />
          </button>
          <div>
            <div className="flex items-center gap-2">
              <Tv className="w-5 h-5 text-blue-400" />
              <h1 className="text-xl font-bold text-white tracking-wide">Channel Master Control</h1>
              <span className="px-2 py-0.5 rounded text-[11px] font-mono bg-blue-950/70 border border-blue-700/50 text-blue-300">
                LCN {formData.lcn}
              </span>
            </div>
            <p className="text-xs text-gray-400 mt-0.5">
              Strictly scheduled playout, raster configuration, egress switches, and Ad & Layout assignment
            </p>
          </div>
        </div>

        {/* Channel Selector & Actions */}
        <div className="flex flex-wrap items-center gap-2">
          <select
            value={activeChannel.id || ""}
            onChange={(e) => onSwitchChannel(e.target.value)}
            className="bg-[#182030] border border-gray-700 text-white rounded-lg px-3 py-2 text-sm font-medium focus:ring-2 focus:ring-blue-500 focus:outline-none"
          >
            {channels.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name} ({c.call_sign || `CH-${c.lcn}`})
              </option>
            ))}
          </select>

          {/* Playout Status Badge */}
          <div className={`px-3 py-1.5 rounded-lg border text-xs font-semibold flex items-center gap-2 ${
            playoutStatus?.state === "ON-AIR"
              ? "bg-red-950/70 border-red-700 text-red-300 animate-pulse"
              : playoutStatus?.state === "EMERGENCY_SLATE"
              ? "bg-amber-950/70 border-amber-700 text-amber-300"
              : "bg-gray-800/80 border-gray-700 text-gray-400"
          }`}>
            <span className={`w-2 h-2 rounded-full ${
              playoutStatus?.state === "ON-AIR" ? "bg-red-500" : playoutStatus?.state === "EMERGENCY_SLATE" ? "bg-amber-500" : "bg-gray-500"
            }`} />
            {playoutStatus?.state || "STANDBY"}
            {playoutStatus?.smpt_timecode && (
              <span className="font-mono text-gray-300 font-normal ml-1">
                {playoutStatus.smpt_timecode}
              </span>
            )}
          </div>

          {/* Emergency Slate Toggle */}
          <button
            onClick={handleToggleSlate}
            className={`px-3 py-2 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-colors border ${
              isSlateActive
                ? "bg-amber-600 hover:bg-amber-500 text-white border-amber-500"
                : "bg-gray-800 hover:bg-gray-700 text-amber-400 border-gray-700"
            }`}
            title="Toggle Technical Difficulties Slate"
          >
            <AlertOctagon className="w-4 h-4" />
            {isSlateActive ? "Release Slate" : "Emergency Slate"}
          </button>

          {/* Live Browser Preview Button */}
          <button
            onClick={handleOpenPreview}
            className="px-3 py-2 bg-indigo-900/40 hover:bg-indigo-900/70 text-indigo-300 border border-indigo-700/60 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-colors"
          >
            <Eye className="w-4 h-4 text-indigo-400" />
            Live Preview
          </button>

          {/* New Channel Button */}
          <button
            onClick={onCreateChannel}
            className="p-2 bg-gray-800 hover:bg-gray-700 text-gray-200 border border-gray-700 rounded-lg transition-colors"
            title="Create New Channel"
          >
            <Plus className="w-4 h-4" />
          </button>

          {/* Save Channel Button */}
          <button
            onClick={handleSave}
            className="px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white rounded-lg text-sm font-semibold flex items-center gap-1.5 shadow-md shadow-blue-900/30 transition-all"
          >
            <Save className="w-4 h-4" />
            Save Changes
          </button>

          {/* Delete Channel Button */}
          <button
            onClick={() => onDeleteChannel(activeChannel.id)}
            className="p-2 bg-red-950/40 hover:bg-red-900/60 text-red-400 border border-red-800/40 rounded-lg transition-colors"
            title="Delete Channel"
          >
            <Trash2 className="w-4 h-4" />
          </button>
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
              <label className="block text-xs font-medium text-gray-400 mb-1">Assigned Ad & Layout Template</label>
              <select
                value={formData.ad_template_id}
                onChange={(e) => handleInputChange("ad_template_id", e.target.value)}
                className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-sm text-white focus:ring-2 focus:ring-purple-500 focus:outline-none"
              >
                <option value="">-- None (No Branding / Overlays) --</option>
                {adTemplates.map((tmpl) => (
                  <option key={tmpl.id} value={tmpl.id}>
                    {tmpl.name} ({tmpl.template_type || 'composite'}) - {tmpl.overlay_elements?.length || 0} Overlays
                  </option>
                ))}
              </select>
              <p className="text-[11px] text-gray-400 mt-1">
                Branding, station watermark logo, dynamic Now Playing, and ad banners are centralized in the template.
              </p>
            </div>

            {/* Collision Detection Warning Banner */}
            {detectedCollisions.length > 0 && (
              <div className="p-3 bg-amber-950/40 border border-amber-700/60 rounded-xl space-y-2">
                <div className="flex items-start gap-2.5">
                  <AlertTriangle className="w-5 h-5 text-amber-400 shrink-0 mt-0.5" />
                  <div>
                    <h4 className="text-xs font-bold text-amber-300">
                      Layout Overlap Collision Detected ({detectedCollisions.length} overlap{detectedCollisions.length > 1 ? 's' : ''})
                    </h4>
                    <p className="text-[11px] text-amber-200/80 mt-0.5">
                      The assigned template has multiple graphic elements occupying overlapping coordinates:
                    </p>
                    <ul className="text-[11px] text-amber-300/90 list-disc list-inside mt-1 space-y-0.5 font-mono">
                      {detectedCollisions.map((col, idx) => (
                        <li key={idx}>
                          {col.a.type.toUpperCase()} ({col.a.x},{col.a.y}) collides with {col.b.type.toUpperCase()} ({col.b.x},{col.b.y})
                        </li>
                      ))}
                    </ul>
                  </div>
                </div>

                <div className="bg-amber-900/30 rounded px-2.5 py-1.5 text-[11px] text-amber-200 flex items-center justify-between">
                  <span>
                    Configured Rule: <strong>
                      {assignedTemplate?.collision_behavior === "alternate"
                        ? `Alternate every ${assignedTemplate.alternate_duration_seconds || 15}s`
                        : assignedTemplate?.collision_behavior === "priority"
                        ? "Priority Precedence Order"
                        : "No resolution rule set (Simultaneous display)"}
                    </strong>
                  </span>
                  {onNavigateToAdStudio && (
                    <button
                      type="button"
                      onClick={onNavigateToAdStudio}
                      className="text-amber-300 hover:text-white underline font-semibold ml-2"
                    >
                      Configure Resolution
                    </button>
                  )}
                </div>
              </div>
            )}

            {/* Template Summary Card */}
            {assignedTemplate && (
              <div className="bg-[#141b2b] border border-gray-800 rounded-lg p-3 text-xs space-y-1.5 text-gray-300">
                <div className="flex items-center justify-between font-semibold text-white">
                  <span>Template: {assignedTemplate.name}</span>
                  <span className="text-[11px] px-2 py-0.5 bg-purple-950 border border-purple-800 text-purple-300 rounded">
                    {assignedTemplate.template_type}
                  </span>
                </div>
                <div className="text-[11px] text-gray-400 grid grid-cols-2 gap-2 pt-1">
                  <div>Logo: {assignedTemplate.logo_path ? assignedTemplate.logo_position || 'top-right' : 'Default Bug'}</div>
                  <div>Overlays: {assignedTemplate.overlay_elements?.length || 0} layers</div>
                  <div>Commercial Breaks: {assignedTemplate.commercial_breaks?.length || 0} scheduled</div>
                  <div>Safe Area: EBU R95 16:9 Broadcast</div>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Card 3: Streaming Output Destinations (Egress Switches) */}
        <div className="bg-[#111622] border border-gray-800 rounded-xl p-5 shadow-lg space-y-4 lg:col-span-2">
          <div className="flex items-center justify-between pb-3 border-b border-gray-800">
            <div className="flex items-center gap-2">
              <Radio className="w-4 h-4 text-emerald-400" />
              <h2 className="text-sm font-bold text-white uppercase tracking-wider">Streaming Output Destinations</h2>
            </div>
            <span className="text-xs text-gray-400">Supported: UDP Multicast, RTMP, HLS</span>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
            {/* 1. UDP Multicast / Unicast */}
            <div className={`p-4 rounded-xl border transition-all ${
              formData.udp_enabled
                ? "bg-[#142028] border-emerald-700/60 shadow-md"
                : "bg-[#141b28]/60 border-gray-800 opacity-70"
            }`}>
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  <span className="w-2.5 h-2.5 rounded-full bg-emerald-500" />
                  <span className="text-xs font-bold text-white uppercase tracking-wide">UDP TS Multicast</span>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={formData.udp_enabled}
                    onChange={(e) => handleInputChange("udp_enabled", e.target.checked)}
                    className="sr-only peer"
                  />
                  <div className="w-9 h-5 bg-gray-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-emerald-600"></div>
                </label>
              </div>

              <label className="block text-[11px] text-gray-400 mb-1">MPEG-TS Multicast / Unicast Address</label>
              <input
                type="text"
                disabled={!formData.udp_enabled}
                value={formData.udp_url}
                onChange={(e) => handleInputChange("udp_url", e.target.value)}
                placeholder="udp://239.255.10.1:5000?pkt_size=1316"
                className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-1.5 text-xs text-white placeholder-gray-500 font-mono focus:ring-2 focus:ring-emerald-500 focus:outline-none disabled:opacity-50"
              />
              <p className="text-[10px] text-gray-400 mt-1">Direct playout to broadcast DVB mux / edge transmitters.</p>
            </div>

            {/* 2. RTMP Egress */}
            <div className={`p-4 rounded-xl border transition-all ${
              formData.rtmp_enabled
                ? "bg-[#1f192b] border-pink-700/60 shadow-md"
                : "bg-[#141b28]/60 border-gray-800 opacity-70"
            }`}>
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  <span className="w-2.5 h-2.5 rounded-full bg-pink-500" />
                  <span className="text-xs font-bold text-white uppercase tracking-wide">RTMP Egress</span>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={formData.rtmp_enabled}
                    onChange={(e) => handleInputChange("rtmp_enabled", e.target.checked)}
                    className="sr-only peer"
                  />
                  <div className="w-9 h-5 bg-gray-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-pink-600"></div>
                </label>
              </div>

              <div className="space-y-2">
                <div>
                  <label className="block text-[11px] text-gray-400 mb-0.5">RTMP Endpoint URL</label>
                  <input
                    type="text"
                    disabled={!formData.rtmp_enabled}
                    value={formData.rtmp_url}
                    onChange={(e) => handleInputChange("rtmp_url", e.target.value)}
                    placeholder="rtmp://live.twitch.tv/app"
                    className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-1.5 text-xs text-white placeholder-gray-500 font-mono focus:ring-2 focus:ring-pink-500 focus:outline-none disabled:opacity-50"
                  />
                </div>
                <div>
                  <label className="block text-[11px] text-gray-400 mb-0.5">Stream Key (Optional)</label>
                  <input
                    type="password"
                    disabled={!formData.rtmp_enabled}
                    value={formData.rtmp_key}
                    onChange={(e) => handleInputChange("rtmp_key", e.target.value)}
                    placeholder="live_secret_key"
                    className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-1.5 text-xs text-white placeholder-gray-500 font-mono focus:ring-2 focus:ring-pink-500 focus:outline-none disabled:opacity-50"
                  />
                </div>
              </div>
            </div>

            {/* 3. HLS Egress */}
            <div className={`p-4 rounded-xl border transition-all ${
              formData.hls_enabled
                ? "bg-[#142338] border-blue-700/60 shadow-md"
                : "bg-[#141b28]/60 border-gray-800 opacity-70"
            }`}>
              <div className="flex items-center justify-between mb-3">
                <div className="flex items-center gap-2">
                  <span className="w-2.5 h-2.5 rounded-full bg-blue-500" />
                  <span className="text-xs font-bold text-white uppercase tracking-wide">HLS Sliding Window</span>
                </div>
                <label className="relative inline-flex items-center cursor-pointer">
                  <input
                    type="checkbox"
                    checked={formData.hls_enabled}
                    onChange={(e) => handleInputChange("hls_enabled", e.target.checked)}
                    className="sr-only peer"
                  />
                  <div className="w-9 h-5 bg-gray-700 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-blue-600"></div>
                </label>
              </div>

              <label className="block text-[11px] text-gray-400 mb-1">Rolling HLS Manifest (master.m3u8)</label>
              <div className="flex items-center gap-1.5">
                <input
                  type="text"
                  readOnly
                  value={fullHlsUrl}
                  className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-1.5 text-xs text-blue-300 font-mono select-all focus:outline-none"
                />
                <button
                  type="button"
                  onClick={() => copyToClipboard(fullHlsUrl, "hls")}
                  className="p-1.5 bg-gray-800 hover:bg-gray-700 text-gray-200 border border-gray-700 rounded-lg transition-colors shrink-0"
                  title="Copy HLS URL"
                >
                  {copiedHls ? <Check className="w-4 h-4 text-green-400" /> : <Copy className="w-4 h-4" />}
                </button>
              </div>
              <p className="text-[10px] text-gray-400 mt-1">10-segment sliding window with 2-second sub-chunks.</p>
            </div>
          </div>
        </div>

        {/* Card 4: Collapsible Advanced Settings (Optional Web Token & EPG) */}
        <div className="bg-[#111622] border border-gray-800 rounded-xl shadow-lg lg:col-span-2 overflow-hidden">
          <button
            type="button"
            onClick={() => setIsAdvancedOpen(!isAdvancedOpen)}
            className="w-full p-4 flex items-center justify-between text-left hover:bg-gray-800/40 transition-colors"
          >
            <div className="flex items-center gap-2.5">
              <Shield className="w-4 h-4 text-amber-400" />
              <div>
                <span className="text-sm font-bold text-white uppercase tracking-wider">Advanced Settings (Optional)</span>
                <span className="text-xs text-gray-400 ml-2">Security Web Tokens & XMLTV EPG Distribution</span>
              </div>
            </div>
            {isAdvancedOpen ? <ChevronUp className="w-5 h-5 text-gray-400" /> : <ChevronDown className="w-5 h-5 text-gray-400" />}
          </button>

          {isAdvancedOpen && (
            <div className="p-5 pt-1 border-t border-gray-800/80 space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
                {/* Optional HLS Token */}
                <div>
                  <label className="block text-xs font-medium text-gray-300 mb-1">
                    HLS Web Access Token <span className="text-gray-400 font-normal">(Optional, default blank)</span>
                  </label>
                  <div className="flex items-center gap-2">
                    <input
                      type="text"
                      value={formData.hls_web_token}
                      onChange={(e) => handleInputChange("hls_web_token", e.target.value)}
                      placeholder="e.g. live_secret_token (leave blank for open access)"
                      className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-xs text-white placeholder-gray-500 font-mono focus:ring-2 focus:ring-amber-500 focus:outline-none"
                    />
                    <button
                      type="button"
                      onClick={() => handleInputChange("hls_web_token", `hls_${Date.now().toString(36)}_${Math.random().toString(36).substring(2, 6)}`)}
                      className="px-2.5 py-2 bg-gray-800 hover:bg-gray-700 text-gray-300 rounded-lg text-xs shrink-0 font-medium border border-gray-700"
                    >
                      Generate
                    </button>
                  </div>
                  <p className="text-[11px] text-gray-400 mt-1">
                    When populated, HLS requests must include <code>?token=...</code> to stream.
                  </p>
                </div>

                {/* Optional EPG Token */}
                <div>
                  <label className="block text-xs font-medium text-gray-300 mb-1">
                    XMLTV EPG Web Token <span className="text-gray-400 font-normal">(Optional, default blank)</span>
                  </label>
                  <div className="flex items-center gap-2">
                    <input
                      type="text"
                      value={formData.epg_web_token}
                      onChange={(e) => handleInputChange("epg_web_token", e.target.value)}
                      placeholder="e.g. epg_secret_token (leave blank for open access)"
                      className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-xs text-white placeholder-gray-500 font-mono focus:ring-2 focus:ring-amber-500 focus:outline-none"
                    />
                    <button
                      type="button"
                      onClick={() => handleInputChange("epg_web_token", `epg_${Date.now().toString(36)}_${Math.random().toString(36).substring(2, 6)}`)}
                      className="px-2.5 py-2 bg-gray-800 hover:bg-gray-700 text-gray-300 rounded-lg text-xs shrink-0 font-medium border border-gray-700"
                    >
                      Generate
                    </button>
                  </div>
                  <p className="text-[11px] text-gray-400 mt-1">
                    When populated, external EPG clients must query <code>/api/v1/epg/{activeChannel.id}.xml?token=...</code>
                  </p>
                </div>
              </div>

              {/* Direct EPG Download link */}
              <div className="pt-2 border-t border-gray-800/60 flex flex-col sm:flex-row sm:items-center justify-between gap-2">
                <div className="text-xs text-gray-300 flex items-center gap-2 font-mono truncate">
                  <span className="text-gray-400">XMLTV Endpoint:</span>
                  <span className="text-blue-300 truncate">{fullEpgUrl}</span>
                </div>
                <button
                  type="button"
                  onClick={() => copyToClipboard(fullEpgUrl, "epg")}
                  className="px-3 py-1.5 bg-gray-800 hover:bg-gray-700 text-gray-300 rounded text-xs flex items-center gap-1.5 shrink-0 border border-gray-700"
                >
                  {copiedEpg ? <Check className="w-3.5 h-3.5 text-green-400" /> : <Copy className="w-3.5 h-3.5" />}
                  Copy EPG URL
                </button>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Live Browser Preview Modal (HLS Exception) */}
      {isPreviewModalOpen && (
        <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-[#111622] border border-gray-800 rounded-2xl w-full max-w-4xl shadow-2xl overflow-hidden animate-in zoom-in-95 duration-200">
            <div className="p-4 border-b border-gray-800 flex items-center justify-between bg-[#151c2c]">
              <div className="flex items-center gap-2">
                <Eye className="w-5 h-5 text-indigo-400" />
                <h3 className="font-bold text-white text-base">
                  Live Browser Preview • {activeChannel.name}
                </h3>
                <span className="px-2 py-0.5 rounded text-[11px] bg-red-950/80 border border-red-800 text-red-300 animate-pulse font-mono font-bold">
                  LIVE HLS
                </span>
              </div>
              <button
                onClick={handleClosePreview}
                className="p-1.5 text-gray-400 hover:text-white rounded-lg hover:bg-gray-800 transition-colors"
                title="Close Live Preview"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="p-6 space-y-4">
              <div className="aspect-video bg-black rounded-xl overflow-hidden border border-gray-800 shadow-inner relative">
                {previewHlsUrl ? (
                  <VideoPlayer src={previewHlsUrl} autoPlay={true} muted={true} />
                ) : (
                  <div className="h-full flex flex-col items-center justify-center text-gray-500">
                    <Radio className="w-10 h-10 animate-pulse mb-2 text-indigo-500" />
                    <p className="text-sm">Initiating low-latency preview stream...</p>
                  </div>
                )}
              </div>

              <div className="bg-[#182030] p-3 rounded-lg flex items-center justify-between text-xs text-gray-300">
                <span className="font-mono text-[11px] text-gray-400 truncate">
                  Stream URL: {previewHlsUrl}
                </span>
                <span className="text-[11px] text-indigo-300 bg-indigo-950/60 px-2 py-0.5 rounded border border-indigo-800 shrink-0">
                  Temporary in-browser session auto-terminates on close
                </span>
              </div>
            </div>

            <div className="p-4 border-t border-gray-800 flex justify-end bg-[#151c2c]">
              <button
                onClick={handleClosePreview}
                className="px-4 py-2 bg-gray-800 hover:bg-gray-700 text-white rounded-lg text-sm font-semibold transition-colors"
              >
                Close Preview
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
