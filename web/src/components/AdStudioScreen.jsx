import React, { useState, useEffect, useRef } from 'react';
import {
  Sparkles,
  Play,
  Layers,
  Clock,
  Eye,
  Sliders,
  Radio,
  Check,
  Plus,
  Trash2,
  Save,
  Move,
  Type,
  Palette,
  Shield,
  Layout,
  RefreshCw,
  Upload,
  AlertTriangle,
  Maximize2,
  Image as ImageIcon,
  Tv,
  ArrowRight
} from 'lucide-react';
import { api } from '../api';

const DEFAULT_OVERLAYS = [
  {
    id: "ov-hdr-1",
    type: "header_banner",
    text: "MCRFLOW LIVE BROADCAST NETWORK",
    sub_text: "24/7 AUTOMATED LINEAR PLAYOUT",
    x: 0,
    y: 0,
    width: 1920,
    height: 60,
    background_color: "navy@0.85",
    text_color: "white",
    font_size: 22,
    entrance_animation: "fade_in",
    is_active: true
  },
  {
    id: "ov-np-1",
    type: "now_playing",
    text: "NOW PLAYING: PATHAAN (2023) 4K",
    sub_text: "5.1 DOLBY ATMOS • ACTION",
    x: 60,
    y: 80,
    width: 380,
    height: 85,
    background_color: "black@0.80",
    text_color: "white",
    font_size: 20,
    entrance_animation: "slide_in_left",
    is_active: true
  },
  {
    id: "ov-un-1",
    type: "up_next",
    text: "UP NEXT: JAWAN (EXTENDED CUT)",
    sub_text: "TONIGHT @ 21:00 IST",
    x: 1480,
    y: 80,
    width: 380,
    height: 85,
    background_color: "black@0.80",
    text_color: "white",
    font_size: 20,
    entrance_animation: "fade_in",
    is_active: true
  },
  {
    id: "ov-prm-1",
    type: "promo",
    text: "DIWALI MOVIE FESTIVAL PREMIERE",
    sub_text: "EXCLUSIVE BROADCAST THIS WEEKEND",
    x: 1460,
    y: 840,
    width: 400,
    height: 95,
    background_color: "purple@0.85",
    text_color: "gold",
    font_size: 18,
    entrance_animation: "zoom_in",
    is_active: true
  },
  {
    id: "ov-tck-1",
    type: "ticker",
    text: "BREAKING: MCRFLOW MASTER CONTROL LAUNCHES NEXT-GEN CLOUD PLAYOUT • ZERO-FRAME SWITCHING • EBU R128 AUDIO NORMALIZATION ACTIVE",
    sub_text: "",
    x: 0,
    y: 1020,
    width: 1920,
    height: 60,
    background_color: "black@0.80",
    text_color: "yellow",
    font_size: 24,
    entrance_animation: "scroll_left",
    is_active: true
  }
];

export function AdStudioScreen({
  adTemplates = [],
  onRefreshTemplates,
  onShowToast,
  t
}) {
  const [activeTab, setActiveTab] = useState("logo"); // "logo", "now_playing", "ads", "general"

  const [activeTemplateId, setActiveTemplateId] = useState("");
  const [templateName, setTemplateName] = useState("");
  const [templateType, setTemplateType] = useState("composite");
  const [overlayElements, setOverlayElements] = useState(DEFAULT_OVERLAYS);
  const [commercialBreaks, setCommercialBreaks] = useState([]);
  const [selectedElementId, setSelectedElementId] = useState("ov-np-1");

  // Logo & Branding state
  const [logoPath, setLogoPath] = useState("data/logos/channel_logo.png");
  const [logoPosition, setLogoPosition] = useState("top-right");
  const [logoX, setLogoX] = useState(1720);
  const [logoY, setLogoY] = useState(40);
  const [logoWidth, setLogoWidth] = useState(140);
  const [logoHeight, setLogoHeight] = useState(90);
  const [logoOpacity, setLogoOpacity] = useState(0.90);
  const [logoFit, setLogoFit] = useState("contain");
  const [isUploadingLogo, setIsUploadingLogo] = useState(false);
  const logoInputRef = useRef(null);

  // Collision Resolution state
  const [collisionBehavior, setCollisionBehavior] = useState("alternate"); // "alternate", "priority"
  const [alternateDuration, setAlternateDuration] = useState(15);
  const [priorityOrder, setPriorityOrder] = useState(["header_banner", "now_playing", "up_next", "promo", "ticker"]);

  const [showSafeGuides, setShowSafeGuides] = useState(true);
  const [animKey, setAnimKey] = useState(0);
  const [isSaving, setIsSaving] = useState(false);

  // Interactive Canvas Drag & Resize State
  const canvasRef = useRef(null);
  const [interactionMode, setInteractionMode] = useState(null); // "drag" or "resize"
  const [activeTargetId, setActiveTargetId] = useState(null); // element id or "logo"
  const [interactionStart, setInteractionStart] = useState({ clientX: 0, clientY: 0 });
  const [initialBox, setInitialBox] = useState({ x: 0, y: 0, width: 0, height: 0 });

  // Sync when adTemplates changes
  useEffect(() => {
    if (adTemplates.length > 0) {
      const match = adTemplates.find((t) => t.id === activeTemplateId) || adTemplates[0];
      if (match) {
        applyTemplateData(match);
      }
    }
  }, [adTemplates, activeTemplateId]);

  const applyTemplateData = (tmpl) => {
    setActiveTemplateId(tmpl.id);
    setTemplateName(tmpl.name || "Default Template");
    setTemplateType(tmpl.template_type || "composite");

    setLogoPath(tmpl.logo_path || "data/logos/channel_logo.png");
    setLogoPosition(tmpl.logo_position || "top-right");
    setLogoX(tmpl.logo_x || 1720);
    setLogoY(tmpl.logo_y || 40);
    setLogoWidth(tmpl.logo_width || 140);
    setLogoHeight(tmpl.logo_height || 90);
    setLogoOpacity(tmpl.logo_opacity || 0.90);
    setLogoFit(tmpl.logo_fit || "contain");

    setCollisionBehavior(tmpl.collision_behavior || "alternate");
    setAlternateDuration(tmpl.alternate_duration_seconds || 15);
    if (Array.isArray(tmpl.priority_order) && tmpl.priority_order.length > 0) {
      setPriorityOrder(tmpl.priority_order);
    }

    const elems = tmpl.overlay_elements && tmpl.overlay_elements.length > 0
      ? tmpl.overlay_elements
      : DEFAULT_OVERLAYS;
    setOverlayElements(elems);
    setCommercialBreaks(tmpl.commercial_breaks || []);
    if (elems.length > 0 && !selectedElementId) {
      setSelectedElementId(elems[0].id);
    }
  };

  const handleSelectTemplate = (id) => {
    setActiveTemplateId(id);
    const tmpl = adTemplates.find((t) => t.id === id);
    if (tmpl) applyTemplateData(tmpl);
  };

  const handleCreateNewTemplate = () => {
    const newId = `tmpl-${Date.now().toString(36)}`;
    setActiveTemplateId(newId);
    setTemplateName("New Broadcast Ad & Layout Template");
    setTemplateType("composite");
    setLogoPath("data/logos/channel_logo.png");
    setLogoPosition("top-right");
    setLogoX(1720);
    setLogoY(40);
    setLogoWidth(140);
    setLogoHeight(90);
    setLogoOpacity(0.90);
    setLogoFit("contain");
    setCollisionBehavior("alternate");
    setAlternateDuration(15);
    setOverlayElements(JSON.parse(JSON.stringify(DEFAULT_OVERLAYS)));
    setCommercialBreaks([
      { break_type: "mid_roll", offset_seconds: 1800, duration_seconds: 60, scte35_cue: true, clips: [] }
    ]);
    setSelectedElementId(DEFAULT_OVERLAYS[0].id);
    onShowToast("Initialized new on-air layout template", "info");
  };

  const handleSaveTemplate = async () => {
    if (!templateName.trim()) {
      onShowToast("Template name is required", "error");
      return;
    }
    setIsSaving(true);
    try {
      const payload = {
        name: templateName.trim(),
        template_type: templateType,
        logo_path: logoPath,
        logo_position: logoPosition,
        logo_x: Number(logoX) || 0,
        logo_y: Number(logoY) || 0,
        logo_width: Number(logoWidth) || 140,
        logo_height: Number(logoHeight) || 90,
        logo_opacity: Number(logoOpacity) || 0.90,
        logo_fit: logoFit,
        collision_behavior: collisionBehavior,
        alternate_duration_seconds: Number(alternateDuration) || 15,
        priority_order: priorityOrder,
        overlay_elements: overlayElements,
        commercial_breaks: commercialBreaks,
        is_active: true
      };

      const existing = adTemplates.find((t) => t.id === activeTemplateId);
      if (existing) {
        await api.updateAdTemplate(activeTemplateId, { ...existing, ...payload });
        onShowToast(`Updated template "${templateName}" successfully!`, "success");
      } else {
        const created = await api.createAdTemplate(payload);
        if (created?.id) setActiveTemplateId(created.id);
        onShowToast(`Created template "${templateName}" successfully!`, "success");
      }
      if (onRefreshTemplates) onRefreshTemplates();
    } catch (err) {
      onShowToast("Failed to save template: " + err.message, "error");
    } finally {
      setIsSaving(false);
    }
  };

  const handleDeleteTemplate = async () => {
    if (!activeTemplateId) return;
    try {
      await api.deleteAdTemplate(activeTemplateId);
      onShowToast("Deleted layout template", "info");
      setActiveTemplateId("");
      if (onRefreshTemplates) onRefreshTemplates();
    } catch (err) {
      onShowToast("Delete failed: " + err.message, "error");
    }
  };

  // Logo file upload handler
  const handleUploadLogoFile = async (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setIsUploadingLogo(true);
    try {
      const res = await api.uploadLogo(file);
      const newPath = res.logo_path || "data/logos/channel_logo.png";
      setLogoPath(newPath);
      onShowToast("Station watermark logo uploaded successfully!", "success");
    } catch (err) {
      onShowToast("Failed to upload logo: " + err.message, "error");
    } finally {
      setIsUploadingLogo(false);
      if (logoInputRef.current) logoInputRef.current.value = "";
    }
  };

  // Overlay Elements Manipulation
  const selectedElement = overlayElements.find((el) => el.id === selectedElementId) || overlayElements[0];

  const updateSelectedElement = (fields) => {
    if (!selectedElement) return;
    setOverlayElements((prev) =>
      prev.map((el) => (el.id === selectedElement.id ? { ...el, ...fields } : el))
    );
  };

  const handleAddElement = (type) => {
    const newId = `ov-${Date.now().toString(36)}`;
    let newEl = {
      id: newId,
      type: type,
      text: "NEW BROADCAST OVERLAY",
      sub_text: "",
      x: 100,
      y: 100,
      width: 400,
      height: 90,
      background_color: "black@0.80",
      text_color: "white",
      font_size: 22,
      entrance_animation: "fade_in",
      is_active: true
    };

    switch (type) {
      case "header_banner":
        newEl.text = "NETWORK LIVE SPECIAL";
        newEl.x = 0;
        newEl.y = 0;
        newEl.width = 1920;
        newEl.height = 60;
        newEl.background_color = "navy@0.85";
        break;
      case "footer_banner":
        newEl.text = "NEXT-GEN TV BROADCAST";
        newEl.x = 0;
        newEl.y = 1010;
        newEl.width = 1920;
        newEl.height = 70;
        newEl.background_color = "darkblue@0.85";
        break;
      case "ticker":
        newEl.text = "BREAKING NEWS TICKER • LIVE SPORTS SCORES • WEATHER UPDATES • MARKET WATCH";
        newEl.x = 0;
        newEl.y = 1020;
        newEl.width = 1920;
        newEl.height = 60;
        newEl.entrance_animation = "scroll_left";
        newEl.background_color = "black@0.80";
        newEl.text_color = "yellow";
        break;
      case "now_playing":
        newEl.text = "NOW PLAYING: BLOCKBUSTER 4K";
        newEl.sub_text = "5.1 DOLBY SURROUND";
        newEl.x = 60;
        newEl.y = 80;
        newEl.width = 380;
        newEl.height = 85;
        break;
      case "up_next":
        newEl.text = "UP NEXT: ACTION EXTRAVAGANZA";
        newEl.sub_text = "COMING UP @ 22:00 IST";
        newEl.x = 1480;
        newEl.y = 80;
        newEl.width = 380;
        newEl.height = 85;
        break;
      case "promo":
        newEl.text = "SPECIAL BROADCAST PROMO";
        newEl.sub_text = "SUBSCRIBE TO PREMIUM HD";
        newEl.x = 1460;
        newEl.y = 840;
        newEl.width = 400;
        newEl.height = 95;
        newEl.background_color = "purple@0.85";
        newEl.text_color = "gold";
        break;
      case "lower_third":
        newEl.text = "GUEST SPEAKER: CHIEF BROADCAST ENGINEER";
        newEl.sub_text = "LIVE TRANSMITTER MONITORING";
        newEl.x = 80;
        newEl.y = 880;
        newEl.width = 650;
        newEl.height = 90;
        break;
    }

    setOverlayElements((prev) => [...prev, newEl]);
    setSelectedElementId(newId);
    onShowToast(`Added ${type.replace(/_/g, ' ')} overlay`, "info");
  };

  const handleRemoveElement = (id) => {
    setOverlayElements((prev) => prev.filter((el) => el.id !== id));
    if (selectedElementId === id) {
      const remaining = overlayElements.filter((el) => el.id !== id);
      setSelectedElementId(remaining[0]?.id || "");
    }
  };

  // Robust Drag and Resize Handlers using Global Window Listeners
  const startDrag = (e, targetId, box) => {
    e.stopPropagation();
    setInteractionMode("drag");
    setActiveTargetId(targetId);
    if (targetId !== "logo") setSelectedElementId(targetId);
    setInteractionStart({ clientX: e.clientX, clientY: e.clientY });
    setInitialBox({ ...box });
  };

  const startResize = (e, targetId, box) => {
    e.stopPropagation();
    setInteractionMode("resize");
    setActiveTargetId(targetId);
    if (targetId !== "logo") setSelectedElementId(targetId);
    setInteractionStart({ clientX: e.clientX, clientY: e.clientY });
    setInitialBox({ ...box });
  };

  useEffect(() => {
    if (!interactionMode) return;

    const handleWindowMouseMove = (e) => {
      if (!canvasRef.current) return;
      const rect = canvasRef.current.getBoundingClientRect();
      const scaleX = 1920 / rect.width;
      const scaleY = 1080 / rect.height;

      const deltaX = (e.clientX - interactionStart.clientX) * scaleX;
      const deltaY = (e.clientY - interactionStart.clientY) * scaleY;

      if (interactionMode === "drag") {
        const newX = Math.max(0, Math.min(1920 - initialBox.width, Math.round(initialBox.x + deltaX)));
        const newY = Math.max(0, Math.min(1080 - initialBox.height, Math.round(initialBox.y + deltaY)));

        if (activeTargetId === "logo") {
          setLogoX(newX);
          setLogoY(newY);
        } else {
          setOverlayElements((prev) =>
            prev.map((el) => (el.id === activeTargetId ? { ...el, x: newX, y: newY } : el))
          );
        }
      } else if (interactionMode === "resize") {
        const newW = Math.max(80, Math.min(1920 - initialBox.x, Math.round(initialBox.width + deltaX)));
        const newH = Math.max(30, Math.min(1080 - initialBox.y, Math.round(initialBox.height + deltaY)));

        if (activeTargetId === "logo") {
          setLogoWidth(newW);
          setLogoHeight(newH);
        } else {
          setOverlayElements((prev) =>
            prev.map((el) => (el.id === activeTargetId ? { ...el, width: newW, height: newH } : el))
          );
        }
      }
    };

    const handleWindowMouseUp = () => {
      setInteractionMode(null);
      setActiveTargetId(null);
    };

    window.addEventListener("mousemove", handleWindowMouseMove);
    window.addEventListener("mouseup", handleWindowMouseUp);
    return () => {
      window.removeEventListener("mousemove", handleWindowMouseMove);
      window.removeEventListener("mouseup", handleWindowMouseUp);
    };
  }, [interactionMode, activeTargetId, interactionStart, initialBox]);

  // Real-time Collision Detection
  const collisions = [];
  const activeElements = overlayElements.filter((el) => el.is_active);
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
        collisions.push({ a, b });
      }
    }
  }

  const isElementColliding = (id) => collisions.some((c) => c.a.id === id || c.b.id === id);

  const handlePlayAnimation = () => {
    setAnimKey((prev) => prev + 1);
    onShowToast("Previewing graphic entrance animations", "info");
  };

  const getElementAnimationClass = (anim) => {
    switch (anim) {
      case 'slide_in_left':
        return 'animate-in slide-in-from-left duration-500';
      case 'slide_in_bottom':
        return 'animate-in slide-in-from-bottom duration-500';
      case 'zoom_in':
        return 'animate-in zoom-in duration-300';
      default:
        return 'animate-in fade-in duration-300';
    }
  };

  return (
    <div className="space-y-6 pb-12 animate-in fade-in duration-300">
      {/* Top Header & Actions Bar */}
      <div className="bg-[#111622] border border-gray-800 rounded-xl p-5 shadow-lg flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <Sparkles className="w-5 h-5 text-purple-400" />
            <h1 className="text-xl font-bold text-white tracking-wide">Ad & Layout Management</h1>
            <span className="px-2 py-0.5 rounded text-[11px] font-mono bg-purple-950/70 border border-purple-700/50 text-purple-300">
              WYSIWYG CG Studio
            </span>
          </div>
          <p className="text-xs text-gray-400 mt-0.5">
            Branding watermark, Now Playing / Up Next graphics, commercial insertions, and collision resolution
          </p>
        </div>

        {/* Template Controls */}
        <div className="flex flex-wrap items-center gap-2">
          <select
            value={activeTemplateId}
            onChange={(e) => handleSelectTemplate(e.target.value)}
            className="bg-[#182030] border border-gray-700 text-white rounded-lg px-3 py-2 text-sm font-medium focus:ring-2 focus:ring-purple-500 focus:outline-none"
          >
            {adTemplates.map((t) => (
              <option key={t.id} value={t.id}>
                {t.name} ({t.template_type || 'composite'})
              </option>
            ))}
          </select>

          <button
            onClick={handleCreateNewTemplate}
            className="p-2 bg-gray-800 hover:bg-gray-700 text-gray-200 border border-gray-700 rounded-lg transition-colors"
            title="Create New Template"
          >
            <Plus className="w-4 h-4" />
          </button>

          <button
            onClick={handleSaveTemplate}
            disabled={isSaving}
            className="px-4 py-2 bg-purple-600 hover:bg-purple-500 text-white rounded-lg text-sm font-semibold flex items-center gap-1.5 shadow-md shadow-purple-900/30 transition-all disabled:opacity-50"
          >
            <Save className="w-4 h-4" />
            Save Template
          </button>

          {adTemplates.length > 1 && (
            <button
              onClick={handleDeleteTemplate}
              className="p-2 bg-red-950/40 hover:bg-red-900/60 text-red-400 border border-red-800/40 rounded-lg transition-colors"
              title="Delete Template"
            >
              <Trash2 className="w-4 h-4" />
            </button>
          )}
        </div>
      </div>

      {/* Template Properties Bar */}
      <div className="bg-[#111622] border border-gray-800 rounded-xl p-4 shadow-sm flex flex-wrap items-center justify-between gap-4">
        <div className="flex flex-wrap items-center gap-4 flex-1">
          <div>
            <label className="block text-[11px] font-medium text-gray-400 mb-1">Template Name</label>
            <input
              type="text"
              value={templateName}
              onChange={(e) => setTemplateName(e.target.value)}
              className="bg-[#182030] border border-gray-700 rounded-lg px-3 py-1.5 text-sm text-white font-semibold focus:outline-none focus:border-purple-500"
            />
          </div>

          <div>
            <label className="block text-[11px] font-medium text-gray-400 mb-1">Template Classification</label>
            <select
              value={templateType}
              onChange={(e) => setTemplateType(e.target.value)}
              className="bg-[#182030] border border-gray-700 rounded-lg px-3 py-1.5 text-sm text-white focus:outline-none focus:border-purple-500"
            >
              <option value="composite">Composite (Branding + Ads + Overlays)</option>
              <option value="overlay">On-Air Graphics Only (CG Overlays)</option>
              <option value="commercial_break">Commercial Break Master Only</option>
            </select>
          </div>
        </div>

        {/* Real-time Collision Status Indicator */}
        <div className="flex items-center gap-2">
          {collisions.length > 0 ? (
            <div className="px-3 py-1.5 bg-amber-950/70 border border-amber-700 rounded-lg text-xs font-semibold text-amber-300 flex items-center gap-2 animate-pulse">
              <AlertTriangle className="w-4 h-4 text-amber-400" />
              <span>{collisions.length} Layout Overlap Collision(s) Active</span>
            </div>
          ) : (
            <div className="px-3 py-1.5 bg-emerald-950/70 border border-emerald-800 rounded-lg text-xs font-semibold text-emerald-300 flex items-center gap-1.5">
              <Check className="w-4 h-4 text-emerald-400" />
              <span>No Overlap Collisions</span>
            </div>
          )}
        </div>
      </div>

      {/* 16:9 WYSIWYG Broadcast Canvas Editor */}
      <div className="bg-[#111622] border border-gray-800 rounded-xl p-5 shadow-lg space-y-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Layout className="w-4 h-4 text-purple-400" />
            <h3 className="text-xs font-bold text-white uppercase tracking-wider">
              16:9 Interactive Broadcast Monitor (1920x1080 Raster)
            </h3>
          </div>
          <div className="flex items-center gap-3">
            <label className="flex items-center gap-1.5 text-xs text-gray-300 cursor-pointer">
              <input
                type="checkbox"
                checked={showSafeGuides}
                onChange={(e) => setShowSafeGuides(e.target.checked)}
                className="rounded border-gray-700 text-purple-600 focus:ring-purple-500 bg-gray-900"
              />
              <span>EBU R95 Safe Area Guides</span>
            </label>
            <button
              onClick={handlePlayAnimation}
              className="px-3 py-1 bg-gray-800 hover:bg-gray-700 text-white rounded text-xs flex items-center gap-1 border border-gray-700 font-semibold"
            >
              <Play className="w-3.5 h-3.5 text-green-400" />
              Preview Animations
            </button>
          </div>
        </div>

        {/* 16:9 Canvas Screen */}
        <div
          ref={canvasRef}
          className="relative w-full aspect-video bg-[#0a0e17] rounded-xl overflow-hidden border border-gray-800 shadow-2xl select-none"
          style={{
            backgroundImage: "radial-gradient(circle at 50% 50%, #151f33 0%, #080c14 100%)"
          }}
        >
          {/* EBU R95 Safe Area Overlays */}
          {showSafeGuides && (
            <div className="absolute inset-0 pointer-events-none z-10">
              {/* Action Safe (90%) */}
              <div
                className="absolute border border-dashed border-gray-600/40 rounded"
                style={{ top: '5%', left: '5%', right: '5%', bottom: '5%' }}
              />
              {/* Title Safe (80%) */}
              <div
                className="absolute border border-dotted border-gray-500/30 rounded"
                style={{ top: '10%', left: '10%', right: '10%', bottom: '10%' }}
              />
              <span className="absolute bottom-2 right-2 text-[10px] font-mono text-gray-400 bg-black/60 px-1 rounded">
                1920x1080 16:9 EBU R95
              </span>
            </div>
          )}

          {/* Station Logo / Watermark Bug */}
          {logoPath && (
            <div
              key={`logo-${animKey}`}
              onMouseDown={(e) => startDrag(e, "logo", { x: logoX, y: logoY, width: logoWidth, height: logoHeight })}
              className={`absolute cursor-move group transition-shadow ${
                activeTargetId === "logo" ? "ring-2 ring-purple-500 z-30" : "hover:ring-1 hover:ring-purple-400 z-20"
              }`}
              style={{
                left: `${(logoX / 1920) * 100}%`,
                top: `${(logoY / 1080) * 100}%`,
                width: `${(logoWidth / 1920) * 100}%`,
                height: `${(logoHeight / 1080) * 100}%`,
                opacity: logoOpacity
              }}
            >
              <div className="w-full h-full bg-black/40 border border-purple-500/60 rounded flex items-center justify-center p-1 relative backdrop-blur-xs">
                <span className="text-[10px] font-bold text-white tracking-widest font-mono uppercase truncate">
                  STATION LOGO
                </span>
                {/* Resize Handle */}
                <div
                  onMouseDown={(e) => startResize(e, "logo", { x: logoX, y: logoY, width: logoWidth, height: logoHeight })}
                  className="absolute bottom-0 right-0 w-3 h-3 bg-purple-500 cursor-nwse-resize rounded-tl opacity-0 group-hover:opacity-100 transition-opacity"
                  title="Resize Logo"
                />
              </div>
            </div>
          )}

          {/* Render Active Overlays on Canvas */}
          {overlayElements.map((el) => {
            if (!el.is_active) return null;
            const isSelected = selectedElementId === el.id;
            const hasCollision = isElementColliding(el.id);

            const leftPct = (el.x / 1920) * 100;
            const topPct = (el.y / 1080) * 100;
            const widthPct = (el.width / 1920) * 100;
            const heightPct = (el.height / 1080) * 100;

            let bgClass = "bg-black/80";
            if (el.background_color?.includes("navy")) bgClass = "bg-blue-950/85";
            if (el.background_color?.includes("darkblue")) bgClass = "bg-indigo-950/85";
            if (el.background_color?.includes("purple")) bgClass = "bg-purple-950/85";

            let textClass = "text-white";
            if (el.text_color === "yellow") textClass = "text-yellow-300";
            if (el.text_color === "gold") textClass = "text-amber-300";

            return (
              <div
                key={`${el.id}-${animKey}`}
                onMouseDown={(e) => startDrag(e, el.id, { x: el.x, y: el.y, width: el.width, height: el.height })}
                className={`absolute cursor-move group transition-shadow ${
                  isSelected
                    ? "ring-2 ring-blue-500 z-30"
                    : hasCollision
                    ? "ring-2 ring-amber-500/80 z-20"
                    : "hover:ring-1 hover:ring-gray-400 z-10"
                } ${getElementAnimationClass(el.entrance_animation)}`}
                style={{
                  left: `${leftPct}%`,
                  top: `${topPct}%`,
                  width: `${widthPct}%`,
                  height: `${heightPct}%`
                }}
              >
                <div className={`w-full h-full ${bgClass} border ${
                  hasCollision ? "border-amber-500" : "border-gray-700/80"
                } rounded p-1.5 flex flex-col justify-center overflow-hidden relative shadow-lg`}>
                  {/* Collision Warning Badge */}
                  {hasCollision && (
                    <span className="absolute top-0.5 right-1 text-[9px] bg-amber-500 text-black px-1 rounded font-bold uppercase tracking-wider">
                      Overlap
                    </span>
                  )}

                  {/* Header / Subtitle typography */}
                  <div className={`text-[11px] font-bold ${textClass} truncate leading-tight`}>
                    {el.text}
                  </div>
                  {el.sub_text && (
                    <div className="text-[9px] text-gray-300 truncate opacity-90 mt-0.5">
                      {el.sub_text}
                    </div>
                  )}

                  {/* Resize Handle */}
                  <div
                    onMouseDown={(e) => startResize(e, el.id, { x: el.x, y: el.y, width: el.width, height: el.height })}
                    className="absolute bottom-0 right-0 w-3.5 h-3.5 bg-blue-500 cursor-nwse-resize rounded-tl opacity-0 group-hover:opacity-100 transition-opacity"
                    title="Drag to Resize"
                  />
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* 4 Organized Navigation Tabs */}
      <div className="flex border-b border-gray-800 gap-2">
        <button
          onClick={() => setActiveTab("logo")}
          className={`px-4 py-2 text-xs font-bold uppercase tracking-wider transition-colors border-b-2 flex items-center gap-2 ${
            activeTab === "logo"
              ? "border-purple-500 text-purple-400"
              : "border-transparent text-gray-400 hover:text-gray-300"
          }`}
        >
          <ImageIcon className="w-4 h-4" />
          Logo / Watermark
        </button>
        <button
          onClick={() => setActiveTab("now_playing")}
          className={`px-4 py-2 text-xs font-bold uppercase tracking-wider transition-colors border-b-2 flex items-center gap-2 ${
            activeTab === "now_playing"
              ? "border-blue-500 text-blue-400"
              : "border-transparent text-gray-400 hover:text-gray-300"
          }`}
        >
          <Tv className="w-4 h-4" />
          Now Playing & Up Next
        </button>
        <button
          onClick={() => setActiveTab("ads")}
          className={`px-4 py-2 text-xs font-bold uppercase tracking-wider transition-colors border-b-2 flex items-center gap-2 ${
            activeTab === "ads"
              ? "border-emerald-500 text-emerald-400"
              : "border-transparent text-gray-400 hover:text-gray-300"
          }`}
        >
          <Layout className="w-4 h-4" />
          Ad Layout & Insertion
        </button>
        <button
          onClick={() => setActiveTab("general")}
          className={`px-4 py-2 text-xs font-bold uppercase tracking-wider transition-colors border-b-2 flex items-center gap-2 ${
            activeTab === "general"
              ? "border-amber-500 text-amber-400"
              : "border-transparent text-gray-400 hover:text-gray-300"
          }`}
        >
          <Shield className="w-4 h-4" />
          General Layout & Collisions
        </button>
      </div>

      {/* TAB 1: Logo / Watermark Configuration */}
      {activeTab === "logo" && (
        <div className="bg-[#111622] border border-gray-800 rounded-xl p-5 shadow-lg space-y-5 animate-in fade-in duration-200">
          <div className="flex items-center justify-between pb-3 border-b border-gray-800">
            <h3 className="text-sm font-bold text-white uppercase tracking-wider flex items-center gap-2">
              <ImageIcon className="w-4 h-4 text-purple-400" />
              Station Watermark Bug & Logo Placement
            </h3>
            <span className="text-xs text-gray-400">Rendered via FFmpeg overlay filter graph</span>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-5">
            {/* Logo File & Upload */}
            <div className="space-y-3">
              <div>
                <label className="block text-xs font-medium text-gray-400 mb-1">Logo File Path</label>
                <div className="flex items-center gap-2">
                  <input
                    type="text"
                    value={logoPath}
                    onChange={(e) => setLogoPath(e.target.value)}
                    placeholder="data/logos/channel_logo.png"
                    className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-xs text-white font-mono focus:outline-none focus:border-purple-500"
                  />
                  <input
                    type="file"
                    ref={logoInputRef}
                    onChange={handleUploadLogoFile}
                    accept="image/png,image/jpeg,image/webp"
                    className="hidden"
                  />
                  <button
                    type="button"
                    disabled={isUploadingLogo}
                    onClick={() => logoInputRef.current?.click()}
                    className="px-3 py-2 bg-purple-600 hover:bg-purple-500 text-white rounded-lg text-xs font-semibold flex items-center gap-1.5 shrink-0"
                  >
                    <Upload className="w-3.5 h-3.5" />
                    Upload
                  </button>
                </div>
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-400 mb-1">Screen Corner Position</label>
                <select
                  value={logoPosition}
                  onChange={(e) => {
                    const pos = e.target.value;
                    setLogoPosition(pos);
                    if (pos === "top-right") { setLogoX(1720); setLogoY(40); }
                    if (pos === "top-left") { setLogoX(40); setLogoY(40); }
                    if (pos === "bottom-right") { setLogoX(1720); setLogoY(950); }
                    if (pos === "bottom-left") { setLogoX(40); setLogoY(950); }
                  }}
                  className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-purple-500"
                >
                  <option value="top-right">Top Right (Broadcast Standard)</option>
                  <option value="top-left">Top Left</option>
                  <option value="bottom-right">Bottom Right</option>
                  <option value="bottom-left">Bottom Left</option>
                  <option value="custom">Custom Canvas Coordinates</option>
                </select>
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-400 mb-1">
                  Logo Opacity ({Math.round(logoOpacity * 100)}%)
                </label>
                <input
                  type="range"
                  min="0.10"
                  max="1.00"
                  step="0.05"
                  value={logoOpacity}
                  onChange={(e) => setLogoOpacity(parseFloat(e.target.value))}
                  className="w-full accent-purple-500"
                />
              </div>
            </div>

            {/* Coordinates & Sizing */}
            <div className="space-y-3">
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-gray-400 mb-1">X Coordinate (px)</label>
                  <input
                    type="number"
                    value={logoX}
                    onChange={(e) => setLogoX(parseInt(e.target.value) || 0)}
                    className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-xs text-white font-mono focus:outline-none focus:border-purple-500"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-400 mb-1">Y Coordinate (px)</label>
                  <input
                    type="number"
                    value={logoY}
                    onChange={(e) => setLogoY(parseInt(e.target.value) || 0)}
                    className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-xs text-white font-mono focus:outline-none focus:border-purple-500"
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="block text-xs font-medium text-gray-400 mb-1">Width (px)</label>
                  <input
                    type="number"
                    value={logoWidth}
                    onChange={(e) => setLogoWidth(parseInt(e.target.value) || 140)}
                    className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-xs text-white font-mono focus:outline-none focus:border-purple-500"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-gray-400 mb-1">Height (px)</label>
                  <input
                    type="number"
                    value={logoHeight}
                    onChange={(e) => setLogoHeight(parseInt(e.target.value) || 90)}
                    className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-xs text-white font-mono focus:outline-none focus:border-purple-500"
                  />
                </div>
              </div>

              <div>
                <label className="block text-xs font-medium text-gray-400 mb-1">Scaling Mode</label>
                <select
                  value={logoFit}
                  onChange={(e) => setLogoFit(e.target.value)}
                  className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-purple-500"
                >
                  <option value="contain">Contain (Preserve aspect ratio)</option>
                  <option value="cover">Cover (Fill bounding box)</option>
                </select>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* TAB 2: Now Playing & Up Next Overlays */}
      {activeTab === "now_playing" && (
        <div className="bg-[#111622] border border-gray-800 rounded-xl p-5 shadow-lg space-y-5 animate-in fade-in duration-200">
          <div className="flex items-center justify-between pb-3 border-b border-gray-800">
            <h3 className="text-sm font-bold text-white uppercase tracking-wider flex items-center gap-2">
              <Tv className="w-4 h-4 text-blue-400" />
              Automated Now Playing & Up Next Dynamic Overlays
            </h3>
            <span className="text-xs text-gray-400">Automatically populated from schedule timeline</span>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {/* Now Playing Settings */}
            {(() => {
              const npElem = overlayElements.find((el) => el.type === "now_playing");
              return (
                <div className="p-4 bg-[#141b2b] border border-gray-800 rounded-xl space-y-3">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-bold text-blue-400 uppercase tracking-wide">
                      Now Playing Overlay
                    </span>
                    <button
                      type="button"
                      onClick={() => {
                        if (npElem) {
                          setOverlayElements((prev) =>
                            prev.map((el) => el.id === npElem.id ? { ...el, is_active: !el.is_active } : el)
                          );
                        } else {
                          handleAddElement("now_playing");
                        }
                      }}
                      className={`px-2.5 py-1 rounded text-xs font-semibold ${
                        npElem?.is_active
                          ? "bg-blue-600 text-white"
                          : "bg-gray-800 text-gray-400"
                      }`}
                    >
                      {npElem?.is_active ? "Enabled" : "Disabled"}
                    </button>
                  </div>

                  {npElem && (
                    <div className="space-y-3 text-xs">
                      <div>
                        <label className="block text-gray-400 mb-1">Display Text Template</label>
                        <input
                          type="text"
                          value={npElem.text}
                          onChange={(e) => {
                            setOverlayElements((prev) =>
                              prev.map((el) => el.id === npElem.id ? { ...el, text: e.target.value } : el)
                            );
                          }}
                          className="w-full bg-[#182030] border border-gray-700 rounded px-2.5 py-1.5 text-white"
                        />
                      </div>
                      <div className="grid grid-cols-2 gap-2">
                        <div>
                          <label className="block text-gray-400 mb-1">Position (X, Y)</label>
                          <span className="font-mono text-gray-300">{npElem.x}px, {npElem.y}px</span>
                        </div>
                        <div>
                          <label className="block text-gray-400 mb-1">Dimensions (W x H)</label>
                          <span className="font-mono text-gray-300">{npElem.width}px x {npElem.height}px</span>
                        </div>
                      </div>
                      <div>
                        <label className="block text-gray-400 mb-1">Entrance Animation</label>
                        <select
                          value={npElem.entrance_animation}
                          onChange={(e) => {
                            setOverlayElements((prev) =>
                              prev.map((el) => el.id === npElem.id ? { ...el, entrance_animation: e.target.value } : el)
                            );
                          }}
                          className="w-full bg-[#182030] border border-gray-700 rounded px-2.5 py-1.5 text-white"
                        >
                          <option value="slide_in_left">Slide In Left</option>
                          <option value="fade_in">Fade In</option>
                          <option value="zoom_in">Zoom In</option>
                          <option value="static">Static (Always Visible)</option>
                        </select>
                      </div>
                    </div>
                  )}
                </div>
              );
            })()}

            {/* Up Next Settings */}
            {(() => {
              const unElem = overlayElements.find((el) => el.type === "up_next");
              return (
                <div className="p-4 bg-[#141b2b] border border-gray-800 rounded-xl space-y-3">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-bold text-amber-400 uppercase tracking-wide">
                      Up Next Overlay
                    </span>
                    <button
                      type="button"
                      onClick={() => {
                        if (unElem) {
                          setOverlayElements((prev) =>
                            prev.map((el) => el.id === unElem.id ? { ...el, is_active: !el.is_active } : el)
                          );
                        } else {
                          handleAddElement("up_next");
                        }
                      }}
                      className={`px-2.5 py-1 rounded text-xs font-semibold ${
                        unElem?.is_active
                          ? "bg-amber-600 text-white"
                          : "bg-gray-800 text-gray-400"
                      }`}
                    >
                      {unElem?.is_active ? "Enabled" : "Disabled"}
                    </button>
                  </div>

                  {unElem && (
                    <div className="space-y-3 text-xs">
                      <div>
                        <label className="block text-gray-400 mb-1">Display Text Template</label>
                        <input
                          type="text"
                          value={unElem.text}
                          onChange={(e) => {
                            setOverlayElements((prev) =>
                              prev.map((el) => el.id === unElem.id ? { ...el, text: e.target.value } : el)
                            );
                          }}
                          className="w-full bg-[#182030] border border-gray-700 rounded px-2.5 py-1.5 text-white"
                        />
                      </div>
                      <div className="grid grid-cols-2 gap-2">
                        <div>
                          <label className="block text-gray-400 mb-1">Position (X, Y)</label>
                          <span className="font-mono text-gray-300">{unElem.x}px, {unElem.y}px</span>
                        </div>
                        <div>
                          <label className="block text-gray-400 mb-1">Dimensions (W x H)</label>
                          <span className="font-mono text-gray-300">{unElem.width}px x {unElem.height}px</span>
                        </div>
                      </div>
                      <div>
                        <label className="block text-gray-400 mb-1">Entrance Animation</label>
                        <select
                          value={unElem.entrance_animation}
                          onChange={(e) => {
                            setOverlayElements((prev) =>
                              prev.map((el) => el.id === unElem.id ? { ...el, entrance_animation: e.target.value } : el)
                            );
                          }}
                          className="w-full bg-[#182030] border border-gray-700 rounded px-2.5 py-1.5 text-white"
                        >
                          <option value="fade_in">Fade In</option>
                          <option value="slide_in_bottom">Slide In Bottom</option>
                          <option value="zoom_in">Zoom In</option>
                          <option value="static">Static (Always Visible)</option>
                        </select>
                      </div>
                    </div>
                  )}
                </div>
              );
            })()}
          </div>
        </div>
      )}

      {/* TAB 3: Ad Layout & Insertion */}
      {activeTab === "ads" && (
        <div className="bg-[#111622] border border-gray-800 rounded-xl p-5 shadow-lg space-y-5 animate-in fade-in duration-200">
          <div className="flex items-center justify-between pb-3 border-b border-gray-800">
            <h3 className="text-sm font-bold text-white uppercase tracking-wider flex items-center gap-2">
              <Layout className="w-4 h-4 text-emerald-400" />
              Ad Components, Banners, Crawls & SCTE-35
            </h3>
            <div className="flex flex-wrap items-center gap-1.5">
              <button
                type="button"
                onClick={() => handleAddElement("header_banner")}
                className="px-2.5 py-1 bg-gray-800 hover:bg-gray-700 text-gray-200 rounded text-xs border border-gray-700"
              >
                + Header Banner
              </button>
              <button
                type="button"
                onClick={() => handleAddElement("footer_banner")}
                className="px-2.5 py-1 bg-gray-800 hover:bg-gray-700 text-gray-200 rounded text-xs border border-gray-700"
              >
                + Footer Banner
              </button>
              <button
                type="button"
                onClick={() => handleAddElement("ticker")}
                className="px-2.5 py-1 bg-gray-800 hover:bg-gray-700 text-gray-200 rounded text-xs border border-gray-700"
              >
                + Ticker Crawl
              </button>
              <button
                type="button"
                onClick={() => handleAddElement("promo")}
                className="px-2.5 py-1 bg-gray-800 hover:bg-gray-700 text-gray-200 rounded text-xs border border-gray-700"
              >
                + Promo Bump
              </button>
              <button
                type="button"
                onClick={() => handleAddElement("lower_third")}
                className="px-2.5 py-1 bg-gray-800 hover:bg-gray-700 text-gray-200 rounded text-xs border border-gray-700"
              >
                + Lower Third
              </button>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
            {/* List of Graphics Elements */}
            <div className="space-y-2">
              <span className="text-xs font-bold text-gray-400 uppercase tracking-wider block mb-1">
                Active Elements ({overlayElements.length})
              </span>
              <div className="max-h-72 overflow-y-auto space-y-1.5 pr-1">
                {overlayElements.map((el) => (
                  <div
                    key={el.id}
                    onClick={() => setSelectedElementId(el.id)}
                    className={`p-2.5 rounded-lg border text-xs cursor-pointer flex items-center justify-between transition-colors ${
                      selectedElementId === el.id
                        ? "bg-purple-950/80 border-purple-600 text-white font-bold"
                        : "bg-[#141b2b] border-gray-800 text-gray-300 hover:bg-gray-800/60"
                    }`}
                  >
                    <div className="truncate mr-2">
                      <span className="uppercase text-[10px] text-gray-400 block">{el.type.replace(/_/g, ' ')}</span>
                      <span className="truncate">{el.text}</span>
                    </div>
                    <div className="flex items-center gap-1.5 shrink-0">
                      <input
                        type="checkbox"
                        checked={el.is_active}
                        onChange={(e) => {
                          e.stopPropagation();
                          setOverlayElements((prev) =>
                            prev.map((item) => item.id === el.id ? { ...item, is_active: e.target.checked } : item)
                          );
                        }}
                        className="rounded accent-purple-600"
                      />
                      <button
                        type="button"
                        onClick={(e) => {
                          e.stopPropagation();
                          handleRemoveElement(el.id);
                        }}
                        className="text-gray-500 hover:text-red-400"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            </div>

            {/* Element Parameters Editor */}
            <div className="md:col-span-2 space-y-3 bg-[#141b2b] p-4 rounded-xl border border-gray-800">
              {selectedElement ? (
                <div className="space-y-3 text-xs">
                  <span className="font-bold text-white uppercase tracking-wider text-xs block border-b border-gray-800 pb-2">
                    Editing: {selectedElement.type.toUpperCase()} ({selectedElement.id})
                  </span>

                  <div>
                    <label className="block text-gray-400 mb-1">Headline Text</label>
                    <input
                      type="text"
                      value={selectedElement.text}
                      onChange={(e) => updateSelectedElement({ text: e.target.value })}
                      className="w-full bg-[#182030] border border-gray-700 rounded px-3 py-1.5 text-white"
                    />
                  </div>

                  <div>
                    <label className="block text-gray-400 mb-1">Subtext / Secondary Line</label>
                    <input
                      type="text"
                      value={selectedElement.sub_text || ""}
                      onChange={(e) => updateSelectedElement({ sub_text: e.target.value })}
                      className="w-full bg-[#182030] border border-gray-700 rounded px-3 py-1.5 text-white"
                    />
                  </div>

                  <div className="grid grid-cols-2 sm:grid-cols-4 gap-2">
                    <div>
                      <label className="block text-gray-400 mb-1">X Pos</label>
                      <input
                        type="number"
                        value={selectedElement.x}
                        onChange={(e) => updateSelectedElement({ x: parseInt(e.target.value) || 0 })}
                        className="w-full bg-[#182030] border border-gray-700 rounded px-2 py-1 text-white font-mono"
                      />
                    </div>
                    <div>
                      <label className="block text-gray-400 mb-1">Y Pos</label>
                      <input
                        type="number"
                        value={selectedElement.y}
                        onChange={(e) => updateSelectedElement({ y: parseInt(e.target.value) || 0 })}
                        className="w-full bg-[#182030] border border-gray-700 rounded px-2 py-1 text-white font-mono"
                      />
                    </div>
                    <div>
                      <label className="block text-gray-400 mb-1">Width</label>
                      <input
                        type="number"
                        value={selectedElement.width}
                        onChange={(e) => updateSelectedElement({ width: parseInt(e.target.value) || 200 })}
                        className="w-full bg-[#182030] border border-gray-700 rounded px-2 py-1 text-white font-mono"
                      />
                    </div>
                    <div>
                      <label className="block text-gray-400 mb-1">Height</label>
                      <input
                        type="number"
                        value={selectedElement.height}
                        onChange={(e) => updateSelectedElement({ height: parseInt(e.target.value) || 50 })}
                        className="w-full bg-[#182030] border border-gray-700 rounded px-2 py-1 text-white font-mono"
                      />
                    </div>
                  </div>
                </div>
              ) : (
                <div className="text-gray-500 py-12 text-center">Select an element to edit properties</div>
              )}
            </div>
          </div>
        </div>
      )}

      {/* TAB 4: General Layout & Collision Resolution */}
      {activeTab === "general" && (
        <div className="bg-[#111622] border border-gray-800 rounded-xl p-5 shadow-lg space-y-5 animate-in fade-in duration-200">
          <div className="flex items-center justify-between pb-3 border-b border-gray-800">
            <h3 className="text-sm font-bold text-white uppercase tracking-wider flex items-center gap-2">
              <Shield className="w-4 h-4 text-amber-400" />
              Layout Conflict Detection & Automated Collision Resolution
            </h3>
            <span className="text-xs text-gray-400">Deterministic broadcast collision rules</span>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {/* Collision Resolution Configuration */}
            <div className="space-y-4">
              <div>
                <label className="block text-xs font-bold text-gray-300 uppercase tracking-wide mb-2">
                  Collision Behavior for Overlapping Elements
                </label>
                <div className="space-y-2">
                  <label className="p-3 bg-[#141b2b] border border-gray-800 rounded-xl flex items-start gap-3 cursor-pointer hover:bg-gray-800/40">
                    <input
                      type="radio"
                      name="collision_behavior"
                      value="alternate"
                      checked={collisionBehavior === "alternate"}
                      onChange={(e) => setCollisionBehavior(e.target.value)}
                      className="mt-1 accent-amber-500"
                    />
                    <div>
                      <span className="text-xs font-bold text-white block">
                        Alternating Overlays (Time-Sliced Display)
                      </span>
                      <p className="text-[11px] text-gray-400 mt-0.5">
                        Cycles between colliding overlays at a configurable duration in seconds so both graphics receive airtime.
                      </p>
                    </div>
                  </label>

                  <label className="p-3 bg-[#141b2b] border border-gray-800 rounded-xl flex items-start gap-3 cursor-pointer hover:bg-gray-800/40">
                    <input
                      type="radio"
                      name="collision_behavior"
                      value="priority"
                      checked={collisionBehavior === "priority"}
                      onChange={(e) => setCollisionBehavior(e.target.value)}
                      className="mt-1 accent-amber-500"
                    />
                    <div>
                      <span className="text-xs font-bold text-white block">
                        Strict Priority Precedence Rule
                      </span>
                      <p className="text-[11px] text-gray-400 mt-0.5">
                        Only the higher-priority overlay renders on-air. Lower-priority overlays are suppressed while colliding.
                      </p>
                    </div>
                  </label>
                </div>
              </div>

              {collisionBehavior === "alternate" && (
                <div>
                  <label className="block text-xs font-medium text-gray-400 mb-1">
                    Alternating Duration: <strong>{alternateDuration} seconds</strong>
                  </label>
                  <input
                    type="range"
                    min="5"
                    max="60"
                    step="5"
                    value={alternateDuration}
                    onChange={(e) => setAlternateDuration(parseInt(e.target.value))}
                    className="w-full accent-amber-500"
                  />
                  <div className="flex justify-between text-[10px] text-gray-500 font-mono mt-1">
                    <span>5s</span>
                    <span>15s (Standard)</span>
                    <span>30s</span>
                    <span>60s</span>
                  </div>
                </div>
              )}
            </div>

            {/* Active Collision Report */}
            <div className="space-y-3">
              <span className="text-xs font-bold text-gray-300 uppercase tracking-wide block">
                Detected Canvas Collisions ({collisions.length})
              </span>

              {collisions.length === 0 ? (
                <div className="p-4 bg-emerald-950/40 border border-emerald-800 rounded-xl text-xs text-emerald-300 flex items-center gap-2">
                  <Check className="w-5 h-5 text-emerald-400 shrink-0" />
                  <span>No overlapping graphics on this template. All active layers have clear raster separation!</span>
                </div>
              ) : (
                <div className="space-y-2 max-h-60 overflow-y-auto">
                  {collisions.map((col, idx) => (
                    <div
                      key={idx}
                      className="p-3 bg-amber-950/40 border border-amber-700/60 rounded-xl text-xs text-amber-200 space-y-1"
                    >
                      <div className="font-bold flex items-center justify-between text-amber-300">
                        <span>Collision #{idx + 1}</span>
                        <span className="text-[10px] uppercase font-mono px-1.5 py-0.5 bg-amber-900 rounded">
                          {collisionBehavior === "alternate" ? `Alternates every ${alternateDuration}s` : "Priority order applied"}
                        </span>
                      </div>
                      <div className="text-[11px] text-gray-300 flex items-center gap-1 font-mono">
                        <span className="text-blue-300">{col.a.type.toUpperCase()}</span>
                        <span>overlaps with</span>
                        <span className="text-purple-300">{col.b.type.toUpperCase()}</span>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
