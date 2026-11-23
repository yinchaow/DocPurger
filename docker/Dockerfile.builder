FROM ubuntu:20.04

ENV DEBIAN_FRONTEND=noninteractive \
    TZ=Asia/Shanghai \
    GOPATH=/root/go \
    GOPROXY=https://goproxy.cn,direct \
    PATH=/usr/local/go/bin:/root/go/bin:/usr/local/node/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

# Use Aliyun Ubuntu 20.04 mirror for fast builds
RUN sed -i 's|http://archive.ubuntu.com/ubuntu/|http://mirrors.aliyun.com/ubuntu/|g; s|http://security.ubuntu.com/ubuntu/|http://mirrors.aliyun.com/ubuntu/|g' /etc/apt/sources.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        git \
        build-essential \
        pkg-config \
        libgtk-3-dev \
        libwebkit2gtk-4.0-dev \
        dpkg-dev \
        fakeroot \
        xz-utils \
        zip \
        unzip \
        nsis \
    && rm -rf /var/lib/apt/lists/*

# Install Go 1.27.4
ARG GO_VERSION=1.27.4
RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -o /tmp/go.tar.gz \
    && tar -C /usr/local -xzf /tmp/go.tar.gz \
    && rm -f /tmp/go.tar.gz

# Install Node.js 20.18.0 (use npmmirror fallback or official)
ARG NODE_VERSION=20.18.0
RUN (curl -fsSL "https://npmmirror.com/mirrors/node/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-x64.tar.xz" -o /tmp/node.tar.xz \
      || curl -fsSL "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-x64.tar.xz" -o /tmp/node.tar.xz) \
    && mkdir -p /usr/local/node \
    && tar -C /usr/local/node --strip-components=1 -xJf /tmp/node.tar.xz \
    && rm -f /tmp/node.tar.xz

# Install Wails v2 CLI
RUN go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.1

WORKDIR /workspace
