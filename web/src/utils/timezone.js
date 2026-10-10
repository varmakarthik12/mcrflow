// MCRFlow Broadcast Timezone & SMPTE Clock Utilities
// Default Broadcast Timezone: Asia/Kolkata (Indian Standard Time, UTC+05:30)

export const DEFAULT_TIMEZONE = "Asia/Kolkata";

export const BROADCAST_TIMEZONES = [
  { id: "Asia/Kolkata", label: "India Standard Time (IST, UTC+05:30)", region: "India", defaultOffset: "+05:30" },
  { id: "UTC", label: "Coordinated Universal Time (UTC, +00:00)", region: "Global", defaultOffset: "+00:00" },
  { id: "Asia/Dubai", label: "Gulf Standard Time (GST, UTC+04:00)", region: "Middle East", defaultOffset: "+04:00" },
  { id: "Asia/Singapore", label: "Singapore Standard Time (SGT, UTC+08:00)", region: "Southeast Asia", defaultOffset: "+08:00" },
  { id: "Asia/Tokyo", label: "Japan Standard Time (JST, UTC+09:00)", region: "East Asia", defaultOffset: "+09:00" },
  { id: "Europe/London", label: "London (GMT / BST, UTC+00/+01)", region: "Europe", defaultOffset: "+00:00" },
  { id: "Europe/Paris", label: "Central European Time (CET / CEST, UTC+01/+02)", region: "Europe", defaultOffset: "+01:00" },
  { id: "America/New_York", label: "US Eastern Time (EST / EDT, UTC-05/-04)", region: "Americas", defaultOffset: "-05:00" },
  { id: "America/Los_Angeles", label: "US Pacific Time (PST / PDT, UTC-08/-07)", region: "Americas", defaultOffset: "-08:00" }
];

const STORAGE_KEY = "mcrflow_broadcast_timezone";

/**
 * Returns currently configured broadcast timezone (defaults to Asia/Kolkata)
 */
export function getTimezone() {
  if (typeof window !== "undefined" && window.localStorage) {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) return saved;
  }
  return DEFAULT_TIMEZONE;
}

/**
 * Persists and updates broadcast timezone
 */
export function setTimezone(tz) {
  if (typeof window !== "undefined" && window.localStorage) {
    localStorage.setItem(STORAGE_KEY, tz);
    window.dispatchEvent(new CustomEvent("mcrflow:timezone_changed", { detail: { timezone: tz } }));
  }
}

/**
 * Extracts "+05:30" or "-04:00" offset string for given timezone and date
 */
export function getTimezoneOffsetString(date = new Date(), tz = getTimezone()) {
  try {
    const formatter = new Intl.DateTimeFormat("en-US", {
      timeZone: tz,
      timeZoneName: "longOffset"
    });
    const parts = formatter.formatToParts(date);
    const tzPart = parts.find((p) => p.type === "timeZoneName");
    if (tzPart && tzPart.value) {
      const m = tzPart.value.match(/GMT([+-]\d{2}):?(\d{2})?/);
      if (m) {
        return `${m[1]}:${m[2] || "00"}`;
      }
    }
  } catch (e) {}

  // Fallback map
  const match = BROADCAST_TIMEZONES.find((b) => b.id === tz);
  return match?.defaultOffset || "+05:30";
}

/**
 * Converts wall-clock Date string ("YYYY-MM-DD") and Time string ("HH:mm:ss")
 * in the configured broadcast timezone into an exact UTC ISO-8601 string.
 */
export function createIsoInTimezone(dateStr, timeStr, tz = getTimezone()) {
  if (!dateStr) return new Date().toISOString();
  const cleanTime = !timeStr ? "00:00:00" : timeStr.length === 5 ? `${timeStr}:00` : timeStr;

  // Approximate offset using noon of the date
  const temp = new Date(`${dateStr}T12:00:00Z`);
  const offset = getTimezoneOffsetString(temp, tz);

  const isoWithOffset = `${dateStr}T${cleanTime}${offset}`;
  const parsed = new Date(isoWithOffset);

  if (isNaN(parsed.getTime())) {
    return new Date().toISOString();
  }
  return parsed.toISOString();
}

/**
 * Formats a Date or ISO string into "YYYY-MM-DD" in the specified timezone
 */
export function formatDateInTimezone(dateOrIso, tz = getTimezone()) {
  if (!dateOrIso) return "";
  const d = new Date(dateOrIso);
  if (isNaN(d.getTime())) return "";

  try {
    return new Intl.DateTimeFormat("en-CA", {
      timeZone: tz,
      year: "numeric",
      month: "2-digit",
      day: "2-digit"
    }).format(d);
  } catch (e) {
    return d.toISOString().slice(0, 10);
  }
}

/**
 * Formats a Date or ISO string into "HH:mm:ss" in the specified timezone
 */
export function formatTimeInTimezone(dateOrIso, tz = getTimezone(), includeSeconds = true) {
  if (!dateOrIso) return "00:00:00";
  const d = new Date(dateOrIso);
  if (isNaN(d.getTime())) return "00:00:00";

  try {
    return new Intl.DateTimeFormat("en-GB", {
      timeZone: tz,
      hour: "2-digit",
      minute: "2-digit",
      second: includeSeconds ? "2-digit" : undefined,
      hour12: false
    }).format(d);
  } catch (e) {
    return d.toTimeString().slice(0, includeSeconds ? 8 : 5);
  }
}

/**
 * Formats full human-readable date & time in the specified timezone
 */
export function formatDateTimeInTimezone(dateOrIso, tz = getTimezone()) {
  if (!dateOrIso) return "";
  const d = new Date(dateOrIso);
  if (isNaN(d.getTime())) return "";

  try {
    return new Intl.DateTimeFormat("en-US", {
      timeZone: tz,
      month: "short",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: false
    }).format(d);
  } catch (e) {
    return d.toLocaleString();
  }
}

/**
 * Returns current Date ("YYYY-MM-DD") and Time ("HH:mm:ss") in the specified timezone
 */
export function getNowInTimezone(tz = getTimezone()) {
  const now = new Date();
  return {
    date: formatDateInTimezone(now, tz),
    time: formatTimeInTimezone(now, tz, true),
    nowDate: now
  };
}
