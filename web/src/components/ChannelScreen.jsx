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
  Upload
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
  onShowToast,
  t
}) {
  const activeChannel = channels.find((c) => c.id === activeChannelId) || channels[0] || {};
  const logoFileInputRef = useRef(null);
  const [isUploadingLogo, setIsUploadingLogo] = useState(false);
  const [isCallSignManual, setIsCallSignManual] = useState(false);

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
    logo_path: "media/logos/channel_logo.png",
    logo_position: "top-right",
    hls_web_token: "",
    epg_web_token: "",
    udp_url: "",
    srt_url: "",
    rtmp_url: ""
  });

  const [isSlateActive, setIsSlateActive] = useState(false);
  const [copiedHls, setCopiedHls] = useState(false);

  // Sync formData with activeChannel
  useEffect(() => {
    if (activeChannel && activeChannel.id) {
      const udp = activeChannel.destinations?.find((d) => d.protocol === "UDP_MULTICAST" || d.protocol === "udp" || d.type === "udp")?.endpoint_url
        || activeChannel.destinations?.find((d) => d.protocol === "UDP_MULTICAST" || d.protocol === "udp" || d.type === "udp")?.url
        || "udp://239.255.10.1:5000?pkt_size=1316";
      const srt = activeChannel.destinations?.find((d) => d.protocol === "SRT" || d.protocol === "srt" || d.type === "srt")?.endpoint_url
        || activeChannel.destinations?.find((d) => d.protocol === "SRT" || d.protocol === "srt" || d.type === "srt")?.url
        || "srt://127.0.0.1:9000?mode=caller";
      const rtmp = activeChannel.destinations?.find((d) => d.protocol === "RTMP" || d.protocol === "rtmp" || d.type === "rtmp")?.endpoint_url
        || activeChannel.destinations?.find((d) => d.protocol === "RTMP" || d.protocol === "rtmp" || d.type === "rtmp")?.url
        || "";

      const rawCallSign = activeChannel.call_sign || "";
      setFormData({
        name: activeChannel.name || "",
        call_sign: rawCallSign || toCallSignSlug(activeChannel.name || ""),
        lcn: activeChannel.lcn || 101,
        resolution_id: activeChannel.resolution_id || "res-in-1080i50",
        video_codec: activeChannel.video_codec || "libx264",
        audio_codec: activeChannel.audio_codec || "aac",
        logo_path: activeChannel.logo_path || "media/logos/channel_logo.png",
        logo_position: activeChannel.logo_position || "top-right",
        hls_web_token: activeChannel.hls_web_token || "",
        epg_web_token: activeChannel.epg_web_token || "",
        udp_url: udp,
        srt_url: srt,
        rtmp_url: rtmp
      });
      setIsCallSignManual(!!rawCallSign && rawCallSign !== toCallSignSlug(activeChannel.name || ""));
    }
  }, [activeChannel]);

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
    onShowToast(`Auto-generated DVB Call Sign slug: ${slug}`, "info");
  };

  const handleLogoUpload = async (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setIsUploadingLogo(true);
    try {
      const res = await api.uploadChannelLogo(activeChannel.id || "ch-01", file);
      const newLogoPath = res.logo_path || `media/logos/${activeChannel.id}_logo.png`;
      handleInputChange("logo_path", newLogoPath);
      onShowToast("Station logo uploaded successfully!", "success");
      onSaveChannel(activeChannel.id, {
        ...activeChannel,
        logo_path: newLogoPath,
        logo_position: formData.logo_position
      });
    } catch (err) {
      onShowToast("Failed to upload logo: " + err.message, "error");
    } finally {
      setIsUploadingLogo(false);
      if (logoFileInputRef.current) logoFileInputRef.current.value = "";
    }
  };

  const handleSave = () => {
    const autoSlug = toCallSignSlug(formData.name) || `CH-${activeChannel.id || '01'}`;
    const effectiveCallSign = formData.call_sign.trim() || autoSlug;
    const updatedChannel = {
      ...activeChannel,
      name: formData.name.trim() || activeChannel.name,
      call_sign: effectiveCallSign,
      lcn: parseInt(formData.lcn, 10) || 101,
      resolution_id: formData.resolution_id,
      video_codec: formData.video_codec,
      audio_codec: formData.audio_codec,
      logo_path: formData.logo_path,
      logo_position: formData.logo_position,
      hls_web_token: formData.hls_web_token,
      epg_web_token: formData.epg_web_token,
      destinations: [
        { type: "udp", protocol: "UDP_MULTICAST", enabled: true, url: formData.udp_url, endpoint_url: formData.udp_url },
        { type: "srt", protocol: "SRT", enabled: true, url: formData.srt_url, endpoint_url: formData.srt_url },
        ...(formData.rtmp_url ? [{ type: "rtmp", protocol: "RTMP", enabled: true, url: formData.rtmp_url, endpoint_url: formData.rtmp_url }] : []),
        { type: "hls", protocol: "HLS", enabled: true, url: `/hls/${activeChannel.id}/master.m3u8`, endpoint_url: `/hls/${activeChannel.id}/master.m3u8` }
      ]
    };
    onSaveChannel(activeChannel.id, updatedChannel);
  };

  const resolvedHlsUrl = activeChannel.id
    ? `${window.location.origin}/hls/${activeChannel.id}/master.m3u8${formData.hls_web_token ? `?token=${formData.hls_web_token}` : ''}`
    : "";

  const handleCopyHls = () => {
    if (!resolvedHlsUrl) return;
    navigator.clipboard.writeText(resolvedHlsUrl).then(() => {
      setCopiedHls(true);
      setTimeout(() => setCopiedHls(false), 2000);
      onShowToast("HLS stream URL copied to clipboard!", "success");
    });
  };

  const generateToken = (field) => {
    const chars = 'abcdef0123456789';
    let tok = field.includes('hls') ? 'hls_sec_' : 'epg_sec_';
    for (let i = 0; i < 16; i++) {
      tok += chars[Math.floor(Math.random() * chars.length)];
    }
    handleInputChange(field, tok);
    onShowToast(`Generated new security token: ${tok}`, "info");
  };

  return (
    <div className="h-full flex flex-col p-3 sm:p-4 space-y-4 overflow-y-auto max-w-full">
      {/* Top Bar with Channel Switcher and CRUD */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 shrink-0 bg-[#111827] border border-[#1F2937] p-3 rounded-lg shadow-sm">
        <div className="flex flex-wrap items-center gap-2 sm:gap-3">
          <div className="flex items-center gap-2">
            <Tv className="w-5 h-5 text-indigo-400 shrink-0" />
            <div>
              <h2 className="text-sm font-bold text-white">
                {t('channel.management_title') || "Channel Master Configuration"}
              </h2>
              <p className="text-[11px] text-gray-400">
                Active: <span className="text-sky-300 font-semibold">{activeChannel.name || "Default Channel"}</span>
              </p>
            </div>
          </div>
          <div className="hidden sm:block h-6 w-px bg-gray-700"></div>

          {/* Channel Selector */}
          <select
            value={activeChannel.id || ""}
            onChange={(e) => onSwitchChannel(e.target.value)}
            className="w-full sm:w-auto bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white font-medium focus:outline-none focus:border-indigo-500"
          >
            {channels.map((ch) => (
              <option key={ch.id} value={ch.id}>
                CH {String(ch.lcn || 1).padStart(2, '0')}: {ch.name} ({ch.call_sign || toCallSignSlug(ch.name)})
              </option>
            ))}
          </select>
        </div>

        <div className="flex items-center gap-2 self-end sm:self-auto">
          <button
            onClick={onCreateChannel}
            className="px-3 py-1.5 bg-gray-800 hover:bg-gray-700 text-gray-200 text-xs font-semibold rounded flex items-center gap-1 border border-gray-700 transition-colors"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>New Channel</span>
          </button>
          <button
            onClick={handleSave}
            className="px-3.5 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold rounded flex items-center gap-1 shadow transition-colors"
          >
            <Save className="w-3.5 h-3.5" />
            <span>Save Channel</span>
          </button>
          {channels.length > 1 && (
            <button
              onClick={() => onDeleteChannel(activeChannel.id)}
              className="px-2.5 py-1.5 bg-rose-950/40 hover:bg-rose-900/60 text-rose-300 text-xs font-semibold rounded border border-rose-500/30 flex items-center gap-1 transition-colors"
            >
              <Trash2 className="w-3.5 h-3.5" />
            </button>
          )}
        </div>
      </div>

      {/* Main Grid: Form Left, Confidence Monitor Right */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-4 flex-1 min-h-0">
        {/* Left Form: Parameters & Destinations (7 cols) */}
        <div className="lg:col-span-7 bg-[#111827] border border-[#1F2937] rounded-lg p-3 sm:p-4 space-y-4 overflow-y-auto text-xs">
          <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-3">
            <div>
              <label className="block text-[11px] text-gray-400 mb-1 font-medium">Channel Name</label>
              <input
                type="text"
                value={formData.name}
                onChange={(e) => handleNameChange(e.target.value)}
                placeholder="e.g. DD National HD"
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white focus:border-indigo-500 focus:outline-none"
              />
            </div>
            <div>
              <div className="flex items-center justify-between mb-1">
                <label className="text-[11px] text-gray-400 font-medium flex items-center gap-1">
                  <span>DVB Call Sign</span>
                  <span className="text-[9px] text-sky-400 font-mono">(slug)</span>
                </label>
                <button
                  type="button"
                  onClick={handleRegenerateCallSign}
                  title="Sync slug with Channel Name"
                  className="text-[10px] text-indigo-400 hover:text-indigo-300 font-mono flex items-center gap-0.5"
                >
                  ⚡ Sync
                </button>
              </div>
              <input
                type="text"
                value={formData.call_sign}
                onChange={(e) => handleCallSignChange(e.target.value)}
                placeholder="Auto-slug from title"
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white font-mono uppercase focus:border-indigo-500 focus:outline-none"
              />
            </div>
            <div>
              <label className="block text-[11px] text-gray-400 mb-1 font-medium">LCN Position</label>
              <input
                type="number"
                value={formData.lcn}
                onChange={(e) => handleInputChange("lcn", e.target.value)}
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white font-mono"
              />
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <label className="block text-[11px] text-gray-400 mb-1 font-medium">Broadcast Resolution Preset</label>
              <select
                value={formData.resolution_id}
                onChange={(e) => handleInputChange("resolution_id", e.target.value)}
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white font-medium focus:outline-none focus:border-indigo-500"
              >
                {resolutions.map((r) => (
                  <option key={r.id} value={r.id}>
                    {r.name} ({r.width}x{r.height} @ {r.frame_rate || 25}fps)
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-[11px] text-gray-400 mb-1 font-medium">Video Codec</label>
              <select
                value={formData.video_codec}
                onChange={(e) => handleInputChange("video_codec", e.target.value)}
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-indigo-500"
              >
                <option value="libx264">libx264 (Software H.264 CPU)</option>
                <option value="h264_nvenc">h264_nvenc (NVIDIA NVENC Hardware)</option>
                <option value="libx265">libx265 (HEVC Main 10)</option>
              </select>
            </div>
          </div>

          {/* On-Air Branding & Station Bug */}
          <div className="pt-2 border-t border-gray-800 space-y-3">
            <h3 className="text-xs font-bold text-sky-400 uppercase tracking-wider flex items-center gap-1.5">
              <span>On-Air Branding & Station Bug</span>
            </h3>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <div>
                <label className="block text-[11px] text-gray-400 mb-1 font-medium">Station Logo / Bug File</label>
                <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-2">
                  <input
                    type="text"
                    value={formData.logo_path}
                    onChange={(e) => handleInputChange("logo_path", e.target.value)}
                    placeholder="media/logos/channel_logo.png"
                    className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white font-mono break-all focus:outline-none focus:border-indigo-500"
                  />
                  <input
                    type="file"
                    ref={logoFileInputRef}
                    onChange={handleLogoUpload}
                    accept="image/png,image/jpeg,image/webp,image/svg+xml"
                    className="hidden"
                  />
                  <button
                    type="button"
                    onClick={() => logoFileInputRef.current?.click()}
                    disabled={isUploadingLogo}
                    className="px-3 py-1.5 bg-gray-700 hover:bg-gray-600 text-xs text-white rounded font-semibold shrink-0 transition-colors"
                  >
                    {isUploadingLogo ? "Uploading..." : "Upload"}
                  </button>
                </div>
              </div>

              <div>
                <label className="block text-[11px] text-gray-400 mb-1 font-medium">Bug Screen Position</label>
                <select
                  value={formData.logo_position}
                  onChange={(e) => handleInputChange("logo_position", e.target.value)}
                  className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white focus:outline-none focus:border-indigo-500"
                >
                  <option value="top-right">Top-Right (Standard)</option>
                  <option value="top-left">Top-Left</option>
                  <option value="bottom-right">Bottom-Right</option>
                  <option value="bottom-left">Bottom-Left</option>
                </select>
              </div>
            </div>
          </div>

          {/* Egress Destinations */}
          <div className="pt-2 border-t border-gray-800 space-y-3">
            <h3 className="text-xs font-bold text-gray-300 uppercase tracking-wider flex items-center gap-1.5">
              <Radio className="w-3.5 h-3.5 text-sky-400" />
              <span>Playout Egress Destinations</span>
            </h3>

            <div>
              <label className="block text-[11px] text-gray-400 mb-1 font-medium">UDP TS Multicast URL (DVB/Cable Mux)</label>
              <input
                type="text"
                value={formData.udp_url}
                onChange={(e) => handleInputChange("udp_url", e.target.value)}
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white font-mono break-all focus:outline-none focus:border-indigo-500"
              />
            </div>

            <div>
              <label className="block text-[11px] text-gray-400 mb-1 font-medium">SRT Caller URL (Remote Transmitter Egress)</label>
              <input
                type="text"
                value={formData.srt_url}
                onChange={(e) => handleInputChange("srt_url", e.target.value)}
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white font-mono break-all focus:outline-none focus:border-indigo-500"
              />
            </div>

            <div>
              <label className="block text-[11px] text-gray-400 mb-1 font-medium">RTMP Push Endpoint (OTT CDN)</label>
              <input
                type="text"
                value={formData.rtmp_url}
                placeholder="rtmp://live.cdn.tv/app/streamkey"
                onChange={(e) => handleInputChange("rtmp_url", e.target.value)}
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white font-mono break-all focus:outline-none focus:border-indigo-500"
              />
            </div>
          </div>

          {/* WebTokens */}
          <div className="pt-2 border-t border-gray-800 grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div>
              <div className="flex items-center justify-between mb-1">
                <label className="text-[11px] text-gray-400 font-medium">HLS Stream WebToken</label>
                <button
                  type="button"
                  onClick={() => generateToken("hls_web_token")}
                  className="text-[10px] text-sky-400 hover:text-sky-300 font-mono"
                >
                  ⚡ Gen
                </button>
              </div>
              <input
                type="text"
                value={formData.hls_web_token}
                onChange={(e) => handleInputChange("hls_web_token", e.target.value)}
                placeholder="Optional security token"
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-indigo-500"
              />
            </div>
            <div>
              <div className="flex items-center justify-between mb-1">
                <label className="text-[11px] text-gray-400 font-medium">EPG XML WebToken</label>
                <button
                  type="button"
                  onClick={() => generateToken("epg_web_token")}
                  className="text-[10px] text-sky-400 hover:text-sky-300 font-mono"
                >
                  ⚡ Gen
                </button>
              </div>
              <input
                type="text"
                value={formData.epg_web_token}
                onChange={(e) => handleInputChange("epg_web_token", e.target.value)}
                placeholder="Optional query token"
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-indigo-500"
              />
            </div>
          </div>

          {/* Resolved HLS URL */}
          <div className="pt-2 border-t border-gray-800">
            <label className="block text-[11px] text-gray-400 mb-1 font-medium">Direct Live HLS Playback URL</label>
            <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-2">
              <input
                type="text"
                readOnly
                value={resolvedHlsUrl}
                className="w-full bg-[#0B0F17] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-emerald-300 font-mono select-all break-all"
              />
              <button
                type="button"
                onClick={handleCopyHls}
                className="px-3 py-1.5 bg-gray-800 hover:bg-gray-700 text-gray-200 rounded text-xs font-semibold flex items-center justify-center gap-1 shrink-0 transition-colors"
              >
                {copiedHls ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                <span>Copy</span>
              </button>
            </div>
          </div>
        </div>

        {/* Right Panel: Confidence Monitor & Master Slate (5 cols) */}
        <div className="lg:col-span-5 flex flex-col space-y-4">
          <div className="bg-[#111827] border border-[#1F2937] rounded-lg p-3 space-y-3">
            <div className="flex items-center justify-between">
              <h3 className="text-xs font-bold text-white uppercase tracking-wider flex items-center gap-1.5">
                <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
                <span>Live Confidence Monitor</span>
              </h3>
              <span className="text-[10px] font-mono text-gray-400">10-Segment Rolling HLS</span>
            </div>

            {/* Video Player */}
            <VideoPlayer
              streamUrl={resolvedHlsUrl}
              isSlate={isSlateActive}
              channelName={activeChannel.name}
              logoPath={formData.logo_path}
              logoPosition={formData.logo_position}
            />

            {/* Emergency Slate Button */}
            <button
              onClick={() => setIsSlateActive(!isSlateActive)}
              className={`w-full py-2 px-3 rounded text-xs font-bold uppercase tracking-wider flex items-center justify-center gap-2 shadow transition-all ${
                isSlateActive
                  ? 'bg-gray-800 text-red-400 border border-red-500/50'
                  : 'bg-rose-600 hover:bg-rose-500 text-white'
              }`}
            >
              <AlertOctagon className="w-4 h-4" />
              <span>{isSlateActive ? "Disengage Emergency Slate" : "Trigger Emergency Slate"}</span>
            </button>
          </div>

          {/* Redundancy & Edge Node Pairing Status */}
          <div className="bg-[#111827] border border-[#1F2937] rounded-lg p-3 space-y-2 text-xs">
            <h4 className="text-[11px] font-bold text-gray-300 uppercase tracking-wider flex items-center gap-1.5">
              <Shield className="w-3.5 h-3.5 text-indigo-400" />
              <span>1+1 Redundant Playout Pair</span>
            </h4>
            <div className="space-y-1.5">
              <div className="flex items-center justify-between p-2 bg-[#1F2937] rounded border border-emerald-500/30">
                <div className="flex items-center gap-2">
                  <span className="w-2 h-2 rounded-full bg-emerald-400"></span>
                  <span className="font-semibold text-white">Primary Agent (delhi-dc1)</span>
                </div>
                <span className="text-[10px] font-mono text-emerald-400">HOT • ON-AIR</span>
              </div>
              <div className="flex items-center justify-between p-2 bg-[#1F2937] rounded border border-gray-700">
                <div className="flex items-center gap-2">
                  <span className="w-2 h-2 rounded-full bg-amber-400"></span>
                  <span className="font-semibold text-white">Standby Agent (mumbai-dc2)</span>
                </div>
                <span className="text-[10px] font-mono text-amber-400">HOT STANDBY</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
