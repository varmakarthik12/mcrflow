---
name: i18n-localization
description: Standard guidelines and automated rules for internationalization (i18n) across the broadcast playout platform, supporting English (default) and all major Indian languages (Hindi, Tamil, Telugu, Bengali, Marathi, Gujarati, Kannada, Malayalam, Punjabi, Odia).
---

# Internationalization (i18n) & Localization Architecture for Playout Platform

This skill governs the internationalization standards for the React frontend, Go backend services, EPG generators, and Telegram bot notification systems within the Stream Playout platform.

## 1. Supported Languages & Locales

The system is configured with **English (en)** as the primary default language and provides native built-in translations for the 10 constitutional Indian languages:

| Language | Code | Script / Font Family | Text Direction | Locale |
| :--- | :--- | :--- | :--- | :--- |
| **English (Default)** | `en` | Inter / Roboto / System Sans | LTR | `en-IN`, `en-US` |
| **Hindi (हिन्दी)** | `hi` | Noto Sans Devanagari | LTR | `hi-IN` |
| **Tamil (தமிழ்)** | `ta` | Noto Sans Tamil | LTR | `ta-IN` |
| **Telugu (తెలుగు)** | `te` | Noto Sans Telugu | LTR | `te-IN` |
| **Bengali (বাংলা)** | `bn` | Noto Sans Bengali | LTR | `bn-IN` |
| **Marathi (मराठी)** | `mr` | Noto Sans Devanagari | LTR | `mr-IN` |
| **Gujarati (ગુજરાતી)** | `gu` | Noto Sans Gujarati | LTR | `gu-IN` |
| **Kannada (ಕನ್ನಡ)** | `kn` | Noto Sans Kannada | LTR | `kn-IN` |
| **Malayalam (മലയാളം)** | `ml` | Noto Sans Malayalam | LTR | `ml-IN` |
| **Punjabi (ਪੰਜਾਬੀ)** | `pa` | Noto Sans Gurmukhi | LTR | `pa-IN` |
| **Odia (ଓଡ଼ିଆ)** | `or` | Noto Sans Oriya | LTR | `or-IN` |

---

## 2. Core Implementation Directives for Developers & AI Agents

Whenever implementing new UI screens, backend error messages, EPG descriptions, or Telegram bot responses:

### 2.1 React Frontend Rules
1. **Never Hardcode User-Facing Text**:
   - All string literals must use translation hooks:
     ```tsx
     import { useTranslation } from 'react-i18next';
     const { t } = useTranslation();
     // Good:
     <h1>{t('dashboard.title', 'Broadcast Control Center')}</h1>
     // Bad:
     <h1>Broadcast Control Center</h1>
     ```
2. **Key Hierarchy Convention**:
   - Keys must follow domain dot-notation:
     - `dashboard.*` (Dashboard screen telemetry, status, alerts)
     - `channel.*` (Channel creation, redundancy agents, logos)
     - `schedule.*` (Playout scheduler, TMDb search, conflict solver, audio/subtitles)
     - `ad_template.*` (Ad template designer, transitions, banner layouts, ad rolls)
     - `settings.*` (Storage mounts, agent crypto token pairing, bot config, Tailscale)
     - `common.*` (Actions: Save, Cancel, Delete, Edit, Status, Live, Standby, Offline)
3. **Indic Typography & Web Fonts**:
   - Load variable fonts or Google Fonts with unicode subsets for Indic scripts:
     ```css
     @import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&family=Noto+Sans+Devanagari:wght@400;600&family=Noto+Sans+Tamil:wght@400;600&family=Noto+Sans+Telugu:wght@400;600&family=Noto+Sans+Bengali:wght@400;600&display=swap');
     ```
   - Ensure line-height is at least `1.5` for Indic scripts to prevent vowel sign (matra) clipping.
4. **Time & Date Localization**:
   - Always display broadcast times in Indian Standard Time (`IST` / `Asia/Kolkata` / UTC+05:30) with 24-hour timecode format `HH:MM:SS:FF` (SMPTE timecode) or localized human-friendly strings:
     ```ts
     new Intl.DateTimeFormat(currentLocale, {
       hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false, timeZone: 'Asia/Kolkata'
     }).format(date);
     ```

### 2.2 Go Backend & gRPC Rules
1. **gRPC Status & Localized Error Payloads**:
   - Internal logs remain in English with structured fields.
   - User-facing error details returned to the frontend or Telegram bot should use `google.rpc.LocalizedMessage` in gRPC status details, keyed by error code.
2. **TMDb & Metadata Localization**:
   - When fetching metadata from TMDb (`/search/movie`, `/movie/{id}`), pass the user's preferred language code (e.g. `language=hi-IN`, `language=ta-IN`, `language=te-IN`) to store localized titles and overviews for multilingual EPG generation.
3. **EPG DVB-SI EIT Multi-Language Tables**:
   - The DVB Event Information Table (EIT) supports multi-lingual event descriptors (`short_event_descriptor`, `extended_event_descriptor` with ISO 639-2 language codes e.g. `hin`, `tam`, `tel`, `ben`, `eng`). The EPG generator must populate both the primary language and secondary English descriptor.

### 2.3 Telegram Bot Natural Language Parser (NLP)
1. **Multilingual Intent & Entity Recognition**:
   - Support English and transliterated Hinglish/Indic commands:
     - English: `"Schedule Avengers at 9:00 PM"`
     - Hindi / Hinglish: `"कल सुबह 9 बजे एवेंजर्स शेड्यूल करो"` / `"Kal subah 9 baje Avengers schedule karo"`
     - Tamil: `"காலை 9 மணிக்கு அவெஞ்சர்ஸ் ஒளிபரப்பு"`
   - Bot prompts, conflict resolution dialogues, and inline buttons must reply in the user's selected language.

---

## 3. Translation File Directory Structure

```text
src/
  locales/
    en/
      common.json
      dashboard.json
      channel.json
      schedule.json
      ad_template.json
      settings.json
    hi/
      ...
    ta/
      ...
    te/
      ...
    bn/
      ...
    mr/
      ...
    gu/
      ...
    kn/
      ...
    ml/
      ...
    pa/
      ...
    or/
      ...
```

---

## 4. Automated Translation Pipeline Hook

When adding new translation strings in `en/*.json`:
1. Run automated key sync utility (`npm run i18n:sync` or Go sync tool).
2. Missing keys in target Indian locales are populated with machine translation or fall back gracefully to English without breaking UI layout.
3. Keep UI test coverage verifying that text strings fit within standard buttons and card headers in scripts with longer compound words (e.g., Malayalam, Telugu, German).
