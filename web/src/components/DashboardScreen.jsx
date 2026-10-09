import React from 'react';
import {
  Activity,
  Cpu,
  HardDrive,
  Radio,
  Tv,
  CheckCircle2,
  Clock,
  ArrowRight,
  ShieldCheck,
  Zap
} from 'lucide-react';

export function DashboardScreen({
  channels = [],
  onManageChannels,
  onSelectChannel,
  t
}) {
  const activeCount = channels.filter((c) => c.is_active !== false).length;
  const totalCount = channels.length;

  return (
    <div className="h-full flex flex-col p-4 space-y-4 overflow-y-auto">
      {/* Top Operational KPI Cards */}
      <div className="grid grid-cols-2 md:grid-cols-5 gap-3 shrink-0">
        <div className="bg-[#111827] border border-[#1F2937] p-3 rounded-lg shadow-sm">
          <div className="text-[11px] text-gray-400 font-medium flex items-center justify-between">
            <span>{t('dash.active_channels') || "Active Broadcast Channels"}</span>
            <Tv className="w-3.5 h-3.5 text-indigo-400" />
          </div>
          <div className="text-xl font-bold text-white mt-1 flex items-baseline gap-1.5 font-mono">
            {activeCount} / {totalCount || 1}
            <span className="text-xs text-emerald-400 font-normal">● 100% On-Air</span>
          </div>
          <div className="text-[10px] text-gray-500 mt-1 flex items-center gap-1">
            <ShieldCheck className="w-3 h-3 text-emerald-400" />
            <span>1+1 Hitless Redundant Standby</span>
          </div>
        </div>

        <div className="bg-[#111827] border border-[#1F2937] p-3 rounded-lg shadow-sm">
          <div className="text-[11px] text-gray-400 font-medium flex items-center justify-between">
            <span>{t('dash.transcode_load') || "GPU NVENC Encoding Load"}</span>
            <Zap className="w-3.5 h-3.5 text-sky-400" />
          </div>
          <div className="text-xl font-bold text-white mt-1 flex items-baseline gap-1.5 font-mono">
            38.4% <span className="text-xs text-sky-400 font-normal">100.0 FPS Aggregate</span>
          </div>
          <div className="text-[10px] text-gray-500 mt-1">NVIDIA RTX A4000 • VAAPI Ready</div>
        </div>

        <div className="bg-[#111827] border border-[#1F2937] p-3 rounded-lg shadow-sm">
          <div className="text-[11px] text-gray-400 font-medium flex items-center justify-between">
            <span>{t('dash.cluster_cpu') || "Edge Nodes CPU / RAM"}</span>
            <Cpu className="w-3.5 h-3.5 text-purple-400" />
          </div>
          <div className="text-xl font-bold text-white mt-1 flex items-baseline gap-1.5 font-mono">
            22.6% <span className="text-xs text-gray-400 font-normal">14.8 / 64 GB</span>
          </div>
          <div className="text-[10px] text-gray-500 mt-1">Distributed Edge Transmitters</div>
        </div>

        <div className="bg-[#111827] border border-[#1F2937] p-3 rounded-lg shadow-sm">
          <div className="text-[11px] text-gray-400 font-medium flex items-center justify-between">
            <span>{t('dash.egress_bandwidth') || "Playout Egress Throughput"}</span>
            <Activity className="w-3.5 h-3.5 text-emerald-400" />
          </div>
          <div className="text-xl font-bold text-white mt-1 flex items-baseline gap-1.5 font-mono">
            184.2 <span className="text-xs text-emerald-400 font-normal">Mbps</span>
          </div>
          <div className="text-[10px] text-gray-500 mt-1">UDP Mux + SRT Caller + HLS</div>
        </div>

        <div className="bg-[#111827] border border-[#1F2937] p-3 rounded-lg shadow-sm">
          <div className="text-[11px] text-gray-400 font-medium flex items-center justify-between">
            <span>{t('dash.active_destinations') || "Active Stream Destinations"}</span>
            <Radio className="w-3.5 h-3.5 text-amber-400" />
          </div>
          <div className="text-xl font-bold text-white mt-1 flex items-baseline gap-1.5 font-mono">
            {channels.reduce((acc, c) => acc + (c.destinations?.length || 2), 0)}
            <span className="text-xs text-amber-400 font-normal">Endpoints</span>
          </div>
          <div className="text-[10px] text-gray-500 mt-1">Multicast TS, SRT & CDN HLS</div>
        </div>
      </div>

      {/* Channels Playout Matrix */}
      <div className="flex-1 flex flex-col space-y-2 min-h-0">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold text-white uppercase tracking-wider flex items-center gap-2">
            <span className="w-2 h-2 rounded-full bg-sky-400 animate-pulse"></span>
            <span>{t('dash.channel_grid_title') || "Live Playout Matrix"}</span>
          </h2>
          <div className="flex items-center gap-2">
            <span className="text-xs text-gray-400 font-mono">Auto-Telemetry: 500ms</span>
            <button
              onClick={onManageChannels}
              className="text-xs text-sky-400 hover:text-sky-300 font-medium flex items-center gap-1"
            >
              <span>+ Manage Channels</span>
              <ArrowRight className="w-3.5 h-3.5" />
            </button>
          </div>
        </div>

        {/* Dynamic Channel Cards Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 flex-1 overflow-y-auto">
          {channels.map((ch, idx) => {
            const hlsDest = ch.destinations?.find((d) => d.protocol === "HLS");
            return (
              <div
                key={ch.id}
                className="bg-[#111827] border border-[#1F2937] hover:border-indigo-500/50 rounded-lg p-3 flex flex-col space-y-3 transition-all shadow-md group"
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className="px-1.5 py-0.5 rounded text-[10px] font-bold bg-indigo-500/20 text-indigo-300 border border-indigo-500/30 font-mono">
                      CH {String(ch.lcn || idx + 1).padStart(2, '0')}
                    </span>
                    <span className="text-sm font-bold text-white truncate max-w-[130px]">
                      {ch.name}
                    </span>
                  </div>
                  <span className="px-2 py-0.5 rounded text-[10px] font-semibold bg-emerald-500/20 text-emerald-400 border border-emerald-500/30 flex items-center gap-1 font-mono">
                    <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
                    ON-AIR
                  </span>
                </div>

                {/* Video Simulation Thumbnail */}
                <div className="relative aspect-video bg-black rounded overflow-hidden border border-gray-800 flex items-center justify-center">
                  <img
                    src={`https://images.unsplash.com/photo-1536440136628-849c177e76a1?w=400&auto=format&fit=crop&q=60`}
                    alt="Channel Feed"
                    className="w-full h-full object-cover opacity-65 group-hover:opacity-80 transition-opacity"
                  />
                  <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-black/40"></div>

                  {/* Channel Bug */}
                  <div className="absolute top-2 right-2 px-1.5 py-0.5 bg-red-600/90 text-[9px] font-black text-white rounded tracking-wider shadow">
                    {ch.call_sign || "MCR"}
                  </div>

                  {/* Program metadata */}
                  <div className="absolute bottom-2 left-2 right-2">
                    <div className="text-[11px] font-bold text-white drop-shadow truncate">
                      Prime Broadcast Event
                    </div>
                    <div className="text-[9px] text-gray-300 drop-shadow font-mono">
                      {ch.resolution_id || "1080i50"} • {ch.video_codec || "H.264"}
                    </div>
                  </div>
                </div>

                {/* Playout Progress */}
                <div className="space-y-1">
                  <div className="flex justify-between text-[10px] text-gray-400 font-mono">
                    <span>Elapsed: 01:24:10</span>
                    <span className="text-amber-400">Rem: 00:35:50</span>
                  </div>
                  <div className="w-full h-1.5 bg-gray-800 rounded-full overflow-hidden">
                    <div className="h-full bg-indigo-500 rounded-full" style={{ width: '70%' }}></div>
                  </div>
                </div>

                {/* Bottom destinations summary */}
                <div className="pt-2 border-t border-gray-800/80 flex items-center justify-between text-[10px]">
                  <div className="flex items-center gap-1 text-gray-400 font-mono truncate max-w-[150px]">
                    <Radio className="w-3 h-3 text-sky-400" />
                    <span>{ch.destinations?.length || 2} Outputs Hot</span>
                  </div>
                  <button
                    onClick={() => onSelectChannel(ch.id)}
                    className="text-sky-400 hover:text-sky-300 font-semibold"
                  >
                    Open Desk →
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </div>
  );
}
