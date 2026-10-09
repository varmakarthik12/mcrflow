import React, { useEffect, useRef, useState } from 'react';
import Hls from 'hls.js';
import { Play, Volume2, VolumeX, AlertCircle } from 'lucide-react';

export function VideoPlayer({
  streamUrl,
  isSlate = false,
  channelName = "Live Channel",
  logoPath = "",
  logoPosition = "top-right"
}) {
  const videoRef = useRef(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [isMuted, setIsMuted] = useState(true);
  const [error, setError] = useState(null);

  const getLogoPositionClass = (pos) => {
    switch (pos) {
      case 'top-left':
        return 'top-3 left-3';
      case 'bottom-right':
        return 'bottom-10 right-3';
      case 'bottom-left':
        return 'bottom-10 left-3';
      case 'top-right':
      default:
        return 'top-3 right-3';
    }
  };

  useEffect(() => {
    const video = videoRef.current;
    if (!video || !streamUrl) return;

    let hls = null;
    setError(null);

    if (Hls.isSupported()) {
      hls = new Hls({
        enableWorker: true,
        lowLatencyMode: true,
        backBufferLength: 30,
      });

      hls.loadSource(streamUrl);
      hls.attachMedia(video);

      hls.on(Hls.Events.MANIFEST_PARSED, () => {
        video.play().then(() => setIsPlaying(true)).catch(() => {
          setIsPlaying(false);
        });
      });

      hls.on(Hls.Events.ERROR, (event, data) => {
        if (data.fatal) {
          switch (data.type) {
            case Hls.ErrorTypes.NETWORK_ERROR:
              hls.startLoad();
              break;
            case Hls.ErrorTypes.MEDIA_ERROR:
              hls.recoverMediaError();
              break;
            default:
              hls.destroy();
              setError("Live stream unavailable");
              break;
          }
        }
      });
    } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
      video.src = streamUrl;
      video.addEventListener('loadedmetadata', () => {
        video.play().then(() => setIsPlaying(true)).catch(() => setIsPlaying(false));
      });
    }

    return () => {
      if (hls) {
        hls.destroy();
      }
    };
  }, [streamUrl]);

  const toggleMute = () => {
    if (videoRef.current) {
      videoRef.current.muted = !isMuted;
      setIsMuted(!isMuted);
    }
  };

  const togglePlay = () => {
    if (videoRef.current) {
      if (videoRef.current.paused) {
        videoRef.current.play();
        setIsPlaying(true);
      } else {
        videoRef.current.pause();
        setIsPlaying(false);
      }
    }
  };

  const normalizedLogoUrl = logoPath
    ? (logoPath.startsWith('/') || logoPath.startsWith('http') ? logoPath : `/${logoPath}`)
    : "";

  return (
    <div className="relative aspect-video bg-black rounded-lg overflow-hidden border border-gray-800 flex items-center justify-center group shadow-inner">
      <video
        ref={videoRef}
        muted={isMuted}
        playsInline
        autoPlay
        className="w-full h-full object-contain"
      />

      {/* Station Logo / Channel Bug Overlay */}
      {normalizedLogoUrl && !isSlate && (
        <div className={`absolute ${getLogoPositionClass(logoPosition)} z-10 pointer-events-none transition-all duration-300`}>
          <img
            src={normalizedLogoUrl}
            alt="Station Logo Bug"
            className="h-7 w-auto max-w-[100px] object-contain drop-shadow-md opacity-90"
            onError={(e) => {
              e.currentTarget.style.display = 'none';
            }}
          />
        </div>
      )}

      {/* Emergency Slate Overlay */}
      {isSlate && (
        <div className="absolute inset-0 bg-red-950/95 z-30 flex flex-col items-center justify-center border-4 border-red-600 animate-pulse">
          <div className="w-16 h-16 rounded-full bg-red-600/30 flex items-center justify-center mb-3">
            <AlertCircle className="w-10 h-10 text-red-500" />
          </div>
          <div className="text-xl font-black text-white tracking-widest uppercase">EMERGENCY SLATE ACTIVE</div>
          <div className="text-xs text-red-300 font-mono mt-1">Egress replaced with fallback technical slide</div>
          <div className="text-[10px] text-gray-400 font-mono mt-3">AUDIO MUTE • SCTE-35 INHIBITED</div>
        </div>
      )}

      {/* Video controls on hover */}
      <div className="absolute bottom-2 left-2 right-2 z-20 flex items-center justify-between opacity-0 group-hover:opacity-100 transition-opacity bg-black/60 px-3 py-1.5 rounded backdrop-blur">
        <div className="flex items-center gap-2">
          <button onClick={togglePlay} className="text-white hover:text-sky-400">
            <Play className={`w-4 h-4 ${isPlaying ? 'fill-current' : ''}`} />
          </button>
          <button onClick={toggleMute} className="text-white hover:text-sky-400">
            {isMuted ? <VolumeX className="w-4 h-4" /> : <Volume2 className="w-4 h-4" />}
          </button>
          <span className="text-[10px] text-emerald-400 font-mono font-semibold flex items-center gap-1">
            <span className="w-1.5 h-1.5 rounded-full bg-emerald-400 animate-pulse"></span>
            LIVE HLS
          </span>
        </div>
        <div className="text-[10px] text-gray-300 font-mono truncate max-w-[200px]">
          {channelName}
        </div>
      </div>

      {error && !isSlate && (
        <div className="absolute inset-0 bg-black/80 flex flex-col items-center justify-center text-gray-400 text-xs p-4 text-center">
          <AlertCircle className="w-6 h-6 text-amber-500 mb-1" />
          <span>{error}</span>
          <span className="text-[10px] text-gray-500 font-mono mt-1">{streamUrl}</span>
        </div>
      )}
    </div>
  );
}
