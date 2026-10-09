import React, { useState } from 'react';
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
  X
} from 'lucide-react';
import { api } from '../api';

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
  currentUser
}) {
  const [activeTab, setActiveTab] = useState("resolutions");

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
    ip_address: "192.168.1.100",
    port: 3082,
    token: ""
  });

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

  const copyPresetFfmpeg = (preset) => {
    const cmd = `-vf "scale=${preset.width}:${preset.height}" -b:v ${preset.video_bitrate_kbps || 6500}k`;
    navigator.clipboard.writeText(cmd);
    onShowToast(`Copied FFmpeg arguments for ${preset.name} to clipboard!`, "info");
  };

  return (
    <div className="h-full flex flex-col p-4 space-y-4 overflow-y-auto">
      {/* Subtab Navigation Bar */}
      <div className="flex items-center gap-1 bg-[#111827] border border-[#1F2937] p-1.5 rounded-lg shrink-0 overflow-x-auto">
        {[
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

      {/* Subtab 1: Resolutions & FFmpeg Profiles */}
      {activeTab === "resolutions" && (
        <div className="flex-1 bg-[#111827] border border-[#1F2937] rounded-lg p-4 space-y-4 overflow-y-auto">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-sm font-bold text-white">Broadcast Resolutions & FFmpeg Stream Profiles</h3>
              <p className="text-[11px] text-gray-400">
                Manage transcode rasters, interlaced field order, DAR, and custom FFmpeg video filter chains
              </p>
            </div>
            <button
              onClick={() => setIsResModalOpen(true)}
              className="px-3.5 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded text-xs font-semibold flex items-center gap-1 shadow"
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
        <div className="flex-1 bg-[#111827] border border-[#1F2937] rounded-lg p-4 space-y-4 overflow-y-auto">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-sm font-bold text-white">System Operators & Cryptographic RBAC</h3>
              <p className="text-[11px] text-gray-400">
                JWT bearer security with 3 privilege levels: Admin, Operator, and Content Scheduler
              </p>
            </div>
            <button
              onClick={() => setIsUserModalOpen(true)}
              className="px-3.5 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded text-xs font-semibold flex items-center gap-1 shadow"
            >
              <Plus className="w-3.5 h-3.5" />
              <span>+ Add User</span>
            </button>
          </div>

          <div className="divide-y divide-[#1F2937] border border-[#1F2937] rounded-lg overflow-hidden">
            {users.map((u) => (
              <div key={u.id} className="p-3 bg-[#161F30] flex items-center justify-between text-xs">
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

                <div className="flex items-center gap-3">
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
        <div className="flex-1 bg-[#111827] border border-[#1F2937] rounded-lg p-4 space-y-4 overflow-y-auto">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-sm font-bold text-white">Distributed Edge Playout Agents</h3>
              <p className="text-[11px] text-gray-400">
                Paired Docker transmitter nodes communicating via gRPC and cryptographic tokens
              </p>
            </div>
            <button
              onClick={() => setIsPairModalOpen(true)}
              className="px-3.5 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded text-xs font-semibold flex items-center gap-1 shadow"
            >
              <Plus className="w-3.5 h-3.5" />
              <span>+ Pair Edge Node</span>
            </button>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            {agents.map((ag) => (
              <div key={ag.id} className="bg-[#1F2937] border border-gray-700/80 rounded-lg p-3 space-y-2 text-xs">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-emerald-400 animate-pulse"></span>
                    <span className="font-bold text-white">{ag.hostname || ag.id}</span>
                  </div>
                  <span className="text-[10px] px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-300 font-mono">
                    {ag.status || "ONLINE"}
                  </span>
                </div>
                <div className="text-[11px] text-gray-300 font-mono">
                  IP: {ag.ip_address || "127.0.0.1"}:{ag.port || 3082}
                </div>
                <div className="pt-2 border-t border-gray-700 flex justify-between text-[10px] text-gray-400 font-mono">
                  <span>CPU: {ag.cpu_usage_percent || ag.cpu_percent || 12.0}%</span>
                  <span>RAM: {ag.memory_usage_percent || ag.memory_percent || 18.0}%</span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}



      {/* Subtab 5: ChatOps Bots & Automation */}
      {activeTab === "bots" && (
        <div className="flex-1 bg-[#111827] border border-[#1F2937] rounded-lg p-4 space-y-4 overflow-y-auto">
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
          <div className="bg-[#111827] border border-[#2D3A54] rounded-xl w-full max-w-lg shadow-2xl p-5 space-y-4 text-xs">
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
          <div className="bg-[#111827] border border-[#2D3A54] rounded-xl w-full max-w-md shadow-2xl p-5 space-y-3.5 text-xs">
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
          <div className="bg-[#111827] border border-[#2D3A54] rounded-xl w-full max-w-md shadow-2xl p-5 space-y-3.5 text-xs">
            <div className="flex items-center justify-between border-b border-gray-800 pb-2">
              <h3 className="text-sm font-bold text-white">Pair Edge Playout Agent Node</h3>
              <button onClick={() => setIsPairModalOpen(false)} className="text-gray-400 hover:text-white">✕</button>
            </div>
            <div>
              <label className="block text-gray-400 mb-1">Hostname</label>
              <input
                type="text"
                value={pairForm.hostname}
                onChange={(e) => setPairForm({ ...pairForm, hostname: e.target.value })}
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1 text-white font-mono"
              />
            </div>
            <div>
              <label className="block text-gray-400 mb-1">IP Address</label>
              <input
                type="text"
                value={pairForm.ip_address}
                onChange={(e) => setPairForm({ ...pairForm, ip_address: e.target.value })}
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1 text-white font-mono"
              />
            </div>
            <div>
              <label className="block text-gray-400 mb-1">Persistent Pairing Token</label>
              <textarea
                rows="2"
                value={pairForm.token}
                onChange={(e) => setPairForm({ ...pairForm, token: e.target.value })}
                placeholder="agt_sec_..."
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2 py-1 text-white font-mono"
              ></textarea>
            </div>
            <div className="flex justify-end gap-2 pt-2 border-t border-gray-800">
              <button onClick={() => setIsPairModalOpen(false)} className="px-3 py-1 bg-gray-800 text-gray-300 rounded">Cancel</button>
              <button onClick={handlePairAgent} className="px-3 py-1 bg-indigo-600 text-white rounded font-semibold">Authenticate & Pair</button>
            </div>
          </div>
        </div>
      )}

    </div>
  );
}
