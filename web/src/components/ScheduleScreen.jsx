import React, { useState, useEffect } from 'react';
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
  X
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
  const [selectedFile, setSelectedFile] = useState(null);

  // Form State
  const [title, setTitle] = useState("");
  const [startTime, setStartTime] = useState("16:00:00");
  const [duration, setDuration] = useState("02:15:00");
  const [endTime, setEndTime] = useState("18:15:00");
  const [tmdbQuery, setTmdbQuery] = useState("");
  const [tmdbResult, setTmdbResult] = useState(null);
  const [isSearchingTmdb, setIsSearchingTmdb] = useState(false);

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

  const handleItemClick = (item) => {
    if (item.is_dir) {
      setCurrentPath(item.path);
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
    handleSearchTmdb(cleanTitle);
  };

  const handleSearchTmdb = async (queryToSearch) => {
    const q = queryToSearch || tmdbQuery;
    if (!q) return;
    setIsSearchingTmdb(true);
    try {
      const results = await api.searchTmdb(q);
      if (Array.isArray(results) && results.length > 0) {
        setTmdbResult(results[0]);
        if (!title) {
          setTitle(results[0].title);
        }
      } else {
        setTmdbResult(null);
      }
    } catch (err) {
      setTmdbResult(null);
    } finally {
      setIsSearchingTmdb(false);
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
        tmdb_metadata: tmdbResult || {
          title: title,
          overview: "Broadcast feature event."
        }
      },
      action: "FORCE_OVERWRITE"
    };

    try {
      await api.createScheduleItem(payload);
      onShowToast(`Scheduled "${title}" successfully!`, "success");
      setIsModalOpen(false);
      onRefreshSchedule(activeChannelId);
    } catch (err) {
      onShowToast("Failed to schedule media: " + err.message, "error");
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
            onClick={() => setIsModalOpen(true)}
            className="px-3.5 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-xs font-semibold rounded text-white shadow-sm flex items-center gap-1.5"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>+ Add Media / Schedule Movie</span>
          </button>
          <button
            onClick={handleExportXmltv}
            className="px-3 py-1.5 bg-[#1F2937] hover:bg-[#374151] text-xs font-medium rounded text-gray-200 border border-gray-700 flex items-center gap-1.5"
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

      {/* Schedule Items Table */}
      <div className="flex-1 bg-[#111827] border border-[#1F2937] rounded-lg overflow-hidden flex flex-col min-h-0">
        <div className="bg-[#1A2234] px-4 py-2 border-b border-[#2D3A54] grid grid-cols-12 text-[11px] font-bold text-gray-300 uppercase tracking-wider">
          <div className="col-span-1">Status</div>
          <div className="col-span-2">Timecode (IST)</div>
          <div className="col-span-1">Duration</div>
          <div className="col-span-4">Program / Movie Title</div>
          <div className="col-span-2">Ad Overlay Precedence</div>
          <div className="col-span-1">Audio Track</div>
          <div className="col-span-1 text-right">Actions</div>
        </div>

        <div className="flex-1 overflow-y-auto divide-y divide-[#1F2937] text-xs">
          {scheduleItems.length === 0 ? (
            <div className="p-8 text-center text-gray-500 font-mono text-xs">
              No schedule items found for this channel. Click "+ Add Media" to populate the linear timeline.
            </div>
          ) : (
            scheduleItems.map((item, idx) => {
              const startStr = item.start_time ? new Date(item.start_time).toLocaleTimeString() : "16:00:00";
              const endStr = item.end_time ? new Date(item.end_time).toLocaleTimeString() : "18:15:00";
              const durSec = item.duration_seconds || 8100;
              const durH = String(Math.floor(durSec / 3600)).padStart(2, '0');
              const durM = String(Math.floor((durSec % 3600) / 60)).padStart(2, '0');
              const durS = String(durSec % 60).padStart(2, '0');

              return (
                <div
                  key={item.id || idx}
                  className={`px-4 py-3 grid grid-cols-12 items-center transition-colors ${
                    idx === 0
                      ? 'bg-emerald-950/20 hover:bg-emerald-900/30'
                      : 'hover:bg-gray-800/40'
                  }`}
                >
                  <div className="col-span-1">
                    {idx === 0 ? (
                      <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-emerald-500/20 text-emerald-400 border border-emerald-500/30 flex items-center gap-1 w-max font-mono">
                        <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-ping"></span>
                        ON-AIR
                      </span>
                    ) : (
                      <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-blue-500/20 text-blue-300 border border-blue-500/30 font-mono">
                        QUEUED
                      </span>
                    )}
                  </div>

                  <div className="col-span-2 font-mono text-gray-200">
                    {startStr} - {endStr}
                  </div>

                  <div className="col-span-1 font-mono text-gray-400">
                    {durH}:{durM}:{durS}
                  </div>

                  <div className="col-span-4 flex items-center gap-2">
                    {item.tmdb_metadata?.poster_url ? (
                      <img
                        src={item.tmdb_metadata.poster_url}
                        alt="Poster"
                        className="w-7 h-10 object-cover rounded shadow shrink-0 border border-gray-700"
                      />
                    ) : (
                      <div className="w-7 h-10 bg-gray-800 rounded flex items-center justify-center shrink-0 border border-gray-700">
                        <Film className="w-3.5 h-3.5 text-gray-500" />
                      </div>
                    )}
                    <div className="truncate">
                      <div className="font-bold text-white truncate">
                        {item.program_title || item.title || item.tmdb_metadata?.title || "Scheduled Broadcast Program"}
                      </div>
                      <div className="text-[10px] text-gray-400 truncate font-mono">
                        {item.media_path || item.media_file_path || "storage://broadcast_vault/feature.mkv"}
                      </div>
                    </div>
                  </div>

                  <div className="col-span-2">
                    <span className="text-[10px] px-2 py-0.5 rounded bg-indigo-500/10 text-indigo-300 border border-indigo-500/20 font-medium">
                      Station Bug + Ticker
                    </span>
                  </div>

                  <div className="col-span-1 font-mono text-[10px] text-emerald-400">
                    PID 101 • 5.1
                  </div>

                  <div className="col-span-1 text-right">
                    <button
                      onClick={() => handleDeleteItem(item.id)}
                      className="text-gray-400 hover:text-rose-400 p-1"
                      title="Delete Schedule Item"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </div>
                </div>
              );
            })
          )}
        </div>
      </div>

      {/* Add Media Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 bg-black/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-[#111827] border border-[#2D3A54] rounded-xl w-full max-w-2xl max-h-[90vh] flex flex-col shadow-2xl overflow-hidden">
            <div className="px-5 py-3.5 bg-[#1A2234] border-b border-[#2D3A54] flex items-center justify-between">
              <h3 className="text-sm font-bold text-white flex items-center gap-2">
                <Film className="w-4 h-4 text-indigo-400" />
                <span>Add Media to Playout Schedule</span>
              </h3>
              <button onClick={() => setIsModalOpen(false)} className="text-gray-400 hover:text-white">
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="p-5 space-y-4 overflow-y-auto text-xs">
              {/* Local Media Library Browser */}
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <label className="text-[11px] font-bold text-gray-300 uppercase tracking-wider flex items-center gap-1.5">
                    <FolderOpen className="w-3.5 h-3.5 text-indigo-400" />
                    <span>Local Media Library (./media{currentPath ? `/${currentPath}` : ''})</span>
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

                <div className="bg-[#0B0F17] border border-gray-800 rounded-lg max-h-48 overflow-y-auto divide-y divide-gray-800">
                  {fileList.length === 0 ? (
                    <div className="p-3 text-gray-500 font-mono text-[11px]">
                      No media files found in ./media directory.
                    </div>
                  ) : (
                    fileList.map((f, i) => (
                      <div
                        key={i}
                        onClick={() => handleItemClick(f)}
                        className={`p-2 flex items-center justify-between hover:bg-[#1E293B] cursor-pointer transition-colors ${
                          selectedFile?.path === f.path ? 'bg-indigo-950/60 border-l-2 border-indigo-500' : ''
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
                    ))
                  )}
                </div>
              </div>

              {/* Program Title */}
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

              {/* Time Parameters */}
              <div className="grid grid-cols-3 gap-3">
                <div>
                  <label className="block text-[11px] text-gray-400 mb-1">Start Time (IST)</label>
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
                  <label className="block text-[11px] text-gray-400 mb-1">Auto End Time</label>
                  <input
                    type="text"
                    readOnly
                    value={endTime}
                    className="w-full bg-[#0B0F17] border border-gray-800 rounded px-2.5 py-1.5 text-xs text-emerald-400 font-mono"
                  />
                </div>
              </div>

              {/* TMDb Search & Poster Artwork */}
              <div className="space-y-2 pt-2 border-t border-gray-800">
                <label className="block text-[11px] font-bold text-gray-300 uppercase tracking-wider">
                  TMDb Metadata & EPG Enrichment
                </label>
                <div className="flex gap-2">
                  <input
                    type="text"
                    value={tmdbQuery}
                    onChange={(e) => setTmdbQuery(e.target.value)}
                    placeholder="Search movie title (e.g. Jawan, RRR, Pathaan)..."
                    className="flex-1 bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white"
                  />
                  <button
                    onClick={() => handleSearchTmdb()}
                    className="px-3 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded text-xs font-semibold flex items-center gap-1 shrink-0"
                  >
                    <Search className="w-3.5 h-3.5" />
                    <span>{isSearchingTmdb ? "Searching..." : "Lookup"}</span>
                  </button>
                </div>

                {tmdbResult && (
                  <div className="bg-[#1A2234] border border-[#2D3A54] p-3 rounded-lg flex gap-3 items-start animate-in fade-in">
                    {tmdbResult.poster_url && (
                      <img
                        src={tmdbResult.poster_url}
                        alt="Poster"
                        className="w-14 h-20 object-cover rounded shadow border border-gray-700 shrink-0"
                      />
                    )}
                    <div className="flex-1 min-w-0">
                      <div className="font-bold text-white text-xs truncate">
                        {tmdbResult.title} ({tmdbResult.release_year || "2023"})
                      </div>
                      <p className="text-[11px] text-gray-300 line-clamp-3 mt-1">
                        {tmdbResult.overview}
                      </p>
                      <div className="text-[10px] text-emerald-400 font-mono mt-1">
                        TMDb ID: {tmdbResult.tmdb_id} • Matched & Cached
                      </div>
                    </div>
                  </div>
                )}
              </div>
            </div>

            <div className="px-5 py-3 bg-[#1A2234] border-t border-[#2D3A54] flex justify-end gap-2">
              <button
                onClick={() => setIsModalOpen(false)}
                className="px-3.5 py-1.5 bg-gray-800 hover:bg-gray-700 text-gray-300 rounded text-xs font-semibold"
              >
                Cancel
              </button>
              <button
                onClick={handleCommitSchedule}
                className="px-4 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded text-xs font-semibold shadow"
              >
                Commit to Timeline
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
