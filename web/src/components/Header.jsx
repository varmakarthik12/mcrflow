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
    <>
      {/* Top Header */}
      <header className="h-14 bg-[#0F172A]/95 backdrop-blur-md border-b border-[#1E293B] px-3 sm:px-4 flex items-center justify-between shrink-0 z-30 select-none shadow-sm">
        <div className="flex items-center gap-2.5 sm:gap-3">
          {/* Brand Logo */}
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-lg bg-gradient-to-tr from-indigo-600 via-sky-500 to-emerald-400 flex items-center justify-center shadow-lg shadow-indigo-500/25 shrink-0">
              <Radio className="w-4.5 h-4.5 text-white" />
            </div>
            <div>
              <div className="text-sm font-black tracking-tight text-white flex items-center gap-1.5 leading-none">
                <span>MCRFLOW</span>
                <span className="hidden sm:inline-block text-[9px] px-1 py-0.5 rounded bg-indigo-500/20 text-indigo-300 font-mono font-bold border border-indigo-500/30">
                  BROADCAST
                </span>
                <span className="flex h-2 w-2 relative">
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-emerald-400 opacity-75"></span>
                  <span className="relative inline-flex rounded-full h-2 w-2 bg-emerald-500"></span>
                </span>
              </div>
              <div className="hidden sm:block text-[10px] text-gray-400 font-medium tracking-wide">
                Master Control Playout
              </div>
            </div>
          </div>

          <div className="hidden xl:block h-6 w-px bg-gray-800 mx-1"></div>

          {/* Live SMPTE PAL Timecode Display */}
          <div className="hidden xl:flex items-center gap-2 bg-[#0B0F17] border border-[#1E293B] px-2.5 py-1 rounded-md shadow-inner">
            <span className="text-[10px] text-gray-400 font-bold font-mono tracking-wider">PAL 25FPS</span>
            <span className="text-xs font-mono font-bold text-sky-400 tracking-widest">{smpteTimecode}</span>
          </div>
        </div>

        {/* Desktop Screen Navigation Buttons (Hidden on mobile) */}
        <nav className="hidden md:flex items-center gap-1.5 bg-[#0B0F17]/90 p-1 rounded-xl border border-[#1E293B] shadow-inner">
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = activeScreen === item.id;
            return (
              <button
                key={item.id}
                onClick={() => onSelectScreen(item.id)}
                title={item.label}
                className={`flex items-center gap-2 px-3.5 py-1.5 rounded-lg text-xs transition-all duration-150 ${
                  isActive
                    ? 'bg-gradient-to-r from-indigo-600 to-indigo-700 text-white font-bold shadow-md shadow-indigo-600/30 border border-indigo-400/40'
                    : 'text-gray-300 hover:text-white hover:bg-[#1E293B] border border-transparent font-medium'
                }`}
              >
                <Icon className={`w-3.5 h-3.5 ${isActive ? 'text-sky-300' : 'text-gray-400'}`} />
                <span>{item.label}</span>
              </button>
            );
          })}
        </nav>

        {/* Right Action Bar */}
        <div className="flex items-center gap-2 sm:gap-3 shrink-0">
          {/* Cluster Status Badge */}
          <div className="hidden lg:flex items-center gap-1.5 bg-emerald-950/40 border border-emerald-500/30 px-2.5 py-1 rounded-full text-[11px] text-emerald-300 font-medium">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
            <span>{t('common.healthy') || "100% Operational"}</span>
          </div>

          {/* Multilingual Localization Selector */}
          <div className="relative flex items-center">
            <Globe className="w-3.5 h-3.5 text-sky-400 absolute left-2 pointer-events-none" />
            <select
              value={currentLanguage}
              onChange={(e) => onChangeLanguage(e.target.value)}
              className="bg-[#1E293B] border border-[#334155] text-xs text-sky-300 font-medium rounded pl-6 pr-2 py-1 focus:outline-none focus:ring-1 focus:ring-sky-500 cursor-pointer max-w-[110px] sm:max-w-none truncate"
            >
              <option value="en">English</option>
              <option value="hi">🇮🇳 हिन्दी</option>
              <option value="ta">🇮🇳 தமிழ்</option>
              <option value="te">🇮🇳 తెలుగు</option>
              <option value="bn">🇮🇳 বাংলা</option>
              <option value="mr">🇮🇳 मराठी</option>
              <option value="gu">🇮🇳 ગુજરાતી</option>
              <option value="kn">🇮🇳 ಕನ್ನಡ</option>
              <option value="ml">🇮🇳 മലയാളം</option>
              <option value="pa">🇮🇳 ਪੰਜਾਬੀ</option>
              <option value="or">🇮🇳 ଓଡ଼ିଆ</option>
            </select>
          </div>

          {/* Operator User Profile Badge */}
          {currentUser ? (
            <div className="flex items-center gap-2 bg-[#1A2234] border border-[#2D3A54] px-2 sm:px-2.5 py-1 rounded-lg">
              <div className="w-6 h-6 rounded-full bg-gradient-to-tr from-indigo-500 to-purple-600 text-white flex items-center justify-center text-[10px] font-bold shrink-0">
                {getInitials(currentUser)}
              </div>
              <div className="text-left hidden sm:block">
                <div className="text-[11px] font-bold text-white leading-tight flex items-center gap-1.5">
                  <span className="truncate max-w-[90px]">{currentUser.display_name || currentUser.username}</span>
                  <span className={`text-[9px] px-1 py-0.2 rounded font-mono border uppercase font-semibold ${getRoleBadgeClass(currentUser.role)}`}>
                    {currentUser.role}
                  </span>
                </div>
              </div>
              <button
                onClick={onLogout}
                title="Sign Out"
                className="text-gray-400 hover:text-rose-400 p-0.5 rounded transition-colors ml-0.5"
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

      {/* Mobile Bottom Navigation Bar (Visible only on mobile md:hidden) */}
      <nav className="md:hidden fixed bottom-0 left-0 right-0 z-40 bg-[#0F172A]/95 backdrop-blur-xl border-t border-[#1E293B] flex items-center justify-around py-1.5 px-1 shadow-2xl safe-area-pb select-none">
        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive = activeScreen === item.id;
          return (
            <button
              key={item.id}
              onClick={() => onSelectScreen(item.id)}
              className={`flex flex-col items-center justify-center flex-1 py-1 px-1 rounded-lg transition-all ${
                isActive
                  ? 'text-indigo-400 font-bold bg-indigo-600/10'
                  : 'text-gray-400 hover:text-gray-200 font-medium'
              }`}
            >
              <div className="relative">
                <Icon className={`w-4.5 h-4.5 ${isActive ? 'text-indigo-400 scale-110' : 'text-gray-400'} transition-transform`} />
                {isActive && (
                  <span className="absolute -bottom-1 left-1/2 -translate-x-1/2 w-1.5 h-1.5 rounded-full bg-indigo-400"></span>
                )}
              </div>
              <span className={`text-[10px] mt-0.5 truncate max-w-[65px] ${isActive ? 'text-indigo-300' : 'text-gray-400'}`}>
                {item.label}
              </span>
            </button>
          );
        })}
      </nav>
    </>
  );
}
