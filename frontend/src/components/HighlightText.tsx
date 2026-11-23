import React from 'react';
import type { HighlightSpan } from '../types';

interface HighlightTextProps {
  text: string;
  highlights?: HighlightSpan[];
  className?: string;
}

/**
 * Safely renders text with rune-indexed warm-amber highlights (zero innerHTML / zero XSS).
 */
export const HighlightText: React.FC<HighlightTextProps> = ({
  text,
  highlights,
  className = '',
}) => {
  if (!highlights || highlights.length === 0) {
    return <span className={className}>{text}</span>;
  }

  const chars = Array.from(text);
  const parts: React.ReactNode[] = [];
  let cursor = 0;

  const sorted = [...highlights].sort((a, b) => a.start - b.start);

  sorted.forEach((span, idx) => {
    const start = Math.max(0, Math.min(span.start, chars.length));
    const end = Math.max(start, Math.min(span.end, chars.length));
    if (start > cursor) {
      parts.push(
        <React.Fragment key={`t-${idx}`}>
          {chars.slice(cursor, start).join('')}
        </React.Fragment>
      );
    }
    if (end > start) {
      parts.push(
        <mark
          key={`h-${idx}`}
          className="rounded px-1 py-[0.5px] font-semibold bg-[#FDE68A]/85 text-slate-900 dark:bg-amber-400/25 dark:text-amber-200"
        >
          {chars.slice(start, end).join('')}
        </mark>
      );
    }
    cursor = Math.max(cursor, end);
  });

  if (cursor < chars.length) {
    parts.push(
      <React.Fragment key="t-tail">{chars.slice(cursor).join('')}</React.Fragment>
    );
  }

  return <span className={className}>{parts}</span>;
};
