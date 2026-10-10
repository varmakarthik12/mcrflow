import React, { useState } from 'react';
import { Lock, Shield } from 'lucide-react';
import { api, setAuthToken } from '../api';

export function LoginModal({ isOpen, onClose, onLoginSuccess, onShowToast, canClose = false }) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
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
      const res = await api.login({ username: username.trim(), password });
      if (res && res.token) {
        setAuthToken(res.token);
        const name = res.user.display_name || res.user.full_name || username;
        onShowToast(`Signed in as ${name}!`, "success");
        onLoginSuccess(res.user);
        if (onClose) onClose();
      }
    } catch (err) {
      onShowToast("Authentication failed: " + err.message, "error");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/90 backdrop-blur-md z-50 flex items-center justify-center p-4">
      <div className="bg-[#111827] border border-[#2D3A54] rounded-xl w-full max-w-md max-h-[88dvh] overflow-y-auto overscroll-contain shadow-2xl animate-in fade-in zoom-in duration-150">
        <div className="px-5 py-4 bg-[#1A2234] border-b border-[#2D3A54] flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-lg bg-indigo-600 flex items-center justify-center shadow">
              <Lock className="w-4 h-4 text-white" />
            </div>
            <div>
              <h3 className="text-sm font-bold text-white">MCRFlow Master Control Login</h3>
              <p className="text-[10px] text-gray-400">Cryptographic JWT authentication required to access playout</p>
            </div>
          </div>
          {canClose && (
            <button onClick={onClose} className="text-gray-400 hover:text-white transition-colors">✕</button>
          )}
        </div>

        <div className="p-6 space-y-4 text-xs">
          <form onSubmit={handleSubmit} className="space-y-4">
            <div>
              <label className="block text-[11px] font-semibold text-gray-300 mb-1">Username / Operator ID</label>
              <input
                type="text"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="Enter operator username"
                autoFocus
                className="w-full bg-[#1F2937] border border-gray-700 rounded-lg px-3 py-2 text-xs text-white font-mono focus:border-indigo-500 focus:outline-none transition-colors"
              />
            </div>
            <div>
              <label className="block text-[11px] font-semibold text-gray-300 mb-1">Password</label>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Enter account password"
                className="w-full bg-[#1F2937] border border-gray-700 rounded-lg px-3 py-2 text-xs text-white font-mono focus:border-indigo-500 focus:outline-none transition-colors"
              />
            </div>

            <button
              type="submit"
              disabled={loading}
              className="w-full py-2.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded-lg text-xs font-semibold shadow-lg shadow-indigo-600/30 transition-all disabled:opacity-50"
            >
              {loading ? "Authenticating..." : "Sign In to Master Control"}
            </button>
          </form>

          <div className="text-center text-[10px] text-gray-500 pt-2 border-t border-gray-800">
            Role and capabilities are enforced based on assigned operator credentials.
          </div>
        </div>
      </div>
    </div>
  );
}
