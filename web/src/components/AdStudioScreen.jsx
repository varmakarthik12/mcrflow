import React, { useState } from 'react';
import {
  Sparkles,
  Play,
  Layers,
  Clock,
  Eye,
  Sliders,
  Radio,
  Check
} from 'lucide-react';

export function AdStudioScreen({
  adTemplates = [],
  onShowToast,
  t
}) {
  const [selectedElement, setSelectedElement] = useState("bug");
  const [chosenAnim, setChosenAnim] = useState("anim-slide-up");
  const [showSafeGuides, setShowSafeGuides] = useState(true);
  const [animKey, setAnimKey] = useState(0);

  const handlePlayAnimation = () => {
    setAnimKey((prev) => prev + 1);
    onShowToast("Playing on-air graphics transition preview", "info");
  };

  return (
    <div className="h-full flex flex-col p-4 space-y-4 overflow-y-auto">
      {/* Top Header */}
      <div className="flex items-center justify-between shrink-0 bg-[#111827] border border-[#1F2937] p-3 rounded-lg">
        <div className="flex items-center gap-3">
          <Sparkles className="w-5 h-5 text-indigo-400" />
          <div>
            <h2 className="text-sm font-bold text-white">
              {t('ad.title') || "WYSIWYG Ad & CG Graphics Studio"}
            </h2>
            <p className="text-[11px] text-gray-400">
              {t('ad.subtitle') || "Animated on-screen bugs, lower-thirds, tickers & SCTE-35 commercial rolls"}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          {/* Safe Guides Toggle */}
          <label className="flex items-center gap-2 cursor-pointer text-xs text-gray-300">
            <input
              type="checkbox"
              checked={showSafeGuides}
              onChange={(e) => setShowSafeGuides(e.target.checked)}
              className="rounded bg-gray-800 border-gray-700 text-indigo-600 focus:ring-0"
            />
            <span>EBU Safe Area Guides (90%/80%)</span>
          </label>

          <button
            onClick={handlePlayAnimation}
            className="px-3.5 py-1.5 bg-indigo-600 hover:bg-indigo-500 text-white rounded text-xs font-semibold flex items-center gap-1.5 shadow"
          >
            <Play className="w-3.5 h-3.5" />
            <span>Play Preview Animation</span>
          </button>
        </div>
      </div>

      {/* Main Grid: Interactive Canvas Left, Properties & Breaks Right */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-4 flex-1 min-h-0">
        {/* Left: WYSIWYG Canvas (8 cols) */}
        <div className="lg:col-span-8 bg-[#111827] border border-[#1F2937] rounded-lg p-4 flex flex-col space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-xs font-bold text-gray-300 uppercase tracking-wider flex items-center gap-1.5">
              <Eye className="w-3.5 h-3.5 text-sky-400" />
              <span>Interactive Broadcast Canvas (1920x1080 16:9 Reference)</span>
            </span>
            <span className="text-[10px] font-mono text-gray-400">
              Selected: <span className="text-sky-300 font-bold uppercase">{selectedElement}</span>
            </span>
          </div>

          {/* Interactive Screen Preview Container */}
          <div
            key={animKey}
            className="relative aspect-video bg-black rounded-lg overflow-hidden border border-gray-800 flex items-center justify-center select-none shadow-2xl"
          >
            {/* Background Feed simulation */}
            <img
              src="https://images.unsplash.com/photo-1518173946687-a4c8a383392e?w=800&auto=format&fit=crop&q=60"
              alt="Video Preview"
              className="w-full h-full object-cover opacity-75"
            />
            <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-black/30 pointer-events-none"></div>

            {/* Safe Area Guides */}
            {showSafeGuides && (
              <div className="absolute inset-0 pointer-events-none">
                {/* 90% Action Safe */}
                <div className="absolute inset-[5%] border border-cyan-500/25 border-dashed">
                  <span className="absolute top-1 left-1 text-[8px] text-cyan-400/50 font-mono">ACTION SAFE (90%)</span>
                </div>
                {/* 80% Title Safe */}
                <div className="absolute inset-[10%] border border-amber-500/25 border-dashed">
                  <span className="absolute top-1 left-1 text-[8px] text-amber-400/50 font-mono">TITLE SAFE (80%)</span>
                </div>
              </div>
            )}

            {/* Element 1: Sponsor Bug (Top Right) */}
            <div
              onClick={() => setSelectedElement("bug")}
              className={`absolute top-6 right-6 z-20 cursor-pointer p-2 rounded transition-all ${chosenAnim} ${
                selectedElement === "bug" ? 'outline outline-2 outline-sky-400 shadow-lg shadow-sky-500/40 bg-black/40' : ''
              }`}
            >
              <div className="px-2.5 py-1 bg-gradient-to-r from-red-600 to-rose-700 text-white rounded font-black text-xs tracking-wider shadow">
                ★ MCRFLOW PRIME
              </div>
            </div>

            {/* Element 2: Lower-Third Teaser (Bottom Left) */}
            <div
              onClick={() => setSelectedElement("lowerthird")}
              className={`absolute bottom-10 left-8 z-20 cursor-pointer p-2 rounded transition-all anim-slide-up ${
                selectedElement === "lowerthird" ? 'outline outline-2 outline-sky-400 shadow-lg shadow-sky-500/40 bg-black/40' : ''
              }`}
            >
              <div className="bg-[#0B0F17]/90 border-l-4 border-indigo-500 px-3.5 py-2 rounded-r shadow-2xl backdrop-blur-md">
                <div className="text-xs font-bold text-white">UP NEXT: PATHAAN (2023)</div>
                <div className="text-[10px] text-sky-300 font-mono">TONIGHT @ 20:00 IST • 5.1 DOLBY ATMOS</div>
              </div>
            </div>

            {/* Element 3: News Ticker Crawl (Bottom Bar) */}
            <div
              onClick={() => setSelectedElement("ticker")}
              className={`absolute bottom-0 inset-x-0 bg-red-950/90 border-t border-red-600/50 h-6 flex items-center overflow-hidden z-20 cursor-pointer ${
                selectedElement === "ticker" ? 'outline outline-2 outline-sky-400' : ''
              }`}
            >
              <div className="bg-red-700 px-2 py-0.5 text-[9px] font-black text-white shrink-0 tracking-wider">
                BREAKING
              </div>
              <div className="anim-ticker text-[10px] font-semibold text-white pl-3 font-mono">
                MCRFLOW CLOUD PLAYOUT LAUNCHES 24/7 BROADCAST AUTOMATION WITH ZERO-FRAME HITLESS SWITCHING • SCTE-35 DAI INGESTION VERIFIED
              </div>
            </div>
          </div>
        </div>

        {/* Right: Graphic Controls & SCTE-35 Commercial Rolls (4 cols) */}
        <div className="lg:col-span-4 space-y-4">
          <div className="bg-[#111827] border border-[#1F2937] rounded-lg p-3 space-y-3 text-xs">
            <h3 className="text-xs font-bold text-white uppercase tracking-wider flex items-center gap-1.5">
              <Sliders className="w-3.5 h-3.5 text-indigo-400" />
              <span>Keyframe Transitions</span>
            </h3>

            <div>
              <label className="block text-[11px] text-gray-400 mb-1">Entrance Animation Style</label>
              <select
                value={chosenAnim}
                onChange={(e) => setChosenAnim(e.target.value)}
                className="w-full bg-[#1F2937] border border-gray-700 rounded px-2.5 py-1.5 text-xs text-white font-medium"
              >
                <option value="anim-slide-up">Slide Up From Bottom</option>
                <option value="anim-slide-left">Slide Left From Right Edge</option>
                <option value="anim-bounce">Bounce In Scale (Spring)</option>
                <option value="anim-zoom">Smooth Zoom In Fade</option>
              </select>
            </div>

            <div>
              <label className="block text-[11px] text-gray-400 mb-1">Target Element Layer</label>
              <div className="grid grid-cols-3 gap-2">
                {["bug", "lowerthird", "ticker"].map((el) => (
                  <button
                    key={el}
                    onClick={() => setSelectedElement(el)}
                    className={`py-1.5 rounded text-[11px] font-semibold uppercase transition-all ${
                      selectedElement === el
                        ? 'bg-indigo-600 text-white shadow'
                        : 'bg-gray-800 text-gray-300 hover:bg-gray-700'
                    }`}
                  >
                    {el}
                  </button>
                ))}
              </div>
            </div>
          </div>

          {/* SCTE-35 Commercial Ad Breaks */}
          <div className="bg-[#111827] border border-[#1F2937] rounded-lg p-3 space-y-2 text-xs">
            <h3 className="text-xs font-bold text-white uppercase tracking-wider flex items-center gap-1.5">
              <Radio className="w-3.5 h-3.5 text-emerald-400" />
              <span>SCTE-35 Digital Program Insertion</span>
            </h3>

            <div className="space-y-2">
              <div className="p-2.5 bg-[#1F2937] rounded border border-gray-700 flex items-center justify-between">
                <div>
                  <div className="font-bold text-white">Pre-Roll Commercial Break</div>
                  <div className="text-[10px] text-gray-400">Plays before feature starts • 30s</div>
                </div>
                <span className="text-[10px] px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-300 font-mono font-semibold">
                  SCTE-35
                </span>
              </div>

              <div className="p-2.5 bg-[#1F2937] rounded border border-gray-700 flex items-center justify-between">
                <div>
                  <div className="font-bold text-white">Mid-Roll Interval Break</div>
                  <div className="text-[10px] text-gray-400">Repeats every 45 mins • 60s Break</div>
                </div>
                <span className="text-[10px] px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-300 font-mono font-semibold">
                  SCTE-35
                </span>
              </div>

              <div className="p-2.5 bg-[#1F2937] rounded border border-gray-700 flex items-center justify-between">
                <div>
                  <div className="font-bold text-white">Post-Roll Station Ident</div>
                  <div className="text-[10px] text-gray-400">Plays immediately after credits • 15s</div>
                </div>
                <span className="text-[10px] px-1.5 py-0.5 rounded bg-emerald-500/20 text-emerald-300 font-mono font-semibold">
                  CUE
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
