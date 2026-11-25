export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const idx = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  const val = bytes / Math.pow(1024, idx);
  return `${val >= 100 || idx === 0 ? val.toFixed(0) : val.toFixed(1)} ${units[idx]}`;
}

export function formatElapsed(ms: number): string {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(2)}s`;
  const mins = Math.floor(ms / 60000);
  const secs = ((ms % 60000) / 1000).toFixed(1);
  return `${mins}m ${secs}s`;
}

export interface CategoryMeta {
  id: string;
  label: string;
  shortLabel: string;
  extHint: string;
  exts: string;
  dotColor: string;
  iconColor: string;
}

export const CATEGORIES: CategoryMeta[] = [
  {
    id: 'word',
    label: 'Word / WPS',
    shortLabel: 'DOC',
    extHint: '.docx 等',
    exts: '.docx .doc .wps .rtf .odt',
    dotColor: 'bg-sky-500',
    iconColor: 'text-sky-600 dark:text-sky-400',
  },
  {
    id: 'excel',
    label: 'Excel 表格',
    shortLabel: 'XLS',
    extHint: '.xlsx 等',
    exts: '.xlsx .xls .et .ods',
    dotColor: 'bg-emerald-500',
    iconColor: 'text-emerald-600 dark:text-emerald-400',
  },
  {
    id: 'ppt',
    label: 'PPT 演示',
    shortLabel: 'PPT',
    extHint: '.pptx 等',
    exts: '.pptx .ppt .dps .odp',
    dotColor: 'bg-orange-500',
    iconColor: 'text-orange-600 dark:text-orange-400',
  },
  {
    id: 'pdf',
    label: 'PDF 文档',
    shortLabel: 'PDF',
    extHint: '.pdf',
    exts: '.pdf',
    dotColor: 'bg-rose-500',
    iconColor: 'text-rose-600 dark:text-rose-400',
  },
  {
    id: 'ofd',
    label: 'OFD 版式',
    shortLabel: 'OFD',
    extHint: '.ofd',
    exts: '.ofd',
    dotColor: 'bg-violet-500',
    iconColor: 'text-violet-600 dark:text-violet-400',
  },
  {
    id: 'text',
    label: '文本 / 代码',
    shortLabel: 'TXT',
    extHint: '.txt 等',
    exts: '.txt .md .csv .json .xml .log',
    dotColor: 'bg-zinc-400',
    iconColor: 'text-zinc-500 dark:text-zinc-400',
  },
  {
    id: 'archive',
    label: '压缩文件',
    shortLabel: 'ZIP',
    extHint: '.zip 等',
    exts: '.zip .rar .7z .tar.gz',
    dotColor: 'bg-amber-500',
    iconColor: 'text-amber-600 dark:text-amber-400',
  },
];

export function getCategoryMeta(cat: string): CategoryMeta {
  return (
    CATEGORIES.find((c) => c.id === cat) || {
      id: 'other',
      label: '其他文件',
      shortLabel: 'FILE',
      extHint: '',
      exts: '',
      dotColor: 'bg-zinc-400',
      iconColor: 'text-zinc-500 dark:text-zinc-400',
    }
  );
}
