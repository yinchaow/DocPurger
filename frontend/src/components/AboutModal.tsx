import React from 'react';
import {
  CheckCircle2,
  Copy,
  ExternalLink,
  Monitor,
  Scale,
  ShieldCheck,
  Sparkles,
  Users,
  X,
} from 'lucide-react';
import type { SystemInfo } from '../types';
import { AppLogo } from './FluentIcons';
import { backend } from '../lib/wails';

interface AboutModalProps {
  open: boolean;
  sysInfo: SystemInfo | null;
  onClose: () => void;
  onNotify: (msg: string, tone?: 'info' | 'success' | 'error') => void;
}

const REPO_URL = 'https://github.com/yinchaow/DocPurger';

export const AboutModal: React.FC<AboutModalProps> = ({ open, sysInfo, onClose, onNotify }) => {
  if (!open) return null;

  const archLabel = sysInfo?.arch === 'arm64' ? 'arm64' : 'x64';
  const platformLabel =
    sysInfo?.os === 'windows'
      ? `Windows 10 / 11 (${archLabel})`
      : sysInfo?.os === 'linux'
      ? `银河麒麟 V10 SP1 (${archLabel})`
      : sysInfo?.os === 'darwin'
      ? `macOS (${archLabel})`
      : 'Windows 10 / 11 (x64)';

  const handleCopyURL = () => {
    navigator.clipboard
      ?.writeText(REPO_URL)
      .then(() => onNotify('已复制项目主页链接', 'info'))
      .catch(() => {});
  };

  return (
    <div
      onClick={onClose}
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-900/35 dark:bg-black/60 backdrop-blur-[6px] animate-fade-in"
    >
      <div
        onClick={(e) => e.stopPropagation()}
        className="w-full max-w-[540px] rounded-2xl acrylic-surface overflow-hidden animate-modal-in"
      >
        {/* Top Hero */}
        <div className="px-6 pt-6 pb-5 flex items-start justify-between gap-4 border-b border-slate-200/70 dark:border-white/10">
          <div className="flex items-center gap-4">
            <AppLogo className="w-14 h-14 shrink-0 drop-shadow-md" />
            <div>
              <div className="flex items-center gap-2.5">
                <h2 className="text-xl font-bold tracking-tight text-slate-900 dark:text-white">
                  文档清匣
                </h2>
                <span className="px-2 py-0.5 rounded-full text-[11px] font-semibold bg-[#0067C0]/12 text-[#0067C0] dark:bg-[#60CDFF]/15 dark:text-[#60CDFF]">
                  v{sysInfo?.appVersion || '1.1.0'}
                </span>
              </div>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                DocPurger · 跨平台文档内容检索与安全批量清理工具
              </p>
            </div>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="p-1.5 rounded-lg text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-black/5 dark:hover:bg-white/10 transition-colors"
            title="关闭"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Metadata Rows */}
        <div className="px-6 py-4 space-y-3 text-xs">
          <div className="rounded-xl bg-white/75 dark:bg-white/[0.04] border border-slate-200/70 dark:border-white/[0.08] divide-y divide-slate-200/60 dark:divide-white/[0.06]">
            {/* 1. 适用平台 */}
            <div className="px-4 py-2.5 flex items-center justify-between gap-3">
              <span className="inline-flex items-center gap-2 text-slate-500 dark:text-slate-400 shrink-0">
                <Monitor className="w-3.5 h-3.5 text-[#0067C0] dark:text-[#60CDFF]" />
                适用平台
              </span>
              <span className="text-right font-medium text-slate-800 dark:text-slate-100">
                {platformLabel}
              </span>
            </div>

            {/* 2. 开发人员 */}
            <div className="px-4 py-2.5 flex items-center justify-between gap-3">
              <span className="inline-flex items-center gap-2 text-slate-500 dark:text-slate-400 shrink-0">
                <Users className="w-3.5 h-3.5 text-[#0067C0] dark:text-[#60CDFF]" />
                开发人员
              </span>
              <span className="font-semibold text-slate-800 dark:text-slate-100">
                yinchaow &amp; Gemini
              </span>
            </div>

            {/* 3. 开源协议 */}
            <div className="px-4 py-2.5 flex items-center justify-between gap-3">
              <span className="inline-flex items-center gap-2 text-slate-500 dark:text-slate-400 shrink-0">
                <Scale className="w-3.5 h-3.5 text-[#0067C0] dark:text-[#60CDFF]" />
                开源协议
              </span>
              <span className="font-mono font-medium text-slate-800 dark:text-slate-100">
                MIT License
              </span>
            </div>

            {/* 4. 项目主页 */}
            <div className="px-4 py-2.5 flex items-center justify-between gap-3">
              <span className="inline-flex items-center gap-2 text-slate-500 dark:text-slate-400 shrink-0">
                <ExternalLink className="w-3.5 h-3.5 text-[#0067C0] dark:text-[#60CDFF]" />
                项目主页
              </span>
              <div className="flex items-center gap-1.5 min-w-0">
                <button
                  type="button"
                  onClick={() => backend.openURL(REPO_URL)}
                  className="font-mono text-[#0067C0] dark:text-[#60CDFF] hover:underline truncate"
                >
                  {REPO_URL}
                </button>
                <button
                  type="button"
                  onClick={handleCopyURL}
                  className="p-1 rounded hover:bg-black/5 dark:hover:bg-white/10 text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 shrink-0"
                  title="复制主页地址"
                >
                  <Copy className="w-3 h-3" />
                </button>
              </div>
            </div>
          </div>

          {/* 5. 核心功能 */}
          <div className="rounded-xl bg-white/75 dark:bg-white/[0.04] border border-slate-200/70 dark:border-white/[0.08] p-4 space-y-2">
            <div className="flex items-center gap-1.5 font-semibold text-slate-800 dark:text-slate-100">
              <Sparkles className="w-3.5 h-3.5 text-[#0067C0] dark:text-[#60CDFF]" />
              核心功能
            </div>
            <ul className="space-y-1.5 text-[11.5px] text-slate-600 dark:text-slate-300 leading-relaxed">
              <li className="flex items-start gap-2">
                <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400 shrink-0 mt-0.5" />
                <span>
                  <strong>多目录多核并发检索</strong>：支持一次添加多个目录并自动去重，灵活切换「同时包含 / 包含任意」与排除词。
                </span>
              </li>
              <li className="flex items-start gap-2">
                <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400 shrink-0 mt-0.5" />
                <span>
                  <strong>全格式零依赖解析</strong>：原生解析 docx/xlsx/pptx、老式 doc/xls/ppt（含伪装 RTF/HTML）、PDF、国产 OFD 与 GBK/UTF-8 文本。
                </span>
              </li>
              <li className="flex items-start gap-2">
                <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400 shrink-0 mt-0.5" />
                <span>
                  <strong>纯内存穿透压缩包</strong>：免解压直读 zip / rar / 7z / tar.gz 内部文件名与文档正文，自动修复中文文件名乱码。
                </span>
              </li>
              <li className="flex items-start gap-2">
                <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400 shrink-0 mt-0.5" />
                <span>
                  <strong>防误删与安全批量处置</strong>：命中上下文高亮核验、原生目录高亮定位、默认移入系统回收站，支持隔离备份与审计日志 CSV 导出。
                </span>
              </li>
            </ul>
          </div>
        </div>

        {/* Footer */}
        <div className="px-6 py-3.5 bg-slate-100/60 dark:bg-white/[0.02] border-t border-slate-200/70 dark:border-white/10 flex items-center justify-between text-[11px] text-slate-500 dark:text-slate-400">
          <span className="inline-flex items-center gap-1.5">
            <ShieldCheck className="w-3.5 h-3.5 text-emerald-600 dark:text-emerald-400" />
            纯本地离线运行 · 零数据上传 · Copyright © 2026 yinchaow
          </span>
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-1.5 rounded-lg text-xs font-medium fluent-btn-primary"
          >
            确定
          </button>
        </div>
      </div>
    </div>
  );
};
