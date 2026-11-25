import React, { useEffect, useState } from 'react';
import {
  CheckCircle2,
  Download,
  History,
  Trash2,
  X,
  XCircle,
} from 'lucide-react';
import type { AuditEntry } from '../types';
import { backend } from '../lib/wails';
import { formatBytes } from '../lib/format';

interface AuditDrawerProps {
  open: boolean;
  onClose: () => void;
  onNotify: (msg: string, tone?: 'info' | 'success' | 'error') => void;
}

export const AuditDrawer: React.FC<AuditDrawerProps> = ({ open, onClose, onNotify }) => {
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [confirmClear, setConfirmClear] = useState(false);

  const load = async () => {
    setLoading(true);
    try {
      const list = await backend.getAuditLogs(200);
      setEntries(list || []);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (open) {
      setConfirmClear(false);
      load();
    }
  }, [open]);

  if (!open) return null;

  const handleExport = async () => {
    try {
      const dest = await backend.exportAuditCSV();
      if (dest) onNotify(`审计日志已导出：${dest}`, 'success');
    } catch (err: any) {
      onNotify(`导出失败：${err?.message || err}`, 'error');
    }
  };

  const handleClear = async () => {
    if (!confirmClear) {
      setConfirmClear(true);
      return;
    }
    await backend.clearAuditLogs();
    setEntries([]);
    setConfirmClear(false);
    onNotify('操作审计日志已清空', 'info');
  };

  const modeLabel = (mode: AuditEntry['mode']) => {
    if (mode === 'permanent') return { text: '彻底删除', cls: 'bg-rose-500/12 text-rose-700 dark:text-rose-300' };
    if (mode === 'quarantine') return { text: '隔离备份', cls: 'bg-amber-500/12 text-amber-800 dark:text-amber-300' };
    return { text: '移入回收站', cls: 'bg-[#0067C0]/12 text-[#0067C0] dark:bg-[#60CDFF]/15 dark:text-[#60CDFF]' };
  };

  return (
    <div
      onClick={onClose}
      className="fixed inset-0 z-50 flex justify-end bg-slate-900/30 dark:bg-black/55 backdrop-blur-[4px] animate-fade-in"
    >
      <div
        onClick={(e) => e.stopPropagation()}
        className="w-full max-w-lg h-full acrylic-surface border-l border-slate-200/80 dark:border-white/10 flex flex-col animate-drawer-in"
      >
        <div className="px-5 py-4 border-b border-slate-200/70 dark:border-white/10 flex items-center justify-between">
          <div>
            <h3 className="text-sm font-semibold text-slate-900 dark:text-white">操作审计日志</h3>
            <p className="text-[11px] text-slate-500 dark:text-slate-400 mt-0.5 tabular-nums">
              最近 {entries.length} 条文件清理记录，支持导出 Excel/WPS 表格
            </p>
          </div>
          <div className="flex items-center gap-1.5">
            {entries.length > 0 && (
              <>
                <button
                  type="button"
                  onClick={handleExport}
                  className="inline-flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium fluent-btn-subtle"
                >
                  <Download className="w-3.5 h-3.5" />
                  导出 CSV
                </button>
                <button
                  type="button"
                  onClick={handleClear}
                  onBlur={() => setConfirmClear(false)}
                  className={`inline-flex items-center gap-1 rounded-lg transition-colors ${
                    confirmClear
                      ? 'px-2.5 py-1.5 text-xs font-medium bg-rose-600 text-white hover:bg-rose-500'
                      : 'p-1.5 text-slate-400 hover:text-rose-600 hover:bg-rose-500/10'
                  }`}
                  title={confirmClear ? '再次点击确认清空全部审计日志' : '清空日志'}
                >
                  <Trash2 className="w-3.5 h-3.5" />
                  {confirmClear && <span>确认清空？</span>}
                </button>
              </>
            )}
            <button
              type="button"
              onClick={onClose}
              className="p-1.5 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-black/5 dark:hover:bg-white/10 transition-colors"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>

        <div className="flex-1 overflow-y-auto p-3 space-y-2">
          {loading ? (
            <div className="py-16 text-center text-xs text-slate-400">加载中…</div>
          ) : entries.length === 0 ? (
            <div className="py-24 text-center px-8">
              <History className="w-9 h-9 mx-auto text-slate-300 dark:text-slate-600 mb-3 stroke-[1.5]" />
              <p className="text-sm font-medium text-slate-600 dark:text-slate-300">暂无清理记录</p>
              <p className="text-xs text-slate-400 dark:text-slate-500 mt-1">
                执行移入回收站、隔离或删除操作后，明细会自动留档在此
              </p>
            </div>
          ) : (
            entries.map((entry, idx) => {
              const badge = modeLabel(entry.mode);
              return (
                <div
                  key={`${entry.timestamp}-${idx}`}
                  className="p-3.5 rounded-xl bg-white/80 dark:bg-white/[0.04] border border-slate-200/70 dark:border-white/[0.07] space-y-1.5"
                >
                  <div className="flex items-center justify-between gap-2 text-xs">
                    <div className="flex items-center gap-2">
                      {entry.success ? (
                        <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400 shrink-0" />
                      ) : (
                        <XCircle className="w-3.5 h-3.5 text-rose-500 shrink-0" />
                      )}
                      <span className={`px-1.5 py-0.5 rounded text-[10px] font-semibold ${badge.cls}`}>
                        {badge.text}
                      </span>
                      <span className="text-[11px] font-mono tabular-nums text-slate-400">
                        {formatBytes(entry.fileSize)}
                      </span>
                    </div>
                    <span className="text-[11px] font-mono text-slate-400 tabular-nums">
                      {new Date(entry.timestamp).toLocaleString('zh-CN', { hour12: false })}
                    </span>
                  </div>
                  <div className="text-xs font-mono text-slate-800 dark:text-slate-200 break-all">
                    {entry.filePath}
                  </div>
                  {entry.destination && (
                    <div className="text-[11px] font-mono text-amber-700 dark:text-amber-300 break-all">
                      → {entry.destination}
                    </div>
                  )}
                  {entry.keywords && entry.keywords.length > 0 && (
                    <div className="flex flex-wrap gap-1 pt-0.5">
                      {entry.keywords.map((kw) => (
                        <span
                          key={kw}
                          className="px-1.5 py-0.5 rounded bg-slate-100 dark:bg-white/[0.07] text-[10px] text-slate-500 dark:text-slate-300"
                        >
                          {kw}
                        </span>
                      ))}
                    </div>
                  )}
                  {entry.error && <div className="text-[11px] text-rose-500">{entry.error}</div>}
                </div>
              );
            })
          )}
        </div>
      </div>
    </div>
  );
};
