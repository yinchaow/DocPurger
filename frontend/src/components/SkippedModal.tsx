import React from 'react';
import { AlertCircle, FolderOpen, X } from 'lucide-react';
import type { SkippedFile } from '../types';
import { backend } from '../lib/wails';

interface SkippedModalProps {
  open: boolean;
  files: SkippedFile[];
  onClose: () => void;
}

export const SkippedModal: React.FC<SkippedModalProps> = ({ open, files, onClose }) => {
  if (!open) return null;

  return (
    <div
      onClick={onClose}
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/35 dark:bg-black/60 backdrop-blur-[6px] animate-fade-in"
    >
      <div
        onClick={(e) => e.stopPropagation()}
        className="w-full max-w-2xl rounded-2xl acrylic-surface overflow-hidden animate-modal-in"
      >
        <div className="px-6 py-4 border-b border-slate-200/70 dark:border-white/10 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-xl bg-amber-500/12 text-amber-600 dark:text-amber-400 flex items-center justify-center">
              <AlertCircle className="w-4.5 h-4.5" />
            </div>
            <div>
              <h3 className="text-sm font-semibold text-slate-900 dark:text-white">
                未能提取正文的受限文件（{files.length}）
              </h3>
              <p className="text-[11px] text-slate-500 dark:text-slate-400">
                加密文档、纯图片扫描件 PDF 或加密压缩包，透明列出以防漏查
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="p-1.5 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-black/5 dark:hover:bg-white/10"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        <div className="max-h-[58vh] overflow-y-auto divide-y divide-slate-200/60 dark:divide-white/[0.06] bg-white/55 dark:bg-transparent">
          {files.map((f, idx) => (
            <div
              key={`${f.filePath}-${idx}`}
              className="px-6 py-3 flex items-center justify-between gap-3 hover:bg-white/80 dark:hover:bg-white/[0.04] transition-colors"
            >
              <div className="min-w-0 flex-1">
                <div className="text-xs font-mono text-slate-800 dark:text-slate-200 truncate">
                  {f.relativePath}
                </div>
                <div className="text-[11px] text-amber-700 dark:text-amber-300 mt-0.5">
                  {f.reason}
                </div>
              </div>
              <button
                type="button"
                onClick={() => backend.revealInFolder(f.filePath)}
                className="shrink-0 inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium fluent-btn-subtle"
              >
                <FolderOpen className="w-3.5 h-3.5" />
                打开目录
              </button>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};
