import React, { useMemo, useState } from 'react';
import {
  AlertTriangle,
  Archive,
  FolderInput,
  FolderOpen,
  ShieldAlert,
  Trash2,
  X,
} from 'lucide-react';
import type { DeleteMode, SearchResult } from '../types';
import { formatBytes } from '../lib/format';
import { backend } from '../lib/wails';

interface DeleteConfirmModalProps {
  open: boolean;
  mode: DeleteMode;
  items: SearchResult[];
  quarantineDir: string;
  onQuarantineDirChange: (dir: string) => void;
  onCancel: () => void;
  onConfirm: () => Promise<void>;
}

export const DeleteConfirmModal: React.FC<DeleteConfirmModalProps> = ({
  open,
  mode,
  items,
  quarantineDir,
  onQuarantineDirChange,
  onCancel,
  onConfirm,
}) => {
  const [permanentAck, setPermanentAck] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const physicalFiles = useMemo(() => {
    const map = new Map<
      string,
      { path: string; relativePath: string; size: number; isArchive: boolean; innerHits: string[] }
    >();
    for (const item of items) {
      const existing = map.get(item.filePath);
      if (existing) {
        if (item.isArchiveEntry) {
          existing.isArchive = true;
          existing.innerHits.push(item.innerPath);
        }
      } else {
        map.set(item.filePath, {
          path: item.filePath,
          relativePath: item.rootLabel ? `[${item.rootLabel}] ${item.relativePath}` : item.relativePath,
          size: item.diskFileSize || item.size,
          isArchive: item.isArchiveEntry || item.category === 'archive',
          innerHits: item.isArchiveEntry ? [item.innerPath] : [],
        });
      }
    }
    return Array.from(map.values());
  }, [items]);

  const totalSize = useMemo(
    () => physicalFiles.reduce((acc, f) => acc + f.size, 0),
    [physicalFiles]
  );

  const archiveCount = useMemo(
    () => physicalFiles.filter((f) => f.innerHits.length > 0).length,
    [physicalFiles]
  );

  if (!open) return null;

  const isTrash = mode === 'trash';
  const isQuarantine = mode === 'quarantine';
  const isPermanent = mode === 'permanent';

  const canSubmit =
    !submitting &&
    physicalFiles.length > 0 &&
    (!isPermanent || permanentAck) &&
    (!isQuarantine || quarantineDir.trim().length > 0);

  const handleConfirm = async () => {
    if (!canSubmit) return;
    setSubmitting(true);
    try {
      await onConfirm();
      setPermanentAck(false);
    } finally {
      setSubmitting(false);
    }
  };

  const handlePickQuarantine = async () => {
    const picked = await backend.selectQuarantineDir(quarantineDir);
    if (picked) onQuarantineDirChange(picked);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/35 dark:bg-black/60 backdrop-blur-[6px] animate-fade-in">
      <div className="w-full max-w-xl rounded-2xl acrylic-surface overflow-hidden animate-modal-in">
        {/* Header */}
        <div className="px-6 pt-5 pb-4 flex items-start justify-between gap-4 border-b border-slate-200/70 dark:border-white/10">
          <div className="flex items-start gap-3.5">
            <div
              className={`w-10 h-10 rounded-xl flex items-center justify-center shrink-0 ${
                isPermanent
                  ? 'bg-rose-500/12 text-rose-600 dark:text-rose-400'
                  : isQuarantine
                  ? 'bg-amber-500/12 text-amber-600 dark:text-amber-400'
                  : 'bg-[#0067C0]/12 text-[#0067C0] dark:bg-[#60CDFF]/15 dark:text-[#60CDFF]'
              }`}
            >
              {isPermanent ? (
                <ShieldAlert className="w-5 h-5" />
              ) : isQuarantine ? (
                <FolderInput className="w-5 h-5" />
              ) : (
                <Trash2 className="w-5 h-5" />
              )}
            </div>
            <div>
              <h3 className="text-[15px] font-semibold text-slate-900 dark:text-white">
                {isTrash && '确认移入系统回收站'}
                {isQuarantine && '确认移至安全隔离目录'}
                {isPermanent && '高危操作：确认彻底永久删除'}
              </h3>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 tabular-nums">
                实际涉及 <strong className="text-slate-800 dark:text-slate-100">{physicalFiles.length}</strong> 个磁盘文件 · 预计释放{' '}
                <strong className="text-slate-800 dark:text-slate-100">{formatBytes(totalSize)}</strong>
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={() => {
              setPermanentAck(false);
              onCancel();
            }}
            className="p-1.5 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-black/5 dark:hover:bg-white/10 transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Body */}
        <div className="px-6 py-4 space-y-3.5">
          {archiveCount > 0 && (
            <div className="flex items-start gap-2.5 p-3 rounded-xl bg-amber-500/12 border border-amber-500/30 text-amber-900 dark:text-amber-200 text-xs leading-relaxed">
              <Archive className="w-4 h-4 shrink-0 mt-0.5 text-amber-600 dark:text-amber-400" />
              <div>
                所选包含 <strong>{archiveCount}</strong> 个压缩包内命中的条目，操作将
                <strong>移除整个外层压缩包</strong>，请确认包内无其他需保留文件。
              </div>
            </div>
          )}

          {isQuarantine && (
            <div className="space-y-1.5">
              <label className="text-xs font-medium text-slate-600 dark:text-slate-300">
                隔离备份目录（自动保留原目录结构，防重名覆盖）
              </label>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={quarantineDir}
                  onChange={(e) => onQuarantineDirChange(e.target.value)}
                  placeholder="请选择隔离备份目录…"
                  className="flex-1 px-3 py-2 rounded-lg text-xs font-mono border border-slate-200 dark:border-white/10 bg-white/90 dark:bg-black/25 text-slate-800 dark:text-slate-100 outline-none focus:border-[#0067C0] dark:focus:border-[#60CDFF]"
                />
                <button
                  type="button"
                  onClick={handlePickQuarantine}
                  className="inline-flex items-center gap-1.5 px-3 py-2 rounded-lg text-xs font-medium fluent-btn-subtle"
                >
                  <FolderOpen className="w-3.5 h-3.5" />
                  浏览
                </button>
              </div>
            </div>
          )}

          <div className="rounded-xl border border-slate-200/80 dark:border-white/10 bg-white/80 dark:bg-black/20 max-h-56 overflow-y-auto divide-y divide-slate-200/60 dark:divide-white/[0.06]">
            {physicalFiles.map((file) => (
              <div key={file.path} className="px-3.5 py-2.5 flex items-start justify-between gap-3">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-1.5">
                    {file.innerHits.length > 0 && (
                      <span className="shrink-0 px-1.5 py-0.5 rounded text-[10px] font-semibold bg-amber-500/18 text-amber-800 dark:text-amber-300">
                        整个压缩包
                      </span>
                    )}
                    <span className="text-xs font-mono text-slate-800 dark:text-slate-200 truncate">
                      {file.relativePath}
                    </span>
                  </div>
                  {file.innerHits.length > 0 && (
                    <div className="text-[11px] text-slate-500 dark:text-slate-400 mt-0.5 truncate">
                      含命中：{file.innerHits.slice(0, 3).join('、')}
                      {file.innerHits.length > 3 ? ` 等 ${file.innerHits.length} 项` : ''}
                    </div>
                  )}
                </div>
                <span className="text-[11px] font-mono tabular-nums text-slate-400 shrink-0">
                  {formatBytes(file.size)}
                </span>
              </div>
            ))}
          </div>

          {isPermanent && (
            <label className="flex items-start gap-2.5 p-3 rounded-xl bg-rose-500/10 border border-rose-500/30 cursor-pointer select-none">
              <input
                type="checkbox"
                checked={permanentAck}
                onChange={(e) => setPermanentAck(e.target.checked)}
                className="mt-0.5 rounded border-rose-400 text-rose-600 focus:ring-rose-500"
              />
              <span className="text-xs text-rose-700 dark:text-rose-200 leading-relaxed">
                <AlertTriangle className="w-3.5 h-3.5 inline mr-1 -mt-0.5" />
                我确认直接从磁盘永久抹除上述文件，知晓无法通过回收站恢复。
              </span>
            </label>
          )}
        </div>

        {/* Footer */}
        <div className="px-6 py-3.5 bg-slate-100/55 dark:bg-white/[0.02] border-t border-slate-200/70 dark:border-white/10 flex items-center justify-between">
          <span className="text-[11px] text-slate-500 dark:text-slate-400">操作将自动记入本机审计日志</span>
          <div className="flex items-center gap-2">
            <button
              type="button"
              disabled={submitting}
              onClick={() => {
                setPermanentAck(false);
                onCancel();
              }}
              className="px-3.5 py-1.5 rounded-lg text-xs font-medium fluent-btn-subtle"
            >
              取消
            </button>
            <button
              type="button"
              disabled={!canSubmit}
              onClick={handleConfirm}
              className={`px-4 py-1.5 rounded-lg text-xs font-medium transition-all disabled:opacity-45 disabled:cursor-not-allowed ${
                isPermanent
                  ? 'bg-rose-600 hover:bg-rose-500 text-white shadow-sm'
                  : isQuarantine
                  ? 'bg-amber-600 hover:bg-amber-500 text-white shadow-sm'
                  : 'fluent-btn-primary'
              }`}
            >
              {submitting
                ? '处理中…'
                : isPermanent
                ? `永久删除 (${physicalFiles.length})`
                : isQuarantine
                ? `移至隔离 (${physicalFiles.length})`
                : `移入回收站 (${physicalFiles.length})`}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};
