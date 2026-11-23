import React, { useRef, useState } from 'react';
import { X } from 'lucide-react';

interface KeywordInputProps {
  values: string[];
  onChange: (next: string[]) => void;
  draft?: string;
  onDraftChange?: (next: string) => void;
  onSubmitSearch?: (extraTokens?: string[]) => void;
  placeholder: string;
  variant?: 'fluent-bar' | 'sidebar-exclude';
}

export const KeywordInput: React.FC<KeywordInputProps> = ({
  values,
  onChange,
  draft: controlledDraft,
  onDraftChange,
  onSubmitSearch,
  placeholder,
  variant = 'fluent-bar',
}) => {
  const [internalDraft, setInternalDraft] = useState('');
  const draft = controlledDraft !== undefined ? controlledDraft : internalDraft;
  const setDraft = (val: string) => {
    if (onDraftChange) onDraftChange(val);
    else setInternalDraft(val);
  };
  const inputRef = useRef<HTMLInputElement>(null);

  const commitTokens = (raw: string): string[] => {
    const tokens = raw
      .split(/[,，;；\n]+/)
      .map((s) => s.trim())
      .filter(Boolean);
    if (tokens.length === 0) return values;

    const next = [...values];
    for (const tok of tokens) {
      if (!next.includes(tok)) {
        next.push(tok);
      }
    }
    onChange(next);
    setDraft('');
    return next;
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      if (draft.trim()) {
        const next = commitTokens(draft);
        if ((e.ctrlKey || e.metaKey) && onSubmitSearch && next.length > 0) {
          onSubmitSearch(next);
        }
      } else if (onSubmitSearch && values.length > 0) {
        onSubmitSearch(values);
      }
      return;
    }
    if (e.key === ',' || e.key === '，' || e.key === ';' || e.key === '；') {
      e.preventDefault();
      if (draft.trim()) {
        commitTokens(draft);
      }
      return;
    }
    if (e.key === 'Backspace' && draft === '' && values.length > 0) {
      onChange(values.slice(0, -1));
    }
  };

  const handlePaste = (e: React.ClipboardEvent<HTMLInputElement>) => {
    const text = e.clipboardData.getData('text');
    if (/[,，;；\n]/.test(text)) {
      e.preventDefault();
      commitTokens(text);
    }
  };

  if (variant === 'sidebar-exclude') {
    return (
      <div
        onClick={() => inputRef.current?.focus()}
        className="flex flex-wrap items-center gap-1 min-h-[34px] px-2.5 py-1.5 rounded-lg border border-slate-200/90 dark:border-white/10 bg-white/80 dark:bg-black/20 focus-within:border-[#0067C0] dark:focus-within:border-[#60CDFF] transition-colors cursor-text"
      >
        {values.map((kw) => (
          <span
            key={kw}
            className="inline-flex items-center gap-1 pl-2 pr-1 py-0.5 rounded-md text-[11px] font-medium bg-rose-500/12 text-rose-700 dark:text-rose-300"
          >
            <span>-{kw}</span>
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                onChange(values.filter((v) => v !== kw));
              }}
              className="p-0.5 rounded hover:bg-black/10 dark:hover:bg-white/15"
            >
              <X className="w-2.5 h-2.5" />
            </button>
          </span>
        ))}
        <input
          ref={inputRef}
          type="text"
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={handleKeyDown}
          onPaste={handlePaste}
          onBlur={() => {
            if (draft.trim()) commitTokens(draft);
          }}
          placeholder={values.length === 0 ? placeholder : '继续添加排除词…'}
          className="flex-1 min-w-[70px] bg-transparent text-xs text-slate-800 dark:text-slate-200 placeholder:text-slate-400 dark:placeholder:text-slate-500 outline-none"
        />
      </div>
    );
  }

  return (
    <div
      onClick={() => inputRef.current?.focus()}
      className="flex-1 flex flex-wrap items-center gap-1.5 min-w-0 py-0.5 cursor-text"
    >
      {values.map((kw) => (
        <span
          key={kw}
          className="inline-flex items-center gap-1 pl-2.5 pr-1.5 py-1 rounded-lg text-xs font-medium bg-[#E8F1FB] text-[#004E98] border border-[#0067C0]/20 dark:bg-[#60CDFF]/15 dark:text-[#A6E3FF] dark:border-[#60CDFF]/25 animate-fade-in"
        >
          <span>{kw}</span>
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              onChange(values.filter((v) => v !== kw));
            }}
            className="p-0.5 rounded hover:bg-[#0067C0]/15 dark:hover:bg-white/15 transition-colors"
            title="移除关键词"
          >
            <X className="w-3 h-3" />
          </button>
        </span>
      ))}
      <input
        ref={inputRef}
        type="text"
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onKeyDown={handleKeyDown}
        onPaste={handlePaste}
        onBlur={() => {
          if (draft.trim()) commitTokens(draft);
        }}
        placeholder={values.length === 0 ? placeholder : '继续输入回车添加更多关键词…'}
        className="flex-1 min-w-[160px] bg-transparent text-sm text-slate-900 dark:text-slate-100 placeholder:text-slate-400 dark:placeholder:text-slate-500 outline-none py-0.5"
      />
      {(values.length > 0 || draft.length > 0) && (
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            onChange([]);
            setDraft('');
          }}
          className="p-1 rounded-md text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 hover:bg-black/5 dark:hover:bg-white/10 transition-colors"
          title="清空全部关键词"
        >
          <X className="w-3.5 h-3.5" />
        </button>
      )}
    </div>
  );
};
