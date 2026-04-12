# docker 部署说明

```yml
# ==============================================================================
# 📚 Flare (极轻量导航页)
# https://github.com/vickai/flare
# https://github.com/soulteary/docker-flare
#
# 📝 简介：轻量、快速、美观的个人导航页面，适用于 HomeLab 或其他注重私密的场景。
# 无任何数据库依赖，应用数据完全开放透明，应用资源消耗极低 (MEM < 30M)。
#
# ------------------------------------------------------------------------------
# ---📦 部署镜像
# 📥 部署时间: 20260412
# 🔖 镜像版本: ghcr.io/vickai/flare-vickai:20260412
# 🏷️ 镜像版本: ghcr.io/vickai/flare-vickai:latest
#
# ---🚀 目录结构
# /data/docker-compose/flare-vickai
#  ├── docker-compose.yml          # 本配置文件
#  └── app/                        # 存储书签数据及项目配置文件
#      ├── icons
#      ├── apps.yml
#      ├── bookmarks.yml
#      └── config.yml
#
# ------------------------------------------------------------------------------
# ---🧠 注意事项
# 1. 资源消耗极低，非常契合老旧硬件。
# 2. Flare 镜像极小 (<10M)，内部无 Shell 环境，切勿添加常规 Healthcheck。
# 3. 环境变量中的开关选项推荐使用 1(开启) 和 0(关闭)，避免 YAML 解析歧义。
#
# ------------------------------------------------------------------------------
# ---🧪 常用运维
# 验证配置是否正确     docker compose config
# 启动服务             docker compose up -d
# 强制重建             docker compose up -d --force-recreate
# 查看运行日志         docker logs flare --tail 50 -f
# 进入容器命令行       docker exec -it flare sh
#
# ------------------------------------------------------------------------------
# ---🔍 其他说明
# 管理界面地址: http://10.11.10.14
# 默认用户: vickai / 密码: [已在环境定义]
# 启用账号登录模式需要先设置 `nologin` 启动参数为 `0`
# command: flare --disable_login=0
# 环境变量参数
# FLARE_PORT=                     修改程序监听端口
# FLARE_USER=                     配置登陆模式下的账号
# FLARE_PASS=                     配置登陆模式下的密码
# FLARE_GUIDE=                    启用或禁用程序向导
# FLARE_DEPRECATED_NOTICE=        启用程序废弃功能提示
# FLARE_MINI_REQUEST=             启用服务端请求合并功能
# FLARE_DISABLE_LOGIN=            禁用登陆模式
# FLARE_OFFLINE=                  启用离线模式
# FLARE_EDITOR=                   启用在线编辑器功能
# FLARE_VISIBILITY=               首页是否需要登陆可见
# ==============================================================================

name: flare-vickai

services:
  flare-vickai:
    # --- 身份与镜像
    container_name: flare-vickai
    image: ghcr.io/vickai/flare-vickai:latest

    # --- 运行策略与系统级配置
    restart: always
    sysctls:
      - net.ipv6.conf.all.disable_ipv6=1
      - net.ipv6.conf.default.disable_ipv6=1

    # --- 网络与标识
    hostname: flare-vickai
    networks:
      macvlan:
        ipv4_address: 10.11.10.13

    # --- 环境配置
    environment:
      - TZ=Asia/Shanghai
      - FLARE_PORT=80
      - FLARE_USER=vickai
      - FLARE_PASS=admin
      - FLARE_COOKIE_SECRET=vickai1234567890vickai1234567890
      # 以下开关统一使用 1 (True) 和 0 (False)
      - FLARE_GUIDE=0
      - FLARE_DEPRECATED_NOTICE=0
      - FLARE_MINI_REQUEST=1
      - FLARE_DISABLE_LOGIN=0
      - FLARE_OFFLINE=0
      - FLARE_EDITOR=0
      - FLARE_VISIBILITY=0

    # --- 存储卷挂载
    volumes:
      - ./app:/app

    # --- 日志与资源限制
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
    cpus: 0.2
    mem_limit: 64m          # 内存限制收紧至 64M，足矣
    mem_reservation: 16m    # 保证基础 16M 内存预留

    # --- 健康检查
    healthcheck:
      test: ["CMD-SHELL", "wget -q -O /dev/null http://127.0.0.1:80/ || exit 1"]
      interval: 60s
      timeout: 5s
      retries: 3
      start_period: 10s

# ------------------------------------------------------------------------------
networks:
  macvlan:
    external: true

```

---

# Flare

Challenge all bookmarking apps and websites directories, Aim to Be a best performance monster.

🚧 **Code is being prepared and refactored, commits are slow.**

## Feature

**Simple**, **Fast**, **Lightweight** and super **Easy** to install and use.

- Written in Go (Golang) and a little Modern vanilla Javascript only.
- HTTP stack: [Echo](https://echo.labstack.com/) v5.
- Doesn't depend on any database or any complicated framework.
- Single executable, no dependencies required, good docker support.
- You can choose whether to enable various functions according to your needs: offline mode, weather, editor, account, and so on.

## ScreenShot

TBD

## Documentation

TBD

- Browse automatically generated program documentation:
    - `godoc --http=localhost:8080`



## Directory

```bash
├── build                   build script
├── cmd                     user cli/env parser
├── config                  config for app
│   ├── data                    data for app running
│   ├── define                  define for app launch
│   └── model                   data model for app
├── docker                  docker
├── embed                   resource (assets, template) for web
├── internal
│   ├── auth                user login
│   ├── fn                  fn utils
│   ├── logger              logger
│   ├── misc
│   │   ├── deprecated
│   │   ├── health
│   │   └── redir
│   ├── pages
│   │   ├── editor
│   │   ├── guide
│   │   └── home
│   ├── resources           static resource after minify
│   ├── server
│   ├── settings
│   └── version
└── main.go


# 下载依赖包
go mod tidy

# 直接运行项目
go run main.go

go run main.go --debug

# 执行编译脚本
go run build/build.go

# Windows PowerShell 调试登录认证
$env:FLARE_USER="admin"
$env:FLARE_PASS="admin"
$env:FLARE_DISABLE_LOGIN="0"
$env:FLARE_COOKIE_SECRET="vickai1234567890vickai1234567890"
go run main.go --debug

```