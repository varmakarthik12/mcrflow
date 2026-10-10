/**
 * MCRFlow REST API Client
 */

const TOKEN_KEY = "mcrflow_jwt";

export function getAuthToken() {
  return localStorage.getItem(TOKEN_KEY) || "";
}

export function setAuthToken(token) {
  if (token) {
    localStorage.setItem(TOKEN_KEY, token);
  } else {
    localStorage.removeItem(TOKEN_KEY);
  }
}

export async function request(path, options = {}) {
  const headers = {
    "Content-Type": "application/json",
    ...(options.headers || {})
  };

  const token = getAuthToken();
  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(path, {
    ...options,
    headers
  });

  let data = null;
  const contentType = res.headers.get("content-type") || "";
  if (contentType.includes("application/json")) {
    data = await res.json().catch(() => null);
  } else {
    data = await res.text().catch(() => null);
  }

  if (!res.ok) {
    if (res.status === 401 && !path.includes("/auth/login")) {
      setAuthToken(null);
      if (typeof window !== "undefined") {
        window.dispatchEvent(new CustomEvent("mcrflow:unauthorized"));
      }
    }
    const errorMsg = data?.error || res.statusText || "Request failed";
    const err = new Error(errorMsg);
    err.status = res.status;
    err.data = data;
    throw err;
  }

  return data;
}

// Auth API
export const api = {
  // Auth
  getSetupStatus: () => request("/api/v1/auth/setup-status"),
  setupRootAdmin: (payload) => request("/api/v1/auth/setup", { method: "POST", body: JSON.stringify(payload) }),
  login: (payload) => request("/api/v1/auth/login", { method: "POST", body: JSON.stringify(payload) }),
  getMe: () => request("/api/v1/auth/me"),

  // Channels
  getChannels: () => request("/api/v1/channels"),
  getChannel: (id) => request(`/api/v1/channels/${id}`),
  createChannel: (data) => request("/api/v1/channels", { method: "POST", body: JSON.stringify(data) }),
  updateChannel: (id, data) => request(`/api/v1/channels/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  deleteChannel: (id) => request(`/api/v1/channels/${id}`, { method: "DELETE" }),
  startPlayout: (id, data = {}) => request(`/api/v1/channels/${id}/playout/start`, { method: "POST", body: JSON.stringify(data) }),
  stopPlayout: (id) => request(`/api/v1/channels/${id}/playout/stop`, { method: "POST" }),
  getPlayoutStatus: (id) => request(`/api/v1/channels/${id}/playout/status`),
  getFFmpegCommand: (id) => request(`/api/v1/channels/${id}/ffmpeg-cmd`),
  getFFmpegLogs: (id, lines = 250) => request(`/api/v1/channels/${id}/ffmpeg-logs?lines=${lines}`),
  startPreview: (id) => request(`/api/v1/channels/${id}/preview/start`, { method: "POST" }),
  stopPreview: (id) => request(`/api/v1/channels/${id}/preview/stop`, { method: "POST" }),

  // Schedule
  getSchedule: (channelId) => request(`/api/v1/schedule?channel_id=${encodeURIComponent(channelId)}`),
  getScheduleItem: (id) => request(`/api/v1/schedule/${id}`),
  createScheduleItem: (data) => request("/api/v1/schedule", { method: "POST", body: JSON.stringify(data) }),
  updateScheduleItem: (id, data) => request(`/api/v1/schedule/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  deleteScheduleItem: (id) => request(`/api/v1/schedule/${id}`, { method: "DELETE" }),
  checkScheduleConflicts: (data) => request("/api/v1/schedule/check-conflicts", { method: "POST", body: JSON.stringify(data) }),
  getScheduleGaps: (channelId, start = "", end = "", fromCurrent = false) => {
    let url = `/api/v1/schedule/gaps?channel_id=${encodeURIComponent(channelId)}`;
    if (start) url += `&start=${encodeURIComponent(start)}`;
    if (end) url += `&end=${encodeURIComponent(end)}`;
    if (fromCurrent) url += `&from_current=true`;
    return request(url);
  },
  autoFillGaps: (data) => request("/api/v1/schedule/auto-fill-gaps", { method: "POST", body: JSON.stringify(data) }),
  toggleChannelSlate: (channelId, enabled) => request(`/api/v1/channels/${channelId}/slate`, { method: "POST", body: JSON.stringify({ enabled }) }),

  // System Settings & Broadcast Timezone
  getTimezoneSetting: () => request("/api/v1/settings/timezone"),
  updateTimezoneSetting: (timezone) => request("/api/v1/settings/timezone", { method: "PUT", body: JSON.stringify({ timezone }) }),

  // Resolutions
  getResolutions: () => request("/api/v1/resolutions"),
  getResolution: (id) => request(`/api/v1/resolutions/${id}`),
  createResolution: (data) => request("/api/v1/resolutions", { method: "POST", body: JSON.stringify(data) }),
  updateResolution: (id, data) => request(`/api/v1/resolutions/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  deleteResolution: (id) => request(`/api/v1/resolutions/${id}`, { method: "DELETE" }),

  // Ad Templates
  getAdTemplates: () => request("/api/v1/ad-templates"),
  getAdTemplate: (id) => request(`/api/v1/ad-templates/${id}`),
  createAdTemplate: (data) => request("/api/v1/ad-templates", { method: "POST", body: JSON.stringify(data) }),
  updateAdTemplate: (id, data) => request(`/api/v1/ad-templates/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  deleteAdTemplate: (id) => request(`/api/v1/ad-templates/${id}`, { method: "DELETE" }),

  // Storage & Media
  browseStorage: (subPath = "") => request(`/api/v1/storage/browse?path=${encodeURIComponent(subPath)}`),
  probeFile: (path) => request(`/api/v1/storage/probe?path=${encodeURIComponent(path)}`),
  uploadChannelLogo: async (channelId, file) => {
    const formData = new FormData();
    formData.append("logo", file);
    const token = getAuthToken();
    const headers = {};
    if (token) headers["Authorization"] = `Bearer ${token}`;
    const res = await fetch(`/api/v1/channels/${channelId}/logo`, {
      method: "POST",
      headers,
      body: formData
    });
    if (!res.ok) {
      const errData = await res.json().catch(() => null);
      throw new Error(errData?.error || "Logo upload failed");
    }
    return res.json();
  },
  uploadLogo: async (file) => {
    const formData = new FormData();
    formData.append("logo", file);
    const token = getAuthToken();
    const headers = {};
    if (token) headers["Authorization"] = `Bearer ${token}`;
    const res = await fetch("/api/v1/media/upload-logo", {
      method: "POST",
      headers,
      body: formData
    });
    if (!res.ok) {
      const errData = await res.json().catch(() => null);
      throw new Error(errData?.error || "Logo upload failed");
    }
    return res.json();
  },

  // TMDb
  searchTmdb: (query) => request(`/api/v1/tmdb/search?query=${encodeURIComponent(query)}`),
  getTmdbDetails: (id) => request(`/api/v1/tmdb/details/${id}`),

  // Edge Agents
  getAgents: () => request("/api/v1/agents"),
  getAgent: (id) => request(`/api/v1/agents/${id}`),
  createAgent: (data) => request("/api/v1/agents", { method: "POST", body: JSON.stringify(data) }),
  updateAgent: (id, data) => request(`/api/v1/agents/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  deleteAgent: (id) => request(`/api/v1/agents/${id}`, { method: "DELETE" }),
  pairAgent: (data) => request("/api/v1/agents/pair", { method: "POST", body: JSON.stringify(data) }),
  testAgentConnection: (data) => request("/api/v1/agents/test-connection", { method: "POST", body: JSON.stringify(data) }),
  pingAgent: (id) => request(`/api/v1/agents/${id}/ping`, { method: "POST" }),

  // Users
  getUsers: () => request("/api/v1/users"),
  getUser: (id) => request(`/api/v1/users/${id}`),
  createUser: (data) => request("/api/v1/users", { method: "POST", body: JSON.stringify(data) }),
  updateUser: (id, data) => request(`/api/v1/users/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  deleteUser: (id) => request(`/api/v1/users/${id}`, { method: "DELETE" }),

  // Bots & ChatOps
  getBots: () => request("/api/v1/bots"),
  getBot: (id) => request(`/api/v1/bots/${id}`),
  createBot: (data) => request("/api/v1/bots", { method: "POST", body: JSON.stringify(data) }),
  updateBot: (id, data) => request(`/api/v1/bots/${id}`, { method: "PUT", body: JSON.stringify(data) }),
  deleteBot: (id) => request(`/api/v1/bots/${id}`, { method: "DELETE" }),
  sendBotCommand: (id, data) => request(`/api/v1/bots/${id}/command`, { method: "POST", body: JSON.stringify(data) }),
  testNlpBot: (command, channelId = "ch-01") => request("/api/v1/bot/nlp-command", {
    method: "POST",
    body: JSON.stringify({ command, channel_id: channelId })
  })
};
