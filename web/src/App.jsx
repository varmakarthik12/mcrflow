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
  const [isSetupOpen, setIsSetupOpen] = useState(false);
  const [isLoginOpen, setIsLoginOpen] = useState(false);

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
      try {
        const setupRes = await api.getSetupStatus();
        if (setupRes && (setupRes.needs_setup ?? setupRes.setup_required)) {
          setIsSetupOpen(true);
          return;
        }

        const token = getAuthToken();
        if (token) {
          try {
            const me = await api.getMe();
            if (me) {
              setCurrentUser(me);
              loadAllData();
              return;
            }
          } catch (e) {
            setAuthToken(null);
          }
        }

        // Try logging in with default demo operator or open modal
        try {
          const loginRes = await api.login({ username: "admin", password: "admin123" });
          if (loginRes && loginRes.token) {
            setAuthToken(loginRes.token);
            setCurrentUser(loginRes.user);
            loadAllData();
            return;
          }
        } catch (e) {
          try {
            const fallbackRes = await api.login({ username: "admin", password: "SuperAdminPass2026!" });
            if (fallbackRes && fallbackRes.token) {
              setAuthToken(fallbackRes.token);
              setCurrentUser(fallbackRes.user);
              loadAllData();
              return;
            }
          } catch (e2) {
            setIsLoginOpen(true);
          }
        }
      } catch (err) {
        showToast("Initialization error: " + err.message, "error");
      }
    };

    initAuth();
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
        { type: "srt", protocol: "SRT", enabled: true, url: `srt://127.0.0.1:${9000 + chNum}?mode=caller`, endpoint_url: `srt://127.0.0.1:${9000 + chNum}?mode=caller` },
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
    showToast("Signed out", "info");
    setIsLoginOpen(true);
  };

  return (
    <div className="h-full w-full flex flex-col bg-[#0B0F17] overflow-hidden">
      {/* Top Header */}
      <Header
        activeScreen={activeScreen}
        onSelectScreen={setActiveScreen}
        currentUser={currentUser}
        onOpenLogin={() => setIsLoginOpen(true)}
        onLogout={handleLogout}
        currentLanguage={currentLanguage}
        onChangeLanguage={setCurrentLanguage}
        t={t}
      />

      {/* Main Screen Views */}
      <main className="flex-1 overflow-hidden relative">
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

      {/* Modals */}
      <SetupModal
        isOpen={isSetupOpen}
        onClose={() => setIsSetupOpen(false)}
        onSetupSuccess={(user) => {
          setCurrentUser(user);
          loadAllData();
        }}
        onShowToast={showToast}
      />

      <LoginModal
        isOpen={isLoginOpen}
        onClose={() => setIsLoginOpen(false)}
        onLoginSuccess={(user) => {
          setCurrentUser(user);
          loadAllData();
        }}
        onShowToast={showToast}
      />

      {/* Toast Notifications */}
      <ToastContainer toasts={toasts} onDismiss={dismissToast} />
    </div>
  );
}
