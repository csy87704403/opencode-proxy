# OpenCode Minimal Go Proxy

这是一个极简的 Go 语言编写的 OpenCode 免费大模型反代工具，仅保留了核心的反代桥接（Bridge Mode A）功能。非常适合部署在 VPS 上，为本地的 Cursor、Cline、VSCode 等 AI 编码工具提供免费的 API 代理。

## 功能特点
- **极轻量**：仅使用 Go 标准库，无外部依赖。打包成 Docker 镜像后仅约 15MB。
- **只保留核心功能**：专门将 OpenAI 兼容接口转换为 OpenCode 的免费大模型接口。
- **支持流式（Streaming SSE）**：完美支持大模型流式打字输出。
- **自动清洗模型前缀**：自动兼容 `oc/` 或 `opencode/` 前缀的模型名。

---

## 1. 本地运行与编译

确保已安装 Go 1.21 或更高版本：

```bash
# 本地运行
go run main.go

# 编译为当前系统可执行文件
go build -o opencode-proxy main.go
```

---

## 2. 提交到自己的 GitHub 仓库

你可以使用以下命令将此项目初始化并推送到你的 GitHub 仓库：

```bash
# 1. 初始化 Git 仓库
git init

# 2. 将所有文件添加到暂存区
git add .

# 3. 提交更改
git commit -m "Initial commit: opencode proxy rewrite in Go"

# 4. 创建分支
git branch -M main

# 5. 关联你的 GitHub 仓库地址 (替换为你的真实仓库 URL)
git remote add origin https://github.com/你的用户名/你的仓库名.git

# 6. 推送到 GitHub
git push -u origin main
```

---

## 3. VPS 部署指南

推荐使用 **Docker & Docker Compose** 在 VPS 上进行一键部署。

### 准备工作
确保你的 VPS 上已安装 Docker 和 Docker Compose。如果未安装，可以在 Ubuntu/Debian 系统上使用以下命令安装：
```bash
sudo apt update
sudo apt install -y docker.io docker-compose
```

### 部署步骤

#### 方法 A：直接克隆仓库并在 VPS 构建运行 (推荐)
1. 在 VPS 上克隆你的 GitHub 仓库：
   ```bash
   git clone https://github.com/你的用户名/你的仓库名.git opencode-proxy
   cd opencode-proxy
   ```
2. 使用 Docker Compose 后台启动服务：
   ```bash
   docker-compose up -d --build
   ```

#### 方法 B：使用极简的运行包
如果你不想在 VPS 上使用 git，只需将 `Dockerfile`、`docker-compose.yml`、`main.go` 拷贝至 VPS 目录，运行：
```bash
docker-compose up -d --build
```

### 验证服务是否成功启动
```bash
# 查看容器运行状态
docker ps

# 检查日志
docker logs -f opencode-proxy
```

---

## 4. 安全鉴权配置 (PROXY_API_KEY)

本代理工具内置了 `AuthMiddleware` 鉴权中间件，部署在公网时默认开启，以防端口被黑客扫描滥用。

### 密钥配置
你可以在 VPS 上的 `docker-compose.yml` 环境变量中修改密钥：
```yaml
    environment:
      - PORT=20128
      - PROXY_API_KEY=你的自定义密钥
```

*若将 `PROXY_API_KEY` 留空，则不启用鉴权（**极不推荐**）。*

---

## 5. 客户端配置示例

配置本地开发工具（如 Cursor / Cline / Claude Code）：
- **接口地址 (Base URL)**: `http://你的VPS_IP:20128/v1` （请在 VPS 防火墙放行 `20128` 端口）
- **API Key / 密码**: 填写你在 `docker-compose.yml` 中设置的 `PROXY_API_KEY`（默认值为 `sk_opencode_proxy_default_key_2026`）
- **大模型名称**: `big-pickle`，`nemotron-3-super-free` 等可用的免费模型名。

