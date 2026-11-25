import type {
  AuditEntry,
  DeleteRequest,
  DeleteResponse,
  SearchOptions,
  SearchProgress,
  SearchResult,
  SearchSummary,
  SystemInfo,
} from '../types';

declare global {
  interface Window {
    go?: {
      main?: {
        App?: {
          GetSystemInfo(): Promise<SystemInfo>;
          SelectDirectory(defaultDir: string): Promise<string>;
          FilterDirectories(paths: string[]): Promise<string[]>;
          SyncWindowCorners(): Promise<void>;
          SelectQuarantineDir(defaultDir: string): Promise<string>;
          StartSearch(opts: SearchOptions): Promise<void>;
          CancelSearch(): Promise<void>;
          RevealInFolder(filePath: string): Promise<void>;
          OpenFile(filePath: string): Promise<void>;
          OpenURL(url: string): Promise<void>;
          DeleteFiles(req: DeleteRequest): Promise<DeleteResponse>;
          GetAuditLogs(limit: number): Promise<AuditEntry[]>;
          ClearAuditLogs(): Promise<void>;
          ExportAuditCSV(): Promise<string>;
        };
      };
    };
    runtime?: {
      EventsOn(eventName: string, callback: (...data: any[]) => void): () => void;
      EventsOff(eventName: string, ...additionalEventNames: string[]): void;
      OnFileDrop?(
        callback: (x: number, y: number, paths: string[]) => void,
        useDropTarget?: boolean
      ): void;
      OnFileDropOff?(): void;
      WindowMinimise?(): void;
      WindowToggleMaximise?(): void;
      WindowIsMaximised?(): Promise<boolean>;
      ClipboardGetText?(): Promise<string>;
      Quit?(): void;
    };
  }
}

export function isWailsEnv(): boolean {
  return Boolean(window.go?.main?.App);
}

type Listener = (payload: any) => void;
const mockBus: Record<string, Set<Listener>> = {};
function emitMock(event: string, payload: any) {
  mockBus[event]?.forEach((fn) => fn(payload));
}

export const backend = {
  async getSystemInfo(): Promise<SystemInfo> {
    if (window.go?.main?.App) {
      return window.go.main.App.GetSystemInfo();
    }
    return {
      os: 'windows',
      arch: 'amd64',
      appVersion: '1.3.0',
      auditLogPath: '%APPDATA%\\DocPurger\\audit.jsonl',
      homeDir: 'D:\\工作文档',
    };
  },

  async selectDirectory(defaultDir: string): Promise<string> {
    if (window.go?.main?.App) {
      return window.go.main.App.SelectDirectory(defaultDir);
    }
    return defaultDir ? `${defaultDir}\\子归档` : 'D:\\工作文档\\2026项目归档';
  },

  async filterDirectories(paths: string[]): Promise<string[]> {
    if (window.go?.main?.App?.FilterDirectories) {
      return window.go.main.App.FilterDirectories(paths);
    }
    return paths;
  },

  async selectQuarantineDir(defaultDir: string): Promise<string> {
    if (window.go?.main?.App) {
      return window.go.main.App.SelectQuarantineDir(defaultDir);
    }
    return defaultDir || 'D:\\文档清匣_隔离备份';
  },

  async startSearch(opts: SearchOptions): Promise<void> {
    if (window.go?.main?.App) {
      return window.go.main.App.StartSearch(opts);
    }
    const base = opts.rootDirs[0] || 'D:\\工作文档\\2026项目归档';
    const kw = opts.keywords[0] || '预算';
    const multi = opts.rootDirs.length > 1;
    const demo: SearchResult[] = [
      {
        id: 'demo-1',
        filePath: `${base}\\2026年度财务预算方案.docx`,
        relativePath: '2026年度财务预算方案.docx',
        rootLabel: multi ? '2026项目归档' : '',
        fileName: '2026年度财务预算方案.docx',
        archivePath: '',
        innerPath: '',
        isArchiveEntry: false,
        ext: '.docx',
        category: 'word',
        size: 151552,
        diskFileSize: 151552,
        modTime: '2026-11-18 14:22',
        modTimeUnix: 1795011720,
        matchInFilename: true,
        matchInContent: true,
        matchedKeywords: opts.keywords,
        matchCount: 6,
        snippets: [
          {
            source: 'filename',
            text: `2026年度财务${kw}方案.docx`,
            highlights: [{ start: 6, end: 6 + kw.length, keyword: kw }],
          },
          {
            source: 'content',
            text: `…一、总体编制原则：为保障 2026 年度各项核心业务平稳推进，本${kw}方案按照零基预算管理要求，重点向数字化基础设施建设倾斜…`,
            highlights: [{ start: 30, end: 30 + kw.length, keyword: kw }],
          },
        ],
      },
      {
        id: 'demo-2',
        filePath: `${base}\\历史结算明细.xls`,
        relativePath: '历史结算明细.xls',
        rootLabel: multi ? '2026项目归档' : '',
        fileName: '历史结算明细.xls',
        archivePath: '',
        innerPath: '',
        isArchiveEntry: false,
        ext: '.xls',
        category: 'excel',
        size: 94208,
        diskFileSize: 94208,
        modTime: '2026-10-09 09:15',
        modTimeUnix: 1791537300,
        matchInFilename: false,
        matchInContent: true,
        matchedKeywords: opts.keywords,
        matchCount: 3,
        snippets: [
          {
            source: 'content',
            text: `…附表二汇总了上年度各季度实际结算金额与原定${kw}偏差对比，请财务复核…`,
            highlights: [{ start: 22, end: 22 + kw.length, keyword: kw }],
          },
        ],
      },
      {
        id: 'demo-3',
        filePath: `${base}\\季度汇报归档.zip`,
        relativePath: '季度汇报归档.zip',
        rootLabel: multi ? '历史资料' : '',
        fileName: '红头审批单.ofd',
        archivePath: `${base}\\季度汇报归档.zip`,
        innerPath: '财务材料/红头审批单.ofd',
        isArchiveEntry: true,
        ext: '.ofd',
        category: 'ofd',
        size: 64200,
        diskFileSize: 1843200,
        modTime: '2026-11-02 16:40',
        modTimeUnix: 1793637600,
        matchInFilename: false,
        matchInContent: true,
        matchedKeywords: opts.keywords,
        matchCount: 2,
        snippets: [
          {
            source: 'content',
            text: `…经办公会审议同意第一季度专项${kw}经费调拨申请，按规定归档留存…`,
            highlights: [{ start: 15, end: 15 + kw.length, keyword: kw }],
          },
        ],
      },
      {
        id: 'demo-4',
        filePath: `${base}\\项目立项报告.pdf`,
        relativePath: '项目立项报告.pdf',
        rootLabel: multi ? '历史资料' : '',
        fileName: '项目立项报告.pdf',
        archivePath: '',
        innerPath: '',
        isArchiveEntry: false,
        ext: '.pdf',
        category: 'pdf',
        size: 428032,
        diskFileSize: 428032,
        modTime: '2026-11-12 11:05',
        modTimeUnix: 1794481500,
        matchInFilename: false,
        matchInContent: true,
        matchedKeywords: opts.keywords,
        matchCount: 4,
        snippets: [
          {
            source: 'content',
            text: `…本项目拟于年初正式启动，总投资${kw}计划数据报告已通过初步评审…`,
            highlights: [{ start: 16, end: 16 + kw.length, keyword: kw }],
          },
        ],
      },
    ];
    setTimeout(() => {
      demo.forEach((item) => emitMock('search:match', item));
      emitMock('search:progress', {
        scannedFiles: 156,
        matchedCount: demo.length,
        skippedCount: 1,
        currentDir: '',
        elapsedMs: 128,
        running: false,
      } satisfies SearchProgress);
      emitMock('search:done', {
        scannedFiles: 156,
        matchedCount: demo.length,
        skippedCount: 1,
        skippedFiles: [
          {
            filePath: `${base}\\盖章扫描件.pdf`,
            relativePath: '盖章扫描件.pdf',
            reason: '扫描件/纯图片 PDF（无文本层）',
          },
        ],
        elapsedMs: 128,
        cancelled: false,
      } satisfies SearchSummary);
    }, 260);
  },

  async cancelSearch(): Promise<void> {
    if (window.go?.main?.App) {
      return window.go.main.App.CancelSearch();
    }
  },

  async revealInFolder(filePath: string): Promise<void> {
    if (window.go?.main?.App) {
      return window.go.main.App.RevealInFolder(filePath);
    }
    console.info('[Demo] RevealInFolder:', filePath);
  },

  async openFile(filePath: string): Promise<void> {
    if (window.go?.main?.App) {
      return window.go.main.App.OpenFile(filePath);
    }
    console.info('[Demo] OpenFile:', filePath);
  },

  async openURL(url: string): Promise<void> {
    if (window.go?.main?.App?.OpenURL) {
      return window.go.main.App.OpenURL(url);
    }
    window.open(url, '_blank', 'noopener,noreferrer');
  },

  async deleteFiles(req: DeleteRequest): Promise<DeleteResponse> {
    if (window.go?.main?.App) {
      return window.go.main.App.DeleteFiles(req);
    }
    return {
      mode: req.mode,
      total: req.paths.length,
      succeeded: req.paths.length,
      failed: 0,
      freedBytes: 245760,
      results: req.paths.map((p) => ({ path: p, success: true })),
      auditLogPath: '%APPDATA%\\DocPurger\\audit.jsonl',
    };
  },

  async getAuditLogs(limit = 200): Promise<AuditEntry[]> {
    if (window.go?.main?.App) {
      return window.go.main.App.GetAuditLogs(limit);
    }
    return [];
  },

  async clearAuditLogs(): Promise<void> {
    if (window.go?.main?.App) {
      return window.go.main.App.ClearAuditLogs();
    }
  },

  async exportAuditCSV(): Promise<string> {
    if (window.go?.main?.App) {
      return window.go.main.App.ExportAuditCSV();
    }
    return '';
  },

  onEvent<T>(eventName: string, cb: (payload: T) => void): () => void {
    if (window.runtime?.EventsOn) {
      return window.runtime.EventsOn(eventName, (data: T) => cb(data));
    }
    if (!mockBus[eventName]) {
      mockBus[eventName] = new Set();
    }
    mockBus[eventName].add(cb as Listener);
    return () => {
      mockBus[eventName]?.delete(cb as Listener);
    };
  },

  onFileDrop(cb: (paths: string[]) => void, useDropTarget = true): () => void {
    if (window.runtime?.OnFileDrop) {
      window.runtime.OnFileDrop((_x, _y, paths) => {
        cb(Array.isArray(paths) ? paths : []);
      }, useDropTarget);
      return () => {
        window.runtime?.OnFileDropOff?.();
      };
    }
    return () => {};
  },

  windowMinimise(): void {
    window.runtime?.WindowMinimise?.();
  },

  windowToggleMaximise(): void {
    window.runtime?.WindowToggleMaximise?.();
  },

  async windowIsMaximised(): Promise<boolean> {
    if (window.runtime?.WindowIsMaximised) {
      return window.runtime.WindowIsMaximised();
    }
    return false;
  },

  async clipboardGetText(): Promise<string> {
    if (window.runtime?.ClipboardGetText) {
      return window.runtime.ClipboardGetText();
    }
    if (navigator.clipboard?.readText) {
      return navigator.clipboard.readText();
    }
    return '';
  },

  syncWindowCorners(): void {
    window.go?.main?.App?.SyncWindowCorners?.().catch(() => {});
  },

  quit(): void {
    window.runtime?.Quit?.();
  },
};
