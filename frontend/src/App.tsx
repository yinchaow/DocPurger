import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  AlertTriangle,
  Archive,
  Check,
  CheckSquare,
  ClipboardPaste,
  Copy,
  Eraser,
  ExternalLink,
  Folder,
  FolderInput,
  FolderOpen,
  FolderPlus,
  FolderSearch,
  History,
  Info,
  Layers,
  Loader2,
  Maximize2,
  Minimize2,
  Minus,
  Moon,
  Plus,
  Search,
  SlidersHorizontal,
  Square,
  SquareMinus,
  StopCircle,
  Sun,
  Trash2,
  X,
} from 'lucide-react';
import type {
  DeleteMode,
  MatchMode,
  SearchOptions,
  SearchProgress,
  SearchResult,
  SearchScope,
  SearchSummary,
  SkippedFile,
  SystemInfo,
} from './types';
import { backend } from './lib/wails';
import { CATEGORIES, formatBytes, formatElapsed } from './lib/format';
import { AppLogo, FluentFileIcon } from './components/FluentIcons';
import { HighlightText } from './components/HighlightText';
import { KeywordInput } from './components/KeywordInput';
import { DeleteConfirmModal } from './components/DeleteConfirmModal';
import { AuditDrawer } from './components/AuditDrawer';
import { SkippedModal } from './components/SkippedModal';
import { AboutModal } from './components/AboutModal';

type FilterTab = 'all' | 'filename' | 'content' | 'archive';
type SortKey = 'hits' | 'name' | 'size' | 'time';

interface ToastState {
  id: number;
  message: string;
  tone: 'info' | 'success' | 'error';
}

const ALL_CATEGORY_IDS = CATEGORIES.map((c) => c.id);

/* Windows 11 Fluent Toggle Switch */
const FluentToggle: React.FC<{ checked: boolean; onChange: (next: boolean) => void }> = ({
  checked,
  onChange,
}) => (
  <button
    type="button"
    role="switch"
    aria-checked={checked}
    onClick={() => onChange(!checked)}
    className={`relative inline-flex h-[20px] w-[38px] shrink-0 items-center rounded-full border transition-all ${
      checked
        ? 'bg-[#0067C0] border-[#0067C0] dark:bg-[#60CDFF] dark:border-[#60CDFF]'
        : 'bg-slate-200/80 border-slate-300/90 dark:bg-white/10 dark:border-white/20'
    }`}
  >
    <span
      className={`inline-block h-[13px] w-[13px] rounded-full shadow-sm transition-transform ${
        checked
          ? 'translate-x-[20px] bg-white dark:bg-[#0A1929]'
          : 'translate-x-[3px] bg-slate-500 dark:bg-slate-300'
      }`}
    />
  </button>
);

export default function App() {
  // Default: Light Windows 11 Fluent Mica theme
  const [darkMode, setDarkMode] = useState<boolean>(() => {
    const saved = localStorage.getItem('docpurger:fluentTheme');
    return saved ? saved === 'dark' : false;
  });

  useEffect(() => {
    document.documentElement.classList.toggle('dark', darkMode);
    localStorage.setItem('docpurger:fluentTheme', darkMode ? 'dark' : 'light');
  }, [darkMode]);

  const [sysInfo, setSysInfo] = useState<SystemInfo | null>(null);
  const [isMaximised, setIsMaximised] = useState(false);

  useEffect(() => {
    backend.getSystemInfo().then(setSysInfo).catch(() => {});
    backend.windowIsMaximised().then(setIsMaximised).catch(() => {});
    backend.syncWindowCorners();
    const onResize = () => {
      backend.windowIsMaximised().then(setIsMaximised).catch(() => {});
      backend.syncWindowCorners();
    };
    window.addEventListener('resize', onResize);
    return () => window.removeEventListener('resize', onResize);
  }, []);

  const handleToggleMaximise = useCallback(() => {
    backend.windowToggleMaximise();
    setTimeout(() => {
      backend.windowIsMaximised().then(setIsMaximised).catch(() => {});
      backend.syncWindowCorners();
    }, 80);
  }, []);

  // Multi-directory targets
  const [rootDirs, setRootDirs] = useState<string[]>(() => {
    try {
      const saved = JSON.parse(localStorage.getItem('docpurger:rootDirs') || '[]');
      if (Array.isArray(saved) && saved.length > 0) return saved;
    } catch {
      // ignore
    }
    const legacy = localStorage.getItem('docpurger:lastDir');
    return legacy ? [legacy] : [];
  });
  const [dirInput, setDirInput] = useState('');
  const [showDirInput, setShowDirInput] = useState(false);
  const rootDirsRef = useRef<string[]>(rootDirs);
  rootDirsRef.current = rootDirs;

  const [recentDirs, setRecentDirs] = useState<string[]>(() => {
    try {
      return JSON.parse(localStorage.getItem('docpurger:recentDirs') || '[]');
    } catch {
      return [];
    }
  });

  const [savedPrefs] = useState<Record<string, any>>(() => {
    try {
      return JSON.parse(localStorage.getItem('docpurger:prefs') || '{}') || {};
    } catch {
      return {};
    }
  });

  const searchInputRef = useRef<HTMLInputElement>(null);
  const [keywords, setKeywords] = useState<string[]>([]);
  const [keywordDraft, setKeywordDraft] = useState<string>('');
  const [excludeKeywords, setExcludeKeywords] = useState<string[]>([]);
  const [excludeKeywordDraft, setExcludeKeywordDraft] = useState<string>('');
  const [matchMode, setMatchMode] = useState<MatchMode>(() =>
    savedPrefs.matchMode === 'AND' ? 'AND' : 'OR'
  );
  const [scope, setScope] = useState<SearchScope>(() =>
    savedPrefs.scope === 'filename' || savedPrefs.scope === 'content' ? savedPrefs.scope : 'both'
  );
  const [caseSensitive, setCaseSensitive] = useState<boolean>(() => Boolean(savedPrefs.caseSensitive));
  const [includeArchives, setIncludeArchives] = useState<boolean>(() =>
    typeof savedPrefs.includeArchives === 'boolean' ? savedPrefs.includeArchives : true
  );
  const [includeHidden, setIncludeHidden] = useState<boolean>(() => Boolean(savedPrefs.includeHidden));
  const [maxFileSizeMB, setMaxFileSizeMB] = useState<number>(() =>
    typeof savedPrefs.maxFileSizeMB === 'number' && savedPrefs.maxFileSizeMB > 0
      ? savedPrefs.maxFileSizeMB
      : 100
  );
  const [selectedCategories, setSelectedCategories] = useState<string[]>(() => {
    if (Array.isArray(savedPrefs.selectedCategories)) {
      const valid = savedPrefs.selectedCategories.filter((c: string) => ALL_CATEGORY_IDS.includes(c));
      if (valid.length > 0) return valid;
    }
    return ALL_CATEGORY_IDS;
  });

  useEffect(() => {
    try {
      localStorage.setItem(
        'docpurger:prefs',
        JSON.stringify({
          matchMode,
          scope,
          caseSensitive,
          includeArchives,
          includeHidden,
          maxFileSizeMB,
          selectedCategories,
        })
      );
    } catch {
      // ignore
    }
  }, [matchMode, scope, caseSensitive, includeArchives, includeHidden, maxFileSizeMB, selectedCategories]);

  // Runtime state
  const [searching, setSearching] = useState(false);
  const [hasSearched, setHasSearched] = useState(false);
  const [results, setResults] = useState<SearchResult[]>([]);
  const [progress, setProgress] = useState<SearchProgress>({
    scannedFiles: 0,
    matchedCount: 0,
    skippedCount: 0,
    currentDir: '',
    elapsedMs: 0,
    running: false,
  });
  const [skippedFiles, setSkippedFiles] = useState<SkippedFile[]>([]);

  // Filtering / Selection / Active Inspector
  const [filterTab, setFilterTab] = useState<FilterTab>('all');
  const [sortBy, setSortBy] = useState<SortKey>('hits');
  const [quickFilter, setQuickFilter] = useState('');
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [activeId, setActiveId] = useState<string | null>(null);
  const anchorIdxRef = useRef<number | null>(null);

  // Modals & Drawers
  const [deleteModal, setDeleteModal] = useState<{ open: boolean; mode: DeleteMode; targets: SearchResult[] }>({
    open: false,
    mode: 'trash',
    targets: [],
  });
  const [quarantineDir, setQuarantineDir] = useState<string>(
    () => localStorage.getItem('docpurger:quarantineDir') || ''
  );
  const [auditOpen, setAuditOpen] = useState(false);
  const [skippedOpen, setSkippedOpen] = useState(false);
  const [aboutOpen, setAboutOpen] = useState(false);
  const [toast, setToast] = useState<ToastState | null>(null);

  const notify = useCallback((message: string, tone: 'info' | 'success' | 'error' = 'info') => {
    const id = Date.now();
    setToast({ id, message, tone });
    setTimeout(() => {
      setToast((prev) => (prev?.id === id ? null : prev));
    }, 3400);
  }, []);

  const persistRootDirs = useCallback((next: string[]) => {
    localStorage.setItem('docpurger:rootDirs', JSON.stringify(next));
    localStorage.removeItem('docpurger:lastDir');
    if (next.length > 0) {
      setRecentDirs((prev) => {
        const merged = [...next, ...prev.filter((d) => !next.includes(d))].slice(0, 8);
        localStorage.setItem('docpurger:recentDirs', JSON.stringify(merged));
        return merged;
      });
    }
  }, []);

  const addDirectory = useCallback(
    (raw: string) => {
      const cleaned = raw.trim();
      if (!cleaned) return;
      setRootDirs((prev) => {
        if (prev.includes(cleaned)) {
          notify('该目录已在搜索列表中', 'info');
          return prev;
        }
        const next = [...prev, cleaned];
        persistRootDirs(next);
        return next;
      });
    },
    [notify, persistRootDirs]
  );

  const removeDirectory = (dir: string) => {
    setRootDirs((prev) => {
      const next = prev.filter((d) => d !== dir);
      persistRootDirs(next);
      return next;
    });
  };

  const clearDirectories = useCallback(() => {
    setRootDirs([]);
    setDirInput('');
    setShowDirInput(false);
    persistRootDirs([]);
    notify('已清空待搜索目录', 'info');
  }, [persistRootDirs, notify]);

  const clearExcludeKeywords = useCallback(() => {
    setExcludeKeywords([]);
    setExcludeKeywordDraft('');
  }, []);

  const canClearAll =
    rootDirs.length > 0 ||
    dirInput.trim().length > 0 ||
    keywords.length > 0 ||
    keywordDraft.trim().length > 0 ||
    excludeKeywords.length > 0 ||
    excludeKeywordDraft.trim().length > 0;

  const handleClearAll = useCallback(() => {
    setRootDirs([]);
    setDirInput('');
    setShowDirInput(false);
    persistRootDirs([]);
    setKeywords([]);
    setKeywordDraft('');
    setExcludeKeywords([]);
    setExcludeKeywordDraft('');
    notify('已清空待搜索目录和关键词', 'info');
  }, [persistRootDirs, notify]);

  const handlePickAddDir = async () => {
    const defaultDir = rootDirs[rootDirs.length - 1] || sysInfo?.homeDir || '';
    const picked = await backend.selectDirectory(defaultDir);
    if (picked) {
      addDirectory(picked);
    }
  };

  const handlePasteDirectories = useCallback(async () => {
    try {
      const raw = (await backend.clipboardGetText()).trim();
      if (!raw) {
        setShowDirInput(true);
        notify('剪贴板为空，已展开路径输入框', 'info');
        return;
      }
      const candidates = raw
        .split(/\r?\n+/)
        .map((line) => line.trim().replace(/^["']|["']$/g, '').trim())
        .filter(Boolean);
      if (candidates.length === 0) {
        setShowDirInput(true);
        return;
      }
      const validDirs = await backend.filterDirectories(candidates);
      if (validDirs.length === 0) {
        setDirInput(candidates[0]);
        setShowDirInput(true);
        notify('剪贴板内容不是有效文件夹路径，可在此手动修改', 'info');
        return;
      }
      const existing = rootDirsRef.current;
      const added = validDirs.filter((d) => !existing.includes(d));
      const ignoredCount = candidates.length - validDirs.length;
      if (added.length > 0) {
        const next = [...existing, ...added];
        setRootDirs(next);
        persistRootDirs(next);
        if (ignoredCount > 0) {
          notify(`已粘贴添加 ${added.length} 个目录，已忽略 ${ignoredCount} 条无效路径`, 'success');
        } else if (added.length === 1) {
          notify(`已粘贴添加目录：${leafName(added[0])}`, 'success');
        } else {
          notify(`已粘贴添加 ${added.length} 个目录`, 'success');
        }
      } else {
        notify('剪贴板中的目录已在搜索列表中', 'info');
      }
    } catch {
      setShowDirInput(true);
      notify('无法直接读取剪贴板，请在此手动粘贴路径', 'info');
    }
  }, [notify, persistRootDirs]);

  // Native OS directory drag-and-drop onto the directory card / dashed box
  useEffect(() => {
    return backend.onFileDrop(async (paths) => {
      if (!paths || paths.length === 0) return;
      try {
        const validDirs = await backend.filterDirectories(paths);
        const ignoredCount = paths.length - validDirs.length;
        if (validDirs.length === 0) {
          notify('请拖入文件夹，不支持直接拖入文件', 'error');
          return;
        }
        const existing = rootDirsRef.current;
        const added = validDirs.filter((d) => !existing.includes(d));
        if (added.length > 0) {
          const next = [...existing, ...added];
          setRootDirs(next);
          persistRootDirs(next);
          if (ignoredCount > 0) {
            notify(`已添加 ${added.length} 个目录，已忽略 ${ignoredCount} 个非目录项`, 'success');
          } else if (added.length === 1) {
            notify(`已添加搜索目录：${leafName(added[0])}`, 'success');
          } else {
            notify(`已添加 ${added.length} 个搜索目录`, 'success');
          }
        } else if (ignoredCount > 0) {
          notify(`所拖目录已在列表中，已忽略 ${ignoredCount} 个非目录项`, 'info');
        } else {
          notify('所拖入的目录已在搜索列表中', 'info');
        }
      } catch (err: any) {
        notify(`识别拖入目录失败：${err?.message || err}`, 'error');
      }
    }, true);
  }, [notify, persistRootDirs]);

  // Stream match batching
  const pendingMatchesRef = useRef<SearchResult[]>([]);
  const flushTimerRef = useRef<number | null>(null);

  const flushMatches = useCallback(() => {
    if (pendingMatchesRef.current.length === 0) return;
    const batch = pendingMatchesRef.current;
    pendingMatchesRef.current = [];
    setResults((prev) => [...prev, ...batch]);
  }, []);

  useEffect(() => {
    const offMatch = backend.onEvent<SearchResult>('search:match', (res) => {
      pendingMatchesRef.current.push(res);
      if (flushTimerRef.current === null) {
        flushTimerRef.current = window.setTimeout(() => {
          flushTimerRef.current = null;
          flushMatches();
        }, 60);
      }
    });

    const offProgress = backend.onEvent<SearchProgress>('search:progress', (prog) => {
      setProgress(prog);
    });

    const offDone = backend.onEvent<SearchSummary>('search:done', (summary) => {
      if (flushTimerRef.current !== null) {
        clearTimeout(flushTimerRef.current);
        flushTimerRef.current = null;
      }
      flushMatches();
      setSearching(false);
      setSkippedFiles(summary.skippedFiles || []);
      setProgress((prev) => ({
        ...prev,
        scannedFiles: summary.scannedFiles,
        matchedCount: summary.matchedCount,
        skippedCount: summary.skippedCount,
        elapsedMs: summary.elapsedMs,
        running: false,
      }));
      if (summary.error) {
        notify(summary.error, 'error');
      } else if (summary.cancelled) {
        notify(`已中止扫描，已检出 ${summary.matchedCount} 个匹配文档`, 'info');
      }
    });

    return () => {
      offMatch();
      offProgress();
      offDone();
      if (flushTimerRef.current !== null) clearTimeout(flushTimerRef.current);
    };
  }, [flushMatches, notify]);

  const handleStartSearch = async (overrideKeywords?: string[]) => {
    if (rootDirs.length === 0) {
      notify('请先在左侧添加至少一个搜索目录', 'error');
      return;
    }
    if (selectedCategories.length === 0) {
      notify('请先在左侧勾选至少一种文档类型', 'error');
      return;
    }
    if (selectedCategories.length === 0) {
      notify('请先在左侧勾选至少一种文档类型', 'error');
      return;
    }

    let finalKeywords = overrideKeywords ? [...overrideKeywords] : [...keywords];
    if (!overrideKeywords && keywordDraft.trim()) {
      const extra = keywordDraft
        .split(/[,，;；\n]+/)
        .map((s) => s.trim())
        .filter(Boolean);
      for (const tok of extra) {
        if (!finalKeywords.includes(tok)) finalKeywords.push(tok);
      }
      setKeywords(finalKeywords);
      setKeywordDraft('');
    }

    let finalExcludeKeywords = [...excludeKeywords];
    if (excludeKeywordDraft.trim()) {
      const extraExclude = excludeKeywordDraft
        .split(/[,，;；\n]+/)
        .map((s) => s.trim())
        .filter(Boolean);
      for (const tok of extraExclude) {
        if (!finalExcludeKeywords.includes(tok)) finalExcludeKeywords.push(tok);
      }
      setExcludeKeywords(finalExcludeKeywords);
      setExcludeKeywordDraft('');
    }

    if (finalKeywords.length === 0) {
      notify('请输入至少一个搜索关键词', 'error');
      return;
    }

    persistRootDirs(rootDirs);
    pendingMatchesRef.current = [];
    setResults([]);
    setSelectedIds(new Set());
    setActiveId(null);
    anchorIdxRef.current = null;
    setSkippedFiles([]);
    setHasSearched(true);
    setSearching(true);
    setProgress({
      scannedFiles: 0,
      matchedCount: 0,
      skippedCount: 0,
      currentDir: rootDirs[0],
      elapsedMs: 0,
      running: true,
    });

    const opts: SearchOptions = {
      rootDir: rootDirs[0],
      rootDirs,
      keywords: finalKeywords,
      excludeKeywords: finalExcludeKeywords,
      mode: matchMode,
      scope,
      caseSensitive,
      includeArchives,
      categories: selectedCategories,
      maxFileSizeMB,
      includeHidden,
      maxArchiveDepth: 3,
    };

    try {
      await backend.startSearch(opts);
    } catch (err: any) {
      setSearching(false);
      notify(`启动搜索失败：${err?.message || err}`, 'error');
    }
  };

  const visibleResults = useMemo(() => {
    const q = quickFilter.trim().toLowerCase();
    const list = results.filter((r) => {
      if (filterTab === 'filename' && !r.matchInFilename) return false;
      if (filterTab === 'content' && !r.matchInContent) return false;
      if (filterTab === 'archive' && !r.isArchiveEntry && r.category !== 'archive') return false;
      if (q) {
        const hay = `${r.fileName} ${r.relativePath} ${r.innerPath} ${r.rootLabel}`.toLowerCase();
        if (!hay.includes(q)) return false;
      }
      return true;
    });

    list.sort((a, b) => {
      switch (sortBy) {
        case 'name':
          return a.fileName.localeCompare(b.fileName, 'zh-CN');
        case 'size':
          return b.diskFileSize - a.diskFileSize;
        case 'time':
          return b.modTimeUnix - a.modTimeUnix;
        case 'hits':
        default:
          if (b.matchCount !== a.matchCount) return b.matchCount - a.matchCount;
          return a.relativePath.localeCompare(b.relativePath, 'zh-CN');
      }
    });
    return list;
  }, [results, filterTab, quickFilter, sortBy]);

  const activeItem = useMemo(() => {
    if (visibleResults.length === 0 || activeId === '') return null;
    return visibleResults.find((r) => r.id === activeId) || visibleResults[0];
  }, [visibleResults, activeId]);

  // Keyboard shortcuts: Esc close modals/selection, Ctrl+K focus search, Ctrl+Enter search, Ctrl+A select all
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        if (deleteModal.open) {
          setDeleteModal({ open: false, mode: 'trash', targets: [] });
          return;
        }
        if (aboutOpen) {
          setAboutOpen(false);
          return;
        }
        if (auditOpen) {
          setAuditOpen(false);
          return;
        }
        if (skippedOpen) {
          setSkippedOpen(false);
          return;
        }
      }

      if (deleteModal.open || aboutOpen || auditOpen || skippedOpen) return;

      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        searchInputRef.current?.focus();
        return;
      }

      if ((e.ctrlKey || e.metaKey) && e.key === 'Enter' && !searching) {
        e.preventDefault();
        handleStartSearch();
        return;
      }

      const tag = (e.target as HTMLElement)?.tagName;
      if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return;

      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'a' && visibleResults.length > 0) {
        e.preventDefault();
        setSelectedIds(new Set(visibleResults.map((r) => r.id)));
      } else if (e.key === 'Escape') {
        if (selectedIds.size > 0) {
          setSelectedIds(new Set());
        } else if (activeItem) {
          setActiveId('');
        }
      }
    };
    window.addEventListener('keydown', onKeyDown);
    return () => window.removeEventListener('keydown', onKeyDown);
  }, [visibleResults, selectedIds.size, activeItem, searching, deleteModal.open, aboutOpen, auditOpen, skippedOpen]);

  const selectedItems = useMemo(
    () => results.filter((r) => selectedIds.has(r.id)),
    [results, selectedIds]
  );

  const selectionStats = useMemo(() => {
    const byPath = new Map<string, number>();
    for (const item of selectedItems) {
      if (!byPath.has(item.filePath)) {
        byPath.set(item.filePath, item.diskFileSize || item.size);
      }
    }
    let totalBytes = 0;
    byPath.forEach((sz) => (totalBytes += sz));
    return {
      selectedCount: selectedItems.length,
      physicalCount: byPath.size,
      totalBytes,
    };
  }, [selectedItems]);

  // Range selection helper used by BOTH row Shift+Click and checkbox Shift+Click
  const applyRangeSelect = (idx: number, targetSelectState: boolean) => {
    const anchor = anchorIdxRef.current !== null ? anchorIdxRef.current : idx;
    const start = Math.min(anchor, idx);
    const end = Math.max(anchor, idx);
    setSelectedIds((prev) => {
      const next = new Set(prev);
      for (let i = start; i <= end; i++) {
        const row = visibleResults[i];
        if (!row) continue;
        if (targetSelectState) next.add(row.id);
        else next.delete(row.id);
      }
      return next;
    });
  };

  const handleCheckboxClick = (e: React.MouseEvent, idx: number, id: string) => {
    e.stopPropagation();
    setActiveId(id);
    if (e.shiftKey && anchorIdxRef.current !== null) {
      const shouldSelect = !selectedIds.has(id);
      applyRangeSelect(idx, shouldSelect);
    } else {
      setSelectedIds((prev) => {
        const next = new Set(prev);
        if (next.has(id)) next.delete(id);
        else next.add(id);
        return next;
      });
      anchorIdxRef.current = idx;
    }
  };

  // Full-row click supports Shift+Click range multi-select & Ctrl/Cmd+Click toggle (Windows Explorer feel)
  const handleRowClick = (e: React.MouseEvent, idx: number, id: string) => {
    setActiveId(id);
    if (e.shiftKey) {
      e.preventDefault();
      window.getSelection()?.removeAllRanges();
      if (anchorIdxRef.current === null) {
        anchorIdxRef.current = idx;
        setSelectedIds((prev) => new Set(prev).add(id));
      } else {
        // Select continuous range from anchor to clicked row
        applyRangeSelect(idx, true);
      }
      return;
    }
    if (e.ctrlKey || e.metaKey) {
      e.preventDefault();
      setSelectedIds((prev) => {
        const next = new Set(prev);
        if (next.has(id)) next.delete(id);
        else next.add(id);
        return next;
      });
      anchorIdxRef.current = idx;
      return;
    }
    // Normal single click: preview in inspector & set Shift-anchor to this row
    anchorIdxRef.current = idx;
  };

  const selectAllVisible = () => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      visibleResults.forEach((r) => next.add(r.id));
      return next;
    });
  };

  const invertVisibleSelection = () => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      visibleResults.forEach((r) => {
        if (next.has(r.id)) next.delete(r.id);
        else next.add(r.id);
      });
      return next;
    });
  };

  const clearSelection = () => setSelectedIds(new Set());

  const openDeleteDialog = (mode: DeleteMode, targets: SearchResult[]) => {
    if (targets.length === 0) return;
    setDeleteModal({ open: true, mode, targets });
  };

  const executeDelete = async () => {
    const { mode, targets } = deleteModal;
    const uniquePaths = Array.from(new Set(targets.map((t) => t.filePath)));

    if (mode === 'quarantine' && quarantineDir) {
      localStorage.setItem('docpurger:quarantineDir', quarantineDir);
    }

    const resp = await backend.deleteFiles({
      paths: uniquePaths,
      mode,
      quarantineDir,
      rootDir: rootDirs[0] || '',
      rootDirs,
      keywords,
    });

    const removedPaths = new Set(resp.results.filter((r) => r.success).map((r) => r.path));
    if (removedPaths.size > 0) {
      setResults((prev) => prev.filter((item) => !removedPaths.has(item.filePath)));
      setSelectedIds((prev) => {
        const next = new Set(prev);
        results.forEach((item) => {
          if (removedPaths.has(item.filePath)) next.delete(item.id);
        });
        return next;
      });
    }

    setDeleteModal({ open: false, mode: 'trash', targets: [] });

    const actionWord =
      mode === 'trash' ? '已移入回收站' : mode === 'quarantine' ? '已移至隔离目录' : '已彻底删除';
    if (resp.failed === 0) {
      notify(`${actionWord} ${resp.succeeded} 个文件，释放 ${formatBytes(resp.freedBytes)}`, 'success');
    } else {
      notify(`成功 ${resp.succeeded} 个，失败 ${resp.failed} 个`, 'error');
    }
  };

  const toggleCategory = (catId: string) => {
    setSelectedCategories((prev) => {
      if (prev.includes(catId)) {
        return prev.filter((c) => c !== catId);
      }
      return [...prev, catId];
    });
  };

  const copyPath = (p: string) => {
    navigator.clipboard?.writeText(p).then(() => notify('已复制完整路径', 'info')).catch(() => {});
  };

  const counts = useMemo(() => {
    let filename = 0;
    let content = 0;
    let archive = 0;
    const byCat: Record<string, number> = {};
    for (const r of results) {
      if (r.matchInFilename) filename++;
      if (r.matchInContent) content++;
      if (r.isArchiveEntry || r.category === 'archive') archive++;
      const statCat = ALL_CATEGORY_IDS.includes(r.category)
        ? r.category
        : r.isArchiveEntry
        ? 'archive'
        : r.category;
      byCat[statCat] = (byCat[statCat] || 0) + 1;
    }
    return { all: results.length, filename, content, archive, byCat };
  }, [results]);

  const allVisibleSelected =
    visibleResults.length > 0 && visibleResults.every((r) => selectedIds.has(r.id));
  const someVisibleSelected =
    visibleResults.some((r) => selectedIds.has(r.id)) && !allVisibleSelected;

  // Dynamic plain-language hint explaining "同时包含" vs "包含任意"
  const effectivePreviewKeywords = useMemo(() => {
    const list = [...keywords];
    if (keywordDraft.trim() && !list.includes(keywordDraft.trim())) {
      list.push(keywordDraft.trim());
    }
    return list;
  }, [keywords, keywordDraft]);

  const ruleHintText = useMemo(() => {
    if (effectivePreviewKeywords.length < 2) return null;
    const sample = effectivePreviewKeywords.slice(0, 3).map((k) => `「${k}」`);
    const extra = effectivePreviewKeywords.length > 3 ? ` 等 ${effectivePreviewKeywords.length} 个词` : '';
    if (matchMode === 'AND') {
      return `匹配规则：文档需同时包含 ${sample.join(' 和 ')}${extra}`;
    }
    return `匹配规则：文档包含 ${sample.join(' 或 ')}${extra} 中任意一个即可`;
  }, [effectivePreviewKeywords, matchMode]);

  const leafName = (p: string) => p.split(/[\\/]/).filter(Boolean).pop() || p;
  const archiveCategorySelected = selectedCategories.includes('archive');

  return (
    <div
      className={`mica-backdrop h-screen w-screen flex flex-col overflow-hidden text-slate-800 dark:text-slate-100 select-none ${
        isMaximised
          ? ''
          : 'rounded-[10px] border border-slate-300/65 dark:border-white/12'
      }`}
    >
      {/* ================= INTEGRATED FRAMELESS FLUENT 2 TITLEBAR ================= */}
      <div
        onDoubleClick={handleToggleMaximise}
        className="wails-drag h-10 shrink-0 px-3.5 flex items-center justify-between border-b border-slate-200/60 dark:border-white/[0.07] bg-white/40 dark:bg-black/20 backdrop-blur-md"
      >
        <div className="flex items-center gap-2.5 min-w-0">
          <AppLogo className="w-6 h-6 shrink-0 drop-shadow-sm" />
          <div className="flex items-center gap-2 min-w-0">
            <span className="text-[13.5px] font-bold tracking-tight text-slate-900 dark:text-white">
              文档清匣
            </span>
            <span className="text-[11px] font-mono text-slate-400 dark:text-slate-400">
              DocPurger v{sysInfo?.appVersion || '1.3.0'}
            </span>
          </div>
        </div>

        <div className="wails-no-drag flex items-center gap-1">
          <button
            type="button"
            onClick={() => setDarkMode((v) => !v)}
            className="w-8 h-7 inline-flex items-center justify-center rounded-md text-slate-500 dark:text-slate-400 hover:text-[#0067C0] dark:hover:text-[#60CDFF] hover:bg-black/5 dark:hover:bg-white/10 transition-colors"
            title={darkMode ? '切换为浅色 Mica 云母' : '切换为深海云母暗色'}
          >
            {darkMode ? <Sun className="w-3.5 h-3.5" /> : <Moon className="w-3.5 h-3.5" />}
          </button>
          <span className="h-3.5 w-px bg-slate-300/70 dark:bg-white/15 mx-1" />
          <button
            type="button"
            onClick={() => backend.windowMinimise()}
            className="w-8 h-7 inline-flex items-center justify-center rounded-md text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-black/5 dark:hover:bg-white/10 transition-colors"
            title="最小化"
          >
            <Minus className="w-3.5 h-3.5" />
          </button>
          <button
            type="button"
            onClick={handleToggleMaximise}
            className="w-8 h-7 inline-flex items-center justify-center rounded-md text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white hover:bg-black/5 dark:hover:bg-white/10 transition-colors"
            title={isMaximised ? '向下还原' : '最大化'}
          >
            {isMaximised ? <Minimize2 className="w-3.5 h-3.5" /> : <Maximize2 className="w-3.5 h-3.5" />}
          </button>
          <button
            type="button"
            onClick={() => backend.quit()}
            className="w-8 h-7 inline-flex items-center justify-center rounded-md text-slate-500 dark:text-slate-400 hover:bg-[#E81123] hover:text-white transition-colors"
            title="关闭"
          >
            <X className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      {/* ================= MAIN THREE-COLUMN WORKSPACE ================= */}
      <div className="flex-1 min-h-0 flex overflow-hidden">
      {/* ================= LEFT MICA NAVIGATION RAIL ================= */}
      <aside className="w-[268px] shrink-0 h-full flex flex-col px-3.5 py-2.5 gap-2 overflow-hidden">
        {/* Scrollable Left Cards */}
        <div className="flex-1 min-h-0 overflow-y-auto space-y-2 pr-0.5">
          {/* Card 1: Multi-Directory Selector */}
          <div className="mica-card dir-drop-zone rounded-xl p-3 space-y-2 transition-all">
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-1.5 min-w-0">
                <FolderSearch className="w-3.5 h-3.5 text-[#0067C0] dark:text-[#60CDFF] shrink-0" />
                <span className="text-xs font-semibold text-slate-800 dark:text-slate-100 whitespace-nowrap">
                  搜索目录
                </span>
                {rootDirs.length > 0 && (
                  <span className="px-1.5 py-[1px] rounded-full text-[10px] font-mono font-semibold bg-[#0067C0]/12 text-[#0067C0] dark:bg-[#60CDFF]/15 dark:text-[#60CDFF] shrink-0">
                    {rootDirs.length}
                  </span>
                )}
              </div>
              {rootDirs.length > 0 && (
                <button
                  type="button"
                  onClick={clearDirectories}
                  className="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded-md text-[11px] text-slate-400 hover:text-rose-500 hover:bg-rose-500/10 transition-colors whitespace-nowrap shrink-0"
                  title="清空全部待搜索目录"
                >
                  <Trash2 className="w-3 h-3" />
                  清空
                </button>
              )}
            </div>

            {showDirInput && (
              <div className="flex gap-1.5 animate-fade-in">
                <input
                  type="text"
                  value={dirInput}
                  onChange={(e) => setDirInput(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && dirInput.trim()) {
                      addDirectory(dirInput);
                      setDirInput('');
                      setShowDirInput(false);
                    }
                  }}
                  placeholder="粘贴目录路径后回车…"
                  className="flex-1 min-w-0 px-2 py-1 rounded-md text-[11px] font-mono bg-white/90 dark:bg-black/25 border border-slate-200 dark:border-white/10 outline-none focus:border-[#0067C0]"
                />
                <button
                  type="button"
                  onClick={() => {
                    if (dirInput.trim()) {
                      addDirectory(dirInput);
                      setDirInput('');
                      setShowDirInput(false);
                    }
                  }}
                  className="px-2 py-1 rounded-md text-[11px] font-medium fluent-btn-primary whitespace-nowrap shrink-0"
                >
                  添加
                </button>
              </div>
            )}

            {rootDirs.length === 0 ? (
              <button
                type="button"
                onClick={handlePickAddDir}
                className="dir-drop-dashed w-full py-2.5 px-2.5 rounded-lg border border-dashed border-slate-300 dark:border-white/15 hover:border-[#0067C0] dark:hover:border-[#60CDFF] text-center text-[11.5px] text-slate-500 dark:text-slate-400 hover:text-[#0067C0] dark:hover:text-[#60CDFF] transition-colors"
              >
                点击或拖入一个或多个目标目录
              </button>
            ) : (
              <div className="space-y-1 max-h-28 overflow-y-auto">
                {rootDirs.map((dir) => (
                  <div
                    key={dir}
                    title={dir}
                    className="group flex items-center gap-2 px-2 py-1.5 rounded-lg bg-white/80 dark:bg-white/[0.05] border border-slate-200/70 dark:border-white/[0.06]"
                  >
                    <Folder className="w-3.5 h-3.5 text-[#E8A838] fill-[#F5C451]/60 shrink-0" />
                    <div className="min-w-0 flex-1">
                      <div className="text-[11.5px] font-medium text-slate-800 dark:text-slate-100 truncate leading-tight">
                        {leafName(dir)}
                      </div>
                      <div className="text-[10px] font-mono text-slate-400 dark:text-slate-400 truncate leading-tight mt-0.5">
                        {dir}
                      </div>
                    </div>
                    <button
                      type="button"
                      onClick={() => removeDirectory(dir)}
                      className="p-0.5 rounded text-slate-400 hover:text-rose-500 hover:bg-rose-500/10 opacity-60 group-hover:opacity-100 transition-all shrink-0"
                      title="移除此目录"
                    >
                      <X className="w-3 h-3" />
                    </button>
                  </div>
                ))}
              </div>
            )}

            {/* Dedicated non-wrapping action row: 添加目录 + 粘贴路径 */}
            <div className="flex items-center gap-1.5">
              <button
                type="button"
                onClick={handlePickAddDir}
                className="flex-1 inline-flex items-center justify-center gap-1 py-1.5 px-2 rounded-lg text-[11.5px] font-medium bg-[#0067C0]/10 hover:bg-[#0067C0]/18 text-[#0067C0] dark:bg-[#60CDFF]/15 dark:hover:bg-[#60CDFF]/25 dark:text-[#60CDFF] transition-colors whitespace-nowrap"
                title="选择待搜索的文件夹"
              >
                <FolderPlus className="w-3.5 h-3.5 shrink-0" />
                添加目录
              </button>
              <button
                type="button"
                onClick={handlePasteDirectories}
                className="inline-flex items-center justify-center gap-1 py-1.5 px-2.5 rounded-lg text-[11.5px] font-medium bg-slate-200/65 hover:bg-slate-200 text-slate-700 dark:bg-white/10 dark:hover:bg-white/15 dark:text-slate-200 transition-colors whitespace-nowrap shrink-0"
                title="从剪贴板一键粘贴目录路径（支持多行及带引号路径）"
              >
                <ClipboardPaste className="w-3.5 h-3.5 shrink-0" />
                粘贴
              </button>
            </div>

            {/* Quick recent dirs not currently selected */}
            {recentDirs.filter((d) => !rootDirs.includes(d)).length > 0 && (
              <div className="pt-1 border-t border-slate-200/60 dark:border-white/[0.06] space-y-1">
                <div className="text-[10px] text-slate-400">最近使用（点击追加）</div>
                <div className="flex flex-wrap gap-1">
                  {recentDirs
                    .filter((d) => !rootDirs.includes(d))
                    .slice(0, 3)
                    .map((dir) => (
                      <button
                        key={dir}
                        type="button"
                        onClick={() => addDirectory(dir)}
                        title={dir}
                        className="max-w-full inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[10.5px] bg-slate-100/80 dark:bg-white/[0.06] hover:bg-[#0067C0]/12 hover:text-[#0067C0] dark:hover:text-[#60CDFF] text-slate-600 dark:text-slate-300 truncate transition-colors"
                      >
                        <Plus className="w-2.5 h-2.5 shrink-0" />
                        <span className="truncate">{leafName(dir)}</span>
                      </button>
                    ))}
                </div>
              </div>
            )}
          </div>

          {/* Card 2: Fluent Navigation Document Type Filter (unified mica-card matching Card 1 & Card 3) */}
          <div className="mica-card rounded-xl p-3 space-y-1.5">
            <div className="flex items-center justify-between">
              <span className="inline-flex items-center gap-1.5 text-xs font-semibold text-slate-800 dark:text-slate-100">
                <Layers className="w-3.5 h-3.5 text-[#0067C0] dark:text-[#60CDFF] shrink-0" />
                文档类型
              </span>
              {selectedCategories.length === ALL_CATEGORY_IDS.length ? (
                <button
                  type="button"
                  onClick={() => setSelectedCategories([])}
                  className="text-[11px] text-slate-400 hover:text-rose-500 transition-colors"
                  title="取消勾选全部文档类型，方便快速单选"
                >
                  清空
                </button>
              ) : (
                <button
                  type="button"
                  onClick={() => setSelectedCategories(ALL_CATEGORY_IDS)}
                  className="text-[11px] font-medium text-[#0067C0] dark:text-[#60CDFF] hover:underline"
                >
                  全选
                </button>
              )}
            </div>

            <div className="space-y-0.5">
              {CATEGORIES.map((cat) => {
                const active = selectedCategories.includes(cat.id);
                const hitCount = counts.byCat[cat.id] || 0;
                return (
                  <button
                    key={cat.id}
                    type="button"
                    onClick={() => toggleCategory(cat.id)}
                    title={cat.exts}
                    className={`relative w-full flex items-center justify-between pl-2.5 pr-2 py-1 rounded-lg text-[12px] transition-all ${
                      active
                        ? 'bg-[#0067C0]/[0.07] dark:bg-[#60CDFF]/[0.11] text-slate-900 dark:text-white font-medium'
                        : 'text-slate-400 dark:text-slate-500 hover:bg-slate-200/45 dark:hover:bg-white/[0.04]'
                    }`}
                  >
                    {/* Signature Windows 11 Fluent 3px vertical accent pill */}
                    {active && (
                      <span className="absolute left-0 top-1/2 -translate-y-1/2 w-[3px] h-3.5 rounded-full bg-[#0067C0] dark:bg-[#60CDFF]" />
                    )}
                    <span className="flex items-center gap-2 min-w-0">
                      <FluentFileIcon
                        category={cat.id}
                        className={`w-[18px] h-[18px] shrink-0 transition-opacity ${active ? 'opacity-100' : 'opacity-40 grayscale'}`}
                      />
                      <span className="truncate">{cat.label}</span>
                    </span>
                    <span className="flex items-center gap-1.5 shrink-0">
                      {hitCount > 0 ? (
                        <span
                          title={`当前检索结果中该类型共命中 ${hitCount} 个文件`}
                          className="px-1.5 py-[1px] rounded-full text-[10.5px] font-medium tabular-nums bg-[#0067C0]/12 text-[#0067C0] dark:bg-[#60CDFF]/15 dark:text-[#60CDFF] whitespace-nowrap"
                        >
                          命中 {hitCount}
                        </span>
                      ) : (
                        <span className="text-[11px] font-normal text-slate-400 dark:text-slate-500 whitespace-nowrap">
                          {cat.extHint}
                        </span>
                      )}
                    </span>
                  </button>
                );
              })}
            </div>
            {selectedCategories.length === 0 && (
              <p className="pt-0.5 text-[10.5px] leading-snug text-amber-700 dark:text-amber-300">
                请至少勾选一种要检索的文档类型
              </p>
            )}
          </div>

          {/* Card 3: Settings Card (Fluent Toggles & Scope) */}
          <div className="mica-card rounded-xl p-3 space-y-2.5">
            <div className="inline-flex items-center gap-1.5 text-xs font-semibold text-slate-800 dark:text-slate-100">
              <SlidersHorizontal className="w-3.5 h-3.5 text-[#0067C0] dark:text-[#60CDFF] shrink-0" />
              检索设置
            </div>

            {/* Scope */}
            <div className="space-y-1">
              <div className="flex items-center justify-between">
                <span className="text-[11px] font-medium text-slate-600 dark:text-slate-300">搜索范围</span>
              </div>
              <div className="flex p-0.5 rounded-lg bg-slate-200/65 dark:bg-black/25">
                {(
                  [
                    {
                      id: 'both',
                      label: '文件名+内容',
                      tip: '同时搜索文件名和文档内部文字内容',
                      grow: 'flex-[1.35]',
                    },
                    {
                      id: 'filename',
                      label: '仅文件名',
                      tip: '只匹配文件名称，不读取文件内容（速度最快）',
                      grow: 'flex-1',
                    },
                    {
                      id: 'content',
                      label: '仅内容',
                      tip: '只搜索文档内部文字内容，忽略文件名',
                      grow: 'flex-1',
                    },
                  ] as { id: SearchScope; label: string; tip: string; grow: string }[]
                ).map((opt) => (
                  <button
                    key={opt.id}
                    type="button"
                    title={opt.tip}
                    onClick={() => setScope(opt.id)}
                    className={`${opt.grow} py-1 px-1 rounded-[6px] text-[11px] font-medium whitespace-nowrap transition-all ${
                      scope === opt.id
                        ? 'bg-white dark:bg-white/15 text-[#0067C0] dark:text-white shadow-sm'
                        : 'text-slate-500 dark:text-slate-400 hover:text-slate-800 dark:hover:text-slate-200'
                    }`}
                  >
                    {opt.label}
                  </button>
                ))}
              </div>
              <p className="text-[10.5px] leading-snug text-slate-400 dark:text-slate-500">
                {scope === 'both'
                  ? '既查文件名，也读文档正文（推荐）'
                  : scope === 'filename'
                  ? '只匹配文件名称，不读正文（极速）'
                  : '只搜索文档内部文字，忽略文件名'}
              </p>
            </div>

            <div className="space-y-2 text-xs text-slate-800 dark:text-slate-100">
              <div className="space-y-1">
                <div className="flex items-center justify-between">
                  <span className={`font-medium ${!archiveCategorySelected ? 'text-slate-400 dark:text-slate-500' : ''}`}>
                    穿透压缩文件
                  </span>
                  <FluentToggle checked={includeArchives} onChange={setIncludeArchives} />
                </div>
                {includeArchives && !archiveCategorySelected ? (
                  <div className="px-2 py-1 rounded-md bg-amber-500/12 border border-amber-500/25 text-[10.5px] text-amber-800 dark:text-amber-300 leading-snug animate-fade-in">
                    未勾选「压缩文件」，此开关当前不生效
                  </div>
                ) : includeArchives ? (
                  <p className="text-[10.5px] leading-snug text-slate-400 dark:text-slate-500">
                    免解压直读 zip/rar/7z/tar.gz 内文档
                  </p>
                ) : null}
              </div>
              <div className="flex items-center justify-between">
                <span className="font-medium">区分大小写</span>
                <FluentToggle checked={caseSensitive} onChange={setCaseSensitive} />
              </div>
              <div className="flex items-center justify-between">
                <span className="font-medium">含隐藏文件</span>
                <FluentToggle checked={includeHidden} onChange={setIncludeHidden} />
              </div>
              <div className="flex items-center justify-between pt-0.5">
                <span className="font-medium">单文件上限</span>
                <select
                  value={maxFileSizeMB}
                  onChange={(e) => setMaxFileSizeMB(Number(e.target.value))}
                  className="px-2 py-0.5 rounded-md text-[11px] font-mono bg-white/90 dark:bg-black/30 border border-slate-200 dark:border-white/10 text-slate-700 dark:text-slate-200 outline-none"
                >
                  <option value={20}>20 MB</option>
                  <option value={50}>50 MB</option>
                  <option value={100}>100 MB</option>
                  <option value={300}>300 MB</option>
                </select>
              </div>
            </div>

            <div className="pt-1 space-y-1">
              <div className="flex items-center justify-between">
                <span className="text-[11px] text-slate-500 dark:text-slate-400">排除关键词</span>
                {(excludeKeywords.length > 0 || excludeKeywordDraft.length > 0) && (
                  <button
                    type="button"
                    onClick={clearExcludeKeywords}
                    className="text-[11px] text-slate-400 hover:text-rose-500 transition-colors"
                    title="清空排除关键词"
                  >
                    清空
                  </button>
                )}
              </div>
              <KeywordInput
                values={excludeKeywords}
                onChange={setExcludeKeywords}
                draft={excludeKeywordDraft}
                onDraftChange={setExcludeKeywordDraft}
                placeholder="不含这些词，回车添加…"
                variant="sidebar-exclude"
              />
            </div>
          </div>
        </div>

        {/* Left Sidebar Footer: Audit Log + About */}
        <div className="pt-1 flex items-center gap-2 shrink-0">
          <button
            type="button"
            onClick={() => setAuditOpen(true)}
            className="flex-1 inline-flex items-center justify-center gap-1.5 py-2 px-2.5 rounded-xl text-xs font-medium mica-card hover:border-[#0067C0]/35 text-slate-700 dark:text-slate-200 transition-all"
          >
            <History className="w-3.5 h-3.5 text-[#0067C0] dark:text-[#60CDFF]" />
            审计日志
          </button>
          <button
            type="button"
            onClick={() => setAboutOpen(true)}
            className="flex-1 inline-flex items-center justify-center gap-1.5 py-2 px-2.5 rounded-xl text-xs font-medium mica-card hover:border-[#0067C0]/35 text-slate-700 dark:text-slate-200 transition-all"
          >
            <Info className="w-3.5 h-3.5 text-[#0067C0] dark:text-[#60CDFF]" />
            关于
          </button>
        </div>
      </aside>

      {/* ================= RIGHT MAIN WORKSPACE ================= */}
      <div className="flex-1 min-w-0 h-full flex flex-col py-3 pr-3.5 pl-1 gap-3 overflow-hidden">
        {/* TOP FLUENT SEARCH COMMAND HEADER */}
        <header className="shrink-0 space-y-1.5">
          <div className="fluent-search-bar rounded-xl px-3.5 py-2 flex items-center gap-2.5">
            <Search className="w-4 h-4 text-slate-400 shrink-0" />

            <KeywordInput
              values={keywords}
              onChange={setKeywords}
              draft={keywordDraft}
              onDraftChange={setKeywordDraft}
              onSubmitSearch={(nextKws) => handleStartSearch(nextKws)}
              placeholder="输入关键词，回车可添加多个，或直接点击检索… (Ctrl+K)"
              variant="fluent-bar"
              inputRefExternal={searchInputRef}
            />

            {/* Match Mode Segmented Control: Elevated neutral white pill so primary Fluent Blue belongs solely to 检索 */}
            <div
              className="h-8 inline-flex items-center p-[3px] rounded-lg bg-slate-200/75 dark:bg-black/30 border border-slate-200/80 dark:border-white/[0.08] shrink-0"
              title="多关键词匹配方式：包含任意 = 出现其中任意一个即匹配；同时包含 = 必须全部出现"
            >
              <button
                type="button"
                onClick={() => setMatchMode('OR')}
                className={`h-full px-2.5 inline-flex items-center justify-center rounded-[5px] text-[11.5px] transition-all whitespace-nowrap ${
                  matchMode === 'OR'
                    ? 'bg-white dark:bg-white/15 text-slate-900 dark:text-white font-semibold shadow-sm'
                    : 'text-slate-500 dark:text-slate-400 font-medium hover:text-slate-800 dark:hover:text-slate-200'
                }`}
              >
                包含任意
              </button>
              <button
                type="button"
                onClick={() => setMatchMode('AND')}
                className={`h-full px-2.5 inline-flex items-center justify-center rounded-[5px] text-[11.5px] transition-all whitespace-nowrap ${
                  matchMode === 'AND'
                    ? 'bg-white dark:bg-white/15 text-slate-900 dark:text-white font-semibold shadow-sm'
                    : 'text-slate-500 dark:text-slate-400 font-medium hover:text-slate-800 dark:hover:text-slate-200'
                }`}
              >
                同时包含
              </button>
            </div>

            <button
              type="button"
              disabled={searching || !canClearAll}
              onClick={handleClearAll}
              className="h-8 inline-flex items-center justify-center gap-1.5 px-3 rounded-lg text-xs font-medium fluent-btn-subtle shrink-0 disabled:opacity-45 disabled:pointer-events-none whitespace-nowrap"
              title="清空待搜索目录、关键词及排除词"
            >
              <Eraser className="w-3.5 h-3.5" />
              清空
            </button>

            {searching ? (
              <button
                type="button"
                onClick={() => backend.cancelSearch()}
                className="h-8 px-4 inline-flex items-center justify-center gap-1.5 rounded-lg text-xs font-semibold bg-rose-600 hover:bg-rose-500 text-white shadow-sm transition-colors shrink-0 whitespace-nowrap"
              >
                <StopCircle className="w-3.5 h-3.5" />
                停止
              </button>
            ) : (
              <button
                type="button"
                onClick={() => handleStartSearch()}
                className="h-8 px-4 inline-flex items-center justify-center gap-1.5 rounded-lg text-xs font-semibold fluent-btn-primary shrink-0 whitespace-nowrap"
              >
                <Search className="w-3.5 h-3.5" />
                检索
              </button>
            )}
          </div>

          {/* Dynamic plain-language match explanation for multi-keyword queries */}
          {ruleHintText && (
            <div className="px-2 text-[11.5px] text-slate-500 dark:text-slate-400 flex items-center gap-1.5 animate-fade-in">
              <span className="w-1.5 h-1.5 rounded-full bg-[#0067C0] dark:bg-[#60CDFF]" />
              <span>{ruleHintText}</span>
            </div>
          )}
        </header>

        {/* CENTER + RIGHT PANELS */}
        <div className="flex-1 min-h-0 flex gap-3 overflow-hidden">
          {/* CENTER PANEL: Results Sheet */}
          <section className="flex-1 min-w-0 mica-panel rounded-2xl flex flex-col overflow-hidden">
            {/* Scanning 2px Fluent accent beam */}
            <div className="h-[2px] w-full overflow-hidden shrink-0">
              {searching && (
                <div className="h-full w-1/3 bg-gradient-to-r from-transparent via-[#0067C0] dark:via-[#60CDFF] to-transparent scan-beam" />
              )}
            </div>

            {/* Results Panel Toolbar */}
            <div className="px-4 py-2.5 border-b border-slate-200/70 dark:border-white/[0.07] flex flex-wrap items-center justify-between gap-2 shrink-0">
              <div className="flex items-center gap-2 min-w-0">
                <span className="text-[13px] font-semibold text-slate-800 dark:text-white">
                  匹配结果
                  {hasSearched && (
                    <span className="ml-1 font-mono text-[#0067C0] dark:text-[#60CDFF] tabular-nums">
                      ({visibleResults.length})
                    </span>
                  )}
                </span>

                {hasSearched && visibleResults.length > 0 && (
                  <>
                    <span className="h-3.5 w-px bg-slate-300/80 dark:bg-white/15 mx-0.5" />
                    <button
                      type="button"
                      onClick={allVisibleSelected ? clearSelection : selectAllVisible}
                      className="inline-flex items-center gap-1 px-2 py-1 rounded-md text-xs text-slate-600 dark:text-slate-300 hover:bg-white/70 dark:hover:bg-white/10 transition-colors"
                    >
                      {allVisibleSelected ? (
                        <CheckSquare className="w-3.5 h-3.5 text-[#0067C0] dark:text-[#60CDFF]" />
                      ) : someVisibleSelected ? (
                        <SquareMinus className="w-3.5 h-3.5 text-[#0067C0] dark:text-[#60CDFF]" />
                      ) : (
                        <Square className="w-3.5 h-3.5 text-slate-400" />
                      )}
                      <span>{allVisibleSelected ? '取消全选' : '全选'}</span>
                    </button>
                    <button
                      type="button"
                      onClick={invertVisibleSelection}
                      className="px-2 py-1 rounded-md text-xs text-slate-500 dark:text-slate-400 hover:text-slate-800 dark:hover:text-white hover:bg-white/70 dark:hover:bg-white/10 transition-colors"
                    >
                      反选
                    </button>
                  </>
                )}

                {hasSearched && (
                  <div className="hidden sm:flex items-center gap-0.5 ml-1">
                    {(
                      [
                        { id: 'all', label: '全部', count: counts.all },
                        { id: 'filename', label: '文件名', count: counts.filename },
                        { id: 'content', label: '正文', count: counts.content },
                        { id: 'archive', label: '压缩文件', count: counts.archive },
                      ] as { id: FilterTab; label: string; count: number }[]
                    ).map((tab) => (
                      <button
                        key={tab.id}
                        type="button"
                        onClick={() => setFilterTab(tab.id)}
                        className={`px-2 py-0.5 rounded-md text-[11.5px] transition-colors ${
                          filterTab === tab.id
                            ? 'bg-[#0067C0]/12 text-[#0067C0] dark:bg-[#60CDFF]/15 dark:text-[#60CDFF] font-semibold'
                            : 'text-slate-500 dark:text-slate-400 hover:bg-white/60 dark:hover:bg-white/8'
                        }`}
                      >
                        {tab.label}
                        <span className="ml-1 font-mono text-[10.5px] opacity-75">{tab.count}</span>
                      </button>
                    ))}
                  </div>
                )}
              </div>

              <div className="flex items-center gap-2 text-xs">
                {hasSearched && (
                  <div className="flex items-center gap-1.5 text-[11px] font-mono tabular-nums text-slate-500 dark:text-slate-400">
                    {searching && <Loader2 className="w-3 h-3 animate-spin text-[#0067C0]" />}
                    <span>已扫 {progress.scannedFiles}</span>
                    <span>·</span>
                    <span>{formatElapsed(progress.elapsedMs)}</span>
                  </div>
                )}

                {skippedFiles.length > 0 && (
                  <button
                    type="button"
                    onClick={() => setSkippedOpen(true)}
                    className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md text-[11px] font-medium bg-amber-500/15 text-amber-800 dark:text-amber-300 hover:bg-amber-500/25 transition-colors"
                  >
                    <AlertTriangle className="w-3 h-3" />
                    {skippedFiles.length} 受限
                  </button>
                )}

                {hasSearched && results.length > 0 && (
                  <>
                    <input
                      type="text"
                      value={quickFilter}
                      onChange={(e) => setQuickFilter(e.target.value)}
                      placeholder="筛选结果…"
                      className="hidden xl:block w-28 px-2 py-1 rounded-md text-xs bg-white/80 dark:bg-black/25 border border-slate-200/80 dark:border-white/10 outline-none focus:border-[#0067C0]"
                    />
                    <select
                      value={sortBy}
                      onChange={(e) => setSortBy(e.target.value as SortKey)}
                      className="px-2 py-1 rounded-md text-xs bg-white/75 dark:bg-white/[0.06] border border-slate-200/70 dark:border-white/10 text-slate-600 dark:text-slate-300 outline-none cursor-pointer"
                    >
                      <option value="hits" className="dark:bg-slate-900">按命中数</option>
                      <option value="time" className="dark:bg-slate-900">按修改时间</option>
                      <option value="size" className="dark:bg-slate-900">按体积</option>
                      <option value="name" className="dark:bg-slate-900">按文件名</option>
                    </select>
                  </>
                )}
              </div>
            </div>

            {/* Subtle Shift / Ctrl multi-select hint bar or live scan dir */}
            {hasSearched && (visibleResults.length > 0 || searching) && (
              <div className="px-4 py-1 bg-white/35 dark:bg-white/[0.02] border-b border-slate-200/50 dark:border-white/[0.05] text-[10.5px] text-slate-400 dark:text-slate-500 flex items-center justify-between gap-3">
                <span className="truncate">
                  {searching && progress.currentDir
                    ? `正在扫描：${progress.currentDir}`
                    : '提示：双击条目可打开文档，Shift 连续勾选，Ctrl 逐项切换，Ctrl+A 全选'}
                </span>
                {selectedIds.size > 0 && (
                  <span className="font-medium text-[#0067C0] dark:text-[#60CDFF] shrink-0">
                    当前已选 {selectionStats.physicalCount} 个磁盘文件
                  </span>
                )}
              </div>
            )}

            {/* Results Body */}
            <div className="flex-1 min-h-0 overflow-y-auto p-3 space-y-2.5">
              {!hasSearched ? (
                <div className="h-full flex flex-col items-center justify-center text-center px-8 py-12 animate-fade-in">
                  <AppLogo className="w-16 h-16 mb-5 drop-shadow-md" />
                  <h2 className="text-xl font-bold tracking-tight text-slate-800 dark:text-white">
                    文档清匣 · 精准检索与批量清理
                  </h2>
                  <p className="text-xs text-slate-500 dark:text-slate-400 max-w-md mt-2 leading-relaxed">
                    在左侧添加一个或多个目标目录，在上方输入关键词点击「检索」。支持 Office / 老式 DOC·XLS·PPT / PDF / 国产 OFD 与压缩文件内正文穿透检索，纯本地高速运行。
                  </p>
                  <div className="mt-4 flex flex-wrap items-center justify-center gap-2 text-[11px] text-slate-400 dark:text-slate-500">
                    <span className="px-2 py-0.5 rounded-md bg-white/70 dark:bg-white/[0.05] border border-slate-200/70 dark:border-white/[0.08] font-mono">
                      Ctrl+K 聚焦搜索
                    </span>
                    <span className="px-2 py-0.5 rounded-md bg-white/70 dark:bg-white/[0.05] border border-slate-200/70 dark:border-white/[0.08] font-mono">
                      Ctrl+Enter 开始检索
                    </span>
                    <span className="px-2 py-0.5 rounded-md bg-white/70 dark:bg-white/[0.05] border border-slate-200/70 dark:border-white/[0.08] font-mono">
                      双击条目打开文档
                    </span>
                  </div>
                  {rootDirs.length === 0 && (
                    <button
                      type="button"
                      onClick={handlePickAddDir}
                      className="mt-5 inline-flex items-center gap-2 px-4 py-2 rounded-xl text-xs font-semibold fluent-btn-primary"
                    >
                      <FolderPlus className="w-4 h-4" />
                      先添加搜索目录
                    </button>
                  )}
                </div>
              ) : visibleResults.length === 0 ? (
                <div className="h-full flex flex-col items-center justify-center text-center px-8 py-12">
                  {searching ? (
                    <div className="space-y-3 max-w-md">
                      <Loader2 className="w-7 h-7 mx-auto animate-spin text-[#0067C0] dark:text-[#60CDFF]" />
                      <p className="text-sm font-medium text-slate-700 dark:text-slate-200">
                        正在多线程检索文档内容…
                      </p>
                      <p className="text-xs font-mono text-slate-400 truncate max-w-md">
                        {progress.currentDir || rootDirs[0]}
                      </p>
                    </div>
                  ) : (
                    <div className="space-y-2 max-w-sm">
                      <p className="text-sm font-semibold text-slate-700 dark:text-slate-200">
                        {results.length === 0 ? '未找到符合条件的文档' : '当前筛选条件下无匹配项'}
                      </p>
                      <p className="text-xs text-slate-500 dark:text-slate-400 leading-relaxed">
                        {results.length === 0
                          ? `已扫描 ${progress.scannedFiles} 个文件。可尝试切换为「包含任意」或检查左侧文档类型筛选。`
                          : '请切换上方分类标签或清空筛选关键词。'}
                      </p>
                      {skippedFiles.length > 0 && (
                        <button
                          type="button"
                          onClick={() => setSkippedOpen(true)}
                          className="mt-2 inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-amber-500/15 text-amber-800 dark:text-amber-300 hover:bg-amber-500/25"
                        >
                          <AlertTriangle className="w-3.5 h-3.5" />
                          查看 {skippedFiles.length} 个加密或扫描件文件
                        </button>
                      )}
                    </div>
                  )}
                </div>
              ) : (
                visibleResults.map((item, idx) => {
                  const isSelected = selectedIds.has(item.id);
                  const isActive = activeItem?.id === item.id;
                  const previewSnippet =
                    item.snippets?.find((s) => s.source === 'content') || item.snippets?.[0];

                  return (
                    <div
                      key={item.id}
                      onClick={(e) => handleRowClick(e, idx, item.id)}
                      onDoubleClick={() =>
                        backend.openFile(item.filePath).catch((err) =>
                          notify(`打开文档失败：${err?.message || err}`, 'error')
                        )
                      }
                      className={`fluent-result-card group rounded-xl px-3.5 py-3 flex items-center gap-3 cursor-pointer ${
                        isSelected ? 'is-selected' : isActive ? 'is-active' : ''
                      }`}
                    >
                      {/* Fluent Checkbox */}
                      <button
                        type="button"
                        onClick={(e) => handleCheckboxClick(e, idx, item.id)}
                        className={`w-[18px] h-[18px] rounded-[5px] border flex items-center justify-center shrink-0 transition-all ${
                          isSelected
                            ? 'bg-[#0067C0] border-[#0067C0] text-white dark:bg-[#60CDFF] dark:border-[#60CDFF] dark:text-[#0A1929]'
                            : 'border-slate-300 dark:border-slate-500 bg-white dark:bg-black/20 hover:border-[#0067C0]'
                        }`}
                        title="勾选（按住 Shift 可连续勾选）"
                      >
                        {isSelected && <Check className="w-3 h-3 stroke-[3]" />}
                      </button>

                      {/* Crisp Fluent File Type Icon */}
                      <div className="relative shrink-0">
                        <FluentFileIcon
                          category={item.isArchiveEntry ? 'archive' : item.category}
                          className="w-9 h-9"
                        />
                        {item.isArchiveEntry && (
                          <span className="absolute -bottom-1 -right-1 px-1 rounded bg-[#0067C0] text-white text-[8px] font-bold leading-tight shadow">
                            内
                          </span>
                        )}
                      </div>

                      {/* Text Content */}
                      <div className="min-w-0 flex-1 space-y-1">
                        <div className="flex items-center gap-2 min-w-0">
                          <h3 className="text-[13.5px] font-semibold text-slate-900 dark:text-white truncate">
                            {item.isArchiveEntry ? (
                              <>
                                <span className="text-slate-500 dark:text-slate-400 font-medium">
                                  {leafName(item.archivePath)} →{' '}
                                </span>
                                {item.matchInFilename && item.snippets?.[0]?.source === 'filename' ? (
                                  <HighlightText
                                    text={item.snippets[0].text}
                                    highlights={item.snippets[0].highlights}
                                  />
                                ) : (
                                  item.fileName
                                )}
                              </>
                            ) : item.matchInFilename && item.snippets?.[0]?.source === 'filename' ? (
                              <HighlightText
                                text={item.snippets[0].text}
                                highlights={item.snippets[0].highlights}
                              />
                            ) : (
                              item.fileName
                            )}
                          </h3>

                          {item.rootLabel && (
                            <span className="shrink-0 px-1.5 py-[1px] rounded text-[10px] font-medium bg-slate-200/75 dark:bg-white/10 text-slate-600 dark:text-slate-300">
                              {item.rootLabel}
                            </span>
                          )}
                        </div>

                        {/* Highlighted Excerpt (matching Fluent Mica preview clarity) */}
                        {previewSnippet && previewSnippet.source === 'content' ? (
                          <p className="text-[12px] text-slate-600 dark:text-slate-300 leading-relaxed line-clamp-2">
                            <HighlightText
                              text={previewSnippet.text}
                              highlights={previewSnippet.highlights}
                            />
                          </p>
                        ) : (
                          <p className="text-[11.5px] font-mono text-slate-400 dark:text-slate-400 truncate">
                            {item.relativePath}
                          </p>
                        )}
                      </div>

                      {/* Right Action Button (matches preview "打开目录" clean button) */}
                      <div className="shrink-0 flex items-center gap-1.5">
                        <button
                          type="button"
                          onClick={(e) => {
                            e.stopPropagation();
                            backend.revealInFolder(item.filePath).catch((err) =>
                              notify(`打开目录失败：${err?.message || err}`, 'error')
                            );
                          }}
                          className="px-3 py-1.5 rounded-lg text-xs font-medium fluent-btn-subtle whitespace-nowrap"
                          title="在文件管理器中打开所在目录并选中该文件"
                        >
                          打开目录
                        </button>
                        <button
                          type="button"
                          onClick={(e) => {
                            e.stopPropagation();
                            openDeleteDialog('trash', [item]);
                          }}
                          className="p-1.5 rounded-lg text-slate-400 hover:text-rose-600 hover:bg-rose-500/10 opacity-0 group-hover:opacity-100 transition-all"
                          title="移入回收站"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      </div>
                    </div>
                  );
                })
              )}
            </div>
          </section>

          {/* RIGHT PANEL: Acrylic Detail Preview Card */}
          <aside className="hidden lg:flex w-[320px] xl:w-[360px] shrink-0 mica-panel rounded-2xl flex-col overflow-hidden">
            {activeItem ? (
              <>
                <div className="p-4 border-b border-slate-200/70 dark:border-white/[0.08] space-y-3.5">
                  <div className="flex items-start gap-2.5">
                    <FluentFileIcon category={activeItem.category} className="w-8 h-8 shrink-0" />
                    <h3 className="text-[14px] font-bold text-slate-900 dark:text-white break-all leading-snug select-text pt-0.5">
                      {activeItem.fileName}
                    </h3>
                  </div>

                  {activeItem.isArchiveEntry && (
                    <div className="p-2.5 rounded-xl bg-amber-500/12 border border-amber-500/25 text-[11px] text-amber-900 dark:text-amber-200 space-y-0.5">
                      <div className="font-semibold flex items-center gap-1.5">
                        <Archive className="w-3.5 h-3.5 shrink-0" />
                        压缩包内文档（删除将移除外层整包）
                      </div>
                      <div className="font-mono opacity-85 break-all">{activeItem.innerPath}</div>
                    </div>
                  )}

                  {/* Clean key-value table matching Fluent Mica preview */}
                  <div className="space-y-1.5 text-xs">
                    <div className="flex items-center justify-between gap-2">
                      <span className="text-slate-500 dark:text-slate-400 shrink-0">文件大小</span>
                      <span className="font-mono font-medium text-slate-800 dark:text-slate-100 tabular-nums">
                        {formatBytes(activeItem.diskFileSize || activeItem.size)}
                      </span>
                    </div>
                    <div className="flex items-start justify-between gap-3">
                      <span className="text-slate-500 dark:text-slate-400 shrink-0">所在路径</span>
                      <div className="flex items-start gap-1 min-w-0 justify-end">
                        <span className="font-mono text-[11px] text-slate-700 dark:text-slate-200 break-all text-right select-text">
                          {activeItem.filePath}
                        </span>
                        <button
                          type="button"
                          onClick={() => copyPath(activeItem.filePath)}
                          className="p-0.5 rounded text-slate-400 hover:text-[#0067C0] shrink-0"
                          title="复制路径"
                        >
                          <Copy className="w-3 h-3" />
                        </button>
                      </div>
                    </div>
                    <div className="flex items-center justify-between gap-2">
                      <span className="text-slate-500 dark:text-slate-400 shrink-0">命中次数</span>
                      <span className="font-semibold text-[#0067C0] dark:text-[#60CDFF] tabular-nums">
                        {activeItem.matchCount} 处匹配
                      </span>
                    </div>
                    <div className="flex items-center justify-between gap-2">
                      <span className="text-slate-500 dark:text-slate-400 shrink-0">修改时间</span>
                      <span className="font-mono text-[11px] text-slate-600 dark:text-slate-300 tabular-nums">
                        {activeItem.modTime}
                      </span>
                    </div>
                  </div>

                  {/* Action Buttons */}
                  <div className="flex items-center gap-1.5 pt-0.5">
                    <button
                      type="button"
                      onClick={() => backend.revealInFolder(activeItem.filePath)}
                      className="flex-1 inline-flex items-center justify-center gap-1 py-2 px-2 rounded-lg text-[11.5px] font-medium fluent-btn-subtle whitespace-nowrap"
                      title="在文件管理器中打开所在目录并选中该文件"
                    >
                      <FolderOpen className="w-3.5 h-3.5 text-[#0067C0] dark:text-[#60CDFF] shrink-0" />
                      打开目录
                    </button>
                    <button
                      type="button"
                      onClick={() => backend.openFile(activeItem.filePath)}
                      className="flex-1 inline-flex items-center justify-center gap-1 py-2 px-2 rounded-lg text-[11.5px] font-medium fluent-btn-subtle whitespace-nowrap"
                      title="使用系统默认程序打开此文档"
                    >
                      <ExternalLink className="w-3.5 h-3.5 text-slate-600 dark:text-slate-300 shrink-0" />
                      打开文件
                    </button>
                    <button
                      type="button"
                      onClick={() => openDeleteDialog('trash', [activeItem])}
                      className="flex-1 inline-flex items-center justify-center gap-1 py-2 px-2 rounded-lg text-[11.5px] font-medium fluent-btn-subtle hover:!text-rose-600 hover:!border-rose-300 whitespace-nowrap"
                      title="将此文件移入系统回收站"
                    >
                      <Trash2 className="w-3.5 h-3.5 shrink-0" />
                      移入回收站
                    </button>
                  </div>
                </div>

                {/* Matched Context Snippets */}
                <div className="flex-1 min-h-0 overflow-y-auto p-4 space-y-3">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-semibold text-slate-800 dark:text-slate-100">
                      匹配上下文摘要
                    </span>
                    <div className="flex flex-wrap gap-1">
                      {activeItem.matchedKeywords?.map((kw) => (
                        <span
                          key={kw}
                          className="px-1.5 py-0.5 rounded text-[10px] font-medium bg-[#FDE68A]/75 text-slate-800 dark:bg-amber-400/20 dark:text-amber-200"
                        >
                          {kw}
                        </span>
                      ))}
                    </div>
                  </div>

                  {activeItem.snippets && activeItem.snippets.length > 0 ? (
                    <div className="space-y-2.5">
                      {activeItem.snippets.map((snip, idx) => (
                        <div
                          key={idx}
                          className="p-3 rounded-xl bg-white/75 dark:bg-white/[0.04] border border-slate-200/70 dark:border-white/[0.07] space-y-1 select-text"
                        >
                          <div className="text-[10px] font-medium text-slate-400">
                            {snip.source === 'filename' ? '文件名命中' : `正文片段 #${idx + 1}`}
                          </div>
                          <p className="text-xs text-slate-700 dark:text-slate-200 leading-relaxed break-words">
                            <HighlightText text={snip.text} highlights={snip.highlights} />
                          </p>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="text-xs text-slate-400 py-8 text-center">无额外正文摘要</p>
                  )}
                </div>
              </>
            ) : (
              <div className="flex-1 flex flex-col items-center justify-center text-center p-6 text-slate-400 dark:text-slate-500">
                <FolderOpen className="w-8 h-8 mb-2.5 opacity-45 stroke-[1.5]" />
                <p className="text-xs font-medium text-slate-500 dark:text-slate-400">文档上下文预览</p>
                <p className="text-[11px] mt-1 leading-relaxed">
                  点击左侧匹配结果，在此核验完整路径与关键词上下文片段
                </p>
              </div>
            )}
          </aside>
        </div>

        {/* BOTTOM ACRYLIC BATCH ACTION BAR (1:1 Fluent Mica preview layout) */}
        <footer className="acrylic-surface rounded-xl px-4 py-2.5 flex flex-wrap items-center justify-between gap-3 shrink-0">
          <div className="flex items-center gap-2.5 text-xs">
            {selectionStats.selectedCount > 0 ? (
              <>
                <span className="inline-flex items-center justify-center min-w-[22px] h-[22px] px-1.5 rounded-full bg-[#0067C0] dark:bg-[#60CDFF] text-white dark:text-[#0A1929] text-[11px] font-bold font-mono tabular-nums">
                  {selectionStats.physicalCount}
                </span>
                <span className="font-semibold text-slate-800 dark:text-white">
                  已选 {selectionStats.physicalCount} 个文件
                  {selectionStats.selectedCount !== selectionStats.physicalCount && (
                    <span className="font-normal text-amber-700 dark:text-amber-300 ml-1">
                      （对应 {selectionStats.selectedCount} 条命中）
                    </span>
                  )}
                </span>
                <span className="text-slate-400">·</span>
                <span className="font-mono tabular-nums text-slate-600 dark:text-slate-300">
                  {formatBytes(selectionStats.totalBytes)}
                </span>
                <button
                  type="button"
                  onClick={clearSelection}
                  className="text-[11px] text-slate-400 hover:text-slate-700 dark:hover:text-slate-200 underline ml-1"
                >
                  清空勾选
                </button>
              </>
            ) : (
              <span className="text-slate-500 dark:text-slate-400">
                勾选匹配文档即可批量移入回收站、隔离备份或彻底删除（支持 Shift 连选）
              </span>
            )}
          </div>

          <div className="flex items-center gap-2">
            <button
              type="button"
              disabled={selectionStats.selectedCount === 0}
              onClick={() => openDeleteDialog('trash', selectedItems)}
              className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg text-xs font-medium fluent-btn-subtle disabled:opacity-45 disabled:pointer-events-none"
            >
              <Trash2 className="w-3.5 h-3.5 text-[#0067C0] dark:text-[#60CDFF]" />
              移入回收站
            </button>
            <button
              type="button"
              disabled={selectionStats.selectedCount === 0}
              onClick={() => openDeleteDialog('quarantine', selectedItems)}
              className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg text-xs font-medium fluent-btn-subtle disabled:opacity-45 disabled:pointer-events-none"
            >
              <FolderInput className="w-3.5 h-3.5 text-amber-600 dark:text-amber-400" />
              隔离备份
            </button>
            <button
              type="button"
              disabled={selectionStats.selectedCount === 0}
              onClick={() => openDeleteDialog('permanent', selectedItems)}
              className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-lg text-xs font-semibold bg-rose-500/15 hover:bg-rose-600 text-rose-700 hover:text-white dark:bg-rose-500/20 dark:text-rose-300 dark:hover:bg-rose-600 dark:hover:text-white border border-rose-500/25 transition-colors disabled:opacity-40 disabled:pointer-events-none"
            >
              彻底删除
            </button>
          </div>
        </footer>
      </div>
      </div>

      {/* TOAST */}
      {toast && (
        <div className="fixed top-12 right-5 z-50 animate-fade-in">
          <div
            className={`px-4 py-2 rounded-xl text-xs font-medium shadow-lg border backdrop-blur-md ${
              toast.tone === 'error'
                ? 'bg-rose-600/95 text-white border-rose-500'
                : toast.tone === 'success'
                ? 'bg-[#0067C0]/95 text-white border-[#005FB8]'
                : 'acrylic-surface text-slate-800 dark:text-white'
            }`}
          >
            {toast.message}
          </div>
        </div>
      )}

      {/* MODALS & DRAWERS */}
      <DeleteConfirmModal
        open={deleteModal.open}
        mode={deleteModal.mode}
        items={deleteModal.targets}
        quarantineDir={quarantineDir}
        onQuarantineDirChange={setQuarantineDir}
        onCancel={() => setDeleteModal({ open: false, mode: 'trash', targets: [] })}
        onConfirm={executeDelete}
      />

      <AuditDrawer open={auditOpen} onClose={() => setAuditOpen(false)} onNotify={notify} />

      <SkippedModal open={skippedOpen} files={skippedFiles} onClose={() => setSkippedOpen(false)} />

      <AboutModal
        open={aboutOpen}
        sysInfo={sysInfo}
        onClose={() => setAboutOpen(false)}
        onNotify={notify}
      />
    </div>
  );
}
