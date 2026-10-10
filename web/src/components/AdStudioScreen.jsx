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
  RefreshCw
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
    text: "PATHAAN (2023) 4K UHD",
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
    text: "JAWAN (EXTENDED CUT)",
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
  const [activeTemplateId, setActiveTemplateId] = useState("");
  const [templateName, setTemplateName] = useState("");
  const [templateType, setTemplateType] = useState("composite");
  const [overlayElements, setOverlayElements] = useState(DEFAULT_OVERLAYS);
  const [commercialBreaks, setCommercialBreaks] = useState([]);
  const [selectedElementId, setSelectedElementId] = useState("ov-np-1");

  const [showSafeGuides, setShowSafeGuides] = useState(true);
  const [animKey, setAnimKey] = useState(0);
  const [isSaving, setIsSaving] = useState(false);

  // Dragging state on canvas
  const canvasRef = useRef(null);
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });
  const [elementStartPos, setElementStartPos] = useState({ x: 0, y: 0 });

  // Sync when adTemplates changes or selection changes
  useEffect(() => {
    if (adTemplates.length > 0) {
      const match = adTemplates.find((t) => t.id === activeTemplateId) || adTemplates[0];
      if (match) {
        setActiveTemplateId(match.id);
        setTemplateName(match.name || "Default Template");
        setTemplateType(match.template_type || "composite");
        const elems = match.overlay_elements && match.overlay_elements.length > 0
          ? match.overlay_elements
          : DEFAULT_OVERLAYS;
        setOverlayElements(elems);
        setCommercialBreaks(match.commercial_breaks || []);
        if (elems.length > 0 && !selectedElementId) {
          setSelectedElementId(elems[0].id);
        }
      }
    }
  }, [adTemplates, activeTemplateId]);

  const handleSelectTemplate = (id) => {
    setActiveTemplateId(id);
    const tmpl = adTemplates.find((t) => t.id === id);
    if (tmpl) {
      setTemplateName(tmpl.name);
      setTemplateType(tmpl.template_type || "composite");
      const elems = tmpl.overlay_elements && tmpl.overlay_elements.length > 0
        ? tmpl.overlay_elements
        : DEFAULT_OVERLAYS;
      setOverlayElements(elems);
      setCommercialBreaks(tmpl.commercial_breaks || []);
      if (elems.length > 0) setSelectedElementId(elems[0].id);
    }
  };

  const handleCreateNewTemplate = () => {
    const newId = `tmpl-${Date.now().toString(36)}`;
    setActiveTemplateId(newId);
    setTemplateName("Custom Layout Studio Template");
    setTemplateType("composite");
    setOverlayElements(JSON.parse(JSON.stringify(DEFAULT_OVERLAYS)));
    setCommercialBreaks([
      { break_type: "mid_roll", offset_seconds: 1800, duration_seconds: 60, scte35_cue: true, clips: [] }
    ]);
    setSelectedElementId(DEFAULT_OVERLAYS[0].id);
    onShowToast("Initialized new on-air graphics template", "info");
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
      onShowToast("Deleted graphics template", "info");
      setActiveTemplateId("");
      if (onRefreshTemplates) onRefreshTemplates();
    } catch (err) {
      onShowToast("Delete failed: " + err.message, "error");
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
        newEl.text = "NETWORK BREAKING UPDATE";
        newEl.x = 0;
        newEl.y = 0;
        newEl.width = 1920;
        newEl.height = 60;
        newEl.background_color = "navy@0.85";
        break;
      case "footer_banner":
        newEl.text = "NEXT-GEN TV PLAYOUT";
        newEl.x = 0;
        newEl.y = 1010;
        newEl.width = 1920;
        newEl.height = 70;
        newEl.background_color = "darkblue@0.85";
        break;
      case "ticker":
        newEl.text = "LIVE NEWS TICKER • SCROLLING HEADLINE ITEM 1 • SCROLLING HEADLINE ITEM 2";
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
        newEl.sub_text = "ORIGINAL THEATRICAL VERSION";
        newEl.x = 60;
        newEl.y = 80;
        newEl.width = 380;
        newEl.height = 85;
        break;
      case "up_next":
        newEl.text = "UP NEXT: ACTION EXTRAVAGANZA";
        newEl.sub_text = "STREAMING TONIGHT @ 22:00";
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
        newEl.text = "DIRECTOR COMMENTARY LIVE";
        newEl.sub_text = "HOST: MCRFLOW ON-AIR CONTROL";
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

  // Dragging handling on 16:9 canvas
  const handleMouseDown = (e, el) => {
    e.stopPropagation();
    setSelectedElementId(el.id);
    setIsDragging(true);
    setDragStart({ x: e.clientX, y: e.clientY });
    setElementStartPos({ x: el.x || 0, y: el.y || 0 });
  };

  const handleMouseMove = (e) => {
    if (!isDragging || !selectedElement || !canvasRef.current) return;
    const rect = canvasRef.current.getBoundingClientRect();
    const scaleX = 1920 / rect.width;
    const scaleY = 1080 / rect.height;

    const deltaX = (e.clientX - dragStart.x) * scaleX;
    const deltaY = (e.clientY - dragStart.y) * scaleY;

    const newX = Math.max(0, Math.min(1920 - (selectedElement.width || 200), Math.round(elementStartPos.x + deltaX)));
    const newY = Math.max(0, Math.min(1080 - (selectedElement.height || 60), Math.round(elementStartPos.y + deltaY)));

    updateSelectedElement({ x: newX, y: newY });
  };

  const handleMouseUp = () => {
    if (isDragging) setIsDragging(false);
  };

  const handlePlayAnimation = () => {
    setAnimKey((prev) => prev + 1);
    onShowToast("Triggering on-air graphics animation preview", "info");
  };

  const getElementAnimationClass = (anim) => {
    switch (anim) {
      case 'slide_in_left':
        return 'animate-in slide-in-from-left duration-500';
      case 'slide_in_bottom':
        return 'animate-in slide-in-from-bottom duration-500';
      case 'zoom_in':
        return 'animate-in zoom-in duration-300';
      case 'scroll_left':
        return '';
      case 'fade_in':
      default:
        return 'animate-in fade-in duration-500';
    }
  };

  const formatBgColor = (bg) => {
    if (!bg) return 'rgba(0,0,0,0.8)';
    if (bg.includes('navy')) return 'rgba(10, 25, 47, 0.88)';
    if (bg.includes('darkblue')) return 'rgba(15, 30, 65, 0.88)';
    if (bg.includes('purple')) return 'rgba(88, 28, 135, 0.88)';
    if (bg.includes('darkred')) return 'rgba(127, 29, 29, 0.88)';
    if (bg.includes('black')) return 'rgba(15, 17, 23, 0.85)';
    return bg;
  };

  return (
    <div
      className="h-full flex flex-col p-3 sm:p-4 space-y-4 overflow-y-auto max-w-full select-none"
      onMouseMove={handleMouseMove}
      onMouseUp={handleMouseUp}
    >
      {/* Top Header & Template Toolbar */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 shrink-0 bg-[#111827] border border-[#1F2937] p-3 rounded-lg shadow-sm">
        <div className="flex flex-wrap items-center gap-3">
          <div className="flex items-center gap-2">
            <Sparkles className="w-5 h-5 text-indigo-400 shrink-0" />
            <div>
              <h2 className="text-sm font-bold text-white">
                {t('ad.title') || "WYSIWYG Ad & Broadcast Graphics Studio"}
              </h2>
              <p className="text-[11px] text-gray-400">
                Visual layout builder for Header/Footer Banners, Tickers, Cards & SCTE-35
              </p>
            </div>
          </div>
          <div className="hidden sm:block h-6 w-px bg-gray-700"></div>

          {/* Template Selector */}
          <select
            value={activeTemplateId}
            onChange={(e) => handleSelectTemplate(e.target.value)}
            className="bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white font-medium focus:outline-none focus:border-indigo-500"
          >
            {adTemplates.map((t) => (
              <option key={t.id} value={t.id}>
                {t.name} ({t.template_type || "composite"})
              </option>
            ))}
          </select>

          <button
            onClick={handleCreateNewTemplate}
            className="px-2.5 py-1.5 bg-gray-800 hover:bg-gray-700 text-gray-200 text-xs font-semibold rounded flex items-center gap-1 border border-gray-700 transition-colors"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>New Template</span>
          </button>
        </div>

        <div className="flex flex-wrap items-center gap-2 self-end sm:self-auto">
          {/* Safe Guides Toggle */}
          <label className="flex items-center gap-1.5 cursor-pointer text-xs text-gray-300 mr-2">
            <input
              type="checkbox"
              checked={showSafeGuides}
              onChange={(e) => setShowSafeGuides(e.target.checked)}
              className="rounded bg-gray-800 border-gray-700 text-indigo-600 focus:ring-0"
            />
            <span className="text-[11px]">EBU Safe Guides</span>
          </label>

          <button
            onClick={handlePlayAnimation}
            className="px-3 py-1.5 bg-gray-800 hover:bg-gray-700 text-sky-300 rounded text-xs font-semibold flex items-center gap-1 border border-gray-700 transition-colors"
          >
            <Play className="w-3.5 h-3.5" />
            <span>Preview Animation</span>
          </button>

          <button
            onClick={handleSaveTemplate}
            disabled={isSaving}
            className="px-3.5 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded text-xs font-semibold flex items-center gap-1 shadow transition-colors"
          >
            <Save className="w-3.5 h-3.5" />
            <span>{isSaving ? "Saving..." : "Save Template"}</span>
          </button>

          {adTemplates.length > 1 && (
            <button
              onClick={handleDeleteTemplate}
              className="px-2.5 py-1.5 bg-rose-950/40 hover:bg-rose-900/60 text-rose-300 rounded text-xs border border-rose-500/30 transition-colors"
              title="Delete this template"
            >
              <Trash2 className="w-3.5 h-3.5" />
            </button>
          )}
        </div>
      </div>

      {/* Main Studio Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-4 flex-1 min-h-0">
        {/* Left: 16:9 Interactive WYSIWYG Canvas (8 cols) */}
        <div className="lg:col-span-8 bg-[#111827] border border-[#1F2937] rounded-lg p-3 sm:p-4 flex flex-col space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-xs font-bold text-gray-300 uppercase tracking-wider flex items-center gap-1.5">
              <Eye className="w-3.5 h-3.5 text-sky-400" />
              <span>Broadcast 16:9 Screen (Drag & Reposition Overlays)</span>
            </span>
            <span className="text-[10px] font-mono text-gray-400">
              Selected: <span className="text-sky-300 font-bold uppercase">{selectedElement?.type || "None"}</span>
            </span>
          </div>

          {/* 16:9 Canvas Container */}
          <div
            ref={canvasRef}
            key={animKey}
            className="relative aspect-video bg-black rounded-lg overflow-hidden border border-gray-800 flex items-center justify-center select-none shadow-2xl cursor-crosshair"
          >
            {/* Background Feed preview */}
            <img
              src="https://images.unsplash.com/photo-1518173946687-a4c8a383392e?w=800&auto=format&fit=crop&q=60"
              alt="Video Preview"
              className="w-full h-full object-cover opacity-70 pointer-events-none"
            />
            <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-black/30 pointer-events-none"></div>

            {/* EBU Safe Area Guides */}
            {showSafeGuides && (
              <div className="absolute inset-0 pointer-events-none z-10">
                <div className="absolute inset-[5%] border border-cyan-500/30 border-dashed">
                  <span className="absolute top-1 left-1 text-[8px] text-cyan-400/60 font-mono">ACTION SAFE (90%)</span>
                </div>
                <div className="absolute inset-[10%] border border-amber-500/30 border-dashed">
                  <span className="absolute top-1 left-1 text-[8px] text-amber-400/60 font-mono">TITLE SAFE (80%)</span>
                </div>
              </div>
            )}

            {/* Render Overlay Elements on Canvas */}
            {overlayElements.map((el) => {
              if (!el.is_active) return null;
              const isSel = el.id === selectedElementId;

              const leftPct = `${((el.x || 0) / 1920) * 100}%`;
              const topPct = `${((el.y || 0) / 1080) * 100}%`;
              const widthPct = el.width ? `${(el.width / 1920) * 100}%` : 'auto';
              const heightPct = el.height ? `${(el.height / 1080) * 100}%` : 'auto';

              return (
                <div
                  key={el.id}
                  onMouseDown={(e) => handleMouseDown(e, el)}
                  style={{
                    position: 'absolute',
                    left: leftPct,
                    top: topPct,
                    width: widthPct,
                    height: heightPct,
                    backgroundColor: formatBgColor(el.background_color),
                    color: el.text_color || 'white',
                    zIndex: isSel ? 30 : 20
                  }}
                  className={`cursor-move transition-shadow flex items-center px-3 select-none ${getElementAnimationClass(el.entrance_animation)} ${
                    isSel
                      ? 'ring-2 ring-sky-400 ring-offset-1 ring-offset-black shadow-xl shadow-sky-500/30'
                      : 'hover:ring-1 hover:ring-gray-400'
                  }`}
                >
                  {el.type === 'ticker' ? (
                    <div className="w-full flex items-center overflow-hidden">
                      <span className="bg-red-700 text-white font-black text-[9px] px-1.5 py-0.5 rounded mr-2 tracking-wider shrink-0">
                        TICKER
                      </span>
                      <div className="font-mono text-xs font-semibold truncate animate-pulse">
                        {el.text}
                      </div>
                    </div>
                  ) : el.type === 'header_banner' || el.type === 'footer_banner' ? (
                    <div className="w-full flex items-center justify-between text-xs font-bold tracking-wider">
                      <span>{el.text}</span>
                      {el.sub_text && <span className="text-[10px] opacity-75 font-mono">{el.sub_text}</span>}
                    </div>
                  ) : el.type === 'now_playing' ? (
                    <div className="flex flex-col justify-center">
                      <span className="text-[9px] font-bold text-yellow-400 uppercase tracking-wider">NOW PLAYING</span>
                      <span className="text-xs font-bold truncate leading-tight">{el.text}</span>
                      {el.sub_text && <span className="text-[9px] text-gray-300 truncate">{el.sub_text}</span>}
                    </div>
                  ) : el.type === 'up_next' ? (
                    <div className="flex flex-col justify-center">
                      <span className="text-[9px] font-bold text-orange-400 uppercase tracking-wider">UP NEXT</span>
                      <span className="text-xs font-bold truncate leading-tight">{el.text}</span>
                      {el.sub_text && <span className="text-[9px] text-gray-300 truncate">{el.sub_text}</span>}
                    </div>
                  ) : el.type === 'promo' ? (
                    <div className="flex flex-col justify-center">
                      <span className="text-[9px] font-bold text-amber-300 uppercase tracking-wider">SPECIAL EVENT</span>
                      <span className="text-xs font-bold truncate leading-tight">{el.text}</span>
                      {el.sub_text && <span className="text-[9px] text-purple-200 truncate">{el.sub_text}</span>}
                    </div>
                  ) : (
                    <div className="flex flex-col justify-center">
                      <span className="text-xs font-bold truncate leading-tight">{el.text}</span>
                      {el.sub_text && <span className="text-[9px] opacity-80 truncate">{el.sub_text}</span>}
                    </div>
                  )}

                  {/* Move tag badge on selected */}
                  {isSel && (
                    <div className="absolute -top-4 -right-1 bg-sky-500 text-black text-[8px] font-black px-1 rounded shadow">
                      {el.x},{el.y}
                    </div>
                  )}
                </div>
              );
            })}
          </div>

          {/* Quick Add Overlay Buttons Bar */}
          <div className="pt-2 border-t border-gray-800 flex flex-wrap items-center justify-between gap-2 text-xs">
            <span className="text-[11px] font-bold text-gray-400 uppercase tracking-wider">Add Overlay Layout:</span>
            <div className="flex flex-wrap items-center gap-1.5">
              <button
                onClick={() => handleAddElement("header_banner")}
                className="px-2.5 py-1 bg-[#1F2937] hover:bg-gray-700 text-sky-300 rounded text-[11px] font-semibold border border-gray-700 transition-colors"
              >
                + Header Banner
              </button>
              <button
                onClick={() => handleAddElement("footer_banner")}
                className="px-2.5 py-1 bg-[#1F2937] hover:bg-gray-700 text-sky-300 rounded text-[11px] font-semibold border border-gray-700 transition-colors"
              >
                + Footer Banner
              </button>
              <button
                onClick={() => handleAddElement("ticker")}
                className="px-2.5 py-1 bg-[#1F2937] hover:bg-gray-700 text-amber-300 rounded text-[11px] font-semibold border border-gray-700 transition-colors"
              >
                + News Ticker
              </button>
              <button
                onClick={() => handleAddElement("now_playing")}
                className="px-2.5 py-1 bg-[#1F2937] hover:bg-gray-700 text-emerald-300 rounded text-[11px] font-semibold border border-gray-700 transition-colors"
              >
                + Now Playing
              </button>
              <button
                onClick={() => handleAddElement("up_next")}
                className="px-2.5 py-1 bg-[#1F2937] hover:bg-gray-700 text-orange-300 rounded text-[11px] font-semibold border border-gray-700 transition-colors"
              >
                + Up Next
              </button>
              <button
                onClick={() => handleAddElement("promo")}
                className="px-2.5 py-1 bg-[#1F2937] hover:bg-gray-700 text-purple-300 rounded text-[11px] font-semibold border border-gray-700 transition-colors"
              >
                + Special Promo
              </button>
            </div>
          </div>
        </div>

        {/* Right: Selected Element Properties & SCTE-35 Breaks (4 cols) */}
        <div className="lg:col-span-4 space-y-4 text-xs overflow-y-auto">
          {/* Template Info Card */}
          <div className="bg-[#111827] border border-[#1F2937] rounded-lg p-3 space-y-2">
            <h3 className="text-xs font-bold text-white uppercase tracking-wider flex items-center gap-1.5">
              <Layout className="w-3.5 h-3.5 text-indigo-400" />
              <span>Template Settings</span>
            </h3>
            <div>
              <label className="block text-[11px] text-gray-400 mb-1">Template Name</label>
              <input
                type="text"
                value={templateName}
                onChange={(e) => setTemplateName(e.target.value)}
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white focus:outline-none focus:border-indigo-500 font-medium"
              />
            </div>
          </div>

          {/* Active Overlay Properties Editor */}
          {selectedElement ? (
            <div className="bg-[#111827] border border-[#1F2937] rounded-lg p-3 space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-xs font-bold text-sky-400 uppercase tracking-wider flex items-center gap-1.5">
                  <Sliders className="w-3.5 h-3.5" />
                  <span>Edit Element: {selectedElement.type.replace(/_/g, ' ')}</span>
                </h3>
                <button
                  onClick={() => handleRemoveElement(selectedElement.id)}
                  className="text-rose-400 hover:text-rose-300 text-[11px] font-semibold flex items-center gap-1"
                >
                  <Trash2 className="w-3 h-3" />
                  <span>Remove</span>
                </button>
              </div>

              <div>
                <label className="block text-[11px] text-gray-400 mb-1">Primary Display Text</label>
                <input
                  type="text"
                  value={selectedElement.text || ""}
                  onChange={(e) => updateSelectedElement({ text: e.target.value })}
                  className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white focus:outline-none focus:border-indigo-500"
                />
              </div>

              {selectedElement.type !== 'ticker' && (
                <div>
                  <label className="block text-[11px] text-gray-400 mb-1">Subtext / Secondary Line</label>
                  <input
                    type="text"
                    value={selectedElement.sub_text || ""}
                    onChange={(e) => updateSelectedElement({ sub_text: e.target.value })}
                    className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white focus:outline-none focus:border-indigo-500"
                  />
                </div>
              )}

              <div className="grid grid-cols-2 gap-2">
                <div>
                  <label className="block text-[11px] text-gray-400 mb-1">Entrance Animation</label>
                  <select
                    value={selectedElement.entrance_animation || "fade_in"}
                    onChange={(e) => updateSelectedElement({ entrance_animation: e.target.value })}
                    className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1.5 text-xs text-white"
                  >
                    <option value="fade_in">Fade In</option>
                    <option value="slide_in_left">Slide In Left</option>
                    <option value="slide_in_bottom">Slide In Bottom</option>
                    <option value="zoom_in">Zoom In Fade</option>
                    <option value="scroll_left">Scroll Left (Ticker)</option>
                    <option value="static">Static (No Motion)</option>
                  </select>
                </div>

                <div>
                  <label className="block text-[11px] text-gray-400 mb-1">Background Preset</label>
                  <select
                    value={selectedElement.background_color || "black@0.80"}
                    onChange={(e) => updateSelectedElement({ background_color: e.target.value })}
                    className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1.5 text-xs text-white"
                  >
                    <option value="black@0.80">Onyx Dark (80%)</option>
                    <option value="navy@0.85">Broadcast Navy (85%)</option>
                    <option value="darkblue@0.85">Deep Royal Blue</option>
                    <option value="purple@0.85">Regal Purple</option>
                    <option value="darkred@0.85">Crimson Red</option>
                  </select>
                </div>
              </div>

              {/* Exact Position & Dimensions */}
              <div className="pt-2 border-t border-gray-800 space-y-2">
                <div className="text-[11px] font-bold text-gray-300">Layout Coordinates (1920x1080)</div>
                <div className="grid grid-cols-4 gap-2 font-mono">
                  <div>
                    <label className="text-[9px] text-gray-500 block">X</label>
                    <input
                      type="number"
                      value={selectedElement.x || 0}
                      onChange={(e) => updateSelectedElement({ x: parseInt(e.target.value, 10) || 0 })}
                      className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1 text-xs text-white"
                    />
                  </div>
                  <div>
                    <label className="text-[9px] text-gray-500 block">Y</label>
                    <input
                      type="number"
                      value={selectedElement.y || 0}
                      onChange={(e) => updateSelectedElement({ y: parseInt(e.target.value, 10) || 0 })}
                      className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1 text-xs text-white"
                    />
                  </div>
                  <div>
                    <label className="text-[9px] text-gray-500 block">Width</label>
                    <input
                      type="number"
                      value={selectedElement.width || 0}
                      onChange={(e) => updateSelectedElement({ width: parseInt(e.target.value, 10) || 0 })}
                      className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1 text-xs text-white"
                    />
                  </div>
                  <div>
                    <label className="text-[9px] text-gray-500 block">Height</label>
                    <input
                      type="number"
                      value={selectedElement.height || 0}
                      onChange={(e) => updateSelectedElement({ height: parseInt(e.target.value, 10) || 0 })}
                      className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1 text-xs text-white"
                    />
                  </div>
                </div>
              </div>
            </div>
          ) : (
            <div className="p-4 bg-[#111827] border border-[#1F2937] rounded-lg text-center text-gray-400">
              Select or add an overlay element on the canvas to configure properties.
            </div>
          )}

          {/* SCTE-35 Commercial Insertion Card */}
          <div className="bg-[#111827] border border-[#1F2937] rounded-lg p-3 space-y-2 text-xs">
            <h3 className="text-xs font-bold text-white uppercase tracking-wider flex items-center justify-between">
              <span className="flex items-center gap-1.5">
                <Radio className="w-3.5 h-3.5 text-emerald-400" />
                <span>SCTE-35 Digital Ad Insertion</span>
              </span>
              <span className="text-[10px] font-mono text-emerald-400">DAI Cue Active</span>
            </h3>

            <div className="space-y-1.5">
              <div className="p-2 bg-[#1F2937] rounded border border-gray-700 flex items-center justify-between">
                <div>
                  <div className="font-semibold text-white">Pre-Roll Break (30s)</div>
                  <div className="text-[10px] text-gray-400 font-mono">Offset: 00:00:00 • SCTE-35 Cue: Yes</div>
                </div>
                <span className="px-1.5 py-0.5 rounded text-[9px] font-mono font-bold bg-emerald-500/20 text-emerald-300">
                  CUE
                </span>
              </div>
              <div className="p-2 bg-[#1F2937] rounded border border-gray-700 flex items-center justify-between">
                <div>
                  <div className="font-semibold text-white">Mid-Roll Break (60s)</div>
                  <div className="text-[10px] text-gray-400 font-mono">Offset: 00:30:00 • SCTE-35 Cue: Yes</div>
                </div>
                <span className="px-1.5 py-0.5 rounded text-[9px] font-mono font-bold bg-emerald-500/20 text-emerald-300">
                  CUE
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
