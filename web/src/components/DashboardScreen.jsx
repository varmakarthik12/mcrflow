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
  agents = [],
  scheduleItems = [],
  onManageChannels,
  onSelectChannel,
  t
}) {
  const activeCount = channels.filter((c) => c.is_active !== false).length;
  const totalCount = channels.length;
  const onlineAgents = agents.filter((a) => a.status === 'online');
  const avgCpu = onlineAgents.length > 0
    ? (onlineAgents.reduce((sum, a) => sum + (a.cpu_percent || a.cpu_usage_percent || 0), 0) / onlineAgents.length).toFixed(1)
    : "0.0";

  const totalDestinations = channels.reduce((acc, c) => acc + (c.destinations?.length || 0), 0);
  const udpCount = channels.reduce((acc, c) => acc + (c.destinations?.filter(d => d.type === 'udp' || d.protocol === 'UDP_MULTICAST')?.length || 0), 0);
  const srtCount = channels.reduce((acc, c) => acc + (c.destinations?.filter(d => d.type === 'srt' || d.protocol === 'SRT')?.length || 0), 0);
  const hlsCount = channels.reduce((acc, c) => acc + (c.destinations?.filter(d => d.type === 'hls' || d.protocol === 'HLS')?.length || 0), 0);

  const estimatedThroughputMbps = (activeCount * 8.5).toFixed(1);

  const getCardLogoPositionClass = (pos) => {
    switch (pos) {
      case 'top-left':
        return 'top-2 left-2';
      case 'bottom-right':
        return 'bottom-8 right-2';
      case 'bottom-left':
        return 'bottom-8 left-2';
      case 'top-right':
      default:
        return 'top-2 right-2';
    }
  };

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
            <span className="text-xs text-emerald-400 font-normal">● {totalCount > 0 ? Math.round((activeCount / totalCount) * 100) : 0}% On-Air</span>
          </div>
          <div className="text-[10px] text-gray-500 mt-1 flex items-center gap-1">
            <ShieldCheck className="w-3 h-3 text-emerald-400" />
            <span>{activeCount} Active Playout Streams</span>
          </div>
        </div>

        <div className="bg-[#111827] border border-[#1F2937] p-3 rounded-lg shadow-sm">
          <div className="text-[11px] text-gray-400 font-medium flex items-center justify-between">
            <span>Scheduled Linear Programs</span>
            <Clock className="w-3.5 h-3.5 text-sky-400" />
          </div>
          <div className="text-xl font-bold text-white mt-1 flex items-baseline gap-1.5 font-mono">
            {scheduleItems.length} <span className="text-xs text-sky-400 font-normal">Items Cued</span>
          </div>
          <div className="text-[10px] text-gray-500 mt-1">24/7 EPG • TMDb Metadata Active</div>
        </div>

        <div className="bg-[#111827] border border-[#1F2937] p-3 rounded-lg shadow-sm">
          <div className="text-[11px] text-gray-400 font-medium flex items-center justify-between">
            <span>{t('dash.cluster_cpu') || "Edge Playout Nodes"}</span>
            <Cpu className="w-3.5 h-3.5 text-purple-400" />
          </div>
          <div className="text-xl font-bold text-white mt-1 flex items-baseline gap-1.5 font-mono">
            {onlineAgents.length} / {agents.length || 1}
            <span className="text-xs text-purple-300 font-normal">● {avgCpu}% CPU</span>
          </div>
          <div className="text-[10px] text-gray-500 mt-1">
            {agents.length > 0 ? `${agents.length} Registered Playout Daemon(s)` : "Local Playout Agent Active"}
          </div>
        </div>

        <div className="bg-[#111827] border border-[#1F2937] p-3 rounded-lg shadow-sm">
          <div className="text-[11px] text-gray-400 font-medium flex items-center justify-between">
            <span>{t('dash.egress_bandwidth') || "Playout Egress Throughput"}</span>
            <Activity className="w-3.5 h-3.5 text-emerald-400" />
          </div>
          <div className="text-xl font-bold text-white mt-1 flex items-baseline gap-1.5 font-mono">
            {estimatedThroughputMbps} <span className="text-xs text-emerald-400 font-normal">Mbps</span>
          </div>
          <div className="text-[10px] text-gray-500 mt-1">Aggregate Linear Output Stream</div>
        </div>

        <div className="bg-[#111827] border border-[#1F2937] p-3 rounded-lg shadow-sm">
          <div className="text-[11px] text-gray-400 font-medium flex items-center justify-between">
            <span>{t('dash.active_destinations') || "Active Stream Destinations"}</span>
            <Radio className="w-3.5 h-3.5 text-amber-400" />
          </div>
          <div className="text-xl font-bold text-white mt-1 flex items-baseline gap-1.5 font-mono">
            {totalDestinations}
            <span className="text-xs text-amber-400 font-normal">Endpoints</span>
          </div>
          <div className="text-[10px] text-gray-500 mt-1">
            {udpCount} UDP Mux • {srtCount} SRT • {hlsCount} HLS
          </div>
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
            const logoUrl = ch.logo_path
              ? (ch.logo_path.startsWith('/') || ch.logo_path.startsWith('http') ? ch.logo_path : `/${ch.logo_path}`)
              : "";

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

                {/* Video Simulation Thumbnail with Logo Bug */}
                <div className="relative aspect-video bg-black rounded overflow-hidden border border-gray-800 flex items-center justify-center">
                  <img
                    src={`https://images.unsplash.com/photo-1536440136628-849c177e76a1?w=400&auto=format&fit=crop&q=60`}
                    alt="Channel Feed"
                    className="w-full h-full object-cover opacity-65 group-hover:opacity-80 transition-opacity"
                  />
                  <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-black/40"></div>

                  {/* Channel Bug / Station Logo positioned dynamically */}
                  {logoUrl ? (
                    <div className={`absolute ${getCardLogoPositionClass(ch.logo_position)} z-10`}>
                      <img
                        src={logoUrl}
                        alt="Logo Bug"
                        className="h-5 w-auto max-w-[70px] object-contain drop-shadow"
                        onError={(e) => {
                          e.currentTarget.style.display = 'none';
                        }}
                      />
                    </div>
                  ) : (
                    <div className="absolute top-2 right-2 px-1.5 py-0.5 bg-red-600/90 text-[9px] font-black text-white rounded tracking-wider shadow">
                      {ch.call_sign || "MCR"}
                    </div>
                  )}

                  {/* Program metadata */}
                  <div className="absolute bottom-2 left-2 right-2">
                    <div className="text-[11px] font-bold text-white drop-shadow truncate">
                      {ch.name} Transmission
                    </div>
                    <div className="text-[9px] text-gray-300 drop-shadow font-mono">
                      {ch.resolution_id || "1080i50"} • {ch.video_codec || "H.264"}
                    </div>
                  </div>
                </div>

                {/* Destinations and Desk Open */}
                <div className="pt-2 border-t border-gray-800/80 flex items-center justify-between text-[10px]">
                  <div className="flex items-center gap-1 text-gray-400 font-mono truncate max-w-[150px]">
                    <Radio className="w-3 h-3 text-sky-400" />
                    <span>{ch.destinations?.length || 0} Outputs Hot</span>
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
