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
  ArrowRight,
  ArrowLeft,
  Copy,
  Search,
  Filter,
  AlertCircle,
  Film,
  ExternalLink,
  ChevronRight
} from 'lucide-react';
import { api } from '../api';

export const resolveAssetUrl = (path) => {
  if (!path) return '';
  if (path.startsWith('http://') || path.startsWith('https://') || path.startsWith('blob:') || path.startsWith('data:')) {
    return path;
  }
  const clean = path.replace(/^[\\\/]+/, '').replace(/\\/g, '/');
  return `/${clean}`;
};

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
  channels = [],
  onRefreshTemplates,
  onShowToast,
  t
}) {
  // Navigation: "list" (Templates Directory) vs "detail" (WYSIWYG CG Studio Editor)
  const [viewMode, setViewMode] = useState("list");
  const [templateSearch, setTemplateSearch] = useState("");

  // Detail View State
  const [activeTab, setActiveTab] = useState("logo"); // "logo", "now_playing", "banners", "commercials", "collision"
  const [activeTemplateId, setActiveTemplateId] = useState("");
  const [templateName, setTemplateName] = useState("");
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
  const [logoImgError, setLogoImgError] = useState(false);
  const [isUploadingLogo, setIsUploadingLogo] = useState(false);
  const logoInputRef = useRef(null);

  // Collision Resolution state
  const [collisionBehavior, setCollisionBehavior] = useState("alternate"); // "alternate", "priority"
  const [alternateDuration, setAlternateDuration] = useState(15);
  const [priorityOrder, setPriorityOrder] = useState(["header_banner", "now_playing", "up_next", "promo", "ticker"]);

  const [showSafeGuides, setShowSafeGuides] = useState(true);
  const [showAdCuesOnCanvas, setShowAdCuesOnCanvas] = useState(true);
  const [animKey, setAnimKey] = useState(0);
  const [isSaving, setIsSaving] = useState(false);

  // Interactive Canvas Drag & Resize State
  const canvasRef = useRef(null);
  const [interactionMode, setInteractionMode] = useState(null); // "drag" or "resize"
  const [activeTargetId, setActiveTargetId] = useState(null); // element id or "logo"
  const [interactionStart, setInteractionStart] = useState({ clientX: 0, clientY: 0 });
  const [initialBox, setInitialBox] = useState({ x: 0, y: 0, width: 0, height: 0 });

  // Reset logo image error when logoPath changes
  useEffect(() => {
    setLogoImgError(false);
  }, [logoPath]);

  // Sync active template when adTemplates change or activeTemplateId is set
  useEffect(() => {
    if (adTemplates.length > 0 && activeTemplateId) {
      const match = adTemplates.find((t) => t.id === activeTemplateId);
      if (match) {
        applyTemplateData(match);
      }
    }
  }, [adTemplates, activeTemplateId]);

  const applyTemplateData = (tmpl) => {
    setActiveTemplateId(tmpl.id);
    setTemplateName(tmpl.name || "Broadcast Layout Template");

    setLogoPath(tmpl.logo_path || "data/logos/channel_logo.png");
    setLogoPosition(tmpl.logo_position || "top-right");
    setLogoX(tmpl.logo_x ?? 1720);
    setLogoY(tmpl.logo_y ?? 40);
    setLogoWidth(tmpl.logo_width || 140);
    setLogoHeight(tmpl.logo_height || 90);
    setLogoOpacity(tmpl.logo_opacity ?? 0.90);
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
    if (elems.length > 0) {
      setSelectedElementId(elems[0].id);
    }
  };

  const handleOpenTemplateDetail = (tmpl) => {
    applyTemplateData(tmpl);
    setViewMode("detail");
  };

  const handleCreateNewTemplate = () => {
    const newId = `tmpl-${Date.now().toString(36)}`;
    setActiveTemplateId(newId);
    setTemplateName("New Broadcast Layout Template");
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
      { break_type: "pre_roll", offset_seconds: 0, duration_seconds: 15, scte35_cue: true, clips: [] },
      { break_type: "mid_roll", offset_seconds: 1800, duration_seconds: 60, scte35_cue: true, clips: [] }
    ]);
    setSelectedElementId(DEFAULT_OVERLAYS[0].id);
    setViewMode("detail");
    onShowToast("Initialized new on-air layout template", "info");
  };

  const handleDuplicateTemplate = async (sourceTmpl) => {
    try {
      const payload = {
        ...sourceTmpl,
        id: `tmpl-${Date.now().toString(36)}`,
        name: `Copy of ${sourceTmpl.name}`
      };
      const created = await api.createAdTemplate(payload);
      if (onRefreshTemplates) onRefreshTemplates();
      onShowToast(`Duplicated "${sourceTmpl.name}"`, "success");
      if (created?.id) {
        handleOpenTemplateDetail(created);
      }
    } catch (err) {
      onShowToast("Duplicate failed: " + err.message, "error");
    }
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
        template_type: "generic",
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

  const handleDeleteTemplate = async (templateIdToDelete) => {
    const targetId = templateIdToDelete || activeTemplateId;
    if (!targetId) return;
    try {
      await api.deleteAdTemplate(targetId);
      onShowToast("Deleted layout template", "info");
      if (onRefreshTemplates) onRefreshTemplates();
      if (targetId === activeTemplateId) {
        setViewMode("list");
      }
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
      setLogoImgError(false);
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
    const newId = `ov-${type.slice(0, 3)}-${Date.now().toString(36)}`;
    let newEl = {
      id: newId,
      type: type,
      text: "NEW BROADCAST OVERLAY",
      sub_text: "",
      x: 100,
      y: 100,
      width: 400,
      height: 80,
      background_color: "black@0.80",
      text_color: "white",
      font_size: 20,
      entrance_animation: "fade_in",
      is_active: true
    };

    switch (type) {
      case "header_banner":
        newEl.text = "NETWORK LIVE SPECIAL EVENT";
        newEl.sub_text = "SIMULCAST IN 4K DOLBY AUDIO";
        newEl.x = 0;
        newEl.y = 0;
        newEl.width = 1920;
        newEl.height = 60;
        newEl.background_color = "navy@0.85";
        break;
      case "footer_banner":
        newEl.text = "WWW.NETWORK.COM/STREAM • WATCH LIVE ANYWHERE";
        newEl.sub_text = "DOWNLOAD THE MCRFLOW BROADCAST APP";
        newEl.x = 0;
        newEl.y = 1010;
        newEl.width = 1920;
        newEl.height = 70;
        newEl.background_color = "navy@0.85";
        break;
      case "ticker":
        newEl.text = "BREAKING NEWS: LIVE BROADCAST FEED ACTIVE • ZERO-FRAME SWITCHING ENABLED";
        newEl.sub_text = "";
        newEl.x = 0;
        newEl.y = 1020;
        newEl.width = 1920;
        newEl.height = 60;
        newEl.background_color = "black@0.80";
        newEl.text_color = "yellow";
        newEl.entrance_animation = "scroll_left";
        break;
      case "now_playing":
        newEl.text = "NOW PLAYING: PRIME MOVIE BLOCK";
        newEl.sub_text = "2024 • ACTION THRILLER";
        newEl.x = 60;
        newEl.y = 80;
        newEl.width = 380;
        newEl.height = 85;
        newEl.entrance_animation = "slide_in_left";
        break;
      case "up_next":
        newEl.text = "UP NEXT: SPECIAL FEATURE";
        newEl.sub_text = "STARTING IN 15 MINUTES";
        newEl.x = 1480;
        newEl.y = 80;
        newEl.width = 380;
        newEl.height = 85;
        newEl.entrance_animation = "fade_in";
        break;
      case "promo":
        newEl.text = "SPECIAL WEEKEND BLOCKBUSTER";
        newEl.sub_text = "WATCH EXCLUSIVELY ON THIS CHANNEL";
        newEl.x = 1460;
        newEl.y = 840;
        newEl.width = 400;
        newEl.height = 95;
        newEl.background_color = "purple@0.85";
        newEl.text_color = "gold";
        newEl.entrance_animation = "zoom_in";
        break;
      case "lower_third":
        newEl.text = "LIVE INTERVIEW: MASTER CONTROL ROOM";
        newEl.sub_text = "NEW DELHI BROADCAST CENTER";
        newEl.x = 60;
        newEl.y = 920;
        newEl.width = 500;
        newEl.height = 90;
        newEl.background_color = "darkblue@0.85";
        newEl.entrance_animation = "slide_in_left";
        break;
      default:
        break;
    }

    setOverlayElements((prev) => [...prev, newEl]);
    setSelectedElementId(newId);
    onShowToast(`Added ${type.replace(/_/g, ' ')} placement`, "info");
  };

  const handleRemoveElement = (id) => {
    setOverlayElements((prev) => prev.filter((el) => el.id !== id));
    if (selectedElementId === id) {
      const remaining = overlayElements.filter((el) => el.id !== id);
      setSelectedElementId(remaining[0]?.id || "");
    }
  };

  // Commercial Break Handlers
  const handleAddCommercialBreak = (type = "mid_roll") => {
    const newBreak = {
      break_type: type,
      offset_seconds: type === "pre_roll" ? 0 : 900,
      duration_seconds: 30,
      scte35_cue: true,
      clips: []
    };
    setCommercialBreaks((prev) => [...prev, newBreak]);
    onShowToast(`Added ${type.replace(/_/g, ' ')} break slot`, "info");
  };

  const handleRemoveCommercialBreak = (index) => {
    setCommercialBreaks((prev) => prev.filter((_, idx) => idx !== index));
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

  const isElementColliding = (elementId) => {
    return collisions.some((col) => col.a.id === elementId || col.b.id === elementId);
  };

  const getElementAnimationClass = (animationType) => {
    switch (animationType) {
      case 'slide_in_left':
        return 'animate-in slide-in-from-left duration-500';
      case 'slide_in_bottom':
        return 'animate-in slide-in-from-bottom duration-500';
      case 'zoom_in':
        return 'animate-in zoom-in-75 duration-300';
      case 'scroll_left':
        return 'animate-pulse';
      case 'fade_in':
      default:
        return 'animate-in fade-in duration-300';
    }
  };

  const getAssignedChannelsForTemplate = (tmplId) => {
    return channels.filter((ch) => ch.ad_template_id === tmplId);
  };

  // Filter templates for List view
  const filteredTemplates = adTemplates.filter((t) => {
    if (!templateSearch.trim()) return true;
    const q = templateSearch.toLowerCase();
    return t.name?.toLowerCase().includes(q) || t.id?.toLowerCase().includes(q);
  });

  // ==========================================
  // VIEW MODE 1: TEMPLATES DIRECTORY (LIST FIRST)
  // ==========================================
  if (viewMode === "list") {
    return (
      <div className="space-y-6 pb-12 animate-in fade-in duration-300">
        {/* Top Header & Actions Bar */}
        <div className="bg-[#111622] border border-gray-800 rounded-xl p-5 shadow-lg flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div>
            <div className="flex items-center gap-2">
              <Sparkles className="w-5 h-5 text-purple-400" />
              <h1 className="text-xl font-bold text-white tracking-wide">Ad & Layout Management</h1>
              <span className="px-2 py-0.5 rounded text-[11px] font-mono bg-purple-950/70 border border-purple-700/50 text-purple-300">
                Directory
              </span>
            </div>
            <p className="text-xs text-gray-400 mt-1">
              Centralized station branding watermark, dynamic Now Playing / Up Next graphics, commercial insertions, and collision rules
            </p>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            <div className="relative min-w-[220px]">
              <Search className="w-4 h-4 text-gray-400 absolute left-3 top-2.5" />
              <input
                type="text"
                value={templateSearch}
                onChange={(e) => setTemplateSearch(e.target.value)}
                placeholder="Search templates..."
                className="w-full bg-[#182030] border border-gray-700 rounded-lg pl-9 pr-3 py-2 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-purple-500"
              />
            </div>

            <button
              onClick={handleCreateNewTemplate}
              className="px-4 py-2 bg-purple-600 hover:bg-purple-500 text-white rounded-lg text-xs font-semibold flex items-center gap-1.5 shadow-md shadow-purple-900/30 transition-all shrink-0"
            >
              <Plus className="w-4 h-4" />
              Create Template
            </button>
          </div>
        </div>

        {/* Metrics Overview Row */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
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
            <div className="p-2.5 bg-blue-950/60 border border-blue-800/40 rounded-lg text-blue-400">
              <Tv className="w-5 h-5" />
            </div>
            <div>
              <div className="text-lg font-bold text-white font-mono">
                {channels.filter((c) => c.ad_template_id).length}
              </div>
              <div className="text-[11px] text-gray-400">Assigned Channels</div>
            </div>
          </div>

          <div className="bg-[#111622] border border-gray-800 rounded-xl p-4 shadow-sm flex items-center gap-3">
            <div className="p-2.5 bg-indigo-950/60 border border-indigo-800/40 rounded-lg text-indigo-400">
              <Layers className="w-5 h-5" />
            </div>
            <div>
              <div className="text-lg font-bold text-white font-mono">
                {adTemplates.reduce((acc, t) => acc + (t.overlay_elements?.length || 0), 0)}
              </div>
              <div className="text-[11px] text-gray-400">Total Active Layers</div>
            </div>
          </div>

          <div className="bg-[#111622] border border-gray-800 rounded-xl p-4 shadow-sm flex items-center gap-3">
            <div className="p-2.5 bg-emerald-950/60 border border-emerald-800/40 rounded-lg text-emerald-400">
              <Shield className="w-5 h-5" />
            </div>
            <div>
              <div className="text-lg font-bold text-emerald-400 font-mono">ACTIVE</div>
              <div className="text-[11px] text-gray-400">Collision Resolver</div>
            </div>
          </div>
        </div>

        {/* Templates Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-5">
          {filteredTemplates.map((tmpl) => {
            const assignedChs = getAssignedChannelsForTemplate(tmpl.id);
            const overlaysCount = tmpl.overlay_elements?.length || 0;
            const breaksCount = tmpl.commercial_breaks?.length || 0;
            const tmplLogoUrl = resolveAssetUrl(tmpl.logo_path || "data/logos/channel_logo.png");

            return (
              <div
                key={tmpl.id}
                className="bg-[#111622] border border-gray-800 hover:border-purple-600/70 rounded-xl p-5 shadow-lg flex flex-col justify-between transition-all group"
              >
                <div>
                  {/* Miniature Visual Layout Preview */}
                  <div className="aspect-video w-full bg-[#0a0d14] rounded-lg border border-gray-800 p-2 relative overflow-hidden mb-4 group-hover:border-purple-800/50 transition-colors">
                    {/* Header banner bar miniature */}
                    <div className="absolute top-0 inset-x-0 h-1.5 bg-blue-700/80" />

                    {/* Logo miniature */}
                    <div
                      className={`absolute w-7 h-4 bg-purple-950/80 border border-purple-500/70 rounded-xs flex items-center justify-center overflow-hidden ${
                        tmpl.logo_position === "top-left"
                          ? "top-2.5 left-2.5"
                          : tmpl.logo_position === "bottom-right"
                          ? "bottom-2.5 right-2.5"
                          : tmpl.logo_position === "bottom-left"
                          ? "bottom-2.5 left-2.5"
                          : "top-2.5 right-2.5"
                      }`}
                    >
                      <img
                        src={tmplLogoUrl}
                        alt="Logo"
                        className="w-full h-full object-contain"
                        onError={(e) => { e.target.style.display = 'none'; }}
                      />
                    </div>

                    {/* Now Playing miniature */}
                    <div className="absolute top-4 left-2.5 w-16 h-3 bg-black/80 border border-blue-500/60 rounded-xs flex items-center px-1">
                      <span className="text-[6px] text-blue-300 font-mono font-bold truncate">NOW PLAYING</span>
                    </div>

                    {/* Up Next miniature */}
                    <div className="absolute top-4 right-11 w-16 h-3 bg-black/80 border border-amber-500/60 rounded-xs flex items-center px-1">
                      <span className="text-[6px] text-amber-300 font-mono font-bold truncate">UP NEXT</span>
                    </div>

                    {/* Ticker bar miniature */}
                    <div className="absolute bottom-0 inset-x-0 h-2 bg-black/90 border-t border-yellow-500/50 flex items-center px-1">
                      <span className="text-[6px] text-yellow-300 font-mono truncate">TICKER CRAWL</span>
                    </div>
                  </div>

                  {/* Card Title & Badges */}
                  <div className="flex items-start justify-between gap-2 mb-2">
                    <h3 className="text-sm font-bold text-white group-hover:text-purple-300 transition-colors line-clamp-1">
                      {tmpl.name}
                    </h3>
                  </div>

                  {/* Component Badges */}
                  <div className="flex flex-wrap items-center gap-1.5 mb-3">
                    <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-purple-950/60 border border-purple-800/40 text-purple-300">
                      Watermark: {tmpl.logo_position || 'top-right'}
                    </span>
                    <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-blue-950/60 border border-blue-800/40 text-blue-300">
                      {overlaysCount} Overlays
                    </span>
                    <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-emerald-950/60 border border-emerald-800/40 text-emerald-300">
                      {breaksCount} Ad Breaks
                    </span>
                  </div>

                  {/* Assigned Channels */}
                  <div className="text-[11px] text-gray-400 mb-4 bg-[#141b2b] rounded-lg p-2 border border-gray-800/70">
                    <span className="text-gray-500 block mb-0.5 font-medium">Assigned Channels:</span>
                    {assignedChs.length > 0 ? (
                      <div className="flex flex-wrap gap-1">
                        {assignedChs.map((c) => (
                          <span key={c.id} className="text-white font-medium bg-gray-800 px-1.5 py-0.5 rounded text-[10px]">
                            {c.name} ({c.call_sign || `CH-${c.lcn}`})
                          </span>
                        ))}
                      </div>
                    ) : (
                      <span className="text-gray-500 italic">Not currently assigned to any channel</span>
                    )}
                  </div>
                </div>

                {/* Card Actions */}
                <div className="pt-3 border-t border-gray-800 flex items-center justify-between gap-2">
                  <div className="flex items-center gap-1.5">
                    <button
                      onClick={() => handleDuplicateTemplate(tmpl)}
                      className="p-1.5 bg-gray-800 hover:bg-gray-700 text-gray-300 rounded-lg transition-colors"
                      title="Duplicate Template"
                    >
                      <Copy className="w-3.5 h-3.5" />
                    </button>
                    {adTemplates.length > 1 && (
                      <button
                        onClick={() => handleDeleteTemplate(tmpl.id)}
                        className="p-1.5 bg-red-950/40 hover:bg-red-900/60 text-red-400 rounded-lg transition-colors border border-red-800/40"
                        title="Delete Template"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    )}
                  </div>

                  <button
                    onClick={() => handleOpenTemplateDetail(tmpl)}
                    className="px-3 py-1.5 bg-purple-600 hover:bg-purple-500 text-white rounded-lg text-xs font-semibold flex items-center gap-1 transition-all shadow-sm"
                  >
                    Open in CG Studio
                    <ChevronRight className="w-3.5 h-3.5" />
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    );
  }

  // ==========================================
  // VIEW MODE 2: WYSIWYG TEMPLATE EDITOR (DETAIL)
  // ==========================================
  const resolvedLogoUrl = resolveAssetUrl(logoPath);

  return (
    <div className="space-y-6 pb-12 animate-in fade-in duration-300">
      {/* Top Header & Breadcrumb Bar */}
      <div className="bg-[#111622] border border-gray-800 rounded-xl p-4 shadow-lg flex flex-col md:flex-row md:items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <button
            onClick={() => setViewMode("list")}
            className="p-2 bg-gray-800 hover:bg-gray-700 text-gray-200 border border-gray-700 rounded-lg transition-colors flex items-center gap-1.5 text-xs font-semibold"
          >
            <ArrowLeft className="w-4 h-4" />
            Templates Directory
          </button>

          <div className="h-6 w-px bg-gray-700" />

          <div>
            <div className="flex items-center gap-2">
              <Sparkles className="w-4 h-4 text-purple-400" />
              <h1 className="text-base font-bold text-white tracking-wide">WYSIWYG CG Studio</h1>
              <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-purple-950/70 border border-purple-700/50 text-purple-300">
                Template Editor
              </span>
            </div>
            <p className="text-[11px] text-gray-400">
              Editing: <span className="text-purple-300 font-semibold">{templateName}</span>
            </p>
          </div>
        </div>

        {/* Template Controls */}
        <div className="flex flex-wrap items-center gap-2">
          <select
            value={activeTemplateId}
            onChange={(e) => {
              const tmpl = adTemplates.find((t) => t.id === e.target.value);
              if (tmpl) applyTemplateData(tmpl);
            }}
            className="bg-[#182030] border border-gray-700 text-white rounded-lg px-3 py-2 text-xs font-medium focus:ring-2 focus:ring-purple-500 focus:outline-none"
          >
            {adTemplates.map((t) => (
              <option key={t.id} value={t.id}>
                {t.name}
              </option>
            ))}
          </select>

          <button
            onClick={handleSaveTemplate}
            disabled={isSaving}
            className="px-4 py-2 bg-purple-600 hover:bg-purple-500 text-white rounded-lg text-xs font-semibold flex items-center gap-1.5 shadow-md shadow-purple-900/30 transition-all disabled:opacity-50"
          >
            <Save className="w-3.5 h-3.5" />
            Save Template
          </button>

          {adTemplates.length > 1 && (
            <button
              onClick={() => handleDeleteTemplate(activeTemplateId)}
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
          <div className="flex-1 max-w-md">
            <label className="block text-[11px] font-medium text-gray-400 mb-1">Template Name</label>
            <input
              type="text"
              value={templateName}
              onChange={(e) => setTemplateName(e.target.value)}
              className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-1.5 text-sm text-white font-semibold focus:outline-none focus:border-purple-500"
            />
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
                className="rounded accent-purple-600"
              />
              <span>Action/Title Safe Guides</span>
            </label>
            <label className="flex items-center gap-1.5 text-xs text-gray-300 cursor-pointer">
              <input
                type="checkbox"
                checked={showAdCuesOnCanvas}
                onChange={(e) => setShowAdCuesOnCanvas(e.target.checked)}
                className="rounded accent-emerald-600"
              />
              <span>Preview Ad Break Cues</span>
            </label>
            <button
              onClick={() => setAnimKey((k) => k + 1)}
              className="p-1.5 bg-gray-800 hover:bg-gray-700 text-gray-300 rounded text-xs flex items-center gap-1 transition-colors"
              title="Re-trigger Entrance Animations"
            >
              <RefreshCw className="w-3.5 h-3.5" />
              <span>Replay</span>
            </button>
          </div>
        </div>

        {/* Canvas Display Viewport */}
        <div
          ref={canvasRef}
          className="relative w-full aspect-video bg-gradient-to-br from-[#070a10] to-[#121722] rounded-lg border-2 border-gray-800 overflow-hidden shadow-2xl select-none"
        >
          {/* Action Safe (90%) and Title Safe (80%) Guides */}
          {showSafeGuides && (
            <>
              <div
                className="absolute inset-[5%] border border-blue-500/20 pointer-events-none z-0"
                title="Action Safe Area (90%)"
              />
              <div
                className="absolute inset-[10%] border border-dashed border-green-500/20 pointer-events-none z-0"
                title="Title Safe Area (80%)"
              />
            </>
          )}

          {/* Background Grid Pattern */}
          <div className="absolute inset-0 bg-[linear-gradient(to_right,#1f293d0f_1px,transparent_1px),linear-gradient(to_bottom,#1f293d0f_1px,transparent_1px)] bg-[size:4rem_4rem] pointer-events-none" />

          {/* Playout Simulated Test Card Watermark */}
          <div className="absolute inset-0 flex items-center justify-center opacity-5 pointer-events-none">
            <span className="text-7xl font-black tracking-widest uppercase font-mono">MCRFLOW LIVE</span>
          </div>

          {/* Preview Ad Break Cues on Canvas */}
          {showAdCuesOnCanvas && commercialBreaks.map((brk, idx) => (
            <div
              key={`cue-${idx}`}
              className="absolute left-4 top-4 bg-emerald-950/90 border border-emerald-500 rounded px-2.5 py-1 text-[10px] font-mono font-bold text-emerald-300 flex items-center gap-1.5 shadow-lg z-20 pointer-events-none animate-in fade-in"
            >
              <Sparkles className="w-3 h-3 text-emerald-400" />
              <span>{brk.break_type.toUpperCase()} AD CUE • {brk.duration_seconds}s {brk.scte35_cue ? "• SCTE-35" : ""}</span>
            </div>
          ))}

          {/* Station Logo / Watermark Bug */}
          {logoPath && (
            <div
              key={`logo-${animKey}`}
              onMouseDown={(e) => startDrag(e, "logo", { x: logoX, y: logoY, width: logoWidth, height: logoHeight })}
              className={`absolute cursor-move group transition-shadow select-none ${
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
              <div className={`w-full h-full relative rounded overflow-hidden flex items-center justify-center ${
                activeTargetId === "logo" ? "bg-black/40 border border-purple-500" : "border border-purple-500/50 hover:border-purple-400"
              }`}>
                {logoPath && !logoImgError ? (
                  <img
                    src={resolvedLogoUrl}
                    alt="Station Watermark"
                    className={`w-full h-full pointer-events-none select-none drop-shadow-md ${
                      logoFit === "cover" ? "object-cover" : "object-contain"
                    }`}
                    onError={() => setLogoImgError(true)}
                  />
                ) : (
                  <div className="w-full h-full bg-black/70 border border-dashed border-purple-500/60 flex flex-col items-center justify-center p-1 text-center">
                    <ImageIcon className="w-4 h-4 text-purple-400 mb-0.5" />
                    <span className="text-[9px] font-bold text-purple-200 tracking-wider font-mono uppercase truncate max-w-full px-1">
                      {logoPath ? logoPath.split('/').pop() : "STATION LOGO"}
                    </span>
                  </div>
                )}
                {/* Resize Handle */}
                <div
                  onMouseDown={(e) => {
                    e.stopPropagation();
                    startResize(e, "logo", { x: logoX, y: logoY, width: logoWidth, height: logoHeight });
                  }}
                  className="absolute bottom-0 right-0 w-3.5 h-3.5 bg-purple-500 cursor-nwse-resize rounded-tl shadow z-10 opacity-0 group-hover:opacity-100 transition-opacity"
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
                className={`absolute cursor-move group transition-shadow select-none ${
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

                  {/* Header Tag for Now Playing & Up Next */}
                  {el.type === "now_playing" && (
                    <div className="flex items-center gap-1 mb-0.5">
                      <span className="w-1.5 h-1.5 rounded-full bg-red-500 animate-pulse" />
                      <span className="text-[8px] font-mono font-bold text-red-300 uppercase tracking-wider">NOW PLAYING</span>
                    </div>
                  )}

                  {el.type === "up_next" && (
                    <div className="flex items-center gap-1 mb-0.5">
                      <Clock className="w-2.5 h-2.5 text-amber-400" />
                      <span className="text-[8px] font-mono font-bold text-amber-300 uppercase tracking-wider">UP NEXT</span>
                    </div>
                  )}

                  {/* Headline & Subtitle typography */}
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
                    onMouseDown={(e) => {
                      e.stopPropagation();
                      startResize(e, el.id, { x: el.x, y: el.y, width: el.width, height: el.height });
                    }}
                    className="absolute bottom-0 right-0 w-3.5 h-3.5 bg-blue-500 cursor-nwse-resize rounded-tl shadow z-10 opacity-0 group-hover:opacity-100 transition-opacity"
                    title="Drag to Resize"
                  />
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Generic Placements Configuration Tabs */}
      <div className="flex flex-wrap items-center gap-2 border-b border-gray-800 pb-3">
        <button
          onClick={() => setActiveTab("logo")}
          className={`px-4 py-2 rounded-lg text-xs font-semibold flex items-center gap-2 transition-colors ${
            activeTab === "logo"
              ? "bg-purple-600 text-white shadow-md shadow-purple-900/30"
              : "bg-[#111622] text-gray-400 hover:text-white border border-gray-800"
          }`}
        >
          <ImageIcon className="w-4 h-4" />
          Logo & Watermark
        </button>

        <button
          onClick={() => setActiveTab("now_playing")}
          className={`px-4 py-2 rounded-lg text-xs font-semibold flex items-center gap-2 transition-colors ${
            activeTab === "now_playing"
              ? "bg-blue-600 text-white shadow-md shadow-blue-900/30"
              : "bg-[#111622] text-gray-400 hover:text-white border border-gray-800"
          }`}
        >
          <Tv className="w-4 h-4" />
          Now Playing & Up Next
        </button>

        <button
          onClick={() => setActiveTab("banners")}
          className={`px-4 py-2 rounded-lg text-xs font-semibold flex items-center gap-2 transition-colors ${
            activeTab === "banners"
              ? "bg-indigo-600 text-white shadow-md shadow-indigo-900/30"
              : "bg-[#111622] text-gray-400 hover:text-white border border-gray-800"
          }`}
        >
          <Layers className="w-4 h-4" />
          Banners, Crawls & Lower-Thirds
        </button>

        <button
          onClick={() => setActiveTab("commercials")}
          className={`px-4 py-2 rounded-lg text-xs font-semibold flex items-center gap-2 transition-colors ${
            activeTab === "commercials"
              ? "bg-emerald-600 text-white shadow-md shadow-emerald-900/30"
              : "bg-[#111622] text-gray-400 hover:text-white border border-gray-800"
          }`}
        >
          <Film className="w-4 h-4" />
          Commercial Breaks & Pre-Roll
        </button>

        <button
          onClick={() => setActiveTab("collision")}
          className={`px-4 py-2 rounded-lg text-xs font-semibold flex items-center gap-2 transition-colors ${
            activeTab === "collision"
              ? "bg-amber-600 text-white shadow-md shadow-amber-900/30"
              : "bg-[#111622] text-gray-400 hover:text-white border border-gray-800"
          }`}
        >
          <Shield className="w-4 h-4" />
          Collision & Layer Rules
          {collisions.length > 0 && (
            <span className="w-2 h-2 rounded-full bg-amber-400 animate-pulse" />
          )}
        </button>
      </div>

      {/* TAB 1: Logo & Watermark */}
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
            {/* Logo File, Upload & Live Preview */}
            <div className="space-y-4">
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
                    accept="image/png,image/jpeg,image/webp,image/svg+xml"
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

              {/* Logo Live Image Preview Box */}
              <div className="p-3 bg-[#182030] border border-gray-800 rounded-xl flex items-center gap-3">
                <div className="w-20 h-14 bg-black/60 border border-gray-700 rounded-lg flex items-center justify-center p-1 overflow-hidden shrink-0">
                  {logoPath && !logoImgError ? (
                    <img src={resolvedLogoUrl} alt="Logo Preview" className="max-w-full max-h-full object-contain" onError={() => setLogoImgError(true)} />
                  ) : (
                    <ImageIcon className="w-6 h-6 text-purple-400/60" />
                  )}
                </div>
                <div className="min-w-0 flex-1">
                  <div className="text-xs font-bold text-white truncate">{logoPath || "No logo file"}</div>
                  <div className="text-[11px] text-gray-400 mt-0.5">
                    Position: <span className="text-purple-300 font-mono capitalize">{logoPosition}</span> ({logoX}x{logoY}px) • Size: {logoWidth}x{logoHeight}px • Opacity: {Math.round(logoOpacity * 100)}%
                  </div>
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
            <div className="space-y-4">
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
            <span className="text-xs text-gray-400">Automatically derived from active and queued schedule items</span>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {/* Now Playing Settings */}
            {(() => {
              const npElem = overlayElements.find((el) => el.type === "now_playing");
              return (
                <div className="p-4 bg-[#141b2b] border border-gray-800 rounded-xl space-y-3">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-bold text-blue-400 uppercase tracking-wide flex items-center gap-1.5">
                      <span className="w-2 h-2 rounded-full bg-red-500 animate-pulse" />
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
                    <span className="text-xs font-bold text-amber-400 uppercase tracking-wide flex items-center gap-1.5">
                      <Clock className="w-3.5 h-3.5 text-amber-400" />
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

      {/* TAB 3: Banners, Crawls & Lower-Thirds */}
      {activeTab === "banners" && (
        <div className="bg-[#111622] border border-gray-800 rounded-xl p-5 shadow-lg space-y-5 animate-in fade-in duration-200">
          <div className="flex items-center justify-between pb-3 border-b border-gray-800">
            <h3 className="text-sm font-bold text-white uppercase tracking-wider flex items-center gap-2">
              <Layers className="w-4 h-4 text-indigo-400" />
              On-Air Graphics: Banners, Crawls, Lower-Thirds & Promos
            </h3>
            <div className="flex flex-wrap items-center gap-1.5">
              <button
                type="button"
                onClick={() => handleAddElement("header_banner")}
                className="px-2.5 py-1 bg-gray-800 hover:bg-gray-700 text-gray-200 rounded text-xs border border-gray-700 flex items-center gap-1"
              >
                <Plus className="w-3 h-3" /> Header Banner
              </button>
              <button
                type="button"
                onClick={() => handleAddElement("footer_banner")}
                className="px-2.5 py-1 bg-gray-800 hover:bg-gray-700 text-gray-200 rounded text-xs border border-gray-700 flex items-center gap-1"
              >
                <Plus className="w-3 h-3" /> Footer Banner
              </button>
              <button
                type="button"
                onClick={() => handleAddElement("ticker")}
                className="px-2.5 py-1 bg-gray-800 hover:bg-gray-700 text-gray-200 rounded text-xs border border-gray-700 flex items-center gap-1"
              >
                <Plus className="w-3 h-3" /> Ticker Crawl
              </button>
              <button
                type="button"
                onClick={() => handleAddElement("promo")}
                className="px-2.5 py-1 bg-gray-800 hover:bg-gray-700 text-gray-200 rounded text-xs border border-gray-700 flex items-center gap-1"
              >
                <Plus className="w-3 h-3" /> Promo Bump
              </button>
              <button
                type="button"
                onClick={() => handleAddElement("lower_third")}
                className="px-2.5 py-1 bg-gray-800 hover:bg-gray-700 text-gray-200 rounded text-xs border border-gray-700 flex items-center gap-1"
              >
                <Plus className="w-3 h-3" /> Lower Third
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

      {/* TAB 4: Commercial Breaks & Pre-Roll / Mid-Roll Insertion */}
      {activeTab === "commercials" && (
        <div className="bg-[#111622] border border-gray-800 rounded-xl p-5 shadow-lg space-y-5 animate-in fade-in duration-200">
          <div className="flex items-center justify-between pb-3 border-b border-gray-800">
            <div>
              <h3 className="text-sm font-bold text-white uppercase tracking-wider flex items-center gap-2">
                <Film className="w-4 h-4 text-emerald-400" />
                Commercial Breaks & SCTE-35 Digital Ad Insertion (DAI)
              </h3>
              <p className="text-xs text-gray-400 mt-0.5">
                Configure pre-roll bumpers, mid-roll ad pods, and SCTE-35 splice cues applied to scheduled programs
              </p>
            </div>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => handleAddCommercialBreak("pre_roll")}
                className="px-3 py-1.5 bg-emerald-700 hover:bg-emerald-600 text-white rounded text-xs font-semibold flex items-center gap-1"
              >
                <Plus className="w-3.5 h-3.5" /> + Pre-Roll Pod
              </button>
              <button
                type="button"
                onClick={() => handleAddCommercialBreak("mid_roll")}
                className="px-3 py-1.5 bg-emerald-700 hover:bg-emerald-600 text-white rounded text-xs font-semibold flex items-center gap-1"
              >
                <Plus className="w-3.5 h-3.5" /> + Mid-Roll Pod
              </button>
            </div>
          </div>

          {commercialBreaks.length === 0 ? (
            <div className="p-8 text-center text-gray-500 border border-dashed border-gray-800 rounded-xl">
              <Film className="w-8 h-8 text-gray-600 mx-auto mb-2" />
              <p className="text-sm font-semibold text-gray-400">No commercial breaks configured in this template</p>
              <p className="text-xs text-gray-500 mt-1">Add pre-roll or mid-roll slots to trigger automated ad insertions and SCTE-35 cues</p>
            </div>
          ) : (
            <div className="space-y-3">
              {commercialBreaks.map((brk, idx) => (
                <div
                  key={idx}
                  className="p-4 bg-[#141b2b] border border-gray-800 rounded-xl flex flex-col md:flex-row md:items-center justify-between gap-4"
                >
                  <div className="flex items-center gap-3">
                    <span className="px-2.5 py-1 rounded text-xs font-mono font-bold bg-emerald-950/80 border border-emerald-700 text-emerald-300 uppercase">
                      {brk.break_type}
                    </span>
                    <div>
                      <div className="text-xs font-bold text-white">
                        Break #{idx + 1} • Duration: {brk.duration_seconds}s
                        {brk.break_type === "mid_roll" && ` • Offset: ${Math.floor(brk.offset_seconds / 60)}m ${brk.offset_seconds % 60}s`}
                      </div>
                      <div className="text-[11px] text-gray-400 flex items-center gap-2 mt-0.5">
                        <span className={brk.scte35_cue ? "text-emerald-400 font-semibold" : "text-gray-500"}>
                          {brk.scte35_cue ? "✓ SCTE-35 Cue Enabled" : "No SCTE-35"}
                        </span>
                        <span>•</span>
                        <span>{brk.clips?.length || 0} designated clips</span>
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center gap-3">
                    <div>
                      <label className="block text-[10px] text-gray-400 mb-0.5">Duration (sec)</label>
                      <input
                        type="number"
                        min="5"
                        max="300"
                        value={brk.duration_seconds}
                        onChange={(e) => {
                          const val = parseInt(e.target.value) || 15;
                          setCommercialBreaks((prev) =>
                            prev.map((b, i) => i === idx ? { ...b, duration_seconds: val } : b)
                          );
                        }}
                        className="w-20 bg-[#182030] border border-gray-700 rounded px-2 py-1 text-xs text-white font-mono"
                      />
                    </div>

                    {brk.break_type === "mid_roll" && (
                      <div>
                        <label className="block text-[10px] text-gray-400 mb-0.5">Offset (sec)</label>
                        <input
                          type="number"
                          min="0"
                          value={brk.offset_seconds}
                          onChange={(e) => {
                            const val = parseInt(e.target.value) || 0;
                            setCommercialBreaks((prev) =>
                              prev.map((b, i) => i === idx ? { ...b, offset_seconds: val } : b)
                            );
                          }}
                          className="w-24 bg-[#182030] border border-gray-700 rounded px-2 py-1 text-xs text-white font-mono"
                        />
                      </div>
                    )}

                    <div className="flex items-center gap-2 pt-3">
                      <label className="flex items-center gap-1.5 text-xs text-gray-300 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={brk.scte35_cue}
                          onChange={(e) => {
                            const checked = e.target.checked;
                            setCommercialBreaks((prev) =>
                              prev.map((b, i) => i === idx ? { ...b, scte35_cue: checked } : b)
                            );
                          }}
                          className="rounded accent-emerald-500"
                        />
                        <span className="text-[11px]">SCTE-35</span>
                      </label>

                      <button
                        type="button"
                        onClick={() => handleRemoveCommercialBreak(idx)}
                        className="p-1.5 text-gray-500 hover:text-red-400 ml-2"
                        title="Delete Break"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* TAB 5: Collision & Layer Rules */}
      {activeTab === "collision" && (
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
