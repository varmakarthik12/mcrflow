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
  Upload,
  ArrowLeft,
  Film,
  FolderOpen,
  FileVideo,
  Search,
  X,
  Play,
  Square,
  Sparkles,
  Move,
  Layout,
  Layers,
  Palette
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
  onShowToast,
  t
}) {
  const activeChannel = channels.find((c) => c.id === activeChannelId) || channels[0] || {};
  const logoFileInputRef = useRef(null);
  const monitorCanvasRef = useRef(null);

  const [isUploadingLogo, setIsUploadingLogo] = useState(false);
  const [isCallSignManual, setIsCallSignManual] = useState(false);
  const [copiedHls, setCopiedHls] = useState(false);
  const [isSlateActive, setIsSlateActive] = useState(false);
  const [isPlayoutRunning, setIsPlayoutRunning] = useState(false);

  // Layout & Graphics Editing State
  const [isEditLayoutMode, setIsEditLayoutMode] = useState(false);
  const [selectedOverlayId, setSelectedOverlayId] = useState("logo"); // "logo" or overlay.id
  const [draggingTarget, setDraggingTarget] = useState(null); // "logo" or overlay.id
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });
  const [elementStartPos, setElementStartPos] = useState({ x: 0, y: 0 });

  // Quick Media Picker Modal State
  const [isMediaPickerOpen, setIsMediaPickerOpen] = useState(false);
  const [mediaList, setMediaList] = useState([]);
  const [mediaFilterQuery, setMediaFilterQuery] = useState("");
  const [mediaCurrentPath, setMediaCurrentPath] = useState("");
  const [selectedMediaPath, setSelectedMediaPath] = useState("sample_movie.mp4");

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
    logo_path: "data/logos/channel_logo.png",
    logo_position: "top-right",
    logo_x: 1720,
    logo_y: 40,
    logo_width: 140,
    logo_height: 90,
    logo_opacity: 0.90,
    logo_fit: "contain",
    ad_template_id: "",
    hls_web_token: "",
    epg_web_token: "",
    udp_url: "",
    srt_url: "",
    rtmp_url: ""
  });

  const [overlays, setOverlays] = useState([]);

  // Sync formData with activeChannel
  useEffect(() => {
    if (activeChannel && activeChannel.id) {
      const udp = activeChannel.destinations?.find((d) => d.protocol === "UDP_MULTICAST" || d.protocol === "udp" || d.type === "udp")?.endpoint_url
        || activeChannel.destinations?.find((d) => d.protocol === "UDP_MULTICAST" || d.protocol === "udp" || d.type === "udp")?.url
        || "udp://239.255.10.1:5000?pkt_size=1316";
      const srt = activeChannel.destinations?.find((d) => d.protocol === "SRT" || d.protocol === "srt" || d.type === "srt")?.endpoint_url
        || activeChannel.destinations?.find((d) => d.protocol === "SRT" || d.protocol === "srt" || d.type === "srt")?.url
        || "";
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
        logo_path: activeChannel.logo_path || "data/logos/channel_logo.png",
        logo_position: activeChannel.logo_position || "top-right",
        logo_x: activeChannel.logo_x || 1720,
        logo_y: activeChannel.logo_y || 40,
        logo_width: activeChannel.logo_width || 140,
        logo_height: activeChannel.logo_height || 90,
        logo_opacity: activeChannel.logo_opacity || 0.90,
        logo_fit: activeChannel.logo_fit || "contain",
        ad_template_id: activeChannel.ad_template_id || "",
        hls_web_token: activeChannel.hls_web_token || "",
        epg_web_token: activeChannel.epg_web_token || "",
        udp_url: udp,
        srt_url: srt,
        rtmp_url: rtmp
      });

      setOverlays(activeChannel.overlays || []);
      setIsCallSignManual(!!rawCallSign && rawCallSign !== toCallSignSlug(activeChannel.name || ""));

      // Check playout status
      api.getPlayoutStatus(activeChannel.id).then((st) => {
        if (st && st.state === "ON-AIR") {
          setIsPlayoutRunning(true);
          if (st.media_path) setSelectedMediaPath(st.media_path);
        } else {
          setIsPlayoutRunning(false);
        }
      }).catch(() => {});
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
      const newLogoPath = res.logo_path || `data/logos/${activeChannel.id}_logo.png`;
      handleInputChange("logo_path", newLogoPath);
      onShowToast("Station logo uploaded successfully!", "success");
      onSaveChannel(activeChannel.id, {
        ...activeChannel,
        ...formData,
        logo_path: newLogoPath,
        overlays: overlays
      });
    } catch (err) {
      onShowToast("Failed to upload logo: " + err.message, "error");
    } finally {
      setIsUploadingLogo(false);
      if (logoFileInputRef.current) logoFileInputRef.current.value = "";
    }
  };

  // Media Picker Logic
  const openMediaPicker = async () => {
    setIsMediaPickerOpen(true);
    try {
      const files = await api.browseStorage(mediaCurrentPath);
      setMediaList(Array.isArray(files) ? files : []);
    } catch (e) {
      onShowToast("Failed to browse media library: " + e.message, "error");
    }
  };

  const handleMediaBrowsePath = async (subPath) => {
    setMediaCurrentPath(subPath);
    try {
      const files = await api.browseStorage(subPath);
      setMediaList(Array.isArray(files) ? files : []);
    } catch (e) {}
  };

  const handleSelectMediaFile = (file) => {
    const p = file.path || file.name;
    setSelectedMediaPath(p);
    setIsMediaPickerOpen(false);
    onShowToast(`Selected playout media: ${file.name}`, "info");
  };

  const handleStartPlayout = async () => {
    try {
      await api.startPlayout(activeChannel.id, {
        media_path: selectedMediaPath
      });
      setIsPlayoutRunning(true);
      onShowToast(`Playout started on ${activeChannel.name} (UDP & HLS active)`, "success");
    } catch (err) {
      onShowToast("Failed to start playout: " + err.message, "error");
    }
  };

  const handleStopPlayout = async () => {
    try {
      await api.stopPlayout(activeChannel.id);
      setIsPlayoutRunning(false);
      onShowToast(`Playout stopped for ${activeChannel.name}`, "info");
    } catch (err) {
      onShowToast("Failed to stop playout: " + err.message, "error");
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
      logo_x: parseInt(formData.logo_x, 10) || 1720,
      logo_y: parseInt(formData.logo_y, 10) || 40,
      logo_width: parseInt(formData.logo_width, 10) || 140,
      logo_height: parseInt(formData.logo_height, 10) || 90,
      logo_opacity: parseFloat(formData.logo_opacity) || 0.90,
      logo_fit: formData.logo_fit || "contain",
      ad_template_id: formData.ad_template_id || "",
      overlays: overlays,
      hls_web_token: formData.hls_web_token,
      epg_web_token: formData.epg_web_token,
      destinations: [
        { type: "udp", protocol: "UDP_MULTICAST", enabled: true, url: formData.udp_url, endpoint_url: formData.udp_url },
        { type: "srt", protocol: "SRT", enabled: !!formData.srt_url && formData.srt_url.trim() !== "" && !formData.srt_url.includes("127.0.0.1:9000"), url: formData.srt_url, endpoint_url: formData.srt_url },
        ...(formData.rtmp_url ? [{ type: "rtmp", protocol: "RTMP", enabled: true, url: formData.rtmp_url, endpoint_url: formData.rtmp_url }] : []),
        { type: "hls", protocol: "HLS", enabled: true, url: `/hls/${activeChannel.id}/master.m3u8`, endpoint_url: `/hls/${activeChannel.id}/master.m3u8` }
      ]
    };
    onSaveChannel(activeChannel.id, updatedChannel);
    onShowToast("Channel parameters, logo geometry & on-air overlays saved!", "success");
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

  // Overlay Management Operations
  const handleAddOverlay = (type) => {
    const newId = `ov-${Date.now().toString(36)}`;
    let newEl = {
      id: newId,
      type: type,
      text: "BROADCAST GRAPHIC",
      sub_text: "",
      x: 60,
      y: 60,
      width: 380,
      height: 85,
      background_color: "black@0.80",
      text_color: "white",
      font_size: 20,
      entrance_animation: "fade_in",
      is_active: true
    };

    switch (type) {
      case "now_playing":
        newEl.text = "NOW PLAYING: MASTERPIECE";
        newEl.sub_text = "5.1 SURROUND SOUND • LIVE";
        newEl.x = 60;
        newEl.y = 60;
        newEl.width = 380;
        newEl.height = 85;
        break;
      case "up_next":
        newEl.text = "UP NEXT: SPECIAL FEATURE";
        newEl.sub_text = "STARTING IN 15 MINUTES";
        newEl.x = 1480;
        newEl.y = 60;
        newEl.width = 380;
        newEl.height = 85;
        break;
      case "promo":
        newEl.text = "EXCLUSIVE BROADCAST PREMIERE";
        newEl.sub_text = "DON'T MISS THE EVENT OF THE SEASON";
        newEl.x = 1460;
        newEl.y = 860;
        newEl.width = 400;
        newEl.height = 95;
        newEl.background_color = "purple@0.85";
        newEl.text_color = "gold";
        break;
      case "ticker":
        newEl.text = "LIVE CHANNEL NEWS TICKER • BREAKING UPDATES STREAMING NOW";
        newEl.x = 0;
        newEl.y = 1020;
        newEl.width = 1920;
        newEl.height = 60;
        newEl.background_color = "black@0.80";
        newEl.text_color = "yellow";
        newEl.entrance_animation = "scroll_left";
        break;
      case "header_banner":
        newEl.text = `${formData.name || 'CHANNEL'} BROADCAST FEED`;
        newEl.x = 0;
        newEl.y = 0;
        newEl.width = 1920;
        newEl.height = 55;
        newEl.background_color = "navy@0.85";
        break;
      case "footer_banner":
        newEl.text = "LIVE SATELLITE FEED • MCRFLOW TRANSMISSION";
        newEl.x = 0;
        newEl.y = 1025;
        newEl.width = 1920;
        newEl.height = 55;
        newEl.background_color = "darkblue@0.85";
        break;
      case "lower_third":
        newEl.text = "LIVE SPECIAL REPORT";
        newEl.sub_text = "NEWSROOM MASTER CONTROL";
        newEl.x = 80;
        newEl.y = 880;
        newEl.width = 600;
        newEl.height = 90;
        break;
    }

    setOverlays((prev) => [...prev, newEl]);
    setSelectedOverlayId(newId);
    onShowToast(`Added ${type.replace(/_/g, ' ')} graphics placeholder`, "info");
  };

  const handleRemoveOverlay = (id) => {
    setOverlays((prev) => prev.filter((o) => o.id !== id));
    if (selectedOverlayId === id) setSelectedOverlayId("logo");
  };

  const updateSelectedOverlay = (fields) => {
    setOverlays((prev) =>
      prev.map((o) => (o.id === selectedOverlayId ? { ...o, ...fields } : o))
    );
  };

  // Canvas Drag Handling
  const handleStartDrag = (e, targetId, startX, startY) => {
    e.stopPropagation();
    setSelectedOverlayId(targetId);
    setDraggingTarget(targetId);
    setDragStart({ x: e.clientX, y: e.clientY });
    setElementStartPos({ x: startX || 0, y: startY || 0 });
  };

  const handleCanvasMouseMove = (e) => {
    if (!draggingTarget || !monitorCanvasRef.current) return;
    const rect = monitorCanvasRef.current.getBoundingClientRect();
    const scaleX = 1920 / rect.width;
    const scaleY = 1080 / rect.height;

    const deltaX = (e.clientX - dragStart.x) * scaleX;
    const deltaY = (e.clientY - dragStart.y) * scaleY;

    if (draggingTarget === "logo") {
      const newX = Math.max(0, Math.min(1920 - (formData.logo_width || 140), Math.round(elementStartPos.x + deltaX)));
      const newY = Math.max(0, Math.min(1080 - (formData.logo_height || 90), Math.round(elementStartPos.y + deltaY)));
      setFormData((prev) => ({ ...prev, logo_x: newX, logo_y: newY }));
    } else {
      const ov = overlays.find((o) => o.id === draggingTarget);
      if (ov) {
        const newX = Math.max(0, Math.min(1920 - (ov.width || 300), Math.round(elementStartPos.x + deltaX)));
        const newY = Math.max(0, Math.min(1080 - (ov.height || 60), Math.round(elementStartPos.y + deltaY)));
        setOverlays((prev) =>
          prev.map((o) => (o.id === draggingTarget ? { ...o, x: newX, y: newY } : o))
        );
      }
    }
  };

  const handleCanvasMouseUp = () => {
    if (draggingTarget) setDraggingTarget(null);
  };

  const selectedOverlay = overlays.find((o) => o.id === selectedOverlayId);

  return (
    <div
      className="h-full flex flex-col p-3 sm:p-4 space-y-4 overflow-y-auto max-w-full select-none"
      onMouseMove={handleCanvasMouseMove}
      onMouseUp={handleCanvasMouseUp}
    >
      {/* Top Bar with Back Link, Channel Selector, Playout Controls and CRUD */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 shrink-0 bg-[#111827] border border-[#1F2937] p-3 rounded-lg shadow-sm">
        <div className="flex flex-wrap items-center gap-2 sm:gap-3">
          {/* Back to Dashboard Navigation Pill */}
          {onBackToDashboard && (
            <button
              onClick={onBackToDashboard}
              className="px-2.5 py-1.5 bg-gray-800 hover:bg-gray-700 text-sky-400 hover:text-sky-300 rounded text-xs font-semibold flex items-center gap-1.5 border border-gray-700 transition-colors"
              title="Return to Matrix Dashboard"
            >
              <ArrowLeft className="w-3.5 h-3.5" />
              <span>Dashboard</span>
            </button>
          )}

          <div className="flex items-center gap-2">
            <Tv className="w-5 h-5 text-indigo-400 shrink-0" />
            <div>
              <h2 className="text-sm font-bold text-white">
                {activeChannel.name || "Channel Master Control"}
              </h2>
              <p className="text-[11px] text-gray-400">
                Call Sign: <span className="text-sky-300 font-mono font-semibold">{formData.call_sign || "MCR"}</span>
                <span className="mx-1.5">•</span>
                Status:{" "}
                <span className={`font-mono font-semibold ${isPlayoutRunning ? "text-emerald-400" : "text-amber-400"}`}>
                  {isPlayoutRunning ? "ON-AIR" : "STANDBY"}
                </span>
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

        <div className="flex flex-wrap items-center gap-2 self-end sm:self-auto">
          {/* Playout Start / Stop Control */}
          {isPlayoutRunning ? (
            <button
              onClick={handleStopPlayout}
              className="px-3 py-1.5 bg-rose-950/60 hover:bg-rose-900/80 text-rose-300 border border-rose-500/40 text-xs font-semibold rounded flex items-center gap-1.5 transition-colors"
            >
              <Square className="w-3 h-3 fill-current" />
              <span>Stop Playout</span>
            </button>
          ) : (
            <button
              onClick={handleStartPlayout}
              className="px-3 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white text-xs font-semibold rounded flex items-center gap-1.5 shadow transition-colors"
            >
              <Play className="w-3.5 h-3.5 fill-current" />
              <span>Start Playout</span>
            </button>
          )}

          <button
            onClick={onCreateChannel}
            className="px-2.5 py-1.5 bg-gray-800 hover:bg-gray-700 text-gray-200 text-xs font-semibold rounded flex items-center gap-1 border border-gray-700 transition-colors"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>New</span>
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
              title="Delete channel"
            >
              <Trash2 className="w-3.5 h-3.5" />
            </button>
          )}
        </div>
      </div>

      {/* Main Grid: Parameters Left, Confidence Monitor & Overlay Studio Right */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-4 flex-1 min-h-0">
        {/* Left: Channel Parameters, Media Source & Destinations (6 cols) */}
        <div className="lg:col-span-6 bg-[#111827] border border-[#1F2937] rounded-lg p-3 sm:p-4 space-y-4 overflow-y-auto text-xs">
          {/* Quick Media Source Picker */}
          <div className="p-3 bg-[#161F30] rounded-lg border border-[#23314B] space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-[11px] font-bold text-sky-300 uppercase tracking-wider flex items-center gap-1.5">
                <Film className="w-3.5 h-3.5 text-indigo-400" />
                <span>Linear Playout Media Source</span>
              </span>
              <button
                type="button"
                onClick={openMediaPicker}
                className="text-[11px] px-2.5 py-1 bg-indigo-600 hover:bg-indigo-500 text-white rounded font-semibold flex items-center gap-1 transition-colors"
              >
                <FolderOpen className="w-3 h-3" />
                <span>Browse ./media</span>
              </button>
            </div>
            <div className="flex items-center justify-between bg-[#0B0F17] p-2 rounded border border-gray-800 font-mono text-xs">
              <span className="text-gray-300 truncate max-w-[280px]">
                {selectedMediaPath || "sample_movie.mp4"}
              </span>
              <span className="text-emerald-400 font-bold shrink-0 text-[10px]">
                ● Ready for Broadcast
              </span>
            </div>
          </div>

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

          {/* Ad & CG Template Association */}
          <div>
            <label className="block text-[11px] text-gray-400 mb-1 font-medium flex items-center justify-between">
              <span>Channel Ad & CG Graphics Template</span>
              <span className="text-[10px] text-sky-400 font-mono">Burn-in Overlays</span>
            </label>
            <select
              value={formData.ad_template_id}
              onChange={(e) => handleInputChange("ad_template_id", e.target.value)}
              className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white focus:outline-none focus:border-indigo-500"
            >
              <option value="">No Global Template (Custom Overlays Only)</option>
              {adTemplates.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name} ({t.template_type || "composite"})
                </option>
              ))}
            </select>
          </div>

          {/* Egress Destinations */}
          <div className="pt-2 border-t border-gray-800 space-y-3">
            <h3 className="text-xs font-bold text-gray-300 uppercase tracking-wider flex items-center gap-1.5">
              <Radio className="w-3.5 h-3.5 text-sky-400" />
              <span>Playout Egress Destinations</span>
            </h3>

            <div>
              <label className="block text-[11px] text-gray-400 mb-1 font-medium">UDP TS Multicast URL (DVB Mux Egress)</label>
              <input
                type="text"
                value={formData.udp_url}
                onChange={(e) => handleInputChange("udp_url", e.target.value)}
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white font-mono break-all focus:outline-none focus:border-indigo-500"
              />
            </div>

            <div>
              <label className="block text-[11px] text-gray-400 mb-1 font-medium">SRT Caller URL (Remote Transmitter)</label>
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

          {/* WebTokens & Live HLS Egress */}
          <div className="pt-2 border-t border-gray-800 space-y-3">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
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
                  placeholder="Security token"
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
                  placeholder="Query token"
                  className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white font-mono focus:outline-none focus:border-indigo-500"
                />
              </div>
            </div>

            <div>
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
        </div>

        {/* Right: Confidence Monitor & Visual Layout Graphics Studio (6 cols) */}
        <div className="lg:col-span-6 flex flex-col space-y-4">
          <div className="bg-[#111827] border border-[#1F2937] rounded-lg p-3 space-y-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <span className={`w-2.5 h-2.5 rounded-full ${isPlayoutRunning ? "bg-emerald-400 animate-pulse" : "bg-gray-500"}`}></span>
                <h3 className="text-xs font-bold text-white uppercase tracking-wider">
                  Live Confidence Monitor
                </h3>
              </div>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setIsEditLayoutMode(!isEditLayoutMode)}
                  className={`px-2.5 py-1 rounded text-[11px] font-semibold flex items-center gap-1 transition-all ${
                    isEditLayoutMode
                      ? "bg-sky-500 text-black shadow-lg shadow-sky-500/20"
                      : "bg-gray-800 hover:bg-gray-700 text-gray-300 border border-gray-700"
                  }`}
                >
                  <Move className="w-3 h-3" />
                  <span>{isEditLayoutMode ? "Exit Layout Mode" : "Drag & Resize Layout"}</span>
                </button>
              </div>
            </div>

            {/* Confidence Monitor Container with Interactive 16:9 Canvas */}
            <div
              ref={monitorCanvasRef}
              className={`relative aspect-video bg-black rounded-lg overflow-hidden border border-gray-800 shadow-2xl select-none ${
                isEditLayoutMode ? "ring-2 ring-sky-500/80 cursor-crosshair" : ""
              }`}
            >
              {/* Native Live Video Stream */}
              <VideoPlayer
                streamUrl={resolvedHlsUrl}
                isSlate={isSlateActive}
                channelName={activeChannel.name}
                logoPath={formData.logo_path}
                logoPosition={formData.logo_position}
              />

              {/* Station Logo / Channel Bug Layer */}
              {formData.logo_path && !isSlateActive && (
                <div
                  onMouseDown={(e) => handleStartDrag(e, "logo", formData.logo_x, formData.logo_y)}
                  style={{
                    position: 'absolute',
                    left: `${((formData.logo_x || 1720) / 1920) * 100}%`,
                    top: `${((formData.logo_y || 40) / 1080) * 100}%`,
                    width: `${((formData.logo_width || 140) / 1920) * 100}%`,
                    height: `${((formData.logo_height || 90) / 1080) * 100}%`,
                    opacity: formData.logo_opacity || 0.9,
                    zIndex: selectedOverlayId === "logo" ? 40 : 25
                  }}
                  className={`transition-all flex items-center justify-center ${
                    isEditLayoutMode
                      ? "cursor-move ring-2 ring-indigo-400 bg-indigo-950/40"
                      : "pointer-events-none"
                  }`}
                >
                  <img
                    src={formData.logo_path.startsWith('/') || formData.logo_path.startsWith('http') ? formData.logo_path : `/${formData.logo_path}`}
                    alt="Logo"
                    style={{
                      objectFit: formData.logo_fit === 'cover' ? 'cover' : 'contain'
                    }}
                    className="w-full h-full drop-shadow"
                    onError={(e) => {
                      e.currentTarget.style.display = 'none';
                    }}
                  />
                  {isEditLayoutMode && selectedOverlayId === "logo" && (
                    <div className="absolute -top-4 left-0 bg-indigo-500 text-black text-[8px] font-black px-1 rounded shadow">
                      LOGO BUG ({formData.logo_x},{formData.logo_y})
                    </div>
                  )}
                </div>
              )}

              {/* Broadcast Overlays Layer (Now Playing, Up Next, Special Promo, Ticker, Banners) */}
              {!isSlateActive && overlays.map((ov) => {
                if (!ov.is_active && !isEditLayoutMode) return null;
                const isSelected = selectedOverlayId === ov.id;

                const leftPct = `${((ov.x || 0) / 1920) * 100}%`;
                const topPct = `${((ov.y || 0) / 1080) * 100}%`;
                const widthPct = ov.width ? `${(ov.width / 1920) * 100}%` : 'auto';
                const heightPct = ov.height ? `${(ov.height / 1080) * 100}%` : 'auto';

                return (
                  <div
                    key={ov.id}
                    onMouseDown={(e) => handleStartDrag(e, ov.id, ov.x, ov.y)}
                    style={{
                      position: 'absolute',
                      left: leftPct,
                      top: topPct,
                      width: widthPct,
                      height: heightPct,
                      zIndex: isSelected ? 40 : 20
                    }}
                    className={`flex items-center px-2.5 transition-all select-none ${
                      isEditLayoutMode ? "cursor-move" : "pointer-events-none"
                    } ${
                      isSelected
                        ? "ring-2 ring-sky-400 bg-black/80 shadow-lg shadow-sky-500/30"
                        : "bg-black/70 hover:ring-1 hover:ring-gray-400"
                    }`}
                  >
                    {ov.type === 'now_playing' ? (
                      <div className="flex flex-col justify-center leading-tight">
                        <span className="text-[8px] font-black text-yellow-400 uppercase tracking-wider">NOW PLAYING</span>
                        <span className="text-[11px] font-bold text-white truncate">{ov.text}</span>
                        {ov.sub_text && <span className="text-[8px] text-gray-300 truncate">{ov.sub_text}</span>}
                      </div>
                    ) : ov.type === 'up_next' ? (
                      <div className="flex flex-col justify-center leading-tight">
                        <span className="text-[8px] font-black text-orange-400 uppercase tracking-wider">UP NEXT</span>
                        <span className="text-[11px] font-bold text-white truncate">{ov.text}</span>
                        {ov.sub_text && <span className="text-[8px] text-gray-300 truncate">{ov.sub_text}</span>}
                      </div>
                    ) : ov.type === 'promo' ? (
                      <div className="flex flex-col justify-center leading-tight">
                        <span className="text-[8px] font-black text-amber-300 uppercase tracking-wider">SPECIAL PROMO</span>
                        <span className="text-[11px] font-bold text-white truncate">{ov.text}</span>
                        {ov.sub_text && <span className="text-[8px] text-purple-200 truncate">{ov.sub_text}</span>}
                      </div>
                    ) : ov.type === 'ticker' ? (
                      <div className="w-full flex items-center overflow-hidden">
                        <span className="bg-red-700 text-white font-black text-[8px] px-1 rounded mr-1.5 shrink-0">NEWS</span>
                        <span className="font-mono text-[10px] text-yellow-300 truncate">{ov.text}</span>
                      </div>
                    ) : (
                      <div className="flex flex-col justify-center leading-tight">
                        <span className="text-[11px] font-bold text-white truncate">{ov.text}</span>
                        {ov.sub_text && <span className="text-[8px] text-gray-300 truncate">{ov.sub_text}</span>}
                      </div>
                    )}

                    {isEditLayoutMode && isSelected && (
                      <div className="absolute -top-3.5 right-0 bg-sky-500 text-black text-[8px] font-black px-1 rounded shadow">
                        {ov.x},{ov.y}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>

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

          {/* Graphics & Logo Positioning / Resizing Control Panel */}
          <div className="bg-[#111827] border border-[#1F2937] rounded-lg p-3 space-y-3 text-xs">
            <div className="flex items-center justify-between">
              <h4 className="text-xs font-bold text-sky-400 uppercase tracking-wider flex items-center gap-1.5">
                <Sliders className="w-3.5 h-3.5" />
                <span>On-Air Graphics & Logo Inspector</span>
              </h4>
              <div className="flex items-center gap-1">
                <button
                  type="button"
                  onClick={() => setSelectedOverlayId("logo")}
                  className={`px-2 py-0.5 rounded text-[10px] font-semibold ${
                    selectedOverlayId === "logo" ? "bg-indigo-600 text-white" : "bg-gray-800 text-gray-300"
                  }`}
                >
                  Logo Bug
                </button>
                {overlays.map((ov) => (
                  <button
                    key={ov.id}
                    type="button"
                    onClick={() => setSelectedOverlayId(ov.id)}
                    className={`px-2 py-0.5 rounded text-[10px] font-semibold ${
                      selectedOverlayId === ov.id ? "bg-sky-600 text-white" : "bg-gray-800 text-gray-300"
                    }`}
                  >
                    {ov.type.replace(/_/g, ' ')}
                  </button>
                ))}
              </div>
            </div>

            {/* LOGO PROPERTIES */}
            {selectedOverlayId === "logo" ? (
              <div className="space-y-2.5 bg-[#161F30] p-3 rounded border border-gray-800">
                <div className="flex items-center justify-between">
                  <span className="font-bold text-white text-[11px]">Station Logo Positioning & Resizing</span>
                  <button
                    type="button"
                    onClick={() => logoFileInputRef.current?.click()}
                    disabled={isUploadingLogo}
                    className="px-2 py-0.5 bg-gray-700 hover:bg-gray-600 text-white rounded text-[10px] font-semibold"
                  >
                    {isUploadingLogo ? "Uploading..." : "Upload New Logo"}
                  </button>
                  <input
                    type="file"
                    ref={logoFileInputRef}
                    onChange={handleLogoUpload}
                    accept="image/png,image/jpeg,image/webp,image/svg+xml"
                    className="hidden"
                  />
                </div>

                <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 font-mono">
                  <div>
                    <label className="text-[10px] text-gray-400 block">X Pos</label>
                    <input
                      type="number"
                      value={formData.logo_x}
                      onChange={(e) => handleInputChange("logo_x", parseInt(e.target.value, 10) || 0)}
                      className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1 text-xs text-white"
                    />
                  </div>
                  <div>
                    <label className="text-[10px] text-gray-400 block">Y Pos</label>
                    <input
                      type="number"
                      value={formData.logo_y}
                      onChange={(e) => handleInputChange("logo_y", parseInt(e.target.value, 10) || 0)}
                      className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1 text-xs text-white"
                    />
                  </div>
                  <div>
                    <label className="text-[10px] text-gray-400 block">Width</label>
                    <input
                      type="number"
                      value={formData.logo_width}
                      onChange={(e) => handleInputChange("logo_width", parseInt(e.target.value, 10) || 80)}
                      className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1 text-xs text-white"
                    />
                  </div>
                  <div>
                    <label className="text-[10px] text-gray-400 block">Height</label>
                    <input
                      type="number"
                      value={formData.logo_height}
                      onChange={(e) => handleInputChange("logo_height", parseInt(e.target.value, 10) || 60)}
                      className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1 text-xs text-white"
                    />
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-3 pt-1">
                  <div>
                    <label className="text-[10px] text-gray-400 block mb-1">
                      Opacity: {Math.round((formData.logo_opacity || 0.9) * 100)}%
                    </label>
                    <input
                      type="range"
                      min="0.1"
                      max="1.0"
                      step="0.05"
                      value={formData.logo_opacity || 0.9}
                      onChange={(e) => handleInputChange("logo_opacity", parseFloat(e.target.value))}
                      className="w-full"
                    />
                  </div>
                  <div>
                    <label className="text-[10px] text-gray-400 block mb-1">Fit & Crop Mode</label>
                    <select
                      value={formData.logo_fit || "contain"}
                      onChange={(e) => handleInputChange("logo_fit", e.target.value)}
                      className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1 text-xs text-white"
                    >
                      <option value="contain">Contain (Preserve Aspect)</option>
                      <option value="cover">Cover (Fill & Crop)</option>
                    </select>
                  </div>
                </div>
              </div>
            ) : selectedOverlay ? (
              /* SELECTED OVERLAY ELEMENT PROPERTIES */
              <div className="space-y-2.5 bg-[#161F30] p-3 rounded border border-gray-800">
                <div className="flex items-center justify-between">
                  <span className="font-bold text-white text-[11px] capitalize">
                    {selectedOverlay.type.replace(/_/g, ' ')} Placeholder
                  </span>
                  <div className="flex items-center gap-2">
                    <label className="flex items-center gap-1 text-[10px] text-emerald-400 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={selectedOverlay.is_active}
                        onChange={(e) => updateSelectedOverlay({ is_active: e.target.checked })}
                      />
                      <span>On-Air</span>
                    </label>
                    <button
                      type="button"
                      onClick={() => handleRemoveOverlay(selectedOverlay.id)}
                      className="text-rose-400 hover:text-rose-300 text-[10px] font-bold"
                    >
                      Delete
                    </button>
                  </div>
                </div>

                <div>
                  <label className="text-[10px] text-gray-400 block mb-0.5">Primary Headline Text</label>
                  <input
                    type="text"
                    value={selectedOverlay.text || ""}
                    onChange={(e) => updateSelectedOverlay({ text: e.target.value })}
                    className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1 text-xs text-white"
                  />
                </div>

                {selectedOverlay.type !== 'ticker' && (
                  <div>
                    <label className="text-[10px] text-gray-400 block mb-0.5">Subtext / Secondary Line</label>
                    <input
                      type="text"
                      value={selectedOverlay.sub_text || ""}
                      onChange={(e) => updateSelectedOverlay({ sub_text: e.target.value })}
                      className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1 text-xs text-white"
                    />
                  </div>
                )}

                <div className="grid grid-cols-4 gap-2 font-mono">
                  <div>
                    <label className="text-[9px] text-gray-400 block">X</label>
                    <input
                      type="number"
                      value={selectedOverlay.x || 0}
                      onChange={(e) => updateSelectedOverlay({ x: parseInt(e.target.value, 10) || 0 })}
                      className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1 text-xs text-white"
                    />
                  </div>
                  <div>
                    <label className="text-[9px] text-gray-400 block">Y</label>
                    <input
                      type="number"
                      value={selectedOverlay.y || 0}
                      onChange={(e) => updateSelectedOverlay({ y: parseInt(e.target.value, 10) || 0 })}
                      className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1 text-xs text-white"
                    />
                  </div>
                  <div>
                    <label className="text-[9px] text-gray-400 block">W</label>
                    <input
                      type="number"
                      value={selectedOverlay.width || 380}
                      onChange={(e) => updateSelectedOverlay({ width: parseInt(e.target.value, 10) || 100 })}
                      className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1 text-xs text-white"
                    />
                  </div>
                  <div>
                    <label className="text-[9px] text-gray-400 block">H</label>
                    <input
                      type="number"
                      value={selectedOverlay.height || 85}
                      onChange={(e) => updateSelectedOverlay({ height: parseInt(e.target.value, 10) || 50 })}
                      className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1 text-xs text-white"
                    />
                  </div>
                </div>
              </div>
            ) : null}

            {/* Quick Add Broadcast Placeholders Bar */}
            <div className="pt-2 border-t border-gray-800 flex flex-wrap items-center gap-1.5">
              <span className="text-[10px] text-gray-400 font-bold uppercase mr-1">Add Graphic:</span>
              <button
                type="button"
                onClick={() => handleAddOverlay("now_playing")}
                className="px-2 py-1 bg-[#1F2937] hover:bg-gray-700 text-yellow-300 rounded text-[10px] font-semibold border border-gray-700"
              >
                + Now Playing
              </button>
              <button
                type="button"
                onClick={() => handleAddOverlay("up_next")}
                className="px-2 py-1 bg-[#1F2937] hover:bg-gray-700 text-orange-300 rounded text-[10px] font-semibold border border-gray-700"
              >
                + Up Next
              </button>
              <button
                type="button"
                onClick={() => handleAddOverlay("promo")}
                className="px-2 py-1 bg-[#1F2937] hover:bg-gray-700 text-purple-300 rounded text-[10px] font-semibold border border-gray-700"
              >
                + Special Promo
              </button>
              <button
                type="button"
                onClick={() => handleAddOverlay("ticker")}
                className="px-2 py-1 bg-[#1F2937] hover:bg-gray-700 text-amber-300 rounded text-[10px] font-semibold border border-gray-700"
              >
                + News Ticker
              </button>
              <button
                type="button"
                onClick={() => handleAddOverlay("header_banner")}
                className="px-2 py-1 bg-[#1F2937] hover:bg-gray-700 text-sky-300 rounded text-[10px] font-semibold border border-gray-700"
              >
                + Header Banner
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* QUICK MEDIA PICKER MODAL */}
      {isMediaPickerOpen && (
        <div className="fixed inset-0 bg-black/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-[#111827] border border-[#2D3A54] rounded-xl w-full max-w-xl max-h-[85vh] flex flex-col shadow-2xl overflow-hidden text-xs">
            <div className="px-5 py-3.5 bg-[#1A2234] border-b border-[#2D3A54] flex items-center justify-between">
              <h3 className="text-sm font-bold text-white flex items-center gap-2">
                <FolderOpen className="w-4 h-4 text-indigo-400" />
                <span>Select Playout Media Source (./media{mediaCurrentPath ? `/${mediaCurrentPath}` : ''})</span>
              </h3>
              <button onClick={() => setIsMediaPickerOpen(false)} className="text-gray-400 hover:text-white">
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="p-4 space-y-3 overflow-y-auto">
              <div className="relative">
                <Search className="w-3.5 h-3.5 text-gray-400 absolute left-2.5 top-2.5" />
                <input
                  type="text"
                  value={mediaFilterQuery}
                  onChange={(e) => setMediaFilterQuery(e.target.value)}
                  placeholder="Filter media files..."
                  className="w-full bg-[#0B0F17] border border-gray-700 rounded pl-8 pr-2.5 py-1.5 text-xs text-white"
                />
              </div>

              {mediaCurrentPath && (
                <button
                  onClick={() => {
                    const parent = mediaCurrentPath.includes('/') ? mediaCurrentPath.substring(0, mediaCurrentPath.lastIndexOf('/')) : '';
                    handleMediaBrowsePath(parent);
                  }}
                  className="text-[10px] text-indigo-400 hover:text-indigo-300 font-mono"
                >
                  ← Up one directory level
                </button>
              )}

              <div className="bg-[#0B0F17] border border-gray-800 rounded-lg max-h-60 overflow-y-auto divide-y divide-gray-800">
                {mediaList
                  .filter((m) => !mediaFilterQuery || m.name.toLowerCase().includes(mediaFilterQuery.toLowerCase()))
                  .map((m, i) => (
                    <div
                      key={i}
                      onClick={() => {
                        if (m.is_dir) {
                          handleMediaBrowsePath(m.path);
                        } else {
                          handleSelectMediaFile(m);
                        }
                      }}
                      className="p-2.5 flex items-center justify-between hover:bg-[#1E293B] cursor-pointer transition-colors"
                    >
                      <div className="flex items-center gap-2 truncate">
                        {m.is_dir ? (
                          <FolderOpen className="w-4 h-4 text-amber-400 shrink-0" />
                        ) : (
                          <FileVideo className="w-4 h-4 text-sky-400 shrink-0" />
                        )}
                        <span className={`font-mono truncate ${m.is_dir ? "text-amber-200 font-bold" : "text-gray-200"}`}>
                          {m.name}
                        </span>
                      </div>
                      {!m.is_dir && (
                        <span className="text-[10px] text-emerald-400 font-mono shrink-0 ml-2">
                          {m.duration_seconds ? `${Math.floor(m.duration_seconds / 60)}m` : "Ready"}
                        </span>
                      )}
                    </div>
                  ))}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
