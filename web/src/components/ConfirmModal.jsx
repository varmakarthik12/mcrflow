import React from 'react';
import { AlertTriangle, X } from 'lucide-react';

export function ConfirmModal({ isOpen, title, message, onConfirm, onCancel, confirmText = "Confirm", isDanger = false }) {
  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-black/80 backdrop-blur-sm z-50 flex items-center justify-center p-4">
      <div className="bg-[#111827] border border-[#2D3A54] rounded-xl w-full max-w-sm shadow-2xl overflow-hidden animate-in fade-in zoom-in duration-150">
        <div className="px-5 py-3.5 bg-[#1A2234] border-b border-[#2D3A54] flex items-center justify-between">
          <div className="flex items-center gap-2">
            <AlertTriangle className={`w-4 h-4 ${isDanger ? 'text-rose-400' : 'text-amber-400'}`} />
            <h3 className="text-sm font-bold text-white">{title || "Confirmation Required"}</h3>
          </div>
          <button onClick={onCancel} className="text-gray-400 hover:text-white">
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="p-5 text-xs text-gray-300 leading-relaxed">
          {message}
        </div>

        <div className="px-5 py-3 bg-[#1A2234] border-t border-[#2D3A54] flex justify-end gap-2">
          <button
            onClick={onCancel}
            className="px-3.5 py-1.5 bg-gray-800 hover:bg-gray-700 text-gray-300 rounded text-xs font-semibold"
          >
            Cancel
          </button>
          <button
            onClick={onConfirm}
            className={`px-4 py-1.5 rounded text-xs font-semibold text-white shadow ${
              isDanger
                ? 'bg-rose-600 hover:bg-rose-500'
                : 'bg-indigo-600 hover:bg-indigo-500'
            }`}
          >
            {confirmText}
          </button>
        </div>
      </div>
    </div>
  );
}
