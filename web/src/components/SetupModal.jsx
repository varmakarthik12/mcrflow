import React, { useState } from 'react';
import { ShieldCheck, X } from 'lucide-react';
import { api, setAuthToken } from '../api';

export function SetupModal({ isOpen, onClose, onSetupSuccess, onShowToast, canCancel = false }) {
  const [username, setUsername] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [loading, setLoading] = useState(false);

  if (!isOpen) return null;

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!username.trim() || !password) {
      onShowToast("Username and password are required", "error");
      return;
    }
    if (password !== confirmPassword) {
      onShowToast("Passwords do not match", "error");
      return;
    }

    setLoading(true);
    try {
      const res = await api.setupRootAdmin({
        username: username.trim(),
        full_name: displayName.trim() || username.trim(),
        display_name: displayName.trim() || username.trim(),
        email: email.trim(),
        password
      });

      if (res && res.token) {
        setAuthToken(res.token);
        onShowToast(`Root Administrator "${username}" initialized!`, "success");
        onSetupSuccess(res.user);
        if (onClose) onClose();
      }
    } catch (err) {
      onShowToast("Setup failed: " + err.message, "error");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/90 backdrop-blur-md z-50 flex items-center justify-center p-4">
      <div className="bg-[#111827] border border-indigo-500/50 rounded-xl w-full max-w-lg shadow-2xl overflow-hidden animate-in fade-in zoom-in duration-200">
        <div className="px-5 py-4 bg-gradient-to-r from-indigo-900/60 to-[#1A2234] border-b border-indigo-500/30 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-lg bg-indigo-600 flex items-center justify-center shadow-md">
              <ShieldCheck className="w-5 h-5 text-white" />
            </div>
            <div>
              <h3 className="text-sm font-bold text-white">First-Launch Master Playout Setup</h3>
              <p className="text-[11px] text-indigo-300">Create the primary Root Administrator account</p>
            </div>
          </div>
          {canCancel && (
            <button onClick={onClose} className="text-gray-400 hover:text-white">✕</button>
          )}
        </div>

        <form onSubmit={handleSubmit} className="p-5 space-y-4 text-xs">
          <div className="bg-indigo-950/40 border border-indigo-500/30 p-3 rounded-lg text-indigo-200 space-y-1">
            <div className="font-bold flex items-center gap-1.5">
              <span>🛡️ Broadcast Cluster Initialization</span>
            </div>
            <p className="text-[11px] text-gray-300">
              No users are registered in database. In compliance with broadcast security standards, initialize the permanent Root Administrator account.
            </p>
          </div>

          <div className="space-y-3">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-[11px] text-gray-400 mb-1">Root Admin Username</label>
                <input
                  type="text"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  placeholder="e.g. admin"
                  autoFocus
                  className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white font-mono focus:border-indigo-500 focus:outline-none"
                />
              </div>
              <div>
                <label className="block text-[11px] text-gray-400 mb-1">Full Display Name</label>
                <input
                  type="text"
                  value={displayName}
                  onChange={(e) => setDisplayName(e.target.value)}
                  placeholder="e.g. Chief Broadcast Engineer"
                  className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white focus:border-indigo-500 focus:outline-none"
                />
              </div>
            </div>

            <div>
              <label className="block text-[11px] text-gray-400 mb-1">Official Engineering Email</label>
              <input
                type="email"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="e.g. chief@mcrflow.tv"
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white focus:border-indigo-500 focus:outline-none"
              />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-[11px] text-gray-400 mb-1">Master Password</label>
                <input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Minimum 8 characters"
                  className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white font-mono focus:border-indigo-500 focus:outline-none"
                />
              </div>
              <div>
                <label className="block text-[11px] text-gray-400 mb-1">Confirm Password</label>
                <input
                  type="password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  placeholder="Re-enter password"
                  className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white font-mono focus:border-indigo-500 focus:outline-none"
                />
              </div>
            </div>
          </div>

          <div className="px-5 py-3 bg-[#1A2234] -mx-5 -mb-5 border-t border-[#2D3A54] flex justify-between items-center">
            <span className="text-[10px] text-gray-400 font-mono">Role: admin (Permanent Root)</span>
            <div className="flex gap-2">
              {canCancel && (
                <button
                  type="button"
                  onClick={onClose}
                  className="px-3.5 py-1.5 bg-gray-800 hover:bg-gray-700 text-gray-300 rounded text-xs font-semibold"
                >
                  Cancel
                </button>
              )}
              <button
                type="submit"
                disabled={loading}
                className="px-4 py-1.5 bg-gradient-to-r from-indigo-600 to-sky-500 hover:from-indigo-500 hover:to-sky-400 text-white rounded text-xs font-semibold shadow-lg disabled:opacity-50"
              >
                {loading ? "Initializing..." : "Initialize & Launch MCRFlow"}
              </button>
            </div>
          </div>
        </form>
      </div>
    </div>
  );
}
