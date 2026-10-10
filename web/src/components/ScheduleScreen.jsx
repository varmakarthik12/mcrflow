import React, { useState, useEffect, useRef } from 'react';
import {
  Calendar,
  Clock,
  Plus,
  Trash2,
  Download,
  Film,
  Search,
  FolderOpen,
  FileVideo,
  Check,
  AlertCircle,
  AlertTriangle,
  X,
  FastForward,
  Scissors,
  ArrowRight,
  ShieldAlert,
  Loader2,
  ChevronRight,
  Sparkles,
  Layers,
  Radio,
  Tv,
  Eye,
  Tag
} from 'lucide-react';
import { api } from '../api';
import {
  formatTimeInTimezone,
  formatDateInTimezone,
  createIsoInTimezone,
  DEFAULT_TIMEZONE
} from '../utils/timezone';

const PROMO_PRESETS = [
  { title: "Diwali Movie Festival Premiere", subtext: "Exclusive 4K Broadcast This Weekend" },
  { title: "Weekend Mega Blockbuster", subtext: "Commercial-Free Television Event" },
  { title: "Director's Special Cut", subtext: "Original Theatrical Version In Dolby Atmos" },
  { title: "Live Sports Special Highlights", subtext: "Action Recap & Expert Analysis" },
  { title: "Prime Time Super Premiere", subtext: "Streaming In High Definition Tonight" }
];

const getFallbackPoster = (title = "Broadcast Feature", year = "") => {
  const initials = (title || "MCR")
    .split(/\s+/)
    .slice(0, 2)
    .map((w) => w[0]?.toUpperCase() || "")
    .join("") || "TV";
  const safeTitle = (title || "").slice(0, 22);
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 300 450" width="300" height="450"><defs><linearGradient id="g" x1="0%" y1="0%" x2="100%" y2="100%"><stop offset="0%" stop-color="#1e293b"/><stop offset="100%" stop-color="#0f172a"/></linearGradient></defs><rect width="300" height="450" fill="url(#g)" rx="12"/><circle cx="150" cy="180" r="54" fill="#3b82f6" opacity="0.25"/><text x="150" y="195" font-family="system-ui, sans-serif" font-size="40" font-weight="bold" fill="#60a5fa" text-anchor="middle">${initials}</text><text x="150" y="280" font-family="system-ui, sans-serif" font-size="16" font-weight="bold" fill="#f8fafc" text-anchor="middle">${safeTitle}</text><text x="150" y="310" font-family="system-ui, sans-serif" font-size="14" fill="#94a3b8" text-anchor="middle">${year}</text></svg>`;
  return "data:image/svg+xml;utf8," + encodeURIComponent(svg);
};

export function ScheduleScreen({
  channels = [],
  activeChannelId,
  onSwitchChannel,
  scheduleItems = [],
  onRefreshSchedule,
  onShowToast,
  adTemplates = [],
  broadcastTimezone = DEFAULT_TIMEZONE,
  t
}) {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [currentPath, setCurrentPath] = useState("");
  const [fileList, setFileList] = useState([]);
  const [fileFilterQuery, setFileFilterQuery] = useState("");
  const [selectedFile, setSelectedFile] = useState(null);

  // Multi-Day Date Navigation State (Synchronized with Broadcast Timezone)
  const todayStr = formatDateInTimezone(new Date(), broadcastTimezone);
  const [selectedDate, setSelectedDate] = useState(todayStr);

  // Form State
  const [title, setTitle] = useState("");
  const [startTime, setStartTime] = useState("16:00:00");
  const [duration, setDuration] = useState("02:15:00");
  const [endTime, setEndTime] = useState("18:15:00");
  const [selectedAdTemplateId, setSelectedAdTemplateId] = useState("");
  const [specialPromoTitle, setSpecialPromoTitle] = useState("");
  const [specialPromoSubtext, setSpecialPromoSubtext] = useState("");
  const [showPromoSuggestions, setShowPromoSuggestions] = useState(false);

  // TMDb Typeahead State
  const [tmdbQuery, setTmdbQuery] = useState("");
  const [tmdbHits, setTmdbHits] = useState([]);
  const [isTmdbDropdownOpen, setIsTmdbDropdownOpen] = useState(false);
  const [isSearchingTmdb, setIsSearchingTmdb] = useState(false);
  const [tmdbResult, setTmdbResult] = useState(null);
  const tmdbRef = useRef(null);
  const promoRef = useRef(null);

  // Conflict State
  const [conflictReport, setConflictReport] = useState(null);
  const [isCheckingConflict, setIsCheckingConflict] = useState(false);
  const [selectedConflictAction, setSelectedConflictAction] = useState(""); // RIPPLE, ADJUST_START, ADJUST_END, FORCE_PREEMPT

  // Auto-Fill Gaps Modal State
  const [isGapModalOpen, setIsGapModalOpen] = useState(false);
  const [gapScope, setGapScope] = useState("current"); // "current" (from current playout time forward) | "full" (entire broadcast day)
  const [detectedGaps, setDetectedGaps] = useState([]);
  const [isDetectingGaps, setIsDetectingGaps] = useState(false);
  const [isFillingGaps, setIsFillingGaps] = useState(false);
  const [fillerTitle, setFillerTitle] = useState("Station Intermission & Reel");
  const [fillerMedia, setFillerMedia] = useState("sample_movie.mp4");

  // Browse files when modal opens or path changes
  useEffect(() => {
    if (isModalOpen) {
      loadFiles(currentPath);
    }
  }, [isModalOpen, currentPath]);

  const loadFiles = async (path = "") => {
    try {
      const files = await api.browseStorage(path);
      setFileList(Array.isArray(files) ? files : []);
    } catch (err) {
      onShowToast("Failed to browse media library: " + err.message, "error");
    }
  };

  // Recalculate End Time
  useEffect(() => {
    try {
      const sParts = startTime.split(':').map(Number);
      const dParts = duration.split(':').map(Number);
      const sSec = (sParts[0] || 0) * 3600 + (sParts[1] || 0) * 60 + (sParts[2] || 0);
      const dSec = (dParts[0] || 0) * 3600 + (dParts[1] || 0) * 60 + (dParts[2] || 0);
      const eSec = (sSec + dSec) % 86400;
      const hh = String(Math.floor(eSec / 3600)).padStart(2, '0');
      const mm = String(Math.floor((eSec % 3600) / 60)).padStart(2, '0');
      const ss = String(eSec % 60).padStart(2, '0');
      setEndTime(`${hh}:${mm}:${ss}`);
    } catch (e) {}
  }, [startTime, duration]);

  // Check conflicts whenever channel, date, start time, or duration changes
  useEffect(() => {
    if (!isModalOpen || !activeChannelId) return;

    const timer = setTimeout(async () => {
      try {
        setIsCheckingConflict(true);
        const dParts = duration.split(':').map(Number);
        const durSecs = (dParts[0] || 0) * 3600 + (dParts[1] || 0) * 60 + (dParts[2] || 0);
        const startIso = createIsoInTimezone(selectedDate, startTime, broadcastTimezone);
        const endIso = new Date(new Date(startIso).getTime() + durSecs * 1000).toISOString();

        const report = await api.checkScheduleConflicts({
          channel_id: activeChannelId,
          start_time: startIso,
          duration_seconds: durSecs,
          end_time: endIso
        });

        if (report && report.has_conflict) {
          setConflictReport(report);
        } else {
          setConflictReport(null);
        }
      } catch (err) {
        checkLocalConflicts();
      } finally {
        setIsCheckingConflict(false);
      }
    }, 200);

    return () => clearTimeout(timer);
  }, [isModalOpen, activeChannelId, selectedDate, startTime, duration, broadcastTimezone]);

  const checkLocalConflicts = () => {
    try {
      const dParts = duration.split(':').map(Number);
      const durSecs = (dParts[0] || 0) * 3600 + (dParts[1] || 0) * 60 + (dParts[2] || 0);
      const startIso = createIsoInTimezone(selectedDate, startTime, broadcastTimezone);
      const sMs = new Date(startIso).getTime();
      const eMs = sMs + durSecs * 1000;

      const collisions = scheduleItems.filter(it => {
        const itemStart = new Date(it.start_time).getTime();
        const itemEnd = new Date(it.end_time).getTime();
        return itemStart < eMs && itemEnd > sMs;
      });

      if (collisions.length > 0) {
        setConflictReport({
          has_conflict: true,
          conflicts: collisions,
          suggested_start: collisions[collisions.length - 1].end_time,
          suggested_duration: Math.max(60, Math.floor((new Date(collisions[0].start_time).getTime() - sMs) / 1000))
        });
      } else {
        setConflictReport(null);
      }
    } catch (e) {}
  };

  // Close TMDb and Promo dropdown on outside click
  useEffect(() => {
    function handleClickOutside(event) {
      if (tmdbRef.current && !tmdbRef.current.contains(event.target)) {
        setIsTmdbDropdownOpen(false);
      }
      if (promoRef.current && !promoRef.current.contains(event.target)) {
        setShowPromoSuggestions(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  // Filtered Media Files
  const filteredFiles = fileList.filter(f => {
    if (!fileFilterQuery) return true;
    return f.name.toLowerCase().includes(fileFilterQuery.toLowerCase());
  });

  const handleItemClick = (item) => {
    if (item.is_dir) {
      setCurrentPath(item.path);
      setFileFilterQuery("");
    } else {
      handleSelectFile(item);
    }
  };

  const handleSelectFile = (file) => {
    setSelectedFile(file);
    const cleanTitle = file.name.replace(/\.[^/.]+$/, "").replace(/[\._]/g, " ");
    setTitle(cleanTitle);
    setTmdbQuery(cleanTitle);

    if (file.duration_seconds) {
      const durSec = file.duration_seconds;
      const hh = String(Math.floor(durSec / 3600)).padStart(2, '0');
      const mm = String(Math.floor((durSec % 3600) / 60)).padStart(2, '0');
      const ss = String(durSec % 60).padStart(2, '0');
      setDuration(`${hh}:${mm}:${ss}`);
    } else if (file.probed_duration) {
      setDuration(file.probed_duration);
    }

    triggerTmdbSearch(cleanTitle);
  };

  const triggerTmdbSearch = async (queryText) => {
    const q = queryText?.trim();
    if (!q || q.length < 2) {
      setTmdbHits([]);
      setIsTmdbDropdownOpen(false);
      return;
    }

    setIsSearchingTmdb(true);
    try {
      const results = await api.searchTmdb(q);
      const hits = Array.isArray(results) ? results : [];
      setTmdbHits(hits);
      setIsTmdbDropdownOpen(hits.length > 0);
      if (hits.length > 0 && !tmdbResult) {
        setTmdbResult(hits[0]);
      }
    } catch (err) {
      setTmdbHits([]);
    } finally {
      setIsSearchingTmdb(false);
    }
  };

  const handleTmdbInputChange = (e) => {
    const val = e.target.value;
    setTmdbQuery(val);
    triggerTmdbSearch(val);
  };

  const handleSelectTmdbHit = (hit) => {
    setTmdbResult(hit);
    setTitle(hit.title);
    setTmdbQuery(hit.title);
    setIsTmdbDropdownOpen(false);
    onShowToast(`Enriched metadata with TMDb hit: "${hit.title}"`, "info");
  };

  // Conflict Resolution Action Handlers
  const handleSnapStartTime = () => {
    if (!conflictReport?.suggested_start) return;
    try {
      const timeStr = formatTimeInTimezone(conflictReport.suggested_start, broadcastTimezone, true);
      setStartTime(timeStr);
      setSelectedConflictAction("ADJUST_START");
      onShowToast(`Snapped start time to ${timeStr} (after colliding program)`, "success");
    } catch (e) {}
  };

  const handleTrimDuration = () => {
    if (!conflictReport?.suggested_duration || conflictReport.suggested_duration <= 0) return;
    const durSec = conflictReport.suggested_duration;
    const hh = String(Math.floor(durSec / 3600)).padStart(2, '0');
    const mm = String(Math.floor((durSec % 3600) / 60)).padStart(2, '0');
    const ss = String(durSec % 60).padStart(2, '0');
    setDuration(`${hh}:${mm}:${ss}`);
    setSelectedConflictAction("ADJUST_END");
    onShowToast(`Trimmed duration to ${hh}:${mm}:${ss} to fit before next program`, "success");
  };

  const handleSelectAction = (action) => {
    setSelectedConflictAction(action);
    if (action === "RIPPLE") {
      onShowToast("Selected: Ripple subsequent programs forward on timeline", "info");
    } else if (action === "FORCE_PREEMPT") {
      onShowToast("Selected: Force playout in this time slot (preempt overlapping)", "warning");
    }
  };

  const handleCommitSchedule = async () => {
    if (!title) {
      onShowToast("Program title is required", "error");
      return;
    }

    const durParts = duration.split(':').map(Number);
    const durSecs = (durParts[0] || 0) * 3600 + (durParts[1] || 0) * 60 + (durParts[2] || 0);

    const startIso = createIsoInTimezone(selectedDate, startTime, broadcastTimezone);
    const endIso = new Date(new Date(startIso).getTime() + durSecs * 1000).toISOString();

    const payload = {
      item: {
        channel_id: activeChannelId,
        program_title: title,
        title: title,
        media_path: selectedFile?.path || selectedFile?.relative_path || selectedFile?.name || "sample_movie.mp4",
        media_file_path: selectedFile?.path || selectedFile?.relative_path || selectedFile?.name || "sample_movie.mp4",
        start_time: startIso,
        duration_seconds: durSecs,
        end_time: endIso,
        ad_template_id: selectedAdTemplateId || "",
        special_promo_title: specialPromoTitle.trim(),
        special_promo_subtext: specialPromoSubtext.trim(),
        tmdb_id: tmdbResult?.id || "",
        tmdb_poster: tmdbResult?.poster_path || getFallbackPoster(title, selectedDate.slice(0, 4)),
        tmdb_overview: tmdbResult?.overview || "Broadcast linear program event.",
        tmdb_metadata: tmdbResult || {
          title: title,
          overview: "Broadcast linear program event."
        }
      },
      action: selectedConflictAction || (conflictReport ? "RIPPLE" : "FORCE_OVERWRITE")
    };

    try {
      await api.createScheduleItem(payload);
      onShowToast(`Scheduled "${title}" on ${selectedDate} successfully!`, "success");
      setIsModalOpen(false);
      setConflictReport(null);
      setSelectedConflictAction("");
      setSpecialPromoTitle("");
      setSpecialPromoSubtext("");
      onRefreshSchedule(activeChannelId);
    } catch (err) {
      if (err.status === 409 && err.data) {
        setConflictReport(err.data);
        onShowToast("Conflict detected! Please select a resolution option below.", "warning");
      } else {
        onShowToast("Failed to schedule media: " + err.message, "error");
      }
    }
  };

  const handleDeleteItem = async (id) => {
    try {
      await api.deleteScheduleItem(id);
      onShowToast("Schedule item removed", "info");
      onRefreshSchedule(activeChannelId);
    } catch (err) {
      onShowToast("Delete failed: " + err.message, "error");
    }
  };

  const handleExportXmltv = () => {
    const url = `/api/v1/epg/${activeChannelId}.xml`;
    window.open(url, '_blank');
    onShowToast("Exporting DVB-SI / XMLTV schedule feed", "info");
  };

  // Gap Detection & Auto-Fill
  const handleOpenGapModal = async (scopeOverride) => {
    const isToday = selectedDate === formatDateInTimezone(new Date(), broadcastTimezone);
    const activeScope = scopeOverride !== undefined ? scopeOverride : (isToday ? "current" : "full");
    setGapScope(activeScope);
    setIsGapModalOpen(true);
    setIsDetectingGaps(true);
    try {
      const fromCurrent = isToday && activeScope === "current";
      const startDay = fromCurrent
        ? new Date().toISOString()
        : createIsoInTimezone(selectedDate, "00:00:00", broadcastTimezone);
      const endDay = createIsoInTimezone(selectedDate, "23:59:59", broadcastTimezone);

      const gaps = await api.getScheduleGaps(activeChannelId, startDay, endDay, fromCurrent);
      setDetectedGaps(Array.isArray(gaps) ? gaps : []);
    } catch (e) {
      onShowToast("Failed to detect schedule gaps: " + e.message, "error");
    } finally {
      setIsDetectingGaps(false);
    }
  };

  const handleAutoFillGaps = async () => {
    setIsFillingGaps(true);
    try {
      const isToday = selectedDate === formatDateInTimezone(new Date(), broadcastTimezone);
      const fromCurrent = isToday && gapScope === "current";
      const startDay = fromCurrent
        ? new Date().toISOString()
        : createIsoInTimezone(selectedDate, "00:00:00", broadcastTimezone);
      const endDay = createIsoInTimezone(selectedDate, "23:59:59", broadcastTimezone);

      const res = await api.autoFillGaps({
        channel_id: activeChannelId,
        start_time: startDay,
        end_time: endDay,
        filler_title: fillerTitle,
        filler_media: fillerMedia,
        from_current_time: fromCurrent
      });

      onShowToast(`Auto-filled ${res.filled_count || detectedGaps.length} schedule gap(s) successfully!`, "success");
      setIsGapModalOpen(false);
      onRefreshSchedule(activeChannelId);
    } catch (e) {
      onShowToast("Failed to auto-fill gaps: " + e.message, "error");
    } finally {
      setIsFillingGaps(false);
    }
  };

  // Now Playing & Up Next Automation Calculation
  const nowMs = Date.now();
  const sortedItems = [...scheduleItems].sort((a, b) => new Date(a.start_time) - new Date(b.start_time));
  const activeNowItem = sortedItems.find((it) => {
    const s = new Date(it.start_time).getTime();
    const e = new Date(it.end_time).getTime();
    return s <= nowMs && e > nowMs;
  });
  const upcomingNextItem = sortedItems.find((it) => new Date(it.start_time).getTime() > nowMs);

  // Filter schedule items for selectedDate in broadcast timezone
  const dayItems = sortedItems.filter((it) => {
    if (!it.start_time) return false;
    const itDate = formatDateInTimezone(it.start_time, broadcastTimezone);
    return itDate === selectedDate;
  });

  // Multi-day quick date navigation pills in broadcast timezone
  const datePills = [0, 1, 2, 3, 4, 5, 6].map((offset) => {
    const d = new Date();
    d.setDate(d.getDate() + offset);
    const iso = formatDateInTimezone(d, broadcastTimezone);
    const label = offset === 0 ? "Today" : offset === 1 ? "Tomorrow" : d.toLocaleDateString('en-US', { timeZone: broadcastTimezone, weekday: 'short', month: 'numeric', day: 'numeric' });
    return { date: iso, label };
  });

  return (
    <div className="w-full flex-1 flex flex-col p-3 sm:p-4 md:p-6 space-y-4 max-w-full">
      {/* Top Scheduling Bar */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-3 shrink-0 bg-[#111827] border border-[#1F2937] p-3 rounded-xl shadow-sm">
        <div className="flex flex-wrap items-center gap-3">
          <div className="flex items-center gap-2">
            <Calendar className="w-5 h-5 text-indigo-400 shrink-0" />
            <div>
              <h2 className="text-sm font-bold text-white">
                {t('sched.title') || "24/7 Playout Schedule & EPG Master"}
              </h2>
              <p className="text-[11px] text-gray-400">
                Multi-day linear timeline, TMDb enrichment, promotions & gap auto-fill
              </p>
            </div>
          </div>
          <div className="hidden sm:block h-6 w-px bg-gray-700"></div>

          {/* Channel selector */}
          <select
            value={activeChannelId}
            onChange={(e) => onSwitchChannel(e.target.value)}
            className="bg-[#1F2937] border border-gray-700 text-xs text-white rounded-lg px-2.5 py-1.5 font-medium focus:outline-none focus:border-indigo-500"
          >
            {channels.map((ch) => (
              <option key={ch.id} value={ch.id}>
                CH {String(ch.lcn || 1).padStart(2, '0')}: {ch.name}
              </option>
            ))}
          </select>
        </div>

        {/* Action Buttons */}
        <div className="flex flex-wrap items-center gap-2">
          {/* Auto-Fill Gaps Button */}
          <button
            onClick={handleOpenGapModal}
            className="px-3 py-1.5 bg-amber-950/60 hover:bg-amber-900/80 text-amber-300 border border-amber-800/70 text-xs font-semibold rounded-lg flex items-center gap-1.5 transition-colors shadow-sm"
            title="Detect and auto-fill unprogrammed schedule slots"
          >
            <Sparkles className="w-3.5 h-3.5 text-amber-400" />
            <span>Auto-Fill Gaps</span>
          </button>

          {/* Add Media Button */}
          <button
            onClick={() => {
              setIsModalOpen(true);
              setConflictReport(null);
              setSelectedConflictAction("");
            }}
            className="px-3.5 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-xs font-semibold rounded-lg text-white shadow-sm flex items-center gap-1.5 transition-colors"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>+ Add Media to Schedule</span>
          </button>

          {/* Export XMLTV Button */}
          <button
            onClick={handleExportXmltv}
            className="px-3 py-1.5 bg-[#1F2937] hover:bg-[#374151] text-xs font-medium rounded-lg text-gray-200 border border-gray-700 flex items-center gap-1.5 transition-colors"
          >
            <Download className="w-3.5 h-3.5" />
            <span>Export XMLTV</span>
          </button>
        </div>
      </div>

      {/* Dynamic Playout Status Strip (Derived from Active Schedule) */}
      <div className="bg-[#101726] border border-blue-900/40 rounded-xl p-3 flex flex-wrap items-center justify-between gap-3 text-xs">
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2">
            <span className="w-2.5 h-2.5 rounded-full bg-red-500 animate-pulse" />
            <span className="font-bold text-gray-400 uppercase tracking-wider text-[11px]">Now On-Air:</span>
            {activeNowItem ? (
              <span className="font-bold text-white bg-red-950/80 px-2 py-0.5 rounded border border-red-800 text-red-200">
                {activeNowItem.program_title}
              </span>
            ) : (
              <span className="text-gray-500 italic">Intermission Playout Reel</span>
            )}
          </div>
          <div className="hidden md:block h-4 w-px bg-gray-800"></div>
          <div className="flex items-center gap-2">
            <span className="font-bold text-gray-400 uppercase tracking-wider text-[11px]">Up Next:</span>
            {upcomingNextItem ? (
              <span className="font-semibold text-amber-200 bg-amber-950/60 px-2 py-0.5 rounded border border-amber-800/80">
                {upcomingNextItem.program_title} (Starts {formatTimeInTimezone(upcomingNextItem.start_time, broadcastTimezone, false)})
              </span>
            ) : (
              <span className="text-gray-500 italic">No scheduled upcoming queue</span>
            )}
          </div>
        </div>

        <div className="text-[11px] text-gray-400 font-mono">
          Single Source of Truth: Automated Master Playout
        </div>
      </div>

      {/* Multi-Day Navigation Bar */}
      <div className="bg-[#111827] border border-[#1F2937] rounded-xl p-3 flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-1.5">
          <span className="text-xs font-bold text-gray-400 uppercase tracking-wider mr-2">Timeline Date:</span>
          {datePills.map((pill) => (
            <button
              key={pill.date}
              onClick={() => setSelectedDate(pill.date)}
              className={`px-3 py-1 rounded-lg text-xs font-semibold transition-all ${
                selectedDate === pill.date
                  ? "bg-blue-600 text-white shadow-md shadow-blue-900/30"
                  : "bg-gray-800/80 hover:bg-gray-700 text-gray-300"
              }`}
            >
              {pill.label}
            </button>
          ))}
        </div>

        {/* Custom Date Picker Input */}
        <div className="flex items-center gap-2">
          <label className="text-xs text-gray-400">Custom Date:</label>
          <input
            type="date"
            value={selectedDate}
            onChange={(e) => setSelectedDate(e.target.value)}
            className="bg-[#1F2937] border border-gray-700 text-xs text-white rounded-lg px-2.5 py-1 font-mono focus:outline-none focus:border-blue-500"
          />
        </div>
      </div>

      {/* Schedule Items List for Selected Day */}
      <div className="flex-1 bg-[#111827] border border-[#1F2937] rounded-xl p-4 space-y-3 overflow-y-auto">
        <div className="flex items-center justify-between border-b border-gray-800 pb-2">
          <div className="flex items-center gap-2">
            <Clock className="w-4 h-4 text-gray-400" />
            <h3 className="text-xs font-bold text-gray-200 uppercase tracking-wider">
              Broadcast Events for {selectedDate} ({dayItems.length})
            </h3>
          </div>
          <span className="text-[11px] font-mono text-gray-400">
            SMPTE Linear Playout Timeline
          </span>
        </div>

        <div className="space-y-2">
          {dayItems.length === 0 ? (
            <div className="text-center py-14 text-gray-500 font-mono text-xs space-y-3">
              <p>No programs scheduled on {selectedDate}.</p>
              <div className="flex justify-center gap-2">
                <button
                  onClick={() => setIsModalOpen(true)}
                  className="px-3 py-1.5 bg-indigo-600/80 hover:bg-indigo-600 text-white rounded-lg text-xs font-semibold"
                >
                  + Add Media to {selectedDate}
                </button>
                <button
                  onClick={handleOpenGapModal}
                  className="px-3 py-1.5 bg-amber-950/60 hover:bg-amber-900 text-amber-300 border border-amber-800 rounded-lg text-xs font-semibold"
                >
                  Auto-Fill Gaps for {selectedDate}
                </button>
              </div>
            </div>
          ) : (
            dayItems.map((item) => {
              const startStr = item.start_time ? formatTimeInTimezone(item.start_time, broadcastTimezone, true) : "16:00:00";
              const endStr = item.end_time ? formatTimeInTimezone(item.end_time, broadcastTimezone, true) : "18:15:00";
              const durMinutes = Math.floor((item.duration_seconds || 3600) / 60);

              const isItemActiveNow = activeNowItem && activeNowItem.id === item.id;
              const isItemNext = upcomingNextItem && upcomingNextItem.id === item.id;

              return (
                <div
                  key={item.id}
                  className={`flex flex-col sm:flex-row items-start sm:items-center justify-between p-3.5 rounded-xl border transition-all gap-3 ${
                    isItemActiveNow
                      ? "bg-[#1f1322] border-red-600/80 shadow-md shadow-red-950/40"
                      : isItemNext
                      ? "bg-[#1d1f2b] border-amber-600/80 shadow-md shadow-amber-950/40"
                      : "bg-[#182030]/60 border-gray-800 hover:border-gray-700"
                  }`}
                >
                  <div className="flex items-center gap-3.5 w-full sm:w-auto">
                    {/* Poster Thumbnail */}
                    <div className="w-12 h-16 bg-gray-900 rounded-lg overflow-hidden border border-gray-800 shrink-0 relative">
                      <img
                        src={item.tmdb_poster || getFallbackPoster(item.program_title)}
                        alt={item.program_title}
                        onError={(e) => {
                          e.currentTarget.src = getFallbackPoster(item.program_title);
                        }}
                        className="w-full h-full object-cover"
                      />
                    </div>

                    <div className="space-y-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="text-sm font-bold text-white">
                          {item.program_title}
                        </span>

                        {isItemActiveNow && (
                          <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-red-600 text-white animate-pulse">
                            🔴 NOW ON-AIR
                          </span>
                        )}

                        {isItemNext && (
                          <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-amber-600 text-white">
                            UP NEXT
                          </span>
                        )}

                        {item.special_promo_title && (
                          <span className="px-2 py-0.5 rounded text-[10px] font-semibold bg-purple-950 border border-purple-800 text-purple-300 flex items-center gap-1">
                            <Tag className="w-3 h-3 text-purple-400" />
                            PROMO: {item.special_promo_title}
                          </span>
                        )}
                      </div>

                      <div className="flex flex-wrap items-center gap-2 text-xs text-gray-400 font-mono">
                        <span className="text-blue-400 font-semibold">{startStr}</span>
                        <span>➔</span>
                        <span className="text-gray-300">{endStr}</span>
                        <span>•</span>
                        <span>{durMinutes} min</span>
                        <span>•</span>
                        <span className="text-gray-400 truncate max-w-xs">{item.media_path}</span>
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center gap-2 self-end sm:self-auto shrink-0">
                    <button
                      onClick={() => handleDeleteItem(item.id)}
                      className="p-1.5 text-gray-400 hover:text-red-400 hover:bg-gray-800 rounded-lg transition-colors"
                      title="Remove from timeline"
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                </div>
              );
            })
          )}
        </div>
      </div>

      {/* Modal: Add Media to Schedule (Integrated Media Picker + TMDb + Promotions) */}
      {isModalOpen && (
        <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-3 sm:p-4 overflow-y-auto">
          <div className="bg-[#111827] border border-gray-800 rounded-2xl w-full max-w-3xl shadow-2xl overflow-hidden animate-in zoom-in-95 duration-200 my-8">
            <div className="p-4 border-b border-gray-800 flex items-center justify-between bg-[#151c2c]">
              <div className="flex items-center gap-2">
                <Film className="w-5 h-5 text-indigo-400" />
                <h3 className="font-bold text-white text-base">Schedule Program to Timeline</h3>
                <span className="text-xs px-2 py-0.5 rounded bg-blue-950 border border-blue-800 text-blue-300 font-mono">
                  {selectedDate}
                </span>
              </div>
              <button
                onClick={() => setIsModalOpen(false)}
                className="p-1.5 text-gray-400 hover:text-white rounded-lg hover:bg-gray-800"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="p-6 space-y-5">
              {/* Conflict Alert Warning Banner */}
              {conflictReport && (
                <div className="p-4 bg-amber-950/60 border border-amber-600/80 rounded-xl space-y-3">
                  <div className="flex items-start gap-2.5">
                    <AlertTriangle className="w-5 h-5 text-amber-400 shrink-0 mt-0.5" />
                    <div>
                      <h4 className="text-xs font-bold text-amber-300 uppercase tracking-wide">
                        Schedule Overlap Conflict Detected
                      </h4>
                      <p className="text-xs text-amber-200/90 mt-0.5">
                        Collides with existing scheduled program(s). Select an automated broadcast conflict resolution:
                      </p>
                    </div>
                  </div>

                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 pt-1">
                    <button
                      type="button"
                      onClick={handleSnapStartTime}
                      className={`p-2.5 rounded-lg border text-left text-xs transition-colors ${
                        selectedConflictAction === "ADJUST_START"
                          ? "bg-amber-600 text-white border-amber-500 font-bold"
                          : "bg-gray-900/80 border-gray-700 text-gray-200 hover:bg-gray-800"
                      }`}
                    >
                      <div className="flex items-center gap-1.5 font-bold mb-0.5">
                        <FastForward className="w-3.5 h-3.5 text-amber-400" />
                        Snap Start Time
                      </div>
                      <span className="text-[11px] opacity-80">
                        Shift start after collision to {conflictReport.suggested_start ? new Date(conflictReport.suggested_start).toLocaleTimeString() : "available slot"}
                      </span>
                    </button>

                    <button
                      type="button"
                      onClick={handleTrimDuration}
                      className={`p-2.5 rounded-lg border text-left text-xs transition-colors ${
                        selectedConflictAction === "ADJUST_END"
                          ? "bg-amber-600 text-white border-amber-500 font-bold"
                          : "bg-gray-900/80 border-gray-700 text-gray-200 hover:bg-gray-800"
                      }`}
                    >
                      <div className="flex items-center gap-1.5 font-bold mb-0.5">
                        <Scissors className="w-3.5 h-3.5 text-amber-400" />
                        Trim Duration
                      </div>
                      <span className="text-[11px] opacity-80">
                        Shorten duration to fit right before the next program
                      </span>
                    </button>

                    <button
                      type="button"
                      onClick={() => handleSelectAction("RIPPLE")}
                      className={`p-2.5 rounded-lg border text-left text-xs transition-colors ${
                        selectedConflictAction === "RIPPLE"
                          ? "bg-amber-600 text-white border-amber-500 font-bold"
                          : "bg-gray-900/80 border-gray-700 text-gray-200 hover:bg-gray-800"
                      }`}
                    >
                      <div className="flex items-center gap-1.5 font-bold mb-0.5">
                        <ArrowRight className="w-3.5 h-3.5 text-amber-400" />
                        Ripple Timeline
                      </div>
                      <span className="text-[11px] opacity-80">
                        Push colliding and all subsequent items downstream
                      </span>
                    </button>

                    <button
                      type="button"
                      onClick={() => handleSelectAction("FORCE_PREEMPT")}
                      className={`p-2.5 rounded-lg border text-left text-xs transition-colors ${
                        selectedConflictAction === "FORCE_PREEMPT"
                          ? "bg-red-600 text-white border-red-500 font-bold"
                          : "bg-gray-900/80 border-gray-700 text-gray-200 hover:bg-gray-800"
                      }`}
                    >
                      <div className="flex items-center gap-1.5 font-bold mb-0.5">
                        <ShieldAlert className="w-3.5 h-3.5 text-red-400" />
                        Force Preempt Playout
                      </div>
                      <span className="text-[11px] opacity-80">
                        Overwrite and truncate colliding slot unconditionally
                      </span>
                    </button>
                  </div>
                </div>
              )}

              {/* 1. Integrated Media File Picker */}
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <label className="block text-xs font-bold text-gray-300 uppercase tracking-wide">
                    1. Select Broadcast Media File
                  </label>
                  <span className="text-[11px] text-gray-400 font-mono">
                    Storage: ./media/{currentPath}
                  </span>
                </div>

                <div className="relative">
                  <Search className="w-4 h-4 absolute left-3 top-2.5 text-gray-500" />
                  <input
                    type="text"
                    value={fileFilterQuery}
                    onChange={(e) => setFileFilterQuery(e.target.value)}
                    placeholder="Search media files in storage folder..."
                    className="w-full bg-[#182030] border border-gray-700 rounded-lg pl-9 pr-3 py-2 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"
                  />
                </div>

                <div className="max-h-40 overflow-y-auto bg-[#141b2b] border border-gray-800 rounded-lg divide-y divide-gray-800/60">
                  {currentPath && (
                    <div
                      onClick={() => setCurrentPath("")}
                      className="p-2 text-xs text-indigo-400 hover:bg-gray-800/60 cursor-pointer flex items-center gap-2"
                    >
                      <FolderOpen className="w-4 h-4" />
                      <span>.. (Back to root)</span>
                    </div>
                  )}
                  {filteredFiles.map((file) => (
                    <div
                      key={file.name}
                      onClick={() => handleItemClick(file)}
                      className={`p-2.5 text-xs flex items-center justify-between cursor-pointer transition-colors ${
                        selectedFile?.name === file.name
                          ? "bg-indigo-950/80 text-white font-bold"
                          : "text-gray-300 hover:bg-gray-800/60"
                      }`}
                    >
                      <div className="flex items-center gap-2 truncate">
                        {file.is_dir ? (
                          <FolderOpen className="w-4 h-4 text-amber-400 shrink-0" />
                        ) : (
                          <FileVideo className="w-4 h-4 text-blue-400 shrink-0" />
                        )}
                        <span className="truncate">{file.name}</span>
                      </div>
                      <div className="flex items-center gap-2 text-[11px] font-mono text-gray-500 shrink-0">
                        {file.probed_duration && <span>{file.probed_duration}</span>}
                        {file.size && <span>{(file.size / 1024 / 1024).toFixed(1)} MB</span>}
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              {/* 2. TMDb Search & Metadata Typeahead */}
              <div className="space-y-2 relative" ref={tmdbRef}>
                <label className="block text-xs font-bold text-gray-300 uppercase tracking-wide">
                  2. TMDb Title Lookup & Poster Auto-Enrichment
                </label>
                <div className="relative">
                  <Search className="w-4 h-4 absolute left-3 top-2.5 text-gray-500" />
                  <input
                    type="text"
                    value={tmdbQuery}
                    onChange={handleTmdbInputChange}
                    placeholder="Type movie or series title for TMDb lookup..."
                    className="w-full bg-[#182030] border border-gray-700 rounded-lg pl-9 pr-3 py-2 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-indigo-500"
                  />
                  {isSearchingTmdb && (
                    <Loader2 className="w-4 h-4 absolute right-3 top-2.5 text-indigo-400 animate-spin" />
                  )}
                </div>

                {/* TMDb Dropdown Hits */}
                {isTmdbDropdownOpen && tmdbHits.length > 0 && (
                  <div className="absolute z-20 left-0 right-0 mt-1 max-h-56 overflow-y-auto bg-[#182030] border border-gray-700 rounded-xl shadow-2xl divide-y divide-gray-700/60">
                    {tmdbHits.map((hit) => {
                      const yr = hit.release_date?.slice(0, 4) || "";
                      return (
                        <div
                          key={hit.id}
                          onClick={() => handleSelectTmdbHit(hit)}
                          className="p-2.5 flex items-center gap-3 hover:bg-indigo-950/70 cursor-pointer transition-colors"
                        >
                          <img
                            src={hit.poster_path || getFallbackPoster(hit.title, yr)}
                            alt={hit.title}
                            onError={(e) => {
                              e.currentTarget.src = getFallbackPoster(hit.title, yr);
                            }}
                            className="w-8 h-12 object-cover rounded bg-gray-900 border border-gray-800 shrink-0"
                          />
                          <div className="truncate">
                            <div className="text-xs font-bold text-white flex items-center gap-2">
                              <span>{hit.title}</span>
                              {yr && <span className="text-[11px] text-gray-400">({yr})</span>}
                            </div>
                            <p className="text-[11px] text-gray-400 truncate mt-0.5">{hit.overview}</p>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>

              {/* 3. Special Promotional Overlays / Bumps (Typeahead) */}
              <div className="space-y-2 relative" ref={promoRef}>
                <div className="flex items-center justify-between">
                  <label className="block text-xs font-bold text-gray-300 uppercase tracking-wide">
                    3. Special Promotional Bump / Overlay (Optional)
                  </label>
                  <button
                    type="button"
                    onClick={() => setShowPromoSuggestions(!showPromoSuggestions)}
                    className="text-[11px] text-purple-400 hover:text-purple-300 underline"
                  >
                    {showPromoSuggestions ? "Hide Suggestions" : "Show Broadcast Presets"}
                  </button>
                </div>

                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                  <div>
                    <input
                      type="text"
                      value={specialPromoTitle}
                      onChange={(e) => setSpecialPromoTitle(e.target.value)}
                      placeholder="Promo Title (e.g. Diwali Premiere Special)"
                      className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-purple-500"
                    />
                  </div>
                  <div>
                    <input
                      type="text"
                      value={specialPromoSubtext}
                      onChange={(e) => setSpecialPromoSubtext(e.target.value)}
                      placeholder="Promo Subtext (e.g. Tonight @ 21:00 IST)"
                      className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-xs text-white placeholder-gray-500 focus:outline-none focus:border-purple-500"
                    />
                  </div>
                </div>

                {showPromoSuggestions && (
                  <div className="bg-[#141b2b] border border-purple-900/60 rounded-xl p-2.5 space-y-1.5">
                    <span className="text-[10px] uppercase font-bold text-purple-300 tracking-wider">
                      Quick Promotional Overlay Presets:
                    </span>
                    <div className="flex flex-wrap gap-1.5">
                      {PROMO_PRESETS.map((p, idx) => (
                        <button
                          key={idx}
                          type="button"
                          onClick={() => {
                            setSpecialPromoTitle(p.title);
                            setSpecialPromoSubtext(p.subtext);
                            setShowPromoSuggestions(false);
                          }}
                          className="px-2 py-1 bg-purple-950/80 hover:bg-purple-900 text-purple-200 border border-purple-800 rounded text-[11px] text-left"
                        >
                          {p.title}
                        </button>
                      ))}
                    </div>
                  </div>
                )}
              </div>

              {/* 4. Scheduling Parameters (Date, Start Time, Duration) */}
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 pt-2 border-t border-gray-800">
                <div>
                  <label className="block text-[11px] font-medium text-gray-400 mb-1">Broadcast Date</label>
                  <input
                    type="date"
                    value={selectedDate}
                    onChange={(e) => setSelectedDate(e.target.value)}
                    className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-xs text-white font-mono focus:outline-none focus:border-indigo-500"
                  />
                </div>
                <div>
                  <label className="block text-[11px] font-medium text-gray-400 mb-1">Start Time (HH:MM:SS)</label>
                  <input
                    type="text"
                    value={startTime}
                    onChange={(e) => setStartTime(e.target.value)}
                    className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-xs text-white font-mono focus:outline-none focus:border-indigo-500"
                  />
                </div>
                <div>
                  <label className="block text-[11px] font-medium text-gray-400 mb-1">Duration (HH:MM:SS)</label>
                  <input
                    type="text"
                    value={duration}
                    onChange={(e) => setDuration(e.target.value)}
                    className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-xs text-white font-mono focus:outline-none focus:border-indigo-500"
                  />
                </div>
              </div>
            </div>

            <div className="p-4 border-t border-gray-800 flex justify-end gap-2 bg-[#151c2c]">
              <button
                onClick={() => setIsModalOpen(false)}
                className="px-4 py-2 bg-gray-800 hover:bg-gray-700 text-gray-300 rounded-lg text-xs font-semibold"
              >
                Cancel
              </button>
              <button
                onClick={handleCommitSchedule}
                className="px-5 py-2 bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg text-xs font-semibold shadow-md shadow-indigo-900/30"
              >
                Save Schedule Event
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Modal: Auto-Fill Gaps */}
      {isGapModalOpen && (
        <div className="fixed inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-4">
          <div className="bg-[#111827] border border-gray-800 rounded-2xl w-full max-w-lg shadow-2xl overflow-hidden animate-in zoom-in-95 duration-200">
            <div className="p-4 border-b border-gray-800 flex items-center justify-between bg-[#151c2c]">
              <div className="flex items-center gap-2">
                <Sparkles className="w-5 h-5 text-amber-400" />
                <div>
                  <h3 className="font-bold text-white text-base">Timeline Gap Detection & Auto-Fill</h3>
                  <div className="flex items-center gap-2 mt-0.5">
                    <span className="text-[11px] text-gray-400">{broadcastTimezone}</span>
                    <span className="text-[11px] font-mono text-amber-400 bg-amber-950/60 px-1.5 py-0.2 rounded border border-amber-800/60">
                      Live Clock: {formatTimeInTimezone(new Date(), broadcastTimezone, true)}
                    </span>
                  </div>
                </div>
              </div>
              <button
                onClick={() => setIsGapModalOpen(false)}
                className="p-1.5 text-gray-400 hover:text-white rounded-lg hover:bg-gray-800"
              >
                <X className="w-5 h-5" />
              </button>
            </div>

            <div className="p-6 space-y-4">
              {/* Scope Selector if selectedDate is today */}
              {selectedDate === formatDateInTimezone(new Date(), broadcastTimezone) && (
                <div className="p-3 bg-[#161f30] border border-gray-800 rounded-xl space-y-2">
                  <span className="text-[11px] font-bold text-gray-300 uppercase tracking-wider block">
                    Auto-Fill Starting Point
                  </span>
                  <div className="grid grid-cols-2 gap-2">
                    <button
                      type="button"
                      onClick={() => handleOpenGapModal("current")}
                      className={`p-2 rounded-lg text-left text-xs transition-all border ${
                        gapScope === "current"
                          ? "bg-amber-950/70 border-amber-500 text-amber-200 shadow-sm"
                          : "bg-gray-800/60 border-gray-700/80 text-gray-400 hover:text-gray-200"
                      }`}
                    >
                      <div className="font-bold">From Current Time (Live)</div>
                      <div className="text-[10px] text-gray-400 mt-0.5">
                        Fills forward from {formatTimeInTimezone(new Date(), broadcastTimezone, false)}
                      </div>
                    </button>

                    <button
                      type="button"
                      onClick={() => handleOpenGapModal("full")}
                      className={`p-2 rounded-lg text-left text-xs transition-all border ${
                        gapScope === "full"
                          ? "bg-amber-950/70 border-amber-500 text-amber-200 shadow-sm"
                          : "bg-gray-800/60 border-gray-700/80 text-gray-400 hover:text-gray-200"
                      }`}
                    >
                      <div className="font-bold">Entire Broadcast Day</div>
                      <div className="text-[10px] text-gray-400 mt-0.5">
                        00:00:00 to 23:59:59
                      </div>
                    </button>
                  </div>
                </div>
              )}

              <p className="text-xs text-gray-300">
                Scanning timeline for unprogrammed intervals on <strong>{selectedDate}</strong>:
              </p>

              {isDetectingGaps ? (
                <div className="py-8 text-center text-gray-400 text-xs flex flex-col items-center gap-2">
                  <Loader2 className="w-6 h-6 animate-spin text-amber-400" />
                  <span>Scanning timeline intervals...</span>
                </div>
              ) : detectedGaps.length === 0 ? (
                <div className="p-4 bg-emerald-950/40 border border-emerald-800 rounded-xl text-xs text-emerald-300 flex items-center gap-2">
                  <Check className="w-5 h-5 text-emerald-400 shrink-0" />
                  <span>No unprogrammed gaps found on {selectedDate}. Schedule timeline is 100% contiguous!</span>
                </div>
              ) : (
                <div className="space-y-3">
                  <div className="max-h-48 overflow-y-auto bg-[#141b2b] border border-gray-800 rounded-xl divide-y divide-gray-800/60 p-2">
                    {detectedGaps.map((gap, idx) => {
                      const sStr = formatTimeInTimezone(gap.start_time, broadcastTimezone, false);
                      const eStr = formatTimeInTimezone(gap.end_time, broadcastTimezone, false);
                      const durMins = Math.floor(gap.duration_seconds / 60);
                      return (
                        <div key={idx} className="p-2 text-xs flex items-center justify-between text-gray-300">
                          <span className="font-mono text-amber-300 font-bold">{sStr} ➔ {eStr}</span>
                          <span className="text-[11px] text-gray-400 font-mono">{durMins} min gap</span>
                        </div>
                      );
                    })}
                  </div>

                  <div className="space-y-3 pt-2 border-t border-gray-800">
                    <div>
                      <label className="block text-xs font-medium text-gray-400 mb-1">Filler Content Title</label>
                      <input
                        type="text"
                        value={fillerTitle}
                        onChange={(e) => setFillerTitle(e.target.value)}
                        className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-xs text-white focus:outline-none focus:border-amber-500"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-gray-400 mb-1">Filler Media Path</label>
                      <input
                        type="text"
                        value={fillerMedia}
                        onChange={(e) => setFillerMedia(e.target.value)}
                        className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-xs text-white font-mono focus:outline-none focus:border-amber-500"
                      />
                    </div>
                  </div>
                </div>
              )}
            </div>

            <div className="p-4 border-t border-gray-800 flex justify-end gap-2 bg-[#151c2c]">
              <button
                onClick={() => setIsGapModalOpen(false)}
                className="px-4 py-2 bg-gray-800 hover:bg-gray-700 text-gray-300 rounded-lg text-xs font-semibold"
              >
                Close
              </button>
              {detectedGaps.length > 0 && (
                <button
                  disabled={isFillingGaps}
                  onClick={handleAutoFillGaps}
                  className="px-4 py-2 bg-amber-600 hover:bg-amber-500 text-white rounded-lg text-xs font-semibold flex items-center gap-1.5 shadow-md shadow-amber-900/30 disabled:opacity-50"
                >
                  {isFillingGaps && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
                  Auto-Fill {detectedGaps.length} Gap(s)
                </button>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
