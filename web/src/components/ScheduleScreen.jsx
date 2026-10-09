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
  ChevronRight
} from 'lucide-react';
import { api } from '../api';

export function ScheduleScreen({
  channels = [],
  activeChannelId,
  onSwitchChannel,
  scheduleItems = [],
  onRefreshSchedule,
  onShowToast,
  adTemplates = [],
  t
}) {
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [currentPath, setCurrentPath] = useState("");
  const [fileList, setFileList] = useState([]);
  const [fileFilterQuery, setFileFilterQuery] = useState("");
  const [selectedFile, setSelectedFile] = useState(null);

  // Form State
  const [title, setTitle] = useState("");
  const [startTime, setStartTime] = useState("16:00:00");
  const [duration, setDuration] = useState("02:15:00");
  const [endTime, setEndTime] = useState("18:15:00");

  // TMDb Typeahead State
  const [tmdbQuery, setTmdbQuery] = useState("");
  const [tmdbHits, setTmdbHits] = useState([]);
  const [isTmdbDropdownOpen, setIsTmdbDropdownOpen] = useState(false);
  const [isSearchingTmdb, setIsSearchingTmdb] = useState(false);
  const [tmdbResult, setTmdbResult] = useState(null);
  const tmdbRef = useRef(null);

  // Conflict State
  const [conflictReport, setConflictReport] = useState(null);
  const [isCheckingConflict, setIsCheckingConflict] = useState(false);
  const [selectedConflictAction, setSelectedConflictAction] = useState(""); // RIPPLE, ADJUST_START, ADJUST_END, FORCE_PREEMPT

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

  // Check conflicts whenever channel, start time, or duration changes
  useEffect(() => {
    if (!isModalOpen || !activeChannelId) return;

    const timer = setTimeout(async () => {
      try {
        setIsCheckingConflict(true);
        const now = new Date();
        const sParts = startTime.split(':').map(Number);
        const dParts = duration.split(':').map(Number);
        const durSecs = (dParts[0] || 0) * 3600 + (dParts[1] || 0) * 60 + (dParts[2] || 0);
        const startIso = new Date(now.getFullYear(), now.getMonth(), now.getDate(), sParts[0] || 0, sParts[1] || 0, sParts[2] || 0).toISOString();
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
        // Fallback local check
        checkLocalConflicts();
      } finally {
        setIsCheckingConflict(false);
      }
    }, 200);

    return () => clearTimeout(timer);
  }, [isModalOpen, activeChannelId, startTime, duration]);

  const checkLocalConflicts = () => {
    try {
      const now = new Date();
      const sParts = startTime.split(':').map(Number);
      const dParts = duration.split(':').map(Number);
      const durSecs = (dParts[0] || 0) * 3600 + (dParts[1] || 0) * 60 + (dParts[2] || 0);
      const sMs = new Date(now.getFullYear(), now.getMonth(), now.getDate(), sParts[0] || 0, sParts[1] || 0, sParts[2] || 0).getTime();
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

  // Close TMDb dropdown on outside click
  useEffect(() => {
    function handleClickOutside(event) {
      if (tmdbRef.current && !tmdbRef.current.contains(event.target)) {
        setIsTmdbDropdownOpen(false);
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

  // Conflict Resolution Action Buttons
  const handleSnapStartTime = () => {
    if (!conflictReport?.suggested_start) return;
    try {
      const dt = new Date(conflictReport.suggested_start);
      const hh = String(dt.getHours()).padStart(2, '0');
      const mm = String(dt.getMinutes()).padStart(2, '0');
      const ss = String(dt.getSeconds()).padStart(2, '0');
      setStartTime(`${hh}:${mm}:${ss}`);
      setSelectedConflictAction("ADJUST_START");
      onShowToast(`Snapped start time to ${hh}:${mm}:${ss} (after colliding program)`, "success");
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

    const now = new Date();
    const sParts = startTime.split(':').map(Number);
    const startIso = new Date(now.getFullYear(), now.getMonth(), now.getDate(), sParts[0] || 0, sParts[1] || 0, sParts[2] || 0).toISOString();
    const endIso = new Date(new Date(startIso).getTime() + durSecs * 1000).toISOString();

    const payload = {
      item: {
        channel_id: activeChannelId,
        program_title: title,
        title: title,
        media_path: selectedFile?.path || selectedFile?.relative_path || selectedFile?.name || "sample_broadcast_promo.mp4",
        media_file_path: selectedFile?.path || selectedFile?.relative_path || selectedFile?.name || "sample_broadcast_promo.mp4",
        start_time: startIso,
        duration_seconds: durSecs,
        end_time: endIso,
        tmdb_id: tmdbResult?.id || tmdbResult?.tmdb_id || "",
        tmdb_poster: tmdbResult?.poster_path || tmdbResult?.poster_url || "",
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
      onShowToast(`Scheduled "${title}" successfully!`, "success");
      setIsModalOpen(false);
      setConflictReport(null);
      setSelectedConflictAction("");
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

  return (
    <div className="h-full flex flex-col p-4 space-y-4 overflow-y-auto">
      {/* Top Scheduling Bar */}
      <div className="flex items-center justify-between shrink-0 bg-[#111827] border border-[#1F2937] p-3 rounded-lg">
        <div className="flex items-center gap-3">
          <Calendar className="w-5 h-5 text-indigo-400" />
          <div>
            <h2 className="text-sm font-bold text-white">
              {t('sched.title') || "24/7 Playout Schedule & EPG Master"}
            </h2>
            <p className="text-[11px] text-gray-400">
              {t('sched.subtitle') || "Manage linear playlists, TMDb metadata, audio PID tracks & conflict resolution"}
            </p>
          </div>
          <div className="h-6 w-px bg-gray-700"></div>

          {/* Channel selector */}
          <select
            value={activeChannelId}
            onChange={(e) => onSwitchChannel(e.target.value)}
            className="bg-[#1F2937] border border-gray-700 text-xs text-white rounded px-2.5 py-1.5 font-medium"
          >
            {channels.map((ch) => (
              <option key={ch.id} value={ch.id}>
                Channel: CH {String(ch.lcn || 1).padStart(2, '0')} - {ch.name}
              </option>
            ))}
          </select>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={() => {
              setIsModalOpen(true);
              setConflictReport(null);
              setSelectedConflictAction("");
            }}
            className="px-3.5 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-xs font-semibold rounded text-white shadow-sm flex items-center gap-1.5 transition-colors"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>+ Add Media / Schedule Movie</span>
          </button>
          <button
            onClick={handleExportXmltv}
            className="px-3 py-1.5 bg-[#1F2937] hover:bg-[#374151] text-xs font-medium rounded text-gray-200 border border-gray-700 flex items-center gap-1.5 transition-colors"
          >
            <Download className="w-3.5 h-3.5" />
            <span>Export XMLTV / DVB-EIT</span>
          </button>
        </div>
      </div>

      {/* 24-Hour Visual Schedule Bar */}
      <div className="bg-[#111827] border border-[#1F2937] rounded-lg p-3 shrink-0 space-y-2">
        <div className="flex items-center justify-between text-[11px] font-mono text-gray-400">
          <span>00:00:00</span>
          <span>06:00:00</span>
          <span>12:00:00</span>
          <span className="text-emerald-400 font-bold underline">16:15:00 (ON-AIR NOW)</span>
          <span>20:00:00</span>
          <span>23:59:59</span>
        </div>
        <div className="h-6 w-full bg-gray-900 rounded flex overflow-hidden border border-gray-800 text-[10px] font-bold text-white select-none">
          <div style={{ width: '25%' }} className="bg-blue-700/80 flex items-center justify-center truncate px-1 border-r border-gray-900">
            Morning Classics
          </div>
          <div style={{ width: '15%' }} className="bg-indigo-700/80 flex items-center justify-center truncate px-1 border-r border-gray-900">
            News Live
          </div>
          <div style={{ width: '28%' }} className="bg-amber-600/80 flex items-center justify-center truncate px-1 border-r border-gray-900">
            Blockbuster Cinema
          </div>
          <div style={{ width: '18%' }} className="bg-emerald-600 flex items-center justify-center truncate px-1 border-r border-gray-900 animate-pulse">
            ▶ On-Air Feature
          </div>
          <div style={{ width: '14%' }} className="bg-blue-700/80 flex items-center justify-center truncate px-1 border-r border-gray-900">
            Prime Special
          </div>
        </div>
      </div>

      {/* Schedule Items List */}
      <div className="flex-1 bg-[#111827] border border-[#1F2937] rounded-lg p-4 space-y-3 overflow-y-auto">
        <div className="flex items-center justify-between border-b border-gray-800 pb-2">
          <div className="flex items-center gap-2">
            <Clock className="w-4 h-4 text-gray-400" />
            <h3 className="text-xs font-bold text-gray-200 uppercase tracking-wider">
              Channel Broadcast Events ({scheduleItems.length})
            </h3>
          </div>
          <span className="text-[11px] font-mono text-gray-400">
            SMPTE Linear Playout Timeline
          </span>
        </div>

        <div className="space-y-2">
          {scheduleItems.length === 0 ? (
            <div className="text-center py-12 text-gray-500 font-mono text-xs">
              No programs scheduled on this channel. Click "+ Add Media / Schedule Movie" above to add.
            </div>
          ) : (
            scheduleItems.map((item) => {
              const startStr = item.start_time ? new Date(item.start_time).toLocaleTimeString() : "16:00:00";
              const endStr = item.end_time ? new Date(item.end_time).toLocaleTimeString() : "18:15:00";
              const durMinutes = Math.floor((item.duration_seconds || 3600) / 60);

              return (
                <div
                  key={item.id}
                  className="bg-[#1F2937] hover:bg-[#253045] border border-gray-700/80 rounded-lg p-3 flex items-center justify-between gap-4 transition-all"
                >
                  <div className="flex items-center gap-3 min-w-0">
                    {/* Poster thumbnail or film icon */}
                    {item.tmdb_poster ? (
                      <img
                        src={item.tmdb_poster}
                        alt="Poster"
                        className="w-10 h-14 object-cover rounded shadow border border-gray-700 shrink-0"
                      />
                    ) : (
                      <div className="w-10 h-14 bg-gray-800 border border-gray-700 rounded flex items-center justify-center shrink-0 text-gray-500">
                        <Film className="w-5 h-5 text-indigo-400" />
                      </div>
                    )}

                    <div className="min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="font-bold text-white text-xs truncate">
                          {item.program_title || item.title || "Untitled Program"}
                        </span>
                        {item.tmdb_id && (
                          <span className="text-[9px] px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-300 font-mono">
                            TMDb Linked
                          </span>
                        )}
                      </div>
                      <div className="text-[11px] text-gray-400 font-mono flex items-center gap-2 mt-0.5">
                        <span className="text-indigo-400 font-semibold">{startStr} - {endStr}</span>
                        <span>•</span>
                        <span>{durMinutes} min ({item.duration_seconds}s)</span>
                      </div>
                      <div className="text-[10px] text-gray-400 truncate mt-0.5 font-mono">
                        Path: {item.media_path || item.media_file_path || "sample_promo.mp4"}
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center gap-2 shrink-0">
                    <button
                      onClick={() => handleDeleteItem(item.id)}
                      className="p-1.5 text-gray-400 hover:text-red-400 hover:bg-red-950/40 rounded transition-colors"
                      title="Remove from Playout Calendar"
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

      {/* Enhanced Add Media / Schedule Modal with Intuitive File Picker & TMDb Typeahead */}
      {isModalOpen && (
        <div className="fixed inset-0 bg-black/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-[#111827] border border-[#2D3A54] rounded-xl w-full max-w-2xl max-h-[92vh] flex flex-col shadow-2xl overflow-hidden">
            {/* Modal Header */}
            <div className="px-5 py-3.5 bg-[#1A2234] border-b border-[#2D3A54] flex items-center justify-between shrink-0">
              <h3 className="text-sm font-bold text-white flex items-center gap-2">
                <Film className="w-4 h-4 text-indigo-400" />
                <span>Add Media to Playout Schedule</span>
              </h3>
              <button
                onClick={() => {
                  setIsModalOpen(false);
                  setConflictReport(null);
                  setSelectedConflictAction("");
                }}
                className="text-gray-400 hover:text-white"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="p-5 space-y-4 overflow-y-auto text-xs">
              {/* SECTION 1: Intuitive Local Media Library File Picker with Type-Ahead Filter */}
              <div className="space-y-2 bg-[#161F30] p-3.5 rounded-lg border border-[#23314B]">
                <div className="flex items-center justify-between">
                  <label className="text-[11px] font-bold text-gray-200 uppercase tracking-wider flex items-center gap-1.5">
                    <FolderOpen className="w-3.5 h-3.5 text-indigo-400" />
                    <span>Media File Picker (./media{currentPath ? `/${currentPath}` : ''})</span>
                  </label>
                  {currentPath && (
                    <button
                      type="button"
                      onClick={() => {
                        const parent = currentPath.includes('/') ? currentPath.substring(0, currentPath.lastIndexOf('/')) : '';
                        setCurrentPath(parent);
                      }}
                      className="text-[10px] text-indigo-400 hover:text-indigo-300 font-mono px-2 py-0.5 bg-gray-800 rounded border border-gray-700"
                    >
                      .. (Up Level)
                    </button>
                  )}
                </div>

                {/* Type-Ahead Filter Input for long lists of media files */}
                <div className="relative">
                  <Search className="w-3.5 h-3.5 text-gray-400 absolute left-2.5 top-2.5" />
                  <input
                    type="text"
                    value={fileFilterQuery}
                    onChange={(e) => setFileFilterQuery(e.target.value)}
                    placeholder="Type-ahead search/filter media files and folders..."
                    className="w-full bg-[#0B0F17] border border-gray-700 rounded pl-8 pr-2.5 py-1.5 text-xs text-white placeholder-gray-500 font-mono"
                  />
                  {fileFilterQuery && (
                    <button
                      type="button"
                      onClick={() => setFileFilterQuery("")}
                      className="absolute right-2.5 top-2 text-gray-400 hover:text-white"
                    >
                      <X className="w-3 h-3" />
                    </button>
                  )}
                </div>

                {/* File / Folder Items List */}
                <div className="bg-[#0B0F17] border border-gray-800 rounded-lg max-h-40 overflow-y-auto divide-y divide-gray-800">
                  {filteredFiles.length === 0 ? (
                    <div className="p-3 text-gray-500 font-mono text-[11px]">
                      {fileList.length === 0 ? "No media files found in ./media directory." : `No media matching "${fileFilterQuery}".`}
                    </div>
                  ) : (
                    filteredFiles.map((f, i) => {
                      const isSelected = selectedFile?.path === f.path || selectedFile?.name === f.name;
                      return (
                        <div
                          key={i}
                          onClick={() => handleItemClick(f)}
                          className={`p-2 flex items-center justify-between hover:bg-[#1E293B] cursor-pointer transition-colors ${
                            isSelected ? 'bg-indigo-950/70 border-l-2 border-indigo-500' : ''
                          }`}
                        >
                          <div className="flex items-center gap-2 truncate">
                            {f.is_dir ? (
                              <FolderOpen className="w-3.5 h-3.5 text-amber-400 shrink-0" />
                            ) : (
                              <FileVideo className="w-3.5 h-3.5 text-sky-400 shrink-0" />
                            )}
                            <span className={`font-mono truncate ${f.is_dir ? 'text-amber-200 font-semibold' : 'text-sky-200'}`}>
                              {f.name}
                            </span>
                          </div>
                          {!f.is_dir && (
                            <span className="text-[10px] text-emerald-400 font-mono shrink-0 ml-2">
                              {f.duration_seconds ? `${Math.floor(f.duration_seconds / 60)}m` : (f.probed_duration || "02:15:00")}
                            </span>
                          )}
                        </div>
                      );
                    })
                  )}
                </div>

                {selectedFile && (
                  <div className="text-[11px] text-sky-300 font-mono bg-sky-950/40 border border-sky-800/60 p-2 rounded flex items-center justify-between">
                    <span className="truncate">Selected File: {selectedFile.name}</span>
                    <span className="text-emerald-400 font-bold shrink-0 ml-2">
                      {selectedFile.duration_seconds ? `${Math.floor(selectedFile.duration_seconds / 60)}m` : "Ready"}
                    </span>
                  </div>
                )}
              </div>

              {/* SECTION 2: Program Title */}
              <div>
                <label className="block text-[11px] text-gray-400 mb-1 font-medium">Program / Feature Title</label>
                <input
                  type="text"
                  value={title}
                  placeholder="Enter program or movie title..."
                  onChange={(e) => setTitle(e.target.value)}
                  className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white font-medium"
                />
              </div>

              {/* SECTION 3: Time Parameters */}
              <div className="grid grid-cols-3 gap-3">
                <div>
                  <label className="block text-[11px] text-gray-400 mb-1">Start Time (IST/UTC)</label>
                  <input
                    type="time"
                    step="1"
                    value={startTime}
                    onChange={(e) => setStartTime(e.target.value)}
                    className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white font-mono"
                  />
                </div>
                <div>
                  <label className="block text-[11px] text-gray-400 mb-1">Duration (HH:MM:SS)</label>
                  <input
                    type="text"
                    value={duration}
                    onChange={(e) => setDuration(e.target.value)}
                    className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white font-mono"
                  />
                </div>
                <div>
                  <label className="block text-[11px] text-gray-400 mb-1">Calculated End Time</label>
                  <input
                    type="text"
                    readOnly
                    value={endTime}
                    className="w-full bg-[#0B0F17] border border-gray-800 rounded px-2.5 py-1.5 text-xs text-emerald-400 font-mono"
                  />
                </div>
              </div>

              {/* SECTION 4: Live Conflict Warning & Resolution Action Panel */}
              {conflictReport && conflictReport.has_conflict && (
                <div className="bg-amber-950/40 border border-amber-600/70 p-3.5 rounded-lg space-y-3 animate-in fade-in">
                  <div className="flex items-start gap-2.5 text-amber-300">
                    <AlertTriangle className="w-5 h-5 text-amber-400 shrink-0 mt-0.5" />
                    <div className="text-xs">
                      <div className="font-bold">Broadcast Schedule Collision Detected</div>
                      <div className="text-[11px] text-amber-200/80 mt-0.5">
                        This time slot overlaps with {conflictReport.conflicts?.length || 1} existing program(s) on Channel timeline:
                        <span className="font-bold text-white ml-1">
                          "{conflictReport.conflicts?.[0]?.program_title || conflictReport.conflicts?.[0]?.title || 'Scheduled Program'}"
                        </span>
                      </div>
                    </div>
                  </div>

                  {/* Conflict Resolution Strategy Options */}
                  <div className="pt-2 border-t border-amber-800/40 space-y-2">
                    <div className="text-[10px] font-bold text-amber-300 uppercase tracking-wider">
                      Select Conflict Resolution Strategy:
                    </div>

                    <div className="grid grid-cols-2 gap-2 text-[11px]">
                      {/* Option 1: Snap Start Time */}
                      <button
                        type="button"
                        onClick={handleSnapStartTime}
                        className={`p-2 rounded border text-left flex items-start gap-2 transition-all ${
                          selectedConflictAction === "ADJUST_START"
                            ? "bg-indigo-600 border-indigo-400 text-white shadow"
                            : "bg-[#1E293B] border-gray-700 text-gray-300 hover:text-white hover:border-gray-600"
                        }`}
                      >
                        <FastForward className="w-3.5 h-3.5 text-sky-400 shrink-0 mt-0.5" />
                        <div>
                          <div className="font-bold">Snap Start Time</div>
                          <div className="text-[10px] text-gray-400">
                            Start immediately after previous program finishes
                          </div>
                        </div>
                      </button>

                      {/* Option 2: Trim Duration */}
                      <button
                        type="button"
                        onClick={handleTrimDuration}
                        className={`p-2 rounded border text-left flex items-start gap-2 transition-all ${
                          selectedConflictAction === "ADJUST_END"
                            ? "bg-indigo-600 border-indigo-400 text-white shadow"
                            : "bg-[#1E293B] border-gray-700 text-gray-300 hover:text-white hover:border-gray-600"
                        }`}
                      >
                        <Scissors className="w-3.5 h-3.5 text-amber-400 shrink-0 mt-0.5" />
                        <div>
                          <div className="font-bold">Trim Duration</div>
                          <div className="text-[10px] text-gray-400">
                            Fit playback exactly before next scheduled item
                          </div>
                        </div>
                      </button>

                      {/* Option 3: Ripple Next Programs */}
                      <button
                        type="button"
                        onClick={() => handleSelectAction("RIPPLE")}
                        className={`p-2 rounded border text-left flex items-start gap-2 transition-all ${
                          selectedConflictAction === "RIPPLE"
                            ? "bg-indigo-600 border-indigo-400 text-white shadow"
                            : "bg-[#1E293B] border-gray-700 text-gray-300 hover:text-white hover:border-gray-600"
                        }`}
                      >
                        <ArrowRight className="w-3.5 h-3.5 text-emerald-400 shrink-0 mt-0.5" />
                        <div>
                          <div className="font-bold">Ripple Next Plays</div>
                          <div className="text-[10px] text-gray-400">
                            Push colliding & future items forward seamlessly
                          </div>
                        </div>
                      </button>

                      {/* Option 4: Force Preempt */}
                      <button
                        type="button"
                        onClick={() => handleSelectAction("FORCE_PREEMPT")}
                        className={`p-2 rounded border text-left flex items-start gap-2 transition-all ${
                          selectedConflictAction === "FORCE_PREEMPT"
                            ? "bg-red-600 border-red-400 text-white shadow"
                            : "bg-[#1E293B] border-gray-700 text-gray-300 hover:text-white hover:border-gray-600"
                        }`}
                      >
                        <ShieldAlert className="w-3.5 h-3.5 text-red-400 shrink-0 mt-0.5" />
                        <div>
                          <div className="font-bold">Force Preempt Slot</div>
                          <div className="text-[10px] text-gray-400">
                            Cut into / overwrite colliding program slot
                          </div>
                        </div>
                      </button>
                    </div>
                  </div>
                </div>
              )}

              {/* SECTION 5: TMDb Metadata Type-Ahead Search with Rich Movie Dropdown */}
              <div className="space-y-2 pt-2 border-t border-gray-800" ref={tmdbRef}>
                <label className="block text-[11px] font-bold text-gray-300 uppercase tracking-wider">
                  TMDb Metadata & EPG Enrichment (Type-Ahead Search)
                </label>

                <div className="relative">
                  <div className="flex gap-2">
                    <div className="relative flex-1">
                      <Search className="w-3.5 h-3.5 text-gray-400 absolute left-2.5 top-2.5" />
                      <input
                        type="text"
                        value={tmdbQuery}
                        onChange={handleTmdbInputChange}
                        onFocus={() => {
                          if (tmdbHits.length > 0) setIsTmdbDropdownOpen(true);
                        }}
                        placeholder="Type movie or series title (e.g. Jawan, RRR, Pathaan, KGF)..."
                        className="w-full bg-[#1F2937] border border-gray-700 rounded pl-8 pr-2.5 py-1.5 text-xs text-white"
                      />
                      {isSearchingTmdb && (
                        <Loader2 className="w-3.5 h-3.5 text-indigo-400 animate-spin absolute right-2.5 top-2.5" />
                      )}
                    </div>
                    <button
                      type="button"
                      onClick={() => triggerTmdbSearch(tmdbQuery)}
                      className="px-3 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded text-xs font-semibold flex items-center gap-1 shrink-0"
                    >
                      <Search className="w-3.5 h-3.5" />
                      <span>Lookup</span>
                    </button>
                  </div>

                  {/* Typeahead Dropdown Menu */}
                  {isTmdbDropdownOpen && tmdbHits.length > 0 && (
                    <div className="absolute left-0 right-0 top-full mt-1 bg-[#1A2234] border border-[#2D3A54] rounded-lg shadow-2xl z-50 max-h-56 overflow-y-auto divide-y divide-gray-800">
                      {tmdbHits.map((hit) => {
                        const year = hit.release_date ? hit.release_date.substring(0, 4) : (hit.release_year || "2023");
                        const posterUrl = hit.poster_path || hit.poster_url;

                        return (
                          <div
                            key={hit.id}
                            onClick={() => handleSelectTmdbHit(hit)}
                            className="p-2.5 flex items-center gap-3 hover:bg-[#25324C] cursor-pointer transition-colors"
                          >
                            {posterUrl ? (
                              <img
                                src={posterUrl}
                                alt="Poster"
                                className="w-9 h-12 object-cover rounded shadow border border-gray-700 shrink-0"
                              />
                            ) : (
                              <div className="w-9 h-12 bg-gray-800 border border-gray-700 rounded flex items-center justify-center shrink-0">
                                <Film className="w-4 h-4 text-gray-500" />
                              </div>
                            )}

                            <div className="flex-1 min-w-0">
                              <div className="flex items-center gap-2">
                                <span className="font-bold text-white text-xs truncate">{hit.title}</span>
                                <span className="text-[10px] text-gray-400 font-mono">({year})</span>
                                {hit.rating && (
                                  <span className="text-[9px] px-1 py-0.2 rounded bg-amber-500/20 text-amber-300 font-mono">
                                    ★ {Number(hit.rating).toFixed(1)}
                                  </span>
                                )}
                              </div>
                              <p className="text-[10px] text-gray-400 truncate mt-0.5">
                                {hit.overview || "Broadcast feature presentation."}
                              </p>
                            </div>
                            <ChevronRight className="w-3.5 h-3.5 text-gray-500 shrink-0" />
                          </div>
                        );
                      })}
                    </div>
                  )}
                </div>

                {/* Enriched TMDb Preview Card */}
                {tmdbResult && (
                  <div className="bg-[#1A2234] border border-[#2D3A54] p-3 rounded-lg flex gap-3 items-start animate-in fade-in">
                    {(tmdbResult.poster_path || tmdbResult.poster_url) && (
                      <img
                        src={tmdbResult.poster_path || tmdbResult.poster_url}
                        alt="Poster"
                        className="w-14 h-20 object-cover rounded shadow border border-gray-700 shrink-0"
                      />
                    )}
                    <div className="flex-1 min-w-0">
                      <div className="font-bold text-white text-xs truncate">
                        {tmdbResult.title} ({tmdbResult.release_date?.substring(0, 4) || tmdbResult.release_year || "2023"})
                      </div>
                      <p className="text-[11px] text-gray-300 line-clamp-3 mt-1">
                        {tmdbResult.overview}
                      </p>
                      <div className="text-[10px] text-emerald-400 font-mono mt-1">
                        TMDb ID: {tmdbResult.id || tmdbResult.tmdb_id} • Enriched EPG Artwork
                      </div>
                    </div>
                  </div>
                )}
              </div>
            </div>

            {/* Modal Actions */}
            <div className="px-5 py-3 bg-[#1A2234] border-t border-[#2D3A54] flex items-center justify-between shrink-0">
              <div className="text-[11px] text-gray-400 font-mono">
                {conflictReport?.has_conflict ? (
                  <span className="text-amber-400 font-semibold flex items-center gap-1">
                    <AlertTriangle className="w-3.5 h-3.5" />
                    <span>Action: {selectedConflictAction || "RIPPLE"}</span>
                  </span>
                ) : (
                  <span className="text-emerald-400 flex items-center gap-1">
                    <Check className="w-3.5 h-3.5" />
                    <span>Timeline slot clean & clear</span>
                  </span>
                )}
              </div>

              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={() => setIsModalOpen(false)}
                  className="px-3.5 py-1.5 bg-gray-800 hover:bg-gray-700 text-gray-300 rounded text-xs font-semibold transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={handleCommitSchedule}
                  className="px-4 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded text-xs font-semibold shadow transition-colors"
                >
                  Commit to Timeline
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
