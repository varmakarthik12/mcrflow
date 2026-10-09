import React, { useState } from 'react';
import { Lock, X, Shield, User, Clock } from 'lucide-react';
import { api, setAuthToken } from '../api';

export function LoginModal({ isOpen, onClose, onLoginSuccess, onShowToast }) {
  const [username, setUsername] = useState("admin");
  const [password, setPassword] = useState("admin123");
  const [loading, setLoading] = useState(false);

  if (!isOpen) return null;

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!username || !password) {
      onShowToast("Enter username and password", "error");
      return;
    }

    setLoading(true);
    try {
      const res = await api.login({ username, password });
      if (res && res.token) {
        setAuthToken(res.token);
        const name = res.user.display_name || res.user.full_name || username;
        onShowToast(`Signed in as ${name}!`, "success");
        onLoginSuccess(res.user);
        onClose();
      }
    } catch (err) {
      onShowToast("Authentication failed: " + err.message, "error");
    } finally {
      setLoading(false);
    }
  };

  const handleSimulateRole = (roleUser, rolePass) => {
    setUsername(roleUser);
    setPassword(rolePass);
  };

  return (
    <div className="fixed inset-0 bg-black/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
      <div className="bg-[#111827] border border-[#2D3A54] rounded-xl w-full max-w-md shadow-2xl overflow-hidden animate-in fade-in zoom-in duration-150">
        <div className="px-5 py-3.5 bg-[#1A2234] border-b border-[#2D3A54] flex items-center justify-between">
          <div className="flex items-center gap-2">
            <div className="w-7 h-7 rounded-lg bg-indigo-600 flex items-center justify-center">
              <Lock className="w-4 h-4 text-white" />
            </div>
            <h3 className="text-sm font-bold text-white">MCRFlow Operator Authentication</h3>
          </div>
          <button onClick={onClose} className="text-gray-400 hover:text-white">✕</button>
        </div>

        <div className="p-5 space-y-4 text-xs">
          <form onSubmit={handleSubmit} className="space-y-3">
            <div>
              <label className="block text-[11px] text-gray-400 mb-1">Username</label>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white font-mono"
              />
            </div>
            <div>
              <label className="block text-[11px] text-gray-400 mb-1">Password</label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white font-mono"
              />
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full py-2 bg-indigo-600 hover:bg-indigo-500 text-white rounded text-xs font-semibold shadow transition-all"
            >
              {loading ? "Authenticating..." : "Sign In with Credentials"}
            </button>
          </form>

          {/* Quick Role Simulation Switcher */}
          <div className="pt-3 border-t border-gray-800 space-y-2">
            <div className="text-[10px] text-gray-400 font-semibold uppercase tracking-wider">
              Quick Role Profile Switcher:
            </div>
            <div className="space-y-1.5">
              <button
                onClick={() => handleSimulateRole("admin", "admin123")}
                className="w-full p-2 bg-[#1A2234] hover:bg-[#25324B] border border-purple-500/30 rounded flex items-center justify-between text-left transition-all"
              >
                <div className="flex items-center gap-2">
                  <span className="w-5 h-5 rounded-full bg-purple-600 flex items-center justify-center text-[9px] font-bold text-white">AD</span>
                  <div>
                    <div className="font-bold text-white text-[11px]">Chief Broadcast Engineer</div>
                    <div className="text-[9px] text-gray-400">Full System Control, User Admin, Edge Nodes</div>
                  </div>
                </div>
                <span className="text-[9px] px-1.5 py-0.5 rounded bg-purple-500/20 text-purple-300 font-mono uppercase font-semibold">ADMIN</span>
              </button>

              <button
                onClick={() => handleSimulateRole("operator", "operator123")}
                className="w-full p-2 bg-[#1A2234] hover:bg-[#25324B] border border-sky-500/30 rounded flex items-center justify-between text-left transition-all"
              >
                <div className="flex items-center gap-2">
                  <span className="w-5 h-5 rounded-full bg-sky-600 flex items-center justify-center text-[9px] font-bold text-white">OP</span>
                  <div>
                    <div className="font-bold text-white text-[11px]">Rajesh Kumar (MCR Desk)</div>
                    <div className="text-[9px] text-gray-400">Channels, Live Playout, Ad Overlays, Emergency Slate</div>
                  </div>
                </div>
                <span className="text-[9px] px-1.5 py-0.5 rounded bg-sky-500/20 text-sky-300 font-mono uppercase font-semibold">OPERATOR</span>
              </button>

              <button
                onClick={() => handleSimulateRole("scheduler", "scheduler123")}
                className="w-full p-2 bg-[#1A2234] hover:bg-[#25324B] border border-emerald-500/30 rounded flex items-center justify-between text-left transition-all"
              >
                <div className="flex items-center gap-2">
                  <span className="w-5 h-5 rounded-full bg-emerald-600 flex items-center justify-center text-[9px] font-bold text-white">SC</span>
                  <div>
                    <div className="font-bold text-white text-[11px]">Rahul Sharma (Scheduler)</div>
                    <div className="text-[9px] text-amber-300">Restricted: Media Scheduling & Storage Only</div>
                  </div>
                </div>
                <span className="text-[9px] px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-300 font-mono uppercase font-semibold">SCHEDULER</span>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
