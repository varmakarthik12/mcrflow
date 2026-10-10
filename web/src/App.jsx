import React, { useState, useEffect, useCallback } from 'react';
import { Header } from './components/Header';
import { DashboardScreen } from './components/DashboardScreen';
import { ChannelScreen } from './components/ChannelScreen';
import { ScheduleScreen } from './components/ScheduleScreen';
import { AdStudioScreen } from './components/AdStudioScreen';
import { SettingsScreen } from './components/SettingsScreen';
import { SetupModal } from './components/SetupModal';
import { LoginModal } from './components/LoginModal';
import { ToastContainer } from './components/Toast';
import { ConfirmModal } from './components/ConfirmModal';
import { api, getAuthToken, setAuthToken } from './api';
import { translations } from './i18n/translations';

export function App() {
  const [activeScreen, setActiveScreen] = useState(1);
  const [currentLanguage, setCurrentLanguage] = useState("en");

  // Auth & User State
  const [currentUser, setCurrentUser] = useState(null);
  const [isSetupRequired, setIsSetupRequired] = useState(false);
  const [authChecking, setAuthChecking] = useState(true);

  // Broadcast Entities State
  const [channels, setChannels] = useState([]);
  const [activeChannelId, setActiveChannelId] = useState("");
  const [scheduleItems, setScheduleItems] = useState([]);
  const [resolutions, setResolutions] = useState([]);
  const [adTemplates, setAdTemplates] = useState([]);
  const [agents, setAgents] = useState([]);
  const [bots, setBots] = useState([]);

  // Toast System
  const [toasts, setToasts] = useState([]);
  const showToast = useCallback((message, type = "info") => {
    const id = Date.now().toString(36) + Math.random().toString(36).substring(2, 6);
    setToasts((prev) => [...prev, { id, message, type }]);
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, 4000);
  }, []);

  const dismissToast = (id) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  };

  // Translation helper
  const t = (key) => {
    const dict = translations[currentLanguage] || translations.en;
    return dict[key] || translations.en[key] || key;
  };

  // Initial Auth Lifecycle
  useEffect(() => {
    const initAuth = async () => {
      setAuthChecking(true);
      try {
        const setupRes = await api.getSetupStatus();
        if (setupRes && (setupRes.needs_setup ?? setupRes.setup_required)) {
          setIsSetupRequired(true);
          setCurrentUser(null);
          setAuthChecking(false);
          return;
        }

        setIsSetupRequired(false);
        const token = getAuthToken();
        if (token) {
          try {
            const me = await api.getMe();
            if (me) {
              setCurrentUser(me);
              loadAllData();
              setAuthChecking(false);
              return;
            }
          } catch (e) {
            setAuthToken(null);
          }
        }
        setCurrentUser(null);
      } catch (err) {
        showToast("Initialization error: " + err.message, "error");
      } finally {
        setAuthChecking(false);
      }
    };

    initAuth();
  }, [showToast]);

  // Reactive Session Guard on HTTP 401 Unauthorized
  useEffect(() => {
    const handleUnauthorized = () => {
      setCurrentUser(null);
      setChannels([]);
      setScheduleItems([]);
      setResolutions([]);
      setAdTemplates([]);
      setAgents([]);
      setBots([]);
      setUsers([]);
      showToast("Session expired or unauthorized. Please sign in.", "warning");
    };

    window.addEventListener("mcrflow:unauthorized", handleUnauthorized);
    return () => window.removeEventListener("mcrflow:unauthorized", handleUnauthorized);
  }, [showToast]);

  // Load All Entities
  const loadAllData = async () => {
    await Promise.allSettled([
      loadChannels(),
      loadResolutions(),
      loadAdTemplates(),
      loadAgents(),
      loadBots(),
      loadUsers(),
    ]);
  };

  // Channels
  const loadChannels = async () => {
    try {
      const data = await api.getChannels();
      if (Array.isArray(data)) {
        setChannels(data);
        if (data.length > 0 && !activeChannelId) {
          setActiveChannelId(data[0].id);
          loadSchedule(data[0].id);
        }
      }
    } catch (e) {}
  };

  const handleSwitchChannel = (chId) => {
    setActiveChannelId(chId);
    loadSchedule(chId);
  };

  const handleSaveChannel = async (id, data) => {
    try {
      await api.updateChannel(id, data);
      showToast("Channel configuration saved!", "success");
      await loadChannels();
    } catch (err) {
      showToast("Failed to save channel: " + err.message, "error");
    }
  };

  const handleCreateChannel = async () => {
    const chNum = channels.length + 1;
    const newCh = {
      name: `MCRFlow Channel ${chNum}`,
      call_sign: `CH-${chNum}`,
      lcn: 100 + chNum,
      resolution_id: resolutions[0]?.id || "res-in-1080i50",
      video_codec: "libx264",
      audio_codec: "aac",
      destinations: [
        { type: "udp", protocol: "UDP_MULTICAST", enabled: true, url: `udp://239.255.10.${chNum}:5000?pkt_size=1316`, endpoint_url: `udp://239.255.10.${chNum}:5000?pkt_size=1316` },
        { type: "rtmp", protocol: "RTMP", enabled: false, url: "rtmp://live.twitch.tv/app", endpoint_url: "rtmp://live.twitch.tv/app", stream_key: "" },
        { type: "hls", protocol: "HLS", enabled: true, url: `/hls/ch-${chNum}/master.m3u8`, endpoint_url: `/hls/ch-${chNum}/master.m3u8` }
      ]
    };
    try {
      const created = await api.createChannel(newCh);
      showToast(`Channel "${created.name}" created!`, "success");
      await loadChannels();
      setActiveChannelId(created.id);
    } catch (err) {
      showToast("Create channel failed: " + err.message, "error");
    }
  };

  const handleDeleteChannel = async (id) => {
    try {
      await api.deleteChannel(id);
      showToast("Channel deleted", "info");
      const remaining = channels.filter((c) => c.id !== id);
      setChannels(remaining);
      if (remaining.length > 0) {
        setActiveChannelId(remaining[0].id);
      }
    } catch (err) {
      showToast("Delete failed: " + err.message, "error");
    }
  };

  // Schedule
  const loadSchedule = async (chId) => {
    const target = chId || activeChannelId;
    if (!target) return;
    try {
      const data = await api.getSchedule(target);
      if (Array.isArray(data)) {
        setScheduleItems(data);
      }
    } catch (e) {}
  };

  // Resolutions
  const loadResolutions = async () => {
    try {
      const data = await api.getResolutions();
      if (Array.isArray(data)) setResolutions(data);
    } catch (e) {}
  };

  // Ad Templates
  const loadAdTemplates = async () => {
    try {
      const data = await api.getAdTemplates();
      if (Array.isArray(data)) setAdTemplates(data);
    } catch (e) {}
  };

  // Agents
  const loadAgents = async () => {
    try {
      const data = await api.getAgents();
      if (Array.isArray(data)) setAgents(data);
    } catch (e) {}
  };

  // Bots
  const loadBots = async () => {
    try {
      const data = await api.getBots();
      if (Array.isArray(data)) setBots(data);
    } catch (e) {}
  };

  // Users
  const [users, setUsers] = useState([]);
  const loadUsers = async () => {
    try {
      const data = await api.getUsers();
      if (Array.isArray(data)) setUsers(data);
    } catch (e) {}
  };

  const handleLogout = () => {
    setAuthToken(null);
    setCurrentUser(null);
    setChannels([]);
    setScheduleItems([]);
    setResolutions([]);
    setAdTemplates([]);
    setAgents([]);
    setBots([]);
    setUsers([]);
    showToast("Signed out of Master Control", "info");
  };

  // 1. Initial auth check loading screen
  if (authChecking) {
    return (
      <div className="min-h-screen min-h-[100dvh] w-full flex items-center justify-center bg-[#0B0F17] text-gray-400 font-mono text-xs p-4">
        <div className="flex flex-col items-center gap-3">
          <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin"></div>
          <span>Initializing MCRFlow Security Guard...</span>
        </div>
      </div>
    );
  }

  // 2. If initial setup is required (0 users in DB)
  if (isSetupRequired) {
    return (
      <div className="min-h-screen min-h-[100dvh] w-full bg-[#0B0F17] flex items-center justify-center relative p-4">
        <SetupModal
          isOpen={true}
          canCancel={false}
          onClose={() => {}}
          onSetupSuccess={(user) => {
            setIsSetupRequired(false);
            setCurrentUser(user);
            loadAllData();
          }}
          onShowToast={showToast}
        />
        <ToastContainer toasts={toasts} onDismiss={dismissToast} />
      </div>
    );
  }

  // 3. If not authenticated, lock down all screens and show Login Modal
  if (!currentUser) {
    return (
      <div className="min-h-screen min-h-[100dvh] w-full bg-[#0B0F17] flex items-center justify-center relative p-4">
        <LoginModal
          isOpen={true}
          canClose={false}
          onClose={() => {}}
          onLoginSuccess={(user) => {
            setCurrentUser(user);
            loadAllData();
          }}
          onShowToast={showToast}
        />
        <ToastContainer toasts={toasts} onDismiss={dismissToast} />
      </div>
    );
  }

  // 4. Authenticated: Render Master Control Playout Workspace
  return (
    <div className="min-h-screen min-h-[100dvh] w-full flex flex-col bg-[#0B0F17]">
      {/* Top Header */}
      <Header
        activeScreen={activeScreen}
        onSelectScreen={setActiveScreen}
        currentUser={currentUser}
        onOpenLogin={() => {}}
        onLogout={handleLogout}
        currentLanguage={currentLanguage}
        onChangeLanguage={setCurrentLanguage}
        t={t}
      />

      {/* Main Screen Views */}
      <main className="flex-1 w-full flex flex-col relative pb-20 md:pb-6">
        {activeScreen === 1 && (
          <DashboardScreen
            channels={channels}
            agents={agents}
            scheduleItems={scheduleItems}
            onManageChannels={() => setActiveScreen(2)}
            onSelectChannel={(chId) => {
              setActiveChannelId(chId);
              setActiveScreen(2);
            }}
            t={t}
          />
        )}

        {activeScreen === 2 && (
          <ChannelScreen
            channels={channels}
            activeChannelId={activeChannelId}
            onSwitchChannel={handleSwitchChannel}
            onSaveChannel={handleSaveChannel}
            onCreateChannel={handleCreateChannel}
            onDeleteChannel={handleDeleteChannel}
            resolutions={resolutions}
            adTemplates={adTemplates}
            onBackToDashboard={() => setActiveScreen(1)}
            onNavigateToAdStudio={() => setActiveScreen(4)}
            onShowToast={showToast}
            t={t}
          />
        )}

        {activeScreen === 3 && (
          <ScheduleScreen
            channels={channels}
            activeChannelId={activeChannelId}
            onSwitchChannel={handleSwitchChannel}
            scheduleItems={scheduleItems}
            onRefreshSchedule={loadSchedule}
            onShowToast={showToast}
            adTemplates={adTemplates}
            t={t}
          />
        )}

        {activeScreen === 4 && (
          <AdStudioScreen
            adTemplates={adTemplates}
            channels={channels}
            onRefreshTemplates={loadAdTemplates}
            onShowToast={showToast}
            t={t}
          />
        )}

        {activeScreen === 5 && (
          <SettingsScreen
            resolutions={resolutions}
            onRefreshResolutions={loadResolutions}
            users={users}
            onRefreshUsers={loadUsers}
            agents={agents}
            onRefreshAgents={loadAgents}
            bots={bots}
            onRefreshBots={loadBots}
            onShowToast={showToast}
            currentUser={currentUser}
          />
        )}
      </main>

      {/* Toast Notifications */}
      <ToastContainer toasts={toasts} onDismiss={dismissToast} />
    </div>
  );
}
