import React, { useState, useEffect, useCallback } from 'react';
import { Maximize2, Minimize2 } from 'lucide-react';

/**
 * FullscreenButton - A reusable fullscreen toggle button.
 * Pass a ref to the container element you want to go fullscreen.
 * If no containerRef is passed, it will fullscreen the entire document.
 */
export default function FullscreenButton({ containerRef, className = '' }) {
  const [isFullscreen, setIsFullscreen] = useState(false);

  // Listen for fullscreen changes (user may press Esc to exit)
  useEffect(() => {
    const handleChange = () => {
      setIsFullscreen(!!document.fullscreenElement);
    };
    document.addEventListener('fullscreenchange', handleChange);
    document.addEventListener('webkitfullscreenchange', handleChange);
    return () => {
      document.removeEventListener('fullscreenchange', handleChange);
      document.removeEventListener('webkitfullscreenchange', handleChange);
    };
  }, []);

  const toggleFullscreen = useCallback(async () => {
    try {
      if (!document.fullscreenElement) {
        const el = containerRef?.current || document.documentElement;
        if (el.requestFullscreen) {
          await el.requestFullscreen();
        } else if (el.webkitRequestFullscreen) {
          await el.webkitRequestFullscreen();
        }
      } else {
        if (document.exitFullscreen) {
          await document.exitFullscreen();
        } else if (document.webkitExitFullscreen) {
          await document.webkitExitFullscreen();
        }
      }
    } catch (err) {
      console.warn('Fullscreen toggle failed:', err);
    }
  }, [containerRef]);

  return (
    <button
      onClick={toggleFullscreen}
      title={isFullscreen ? '退出全屏' : '全屏显示'}
      className={`flex items-center gap-1.5 px-3 py-2 rounded-xl bg-white/70 backdrop-blur-sm border border-gray-200/50 text-sm font-medium text-gray-600 hover:bg-indigo-50 hover:text-indigo-600 hover:border-indigo-200 transition-all shadow-sm ${className}`}
    >
      {isFullscreen ? <Minimize2 size={16} /> : <Maximize2 size={16} />}
      <span className="hidden sm:inline">{isFullscreen ? '退出全屏' : '全屏'}</span>
    </button>
  );
}
