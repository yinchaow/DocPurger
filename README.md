# 文档清匣 (DocPurger)

> 跨平台现代文档内容检索与安全批量清理工具 —— 专为 **Windows 10/11 (x64)** 与 **银河麒麟 V10 SP1 (x64)** 设计。

- **开发人员**：yinchaow & Gemini
- **开源协议**：[MIT License](./LICENSE)
- **项目主页**：https://github.com/yinchaow/DocPurger

## 核心特性

- **Windows 11 Fluent Mica 磨砂美学**：通透云母柔彩底层 + 分层半透明卡片 + Fluent 蓝强调色与统一的高清品牌/文档图标，支持浅色云母与深海云母一键切换。
- **一次搜索多个目录**：支持连续添加多个目标目录，自动执行父子目录包含去重与多核并发扫描。
- **零门槛多关键词逻辑**：直观切换 **「同时包含」** / **「包含任意」**，配合实时白话规则提示与排除词过滤。
- **全格式零依赖正文解析**：
  - 现代 Office & 信创版式：`.docx` / `.xlsx` / `.pptx` / `.ofd` / `.odt` / `.ods` / `.odp`
  - 老式 Office（含文件头魔数自动分流）：`.doc`（OLE2 PieceTable + 伪 `.doc` RTF/HTML 自动识别）、`.xls`、`.ppt`、`.wps`
  - 版式与文本：`.pdf`（自动识别加密/纯图片扫描件并透明列出）、`.txt` / `.md` / `.csv` / `.json` / `.xml` / `.log`（自动识别 UTF-8 / UTF-16 / GB18030）
- **纯内存穿透压缩包**：免解压直读 `.zip`（修复 GBK 中文文件名乱码）、`.rar`（RAR4/RAR5）、`.7z`、`.tar` / `.tar.gz` 内部文件名与文档正文。
- **原生级多选与防误删清理**：
  - 支持整行 `Shift` 连选、`Ctrl` 多选、`Ctrl+A` 全选、`Esc` 清空；
  - 右侧实时高亮上下文核验、一键打开所在目录并选中文件（Windows `explorer /select`，银河麒麟 `peony --show-items`）；
  - 默认**移入系统回收站**，可选**隔离备份目录**与**彻底删除**（二次勾选确认），自动记录审计日志并支持导出 CSV。

## 一键双平台构建

```bash
chmod +x scripts/build-all.sh
./scripts/build-all.sh
```

产物输出在 `dist/` 目录：
- `dist/docpurger_1.1.0_windows_amd64.exe` — Windows 10/11 x64 免安装单文件
- `dist/docpurger_1.1.0_kylin_v10_sp1_amd64.deb` — 银河麒麟 V10 SP1 x64 安装包
- `dist/docpurger_1.1.0_kylin_v10_sp1_amd64.tar.gz` — 银河麒麟 V10 SP1 x64 绿色压缩包
