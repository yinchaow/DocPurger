export type MatchMode = 'AND' | 'OR';
export type SearchScope = 'both' | 'filename' | 'content';
export type DeleteMode = 'trash' | 'quarantine' | 'permanent';

export interface SearchOptions {
  rootDir?: string;
  rootDirs: string[];
  keywords: string[];
  excludeKeywords: string[];
  mode: MatchMode;
  scope: SearchScope;
  caseSensitive: boolean;
  includeArchives: boolean;
  categories: string[];
  maxFileSizeMB: number;
  includeHidden: boolean;
  maxArchiveDepth: number;
}

export interface HighlightSpan {
  start: number;
  end: number;
  keyword: string;
}

export interface MatchSnippet {
  source: 'filename' | 'content';
  text: string;
  highlights: HighlightSpan[];
}

export interface SearchResult {
  id: string;
  filePath: string;
  relativePath: string;
  rootLabel: string;
  fileName: string;
  archivePath: string;
  innerPath: string;
  isArchiveEntry: boolean;
  ext: string;
  category: string;
  size: number;
  diskFileSize: number;
  modTime: string;
  modTimeUnix: number;
  matchInFilename: boolean;
  matchInContent: boolean;
  matchedKeywords: string[];
  matchCount: number;
  snippets: MatchSnippet[];
}

export interface SkippedFile {
  filePath: string;
  relativePath: string;
  reason: string;
}

export interface SearchProgress {
  scannedFiles: number;
  matchedCount: number;
  skippedCount: number;
  currentDir: string;
  elapsedMs: number;
  running: boolean;
}

export interface SearchSummary {
  scannedFiles: number;
  matchedCount: number;
  skippedCount: number;
  skippedFiles: SkippedFile[];
  elapsedMs: number;
  cancelled: boolean;
  error?: string;
}

export interface DeleteRequest {
  paths: string[];
  mode: DeleteMode;
  quarantineDir: string;
  rootDir?: string;
  rootDirs: string[];
  keywords: string[];
}

export interface DeleteItemResult {
  path: string;
  success: boolean;
  destination?: string;
  error?: string;
}

export interface DeleteResponse {
  mode: DeleteMode;
  total: number;
  succeeded: number;
  failed: number;
  freedBytes: number;
  results: DeleteItemResult[];
  auditLogPath: string;
}

export interface AuditEntry {
  timestamp: string;
  mode: DeleteMode;
  filePath: string;
  destination?: string;
  fileSize: number;
  keywords: string[];
  success: boolean;
  error?: string;
}

export interface SystemInfo {
  os: string;
  arch: string;
  appVersion: string;
  auditLogPath: string;
  homeDir: string;
}
