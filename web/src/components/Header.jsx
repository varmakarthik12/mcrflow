import React, { useEffect, useState } from 'react';
import {
  LayoutDashboard,
  Tv,
  Calendar,
  Sparkles,
  Settings,
  Globe,
  Radio,
  LogOut,
  UserCheck
} from 'lucide-react';

export function Header({
  activeScreen,
  onSelectScreen,
  currentUser,
  onOpenLogin,
  onLogout,
  currentLanguage,
  onChangeLanguage,
  t
}) {
  const [smpteTimecode, setSmpteTimecode] = useState("00:00:00:00");

  // Live SMPTE 25fps PAL clock
  useEffect(() => {
    let frame = 0;
    const interval = setInterval(() => {
      const now = new Date();
      frame = (frame + 1) % 25;
      const hh = String(now.getHours()).padStart(2, '0');
      const mm = String(now.getMinutes()).padStart(2, '0');
      const ss = String(now.getSeconds()).padStart(2, '0');
      const ff = String(frame).padStart(2, '0');
      setSmpteTimecode(`${hh}:${mm}:${ss}:${ff}`);
    }, 40);

    return () => clearInterval(interval);
  }, []);

  const navItems = [
    { id: 1, label: t('nav.dashboard') || "Dashboard", icon: LayoutDashboard },
    { id: 2, label: t('nav.channels') || "Channels", icon: Tv },
    { id: 3, label: t('nav.scheduling') || "Scheduling & EPG", icon: Calendar },
    { id: 4, label: t('nav.ad_templates') || "Ad Studio", icon: Sparkles },
    { id: 5, label: t('nav.settings') || "Settings", icon: Settings },
  ];

  const getRoleBadgeClass = (role) => {
    if (role === 'admin') return 'bg-purple-500/20 text-purple-300 border-purple-500/30';
    if (role === 'operator') return 'bg-sky-500/20 text-sky-300 border-sky-500/30';
    return 'bg-emerald-500/20 text-emerald-300 border-emerald-500/30';
  };

  const getInitials = (user) => {
    if (!user) return "OP";
    const name = user.display_name || user.username || "Op";
    return name.slice(0, 2).toUpperCase();
  };

  return (
    <header className="h-14 bg-[#111827] border-b border-[#1F2937] px-4 flex items-center justify-between shrink-0 z-30 select-none">
      <div className="flex items-center gap-3">
        {/* Brand Logo */}
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-lg bg-gradient-to-tr from-indigo-600 via-sky-500 to-emerald-400 flex items-center justify-center shadow-lg shadow-sky-500/20">
            <Radio className="w-5 h-5 text-white" />
          </div>
          <div>
            <div className="text-sm font-black tracking-tight text-white flex items-center gap-1.5 leading-none">
              MCRFLOW <span className="text-[10px] px-1 py-0.2 rounded bg-indigo-500/20 text-indigo-400 font-mono font-semibold border border-indigo-500/30">BROADCAST</span>
            </div>
            <div className="text-[10px] text-gray-400 font-medium tracking-wide">Master Control Playout</div>
          </div>
        </div>

        <div className="h-6 w-px bg-gray-800 mx-1"></div>

        {/* Live SMPTE PAL Timecode Display */}
        <div className="flex items-center gap-2 bg-[#0B0F17] border border-[#1F2937] px-2.5 py-1 rounded-md">
          <span className="w-2 h-2 rounded-full bg-emerald-400 animate-ping"></span>
          <span className="text-[10px] text-gray-400 font-semibold font-mono tracking-wider">PAL 25FPS</span>
          <span className="text-xs font-mono font-bold text-white tracking-widest">{smpteTimecode}</span>
        </div>
      </div>

      {/* Main Screen Navigation Buttons */}
      <nav className="flex items-center gap-1 bg-[#0B0F17] p-1 rounded-lg border border-[#1F2937]">
        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive = activeScreen === item.id;
          return (
            <button
              key={item.id}
              onClick={() => onSelectScreen(item.id)}
              className={`flex items-center gap-1.5 px-3 py-1 rounded text-xs font-semibold transition-all ${
                isActive
                  ? 'bg-indigo-600 text-white shadow-sm'
                  : 'text-gray-300 hover:text-white hover:bg-gray-800/60'
              }`}
            >
              <Icon className="w-3.5 h-3.5" />
              <span>{item.label}</span>
            </button>
          );
        })}
      </nav>

      {/* Right Action Bar */}
      <div className="flex items-center gap-3">
        {/* Cluster Status Badge */}
        <div className="hidden lg:flex items-center gap-1.5 bg-emerald-950/40 border border-emerald-500/30 px-2.5 py-1 rounded-full text-[11px] text-emerald-300 font-medium">
          <span className="w-1.5 h-1.5 rounded-full bg-emerald-400"></span>
          <span>{t('common.healthy') || "Cluster: 100% Operational"}</span>
        </div>

        {/* Multilingual Localization Selector */}
        <div className="relative flex items-center">
          <Globe className="w-3.5 h-3.5 text-sky-400 absolute left-2 pointer-events-none" />
          <select
            value={currentLanguage}
            onChange={(e) => onChangeLanguage(e.target.value)}
            className="bg-[#1E293B] border border-[#334155] text-xs text-sky-300 font-medium rounded pl-6 pr-2 py-1 focus:outline-none focus:ring-1 focus:ring-sky-500 cursor-pointer"
          >
            <option value="en">English (Broadcast Standard)</option>
            <option value="hi">🇮🇳 हिन्दी (Hindi)</option>
            <option value="ta">🇮🇳 தமிழ் (Tamil)</option>
            <option value="te">🇮🇳 తెలుగు (Telugu)</option>
            <option value="bn">🇮🇳 বাংলা (Bengali)</option>
            <option value="mr">🇮🇳 मराठी (Marathi)</option>
            <option value="gu">🇮🇳 ગુજરાતી (Gujarati)</option>
            <option value="kn">🇮🇳 ಕನ್ನಡ (Kannada)</option>
            <option value="ml">🇮🇳 മലയാളം (Malayalam)</option>
            <option value="pa">🇮🇳 ਪੰਜਾਬੀ (Punjabi)</option>
            <option value="or">🇮🇳 ଓଡ଼ିଆ (Odia)</option>
          </select>
        </div>

        {/* Operator User Profile Badge */}
        {currentUser ? (
          <div className="flex items-center gap-2 bg-[#1A2234] border border-[#2D3A54] px-2.5 py-1 rounded-lg">
            <div className="w-6 h-6 rounded-full bg-gradient-to-tr from-indigo-500 to-purple-600 text-white flex items-center justify-center text-[10px] font-bold">
              {getInitials(currentUser)}
            </div>
            <div className="text-left">
              <div className="text-[11px] font-bold text-white leading-tight flex items-center gap-1.5">
                {currentUser.display_name || currentUser.username}
                <span className={`text-[9px] px-1 py-0.2 rounded font-mono border uppercase font-semibold ${getRoleBadgeClass(currentUser.role)}`}>
                  {currentUser.role}
                </span>
              </div>
            </div>
            <button
              onClick={onLogout}
              title="Sign Out"
              className="text-gray-400 hover:text-rose-400 ml-1 p-0.5 rounded transition-colors"
            >
              <LogOut className="w-3.5 h-3.5" />
            </button>
          </div>
        ) : (
          <button
            onClick={onOpenLogin}
            className="flex items-center gap-1.5 px-3 py-1 bg-indigo-600 hover:bg-indigo-500 text-white rounded text-xs font-semibold shadow"
          >
            <UserCheck className="w-3.5 h-3.5" />
            <span>Sign In</span>
          </button>
        )}
      </div>
    </header>
  );
}
