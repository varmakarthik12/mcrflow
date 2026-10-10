import React, { useState, useEffect } from 'react';
import {
  Sliders,
  Users,
  Radio,
  Bot,
  Plus,
  Trash2,
  Copy,
  Check,
  Terminal,
  Send,
  ShieldAlert,
  X,
  Pencil,
  Activity,
  Wifi,
  WifiOff,
  CheckCircle2,
  AlertTriangle,
  Loader2,
  Globe,
  Clock,
  Save
} from 'lucide-react';
import { api } from '../api';
import { BROADCAST_TIMEZONES, DEFAULT_TIMEZONE, formatTimeInTimezone, formatDateInTimezone } from '../utils/timezone';

export function SettingsScreen({
  resolutions = [],
  onRefreshResolutions,
  users = [],
  onRefreshUsers,
  agents = [],
  onRefreshAgents,
  bots = [],
  onRefreshBots,
  onShowToast,
  currentUser,
  broadcastTimezone = "Asia/Kolkata",
  onUpdateTimezone
}) {
  const [activeTab, setActiveTab] = useState("timezone");

  // Broadcast Timezone State
  const [selectedTimezone, setSelectedTimezone] = useState(broadcastTimezone || DEFAULT_TIMEZONE);
  const [isSavingTz, setIsSavingTz] = useState(false);
  const [liveClock, setLiveClock] = useState({ time: "00:00:00", date: "" });

  useEffect(() => {
    if (broadcastTimezone) {
      setSelectedTimezone(broadcastTimezone);
    }
  }, [broadcastTimezone]);

  // Live clock ticker in selected timezone
  useEffect(() => {
    const tick = () => {
      const now = new Date();
      setLiveClock({
        time: formatTimeInTimezone(now, selectedTimezone, true),
        date: formatDateInTimezone(now, selectedTimezone)
      });
    };
    tick();
    const interval = setInterval(tick, 1000);
    return () => clearInterval(interval);
  }, [selectedTimezone]);

  // Custom Resolution Modal
  const [isResModalOpen, setIsResModalOpen] = useState(false);
  const [resForm, setResForm] = useState({
    name: "Custom 1080i50 PAL Deinterlaced",
    width: 1920,
    height: 1080,
    fps: 25.0,
    interlaced: true,
    vbitrate: 6500,
    abitrate: 192,
    vf: "yadif=0:-1:1,scale=1920:1080"
  });

  // User Creation Modal
  const [isUserModalOpen, setIsUserModalOpen] = useState(false);
  const [userForm, setUserForm] = useState({
    username: "",
    display_name: "",
    email: "",
    password: "",
    role: "operator"
  });

  // Agent Pairing Modal
  const [isPairModalOpen, setIsPairModalOpen] = useState(false);
  const [pairForm, setPairForm] = useState({
    hostname: "edge-transmitter-01",
    ip_address: "127.0.0.1",
    port: 3082,
    token: ""
  });

  // Agent Editing & Testing Modal
  const [isEditAgentModalOpen, setIsEditAgentModalOpen] = useState(false);
  const [editingAgent, setEditingAgent] = useState(null);
  const [editAgentForm, setEditAgentForm] = useState({
    hostname: "",
    ip_address: "127.0.0.1",
    port: 3082,
    token: ""
  });

  // Agent Connection Test State
  const [isTestingConn, setIsTestingConn] = useState(false);
  const [testConnResult, setTestConnResult] = useState(null);
  const [pingingAgentId, setPingingAgentId] = useState(null);

  // NLP Bot Tester
  const [nlpQuery, setNlpQuery] = useState("Cue blockbuster movie at 20:00");
  const [nlpOutput, setNlpOutput] = useState("");
  const [isTestingNlp, setIsTestingNlp] = useState(false);

  // Resolution CRUD
  const handleSaveResolution = async () => {
    if (!resForm.name) {
      onShowToast("Preset name is required", "error");
      return;
    }
    try {
      await api.createResolution({
        name: resForm.name,
        width: parseInt(resForm.width, 10),
        height: parseInt(resForm.height, 10),
        frame_rate: parseFloat(resForm.fps),
        fps: parseFloat(resForm.fps),
        interlaced: !!resForm.interlaced,
        scanning_mode: resForm.interlaced ? "interlaced" : "progressive",
        video_bitrate_kbps: parseInt(resForm.vbitrate, 10),
        audio_bitrate_kbps: parseInt(resForm.abitrate, 10),
        extra_ffmpeg_args: `-vf "${resForm.vf}"`,
        extra_ffmpeg_video_args: `-vf "${resForm.vf}"`
      });
      onShowToast(`Preset "${resForm.name}" registered!`, "success");
      setIsResModalOpen(false);
      onRefreshResolutions();
    } catch (err) {
      onShowToast("Failed to save resolution: " + err.message, "error");
    }
  };

  // User CRUD
  const handleCreateUser = async () => {
    if (!userForm.username || !userForm.password) {
      onShowToast("Username and password are required", "error");
      return;
    }
    try {
      await api.createUser({
        ...userForm,
        full_name: userForm.display_name || userForm.full_name || userForm.username,
        display_name: userForm.display_name || userForm.full_name || userForm.username
      });
      onShowToast(`User "${userForm.username}" created with role [${userForm.role}]`, "success");
      setIsUserModalOpen(false);
      setUserForm({ username: "", display_name: "", email: "", password: "", role: "operator" });
      onRefreshUsers();
    } catch (err) {
      onShowToast("Failed to create user: " + err.message, "error");
    }
  };

  const handleDeleteUser = async (id, username) => {
    try {
      await api.deleteUser(id);
      onShowToast(`User "${username}" deleted`, "info");
      onRefreshUsers();
    } catch (err) {
      onShowToast("Failed to delete user: " + err.message, "error");
    }
  };

  // Agent Testing & Management Handlers
  const handleTestConnection = async (ip, port, token) => {
    setIsTestingConn(true);
    setTestConnResult(null);
    try {
      const res = await api.testAgentConnection({
        ip_address: ip || "127.0.0.1",
        port: parseInt(port || 3082, 10),
        token: token || ""
      });
      setTestConnResult(res);
      if (res.reachable && res.authenticated) {
        onShowToast(`Agent reachable (${res.latency_ms}ms) and authenticated!`, "success");
      } else if (res.reachable && !res.authenticated) {
        onShowToast(`Agent reachable (${res.latency_ms}ms) but authentication failed!`, "warning");
      } else {
        onShowToast(`Agent connection failed: ${res.error || 'Connection refused'}`, "error");
      }
    } catch (err) {
      setTestConnResult({ reachable: false, authenticated: false, error: err.message });
      onShowToast(`Test connection error: ${err.message}`, "error");
    } finally {
      setIsTestingConn(false);
    }
  };

  const handlePingAgent = async (agentId) => {
    setPingingAgentId(agentId);
    try {
      const res = await api.pingAgent(agentId);
      if (res.reachable) {
        onShowToast(`Ping response: ${res.latency_ms}ms latency (Authenticated)`, "success");
      } else {
        onShowToast(`Agent unreachable: ${res.error || 'Connection refused'}`, "error");
      }
      onRefreshAgents();
    } catch (err) {
      onShowToast(`Ping failed: ${err.message}`, "error");
    } finally {
      setPingingAgentId(null);
    }
  };

  const openEditAgentModal = (ag) => {
    setEditingAgent(ag);
    setEditAgentForm({
      hostname: ag.hostname || "",
      ip_address: ag.ip_address || "127.0.0.1",
      port: ag.port || 3082,
      token: ag.pairing_token || ag.token || ""
    });
    setTestConnResult(null);
    setIsEditAgentModalOpen(true);
  };

  const handleUpdateAgent = async () => {
    if (!editingAgent) return;
    if (!editAgentForm.hostname) {
      onShowToast("Hostname is required", "error");
      return;
    }
    try {
      await api.updateAgent(editingAgent.id, {
        hostname: editAgentForm.hostname,
        ip_address: editAgentForm.ip_address,
        port: parseInt(editAgentForm.port, 10),
        token: editAgentForm.token,
        pairing_token: editAgentForm.token
      });
      onShowToast(`Edge agent "${editAgentForm.hostname}" updated!`, "success");
      setIsEditAgentModalOpen(false);
      setEditingAgent(null);
      onRefreshAgents();
    } catch (err) {
      onShowToast(`Failed to update agent: ${err.message}`, "error");
    }
  };

  const handleDeleteAgent = async (id, hostname) => {
    try {
      await api.deleteAgent(id);
      onShowToast(`Edge agent "${hostname || id}" deleted`, "info");
      onRefreshAgents();
    } catch (err) {
      onShowToast(`Failed to delete agent: ${err.message}`, "error");
    }
  };

  // Agent Pairing
  const handlePairAgent = async () => {
    if (!pairForm.token) {
      onShowToast("Pairing token is required", "error");
      return;
    }
    try {
      await api.pairAgent({
        agent_id: `agent-${Date.now().toString(36)}`,
        hostname: pairForm.hostname,
        ip_address: pairForm.ip_address,
        port: parseInt(pairForm.port, 10),
        token: pairForm.token,
        pairing_token: pairForm.token
      });
      onShowToast(`Edge agent "${pairForm.hostname}" paired successfully!`, "success");
      setIsPairModalOpen(false);
      onRefreshAgents();
    } catch (err) {
      onShowToast("Failed to pair agent: " + err.message, "error");
    }
  };

  // NLP Bot Command Tester
  const handleTestNlp = async () => {
    if (!nlpQuery) return;
    setIsTestingNlp(true);
    try {
      const res = await api.testNlpBot(nlpQuery);
      setNlpOutput(JSON.stringify(res, null, 2));
      onShowToast("ChatOps natural language command executed!", "success");
    } catch (err) {
      setNlpOutput("Execution error: " + err.message);
      onShowToast("NLP error: " + err.message, "error");
    } finally {
      setIsTestingNlp(false);
    }
  };

  const handleSaveTimezone = async () => {
    setIsSavingTz(true);
    try {
      await api.updateTimezoneSetting(selectedTimezone);
      if (onUpdateTimezone) {
        onUpdateTimezone(selectedTimezone);
      }
      onShowToast(`Broadcast reference timezone updated to ${selectedTimezone}`, "success");
    } catch (err) {
      onShowToast("Failed to update timezone: " + err.message, "error");
    } finally {
      setIsSavingTz(false);
    }
  };

  const handleResetTimezoneToIndia = async () => {
    setSelectedTimezone("Asia/Kolkata");
    setIsSavingTz(true);
    try {
      await api.updateTimezoneSetting("Asia/Kolkata");
      if (onUpdateTimezone) {
        onUpdateTimezone("Asia/Kolkata");
      }
      onShowToast("Broadcast timezone restored to India Standard Time (Asia/Kolkata, UTC+05:30)", "success");
    } catch (err) {
      onShowToast("Failed to reset timezone: " + err.message, "error");
    } finally {
      setIsSavingTz(false);
    }
  };

  const copyPresetFfmpeg = (preset) => {
    const cmd = `-vf "scale=${preset.width}:${preset.height}" -b:v ${preset.video_bitrate_kbps || 6500}k`;
    navigator.clipboard.writeText(cmd);
    onShowToast(`Copied FFmpeg arguments for ${preset.name} to clipboard!`, "info");
  };

  return (
    <div className="w-full flex-1 flex flex-col p-3 sm:p-4 md:p-6 space-y-4 max-w-full">
      {/* Subtab Navigation Bar */}
      <div className="flex items-center gap-1 bg-[#111827] border border-[#1F2937] p-1.5 rounded-lg shrink-0 overflow-x-auto">
        {[
          { id: "timezone", label: "Broadcast Timezone & Clock (IST)", icon: Globe },
          { id: "resolutions", label: "Resolutions & FFmpeg", icon: Sliders },
          { id: "users", label: `User Management (${users.length})`, icon: Users },
          { id: "agents", label: `Edge Agents (${agents.length})`, icon: Radio },
          { id: "bots", label: `ChatOps Bots (${bots.length})`, icon: Bot },
        ].map((tab) => {
          const Icon = tab.icon;
          const isActive = activeTab === tab.id;
          return (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`flex items-center gap-1.5 px-3 py-1.5 rounded text-xs font-semibold whitespace-nowrap transition-all ${
                isActive
                  ? 'bg-indigo-600 text-white shadow-sm'
                  : 'text-gray-300 hover:text-white hover:bg-gray-800'
              }`}
            >
              <Icon className="w-3.5 h-3.5" />
              <span>{tab.label}</span>
            </button>
          );
        })}
      </div>

      {/* Subtab 0: Broadcast Timezone & Master Clock Configuration */}
      {activeTab === "timezone" && (
        <div className="w-full bg-[#111827] border border-[#1F2937] rounded-xl p-4 sm:p-6 space-y-6 shadow-lg">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-gray-800 pb-4">
            <div>
              <div className="flex items-center gap-2">
                <Globe className="w-5 h-5 text-indigo-400" />
                <h3 className="text-base font-bold text-white tracking-wide">
                  Broadcast Timezone & Master Reference Clock
                </h3>
                <span className="px-2 py-0.5 rounded text-[11px] font-mono bg-indigo-950 border border-indigo-700 text-indigo-300">
                  Default: India (IST)
                </span>
              </div>
              <p className="text-xs text-gray-400 mt-1">
                Configures the reference broadcast timezone for all 24/7 linear schedules, automated gap filling, TMDb dates, and EPG tables
              </p>
            </div>

            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={handleResetTimezoneToIndia}
                disabled={isSavingTz}
                className="px-3.5 py-2 bg-gray-800 hover:bg-gray-700 text-gray-200 border border-gray-700 rounded-lg text-xs font-semibold flex items-center gap-1.5 transition-colors"
                title="Reset reference timezone to India Standard Time"
              >
                Reset to India (IST)
              </button>

              <button
                type="button"
                onClick={handleSaveTimezone}
                disabled={isSavingTz}
                className="px-4 py-2 bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg text-xs font-semibold flex items-center gap-1.5 shadow-md shadow-indigo-900/30 transition-all disabled:opacity-50"
              >
                <Save className="w-4 h-4" />
                {isSavingTz ? "Saving..." : "Save Timezone"}
              </button>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {/* Timezone Selection Card */}
            <div className="p-4 bg-[#141b2b] border border-gray-800 rounded-xl space-y-4">
              <div>
                <label className="block text-xs font-bold text-gray-300 uppercase tracking-wide mb-1.5">
                  Reference Broadcast Timezone
                </label>
                <select
                  value={selectedTimezone}
                  onChange={(e) => setSelectedTimezone(e.target.value)}
                  className="w-full bg-[#182030] border border-gray-700 rounded-lg px-3 py-2 text-sm text-white font-semibold focus:outline-none focus:border-indigo-500"
                >
                  {BROADCAST_TIMEZONES.map((tz) => (
                    <option key={tz.id} value={tz.id}>
                      {tz.label} ({tz.region})
                    </option>
                  ))}
                </select>
                <p className="text-[11px] text-gray-400 mt-1.5">
                  Current ID: <code className="text-indigo-400 font-mono">{selectedTimezone}</code>
                </p>
              </div>

              <div className="p-3 bg-[#182030] border border-gray-800 rounded-lg text-xs text-gray-300 space-y-1.5">
                <div className="font-bold text-white flex items-center gap-1.5">
                  <CheckCircle2 className="w-4 h-4 text-emerald-400" />
                  Clock Synchronization Status
                </div>
                <div className="text-[11px] text-gray-400">
                  Playout engine, automated gap bridge, and browser timeline are locked to this reference clock. All schedule items created will represent this wall-clock time.
                </div>
              </div>
            </div>

            {/* Live Master Clock Card */}
            <div className="p-5 bg-gradient-to-br from-[#131a29] to-[#0d121c] border border-gray-800 rounded-xl flex flex-col justify-between">
              <div>
                <div className="flex items-center justify-between text-xs text-gray-400 mb-2">
                  <span className="font-bold uppercase tracking-wider flex items-center gap-1.5">
                    <Clock className="w-4 h-4 text-sky-400" />
                    Live Master Broadcast Clock
                  </span>
                  <span className="px-2 py-0.5 rounded text-[10px] font-mono bg-sky-950/80 border border-sky-800 text-sky-300">
                    {selectedTimezone === 'Asia/Kolkata' ? 'IST +05:30' : selectedTimezone}
                  </span>
                </div>

                <div className="text-4xl sm:text-5xl font-black font-mono text-white tracking-widest my-2 text-sky-400 drop-shadow">
                  {liveClock.time}
                </div>

                <div className="text-sm font-semibold text-gray-300 font-mono mt-1">
                  Date: {liveClock.date}
                </div>
              </div>

              <div className="pt-3 border-t border-gray-800/80 text-[11px] text-gray-400 flex items-center justify-between">
                <span>Standard: SMPTE PAL 25.00 FPS</span>
                <span className="text-emerald-400 font-semibold flex items-center gap-1">
                  <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
                  Master Playout Locked
                </span>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Subtab 1: Resolutions & FFmpeg Profiles */}
      {activeTab === "resolutions" && (
        <div className="w-full bg-[#111827] border border-[#1F2937] rounded-lg p-3 sm:p-4 space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div>
              <h3 className="text-sm font-bold text-white">Broadcast Resolutions & FFmpeg Stream Profiles</h3>
              <p className="text-[11px] text-gray-400">
                Manage transcode rasters, interlaced field order, DAR, and custom FFmpeg video filter chains
              </p>
            </div>
            <button
              onClick={() => setIsResModalOpen(true)}
              className="px-3.5 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded text-xs font-semibold flex items-center gap-1 shadow self-start sm:self-auto transition-colors"
            >
              <Plus className="w-3.5 h-3.5" />
              <span>+ Add Custom Resolution</span>
            </button>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
            {resolutions.map((r) => (
              <div key={r.id} className="bg-[#1F2937] border border-gray-700/80 rounded-lg p-3 space-y-2">
                <div className="flex items-center justify-between">
                  <span className="font-bold text-white text-xs">{r.name}</span>
                  <span className="text-[9px] px-1.5 py-0.5 rounded bg-sky-500/20 text-sky-300 font-mono">
                    {r.is_preset ? "BUILT-IN PRESET" : "CUSTOM"}
                  </span>
                </div>
                <div className="text-[11px] text-gray-300 font-mono">
                  {r.width}x{r.height} @ {r.frame_rate || 25}fps • {r.scanning_mode || "progressive"}
                </div>
                <div className="flex justify-between items-center pt-2 border-t border-gray-700 text-[10px]">
                  <span className="text-gray-400 font-mono">Bitrate: {r.video_bitrate_kbps || 6500}k</span>
                  <button
                    onClick={() => copyPresetFfmpeg(r)}
                    className="text-indigo-400 hover:text-indigo-300 font-semibold flex items-center gap-1"
                  >
                    <Copy className="w-3 h-3" />
                    <span>Copy FFmpeg</span>
                  </button>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Subtab 2: User Management & RBAC */}
      {activeTab === "users" && (
        <div className="w-full bg-[#111827] border border-[#1F2937] rounded-lg p-3 sm:p-4 space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div>
              <h3 className="text-sm font-bold text-white">System Operators & Cryptographic RBAC</h3>
              <p className="text-[11px] text-gray-400">
                JWT bearer security with 3 privilege levels: Admin, Operator, and Content Scheduler
              </p>
            </div>
            <button
              onClick={() => setIsUserModalOpen(true)}
              className="px-3.5 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded text-xs font-semibold flex items-center gap-1 shadow self-start sm:self-auto transition-colors"
            >
              <Plus className="w-3.5 h-3.5" />
              <span>+ Add User</span>
            </button>
          </div>

          <div className="divide-y divide-[#1F2937] border border-[#1F2937] rounded-lg overflow-hidden">
            {users.map((u) => (
              <div key={u.id} className="p-3 bg-[#161F30] flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs">
                <div className="flex items-center gap-3">
                  <div className="w-8 h-8 rounded-full bg-gradient-to-tr from-indigo-600 to-purple-600 text-white flex items-center justify-center font-bold text-xs">
                    {(u.display_name || u.username).slice(0, 2).toUpperCase()}
                  </div>
                  <div>
                    <div className="font-bold text-white flex items-center gap-2">
                      <span>{u.display_name || u.username}</span>
                      <span className="font-mono text-[10px] text-gray-400">(@{u.username})</span>
                    </div>
                    <div className="text-[10px] text-gray-400">{u.email || "official@mcrflow.tv"}</div>
                  </div>
                </div>

                <div className="flex items-center justify-between sm:justify-end gap-3 w-full sm:w-auto">
                  <span className={`text-[10px] px-2 py-0.5 rounded font-mono font-bold uppercase ${
                    u.role === 'admin'
                      ? 'bg-purple-500/20 text-purple-300 border border-purple-500/30'
                      : u.role === 'operator'
                      ? 'bg-sky-500/20 text-sky-300 border border-sky-500/30'
                      : 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30'
                  }`}>
                    {u.role}
                  </span>
                  {users.length > 1 && u.username !== currentUser?.username && (
                    <button
                      onClick={() => handleDeleteUser(u.id, u.username)}
                      className="text-gray-400 hover:text-rose-400 p-1"
                      title="Delete User"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Subtab 3: Edge Playout Agents */}
      {activeTab === "agents" && (
        <div className="w-full bg-[#111827] border border-[#1F2937] rounded-lg p-3 sm:p-4 space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
            <div>
              <h3 className="text-sm font-bold text-white">Distributed Edge Playout Agents</h3>
              <p className="text-[11px] text-gray-400">
                Paired Docker transmitter nodes communicating via gRPC and cryptographic tokens
              </p>
            </div>
            <button
              onClick={() => setIsPairModalOpen(true)}
              className="px-3.5 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded text-xs font-semibold flex items-center gap-1 shadow self-start sm:self-auto transition-colors"
            >
              <Plus className="w-3.5 h-3.5" />
              <span>+ Pair Edge Node</span>
            </button>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {agents.length === 0 ? (
              <div className="col-span-2 text-center py-8 text-gray-500 text-xs bg-[#161F30] rounded-lg border border-dashed border-gray-700">
                No edge playout agents registered yet. Click "+ Pair Edge Node" to pair a daemon.
              </div>
            ) : (
              agents.map((ag) => {
                const isOnline = (ag.status === 'online' || ag.status === 'ONLINE') && 
                  (!ag.last_heartbeat || (Date.now() - new Date(ag.last_heartbeat).getTime() < 90000));
                const cpuVal = ag.cpu_usage_percent !== undefined ? ag.cpu_usage_percent : (ag.cpu_percent !== undefined ? ag.cpu_percent : null);
                const memVal = ag.memory_usage_percent !== undefined ? ag.memory_usage_percent : (ag.memory_percent !== undefined ? ag.memory_percent : null);
                const isPinging = pingingAgentId === ag.id;

                return (
                  <div key={ag.id} className="bg-[#1F2937] border border-gray-700/80 rounded-lg p-3.5 space-y-2.5 text-xs shadow">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <span className={`w-2.5 h-2.5 rounded-full ${isOnline ? 'bg-emerald-400 animate-pulse' : 'bg-gray-500'}`}></span>
                        <span className="font-bold text-white text-sm">{ag.hostname || ag.id}</span>
                      </div>
                      <span className={`text-[10px] px-2 py-0.5 rounded font-mono font-bold uppercase ${
                        isOnline 
                          ? 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/30' 
                          : 'bg-rose-500/20 text-rose-300 border border-rose-500/30'
                      }`}>
                        {isOnline ? "ONLINE" : "OFFLINE"}
                      </span>
                    </div>

                    <div className="grid grid-cols-2 gap-2 text-[11px] text-gray-300 font-mono bg-[#161F30] p-2 rounded border border-gray-800">
                      <div>
                        <span className="text-gray-500 text-[10px] block">ENDPOINT</span>
                        <span>{ag.ip_address || "127.0.0.1"}:{ag.port || 3082}</span>
                      </div>
                      <div>
                        <span className="text-gray-500 text-[10px] block">LAST HEARTBEAT</span>
                        <span className="text-[10px]">
                          {ag.last_heartbeat ? new Date(ag.last_heartbeat).toLocaleTimeString() : 'Never'}
                        </span>
                      </div>
                    </div>

                    <div className="flex justify-between text-[11px] font-mono text-gray-400">
                      <span>CPU: <strong className="text-gray-200">{cpuVal !== null ? `${cpuVal}%` : (isOnline ? 'Active' : 'Offline')}</strong></span>
                      <span>RAM: <strong className="text-gray-200">{memVal !== null ? `${memVal}%` : (isOnline ? 'Active' : 'Offline')}</strong></span>
                    </div>

                    {/* Actions: Ping, Edit, Delete */}
                    <div className="pt-2 border-t border-gray-700/80 flex items-center justify-between">
                      <button
                        onClick={() => handlePingAgent(ag.id)}
                        disabled={isPinging}
                        className="px-2.5 py-1 bg-sky-950/40 hover:bg-sky-900/60 border border-sky-800/60 text-sky-300 rounded text-[11px] font-medium flex items-center gap-1.5 transition-colors disabled:opacity-50"
                        title="Test agent reachability and token"
                      >
                        {isPinging ? <Loader2 className="w-3 h-3 animate-spin" /> : <Activity className="w-3 h-3 text-sky-400" />}
                        <span>{isPinging ? "Pinging..." : "Test Connection"}</span>
                      </button>

                      <div className="flex items-center gap-1.5">
                        <button
                          onClick={() => openEditAgentModal(ag)}
                          className="px-2.5 py-1 bg-gray-800 hover:bg-gray-700 text-indigo-300 rounded text-[11px] font-medium flex items-center gap-1 transition-colors"
                          title="Edit agent details"
                        >
                          <Pencil className="w-3 h-3" />
                          <span>Edit</span>
                        </button>
                        <button
                          onClick={() => handleDeleteAgent(ag.id, ag.hostname)}
                          className="px-2.5 py-1 bg-rose-950/40 hover:bg-rose-900/60 border border-rose-800/40 text-rose-300 rounded text-[11px] font-medium flex items-center gap-1 transition-colors"
                          title="Delete agent"
                        >
                          <Trash2 className="w-3 h-3" />
                          <span>Delete</span>
                        </button>
                      </div>
                    </div>
                  </div>
                );
              })
            )}
          </div>
        </div>
      )}



      {/* Subtab 5: ChatOps Bots & Automation */}
      {activeTab === "bots" && (
        <div className="w-full bg-[#111827] border border-[#1F2937] rounded-lg p-3 sm:p-4 space-y-4">
          <div>
            <h3 className="text-sm font-bold text-white">ChatOps Bots & Natural Language Automation</h3>
            <p className="text-[11px] text-gray-400">
              Trigger schedule cues and conflict resolution via natural language Telegram/Slack commands
            </p>
          </div>

          {/* NLP Command Tester */}
          <div className="bg-[#1F2937] border border-gray-700 rounded-lg p-4 space-y-3">
            <h4 className="text-xs font-bold text-white flex items-center gap-1.5">
              <Terminal className="w-4 h-4 text-emerald-400" />
              <span>Interactive Natural Language Playout Console</span>
            </h4>
            <div className="flex gap-2">
              <input
                type="text"
                value={nlpQuery}
                onChange={(e) => setNlpQuery(e.target.value)}
                placeholder="e.g. Schedule Pathaan at 20:00 on CH 01"
                className="flex-1 bg-[#0B0F17] border border-gray-700 rounded px-3 py-1.5 text-xs text-white font-mono"
              />
              <button
                onClick={handleTestNlp}
                disabled={isTestingNlp}
                className="px-4 py-1.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded text-xs font-semibold flex items-center gap-1 shadow"
              >
                <Send className="w-3.5 h-3.5" />
                <span>{isTestingNlp ? "Executing..." : "Execute"}</span>
              </button>
            </div>

            {nlpOutput && (
              <pre className="bg-[#0B0F17] border border-gray-800 p-3 rounded text-[11px] font-mono text-emerald-300 overflow-x-auto">
                {nlpOutput}
              </pre>
            )}
          </div>
        </div>
      )}



      {/* MODAL: Custom Resolution */}
      {isResModalOpen && (
        <div className="fixed inset-0 bg-black/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-[#111827] border border-[#2D3A54] rounded-xl w-full max-w-lg max-h-[88dvh] overflow-y-auto overscroll-contain shadow-2xl p-5 space-y-4 text-xs">
            <div className="flex items-center justify-between border-b border-gray-800 pb-2">
              <h3 className="text-sm font-bold text-white">Add Custom Resolution Profile</h3>
              <button onClick={() => setIsResModalOpen(false)} className="text-gray-400 hover:text-white">✕</button>
            </div>
            <div>
              <label className="block text-gray-400 mb-1">Preset Name</label>
              <input
                type="text"
                value={resForm.name}
                onChange={(e) => setResForm({ ...resForm, name: e.target.value })}
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1 text-white"
              />
            </div>
            <div className="grid grid-cols-3 gap-2">
              <div>
                <label className="block text-gray-400 mb-1">Width</label>
                <input
                  type="number"
                  value={resForm.width}
                  onChange={(e) => setResForm({ ...resForm, width: e.target.value })}
                  className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1 text-white font-mono"
                />
              </div>
              <div>
                <label className="block text-gray-400 mb-1">Height</label>
                <input
                  type="number"
                  value={resForm.height}
                  onChange={(e) => setResForm({ ...resForm, height: e.target.value })}
                  className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1 text-white font-mono"
                />
              </div>
              <div>
                <label className="block text-gray-400 mb-1">FPS</label>
                <input
                  type="number"
                  value={resForm.fps}
                  onChange={(e) => setResForm({ ...resForm, fps: e.target.value })}
                  className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1 text-white font-mono"
                />
              </div>
            </div>
            <div>
              <label className="block text-gray-400 mb-1">Custom FFmpeg Filter (-vf)</label>
              <input
                type="text"
                value={resForm.vf}
                onChange={(e) => setResForm({ ...resForm, vf: e.target.value })}
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1 text-white font-mono"
              />
            </div>
            <div className="flex justify-end gap-2 pt-2 border-t border-gray-800">
              <button onClick={() => setIsResModalOpen(false)} className="px-3 py-1 bg-gray-800 text-gray-300 rounded">Cancel</button>
              <button onClick={handleSaveResolution} className="px-3 py-1 bg-indigo-600 text-white rounded font-semibold">Save Preset</button>
            </div>
          </div>
        </div>
      )}

      {/* MODAL: Create User */}
      {isUserModalOpen && (
        <div className="fixed inset-0 bg-black/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-[#111827] border border-[#2D3A54] rounded-xl w-full max-w-md max-h-[88dvh] overflow-y-auto overscroll-contain shadow-2xl p-5 space-y-3.5 text-xs">
            <div className="flex items-center justify-between border-b border-gray-800 pb-2">
              <h3 className="text-sm font-bold text-white">Add System Operator / User</h3>
              <button onClick={() => setIsUserModalOpen(false)} className="text-gray-400 hover:text-white">✕</button>
            </div>
            <div>
              <label className="block text-gray-400 mb-1">Username (Login ID)</label>
              <input
                type="text"
                value={userForm.username}
                onChange={(e) => setUserForm({ ...userForm, username: e.target.value })}
                placeholder="e.g. operator_rahul"
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-white font-mono"
              />
            </div>
            <div>
              <label className="block text-gray-400 mb-1">Display Name</label>
              <input
                type="text"
                value={userForm.display_name}
                onChange={(e) => setUserForm({ ...userForm, display_name: e.target.value })}
                placeholder="e.g. Rahul Sharma"
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-white"
              />
            </div>
            <div>
              <label className="block text-gray-400 mb-1">Email</label>
              <input
                type="email"
                value={userForm.email}
                onChange={(e) => setUserForm({ ...userForm, email: e.target.value })}
                placeholder="e.g. rahul@mcrflow.tv"
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-white"
              />
            </div>
            <div>
              <label className="block text-gray-400 mb-1">Password</label>
              <input
                type="password"
                value={userForm.password}
                onChange={(e) => setUserForm({ ...userForm, password: e.target.value })}
                placeholder="Min 8 characters"
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-white font-mono"
              />
            </div>
            <div>
              <label className="block text-gray-400 mb-1">Role</label>
              <select
                value={userForm.role}
                onChange={(e) => setUserForm({ ...userForm, role: e.target.value })}
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-white"
              >
                <option value="content_scheduler">content_scheduler (Media & Timelines Only)</option>
                <option value="operator">operator (Channel Management & Slate)</option>
                <option value="admin">admin (Full System Control)</option>
              </select>
            </div>
            <div className="flex justify-end gap-2 pt-2 border-t border-gray-800">
              <button onClick={() => setIsUserModalOpen(false)} className="px-3 py-1 bg-gray-800 text-gray-300 rounded">Cancel</button>
              <button onClick={handleCreateUser} className="px-3 py-1 bg-indigo-600 text-white rounded font-semibold">Create User</button>
            </div>
          </div>
        </div>
      )}

      {/* MODAL: Pair Edge Agent */}
      {isPairModalOpen && (
        <div className="fixed inset-0 bg-black/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-[#111827] border border-[#2D3A54] rounded-xl w-full max-w-md max-h-[88dvh] overflow-y-auto overscroll-contain shadow-2xl p-5 space-y-3.5 text-xs">
            <div className="flex items-center justify-between border-b border-gray-800 pb-2">
              <h3 className="text-sm font-bold text-white flex items-center gap-1.5">
                <Radio className="w-4 h-4 text-indigo-400" />
                <span>Pair Edge Playout Agent Node</span>
              </h3>
              <button onClick={() => { setIsPairModalOpen(false); setTestConnResult(null); }} className="text-gray-400 hover:text-white">✕</button>
            </div>
            <div>
              <label className="block text-gray-400 mb-1">Hostname / Node ID</label>
              <input
                type="text"
                value={pairForm.hostname}
                onChange={(e) => setPairForm({ ...pairForm, hostname: e.target.value })}
                placeholder="e.g. edge-transmitter-mumbai"
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-white font-mono"
              />
            </div>
            <div className="grid grid-cols-3 gap-2">
              <div className="col-span-2">
                <label className="block text-gray-400 mb-1">IP Address / Host</label>
                <input
                  type="text"
                  value={pairForm.ip_address}
                  onChange={(e) => setPairForm({ ...pairForm, ip_address: e.target.value })}
                  placeholder="127.0.0.1"
                  className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-white font-mono"
                />
              </div>
              <div>
                <label className="block text-gray-400 mb-1">Daemon Port</label>
                <input
                  type="number"
                  value={pairForm.port}
                  onChange={(e) => setPairForm({ ...pairForm, port: e.target.value })}
                  placeholder="3082"
                  className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-white font-mono"
                />
              </div>
            </div>
            <div>
              <label className="block text-gray-400 mb-1">Cryptographic Token (MCRFLOW_TOKEN)</label>
              <input
                type="text"
                value={pairForm.token}
                onChange={(e) => setPairForm({ ...pairForm, token: e.target.value })}
                placeholder="agt_sec_..."
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-white font-mono"
              />
            </div>

            {/* Test Connection Inline Banner */}
            {testConnResult && (
              <div className={`p-2.5 rounded border text-[11px] font-mono flex items-start gap-2 ${
                testConnResult.reachable && testConnResult.authenticated
                  ? 'bg-emerald-950/40 border-emerald-700/60 text-emerald-300'
                  : testConnResult.reachable
                  ? 'bg-amber-950/40 border-amber-700/60 text-amber-300'
                  : 'bg-rose-950/40 border-rose-700/60 text-rose-300'
              }`}>
                {testConnResult.reachable && testConnResult.authenticated ? (
                  <CheckCircle2 className="w-4 h-4 text-emerald-400 shrink-0 mt-0.5" />
                ) : (
                  <AlertTriangle className="w-4 h-4 text-amber-400 shrink-0 mt-0.5" />
                )}
                <div className="space-y-0.5">
                  <div className="font-semibold">
                    {testConnResult.reachable && testConnResult.authenticated
                      ? `Reachable (${testConnResult.latency_ms}ms) & Authenticated`
                      : testConnResult.reachable
                      ? `Reachable (${testConnResult.latency_ms}ms) - Auth Failed`
                      : `Daemon Unreachable`}
                  </div>
                  {testConnResult.error && (
                    <div className="text-[10px] opacity-80">{testConnResult.error}</div>
                  )}
                </div>
              </div>
            )}

            <div className="flex items-center justify-between pt-2 border-t border-gray-800">
              <button
                type="button"
                onClick={() => handleTestConnection(pairForm.ip_address, pairForm.port, pairForm.token)}
                disabled={isTestingConn}
                className="px-3 py-1.5 bg-gray-800 hover:bg-gray-700 text-sky-300 rounded font-semibold flex items-center gap-1.5 transition-colors disabled:opacity-50"
              >
                {isTestingConn ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Activity className="w-3.5 h-3.5 text-sky-400" />}
                <span>{isTestingConn ? "Testing..." : "Test Connection"}</span>
              </button>

              <div className="flex gap-2">
                <button
                  onClick={() => { setIsPairModalOpen(false); setTestConnResult(null); }}
                  className="px-3 py-1.5 bg-gray-800 text-gray-300 hover:text-white rounded"
                >
                  Cancel
                </button>
                <button
                  onClick={handlePairAgent}
                  className="px-3.5 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded font-semibold shadow"
                >
                  Authenticate & Pair
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* MODAL: Edit Edge Agent */}
      {isEditAgentModalOpen && (
        <div className="fixed inset-0 bg-black/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
          <div className="bg-[#111827] border border-[#2D3A54] rounded-xl w-full max-w-md max-h-[88dvh] overflow-y-auto overscroll-contain shadow-2xl p-5 space-y-3.5 text-xs">
            <div className="flex items-center justify-between border-b border-gray-800 pb-2">
              <h3 className="text-sm font-bold text-white flex items-center gap-1.5">
                <Pencil className="w-4 h-4 text-indigo-400" />
                <span>Edit Edge Playout Agent Node</span>
              </h3>
              <button onClick={() => { setIsEditAgentModalOpen(false); setEditingAgent(null); setTestConnResult(null); }} className="text-gray-400 hover:text-white">✕</button>
            </div>
            <div>
              <label className="block text-gray-400 mb-1">Hostname / Node ID</label>
              <input
                type="text"
                value={editAgentForm.hostname}
                onChange={(e) => setEditAgentForm({ ...editAgentForm, hostname: e.target.value })}
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-white font-mono"
              />
            </div>
            <div className="grid grid-cols-3 gap-2">
              <div className="col-span-2">
                <label className="block text-gray-400 mb-1">IP Address / Host</label>
                <input
                  type="text"
                  value={editAgentForm.ip_address}
                  onChange={(e) => setEditAgentForm({ ...editAgentForm, ip_address: e.target.value })}
                  className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-white font-mono"
                />
              </div>
              <div>
                <label className="block text-gray-400 mb-1">Daemon Port</label>
                <input
                  type="number"
                  value={editAgentForm.port}
                  onChange={(e) => setEditAgentForm({ ...editAgentForm, port: e.target.value })}
                  className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-white font-mono"
                />
              </div>
            </div>
            <div>
              <label className="block text-gray-400 mb-1">Cryptographic Token (Update or Keep)</label>
              <input
                type="text"
                value={editAgentForm.token}
                onChange={(e) => setEditAgentForm({ ...editAgentForm, token: e.target.value })}
                placeholder="agt_sec_..."
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-white font-mono"
              />
            </div>

            {/* Test Connection Inline Banner */}
            {testConnResult && (
              <div className={`p-2.5 rounded border text-[11px] font-mono flex items-start gap-2 ${
                testConnResult.reachable && testConnResult.authenticated
                  ? 'bg-emerald-950/40 border-emerald-700/60 text-emerald-300'
                  : testConnResult.reachable
                  ? 'bg-amber-950/40 border-amber-700/60 text-amber-300'
                  : 'bg-rose-950/40 border-rose-700/60 text-rose-300'
              }`}>
                {testConnResult.reachable && testConnResult.authenticated ? (
                  <CheckCircle2 className="w-4 h-4 text-emerald-400 shrink-0 mt-0.5" />
                ) : (
                  <AlertTriangle className="w-4 h-4 text-amber-400 shrink-0 mt-0.5" />
                )}
                <div className="space-y-0.5">
                  <div className="font-semibold">
                    {testConnResult.reachable && testConnResult.authenticated
                      ? `Reachable (${testConnResult.latency_ms}ms) & Authenticated`
                      : testConnResult.reachable
                      ? `Reachable (${testConnResult.latency_ms}ms) - Auth Failed`
                      : `Daemon Unreachable`}
                  </div>
                  {testConnResult.error && (
                    <div className="text-[10px] opacity-80">{testConnResult.error}</div>
                  )}
                </div>
              </div>
            )}

            <div className="flex items-center justify-between pt-2 border-t border-gray-800">
              <button
                type="button"
                onClick={() => handleTestConnection(editAgentForm.ip_address, editAgentForm.port, editAgentForm.token)}
                disabled={isTestingConn}
                className="px-3 py-1.5 bg-gray-800 hover:bg-gray-700 text-sky-300 rounded font-semibold flex items-center gap-1.5 transition-colors disabled:opacity-50"
              >
                {isTestingConn ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <Activity className="w-3.5 h-3.5 text-sky-400" />}
                <span>{isTestingConn ? "Testing..." : "Test Connection"}</span>
              </button>

              <div className="flex gap-2">
                <button
                  onClick={() => { setIsEditAgentModalOpen(false); setEditingAgent(null); setTestConnResult(null); }}
                  className="px-3 py-1.5 bg-gray-800 text-gray-300 hover:text-white rounded"
                >
                  Cancel
                </button>
                <button
                  onClick={handleUpdateAgent}
                  className="px-3.5 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded font-semibold shadow"
                >
                  Save Changes
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

    </div>
  );
}
