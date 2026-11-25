#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
IMAGE_NAME="docpurger-builder:20.04"
VERSION="1.3.0"

echo "==> [1/4] Checking Docker builder image ($IMAGE_NAME)..."
if ! docker image inspect "$IMAGE_NAME" >/dev/null 2>&1; then
  echo "Building $IMAGE_NAME (Ubuntu 20.04 / glibc 2.31 / webkit2gtk-4.0)..."
  docker build -t "$IMAGE_NAME" -f "$PROJECT_DIR/docker/Dockerfile.builder" "$PROJECT_DIR/docker"
fi

mkdir -p "$PROJECT_DIR/dist"
# Remove older dist artifacts and cached Windows .ico so Wails always regenerates from the latest build/appicon.png
rm -f "$PROJECT_DIR"/dist/docpurger_* "$PROJECT_DIR"/dist/DocPurger_* "$PROJECT_DIR/build/windows/icon.ico"

echo "==> [2/4] Generating icon, running tests, frontend build, Kylin V10 SP1 (linux/amd64) & Windows (windows/amd64) builds..."
docker run --rm \
  -v "$PROJECT_DIR:/workspace" \
  -v docpurger-gomod-cache:/root/go/pkg/mod \
  -e VERSION="$VERSION" \
  -e HOST_UID="$(id -u)" \
  -e HOST_GID="$(id -g)" \
  "$IMAGE_NAME" \
  bash -lc '
    set -euo pipefail
    cd /workspace

    echo "--- Generating 512x512 Fluent brand icon ---"
    go run scripts/genicon.go
    rm -f build/windows/icon.ico

    echo "--- Frontend build (React + Vite + Tailwind) ---"
    cd /workspace/frontend
    if [ ! -d node_modules ]; then
      npm install --no-audit --no-fund
    fi
    npm run build

    cd /workspace
    echo "--- Go mod tidy & unit tests ---"
    go mod tidy
    go test -v ./...

    echo "--- Building Kylin V10 SP1 x64 binary (Ubuntu 20.04 / glibc 2.31 / webkit2gtk-4.0) ---"
    wails build -clean -skipbindings -s -ldflags "-s -w" -platform linux/amd64 -o docpurger

    echo "--- Verifying Linux binary GLIBC max version & dynamic libraries ---"
    (ldd build/bin/docpurger || true) | head -n 15
    MAX_GLIBC=$(objdump -T build/bin/docpurger 2>/dev/null | grep -oE "GLIBC_[0-9]+\.[0-9]+" | sort -Vu | tail -n 1 || true)
    echo "Max required GLIBC symbol: ${MAX_GLIBC:-unknown}"

    echo "--- Packaging Kylin V10 SP1 .deb installer ---"
    DEB_ROOT="/tmp/docpurger-deb"
    rm -rf "$DEB_ROOT"
    mkdir -p "$DEB_ROOT/DEBIAN" \
             "$DEB_ROOT/usr/bin" \
             "$DEB_ROOT/usr/share/applications" \
             "$DEB_ROOT/usr/share/icons/hicolor/512x512/apps"

    cp build/bin/docpurger "$DEB_ROOT/usr/bin/docpurger"
    chmod 0755 "$DEB_ROOT/usr/bin/docpurger"
    cp build/appicon.png "$DEB_ROOT/usr/share/icons/hicolor/512x512/apps/docpurger.png"

    cat > "$DEB_ROOT/usr/share/applications/docpurger.desktop" <<EOF
[Desktop Entry]
Name=DocPurger
Name[zh_CN]=文档清匣
Comment=Cross-platform document keyword search and safe batch cleanup tool
Comment[zh_CN]=跨平台文档内容检索与安全批量清理工具
Exec=/usr/bin/docpurger
Icon=docpurger
Terminal=false
Type=Application
Categories=Utility;Office;
Keywords=search;document;clean;pdf;office;ofd;文档清匣;
EOF

    cat > "$DEB_ROOT/DEBIAN/control" <<EOF
Package: docpurger
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: amd64
Depends: libgtk-3-0, libwebkit2gtk-4.0-37
Maintainer: yinchaow <https://github.com/yinchaow/DocPurger>
Homepage: https://github.com/yinchaow/DocPurger
Description: 文档清匣 (DocPurger) - 文档检索与批量清理工具
 适用于银河麒麟 V10 SP1 x64 的现代跨平台文档检索与批量清理工具，
 支持多目录扫描、文件名及内容多关键词同时包含/包含任意检索、老式
 doc/xls/ppt、docx/xlsx/pptx、PDF、OFD 及压缩包穿透检索、高亮预览、
 打开所在目录与安全批量删除。
EOF

    fakeroot dpkg-deb --build "$DEB_ROOT" "/workspace/dist/DocPurger_${VERSION}_kylin_v10_sp1_amd64.deb"

    echo "--- Packaging Kylin V10 SP1 portable tarball ---"
    PORTABLE_DIR="/tmp/DocPurger-kylin-portable"
    rm -rf "$PORTABLE_DIR"
    mkdir -p "$PORTABLE_DIR"
    cp build/bin/docpurger "$PORTABLE_DIR/DocPurger"
    cp build/appicon.png "$PORTABLE_DIR/DocPurger.png"
    cat > "$PORTABLE_DIR/README.txt" <<EOF
文档清匣 (DocPurger) v${VERSION} — 银河麒麟 V10 SP1 x64 绿色免安装版
运行方式：双击或终端执行 ./DocPurger
项目主页：https://github.com/yinchaow/DocPurger
开源协议：MIT License
EOF
    tar -C /tmp -czf "/workspace/dist/DocPurger_${VERSION}_kylin_v10_sp1_amd64.tar.gz" DocPurger-kylin-portable

    echo "--- Cross-compiling Windows x64 portable .exe (zero CGO) ---"
    wails build -skipbindings -s -ldflags "-s -w" -platform windows/amd64 -o DocPurger.exe
    cp build/bin/DocPurger.exe "/workspace/dist/DocPurger_${VERSION}_windows_amd64.exe"

    # Align ownership of generated artifacts with host user
    chown -R "${HOST_UID}:${HOST_GID}" /workspace/dist /workspace/build /workspace/frontend/dist /workspace/go.mod /workspace/go.sum 2>/dev/null || true
  '

echo "==> [3/4] Distribution artifacts:"
ls -lh "$PROJECT_DIR/dist/"
echo "==> [4/4] All builds completed successfully!"
