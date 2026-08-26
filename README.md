# Trae-Sign

TRAE 每日自动签到 Web 服务。独立运行，不依赖 TRAE 客户端——通过官方 OAuth 流程登录账号，使用设备密钥签名刷新 token，自动完成每日签到与积分查询。

## 功能

- **独立签到**：官方 OAuth 网页登录添加账号，内置设备密钥签名（ECDSA P-256），token 过期自动刷新
- **定时签到**：每个账号可独立设置签到时间（默认 08:00），到点自动执行
- **风控自愈**：遇 9074（参与用户太多）/ 9095 设备级风控时自动轮换数字设备 ID 重试
- **积分查询**：签到后自动更新余额，支持手动刷新
- **消息推送**：PushPlus 签到结果推送（支持全局默认 token + 账号级覆盖）
- **Web 登录**：首次使用设置管理账号密码，会话 7 天有效，设置页可随时修改
- **零依赖后端**：纯 Go 标准库，单二进制 + JSON 文件存储，内存占用极低

## 快速部署

### Docker（推荐）

```bash
docker run -d \
  --name trae-signin \
  -p 9090:9090 \
  -v /opt/trae-signin/data:/app/data \
  -e TZ=Asia/Shanghai \
  --restart unless-stopped \
  ghcr.io/lijsjust2/trae-signin-web:latest
```

支持 `linux/amd64` 与 `linux/arm64`（树莓派、ARM 云主机）。

### 离线部署（Release 镜像文件）

从 [Releases](https://github.com/lijsjust2/Trae-Sign/releases) 下载对应架构的 `tar.gz`：

```bash
docker load -i trae-signin-web_1.0.0_linux_amd64.tar.gz
docker tag trae-signin:amd64 trae-signin-web:latest
docker run -d --name trae-signin -p 9090:9090 \
  -v /opt/trae-signin/data:/app/data \
  -e TZ=Asia/Shanghai --restart unless-stopped \
  trae-signin-web:latest
```

### 二进制直接运行

```bash
./trae-signin-web -addr :9090 -data ./data/data.json -dist ./web/dist
```

## 使用

1. 打开 `http://服务器IP:9090`，首次访问设置管理账号密码
2. 「账号」页 → 添加账号：生成登录链接 → 浏览器登录 TRAE → 粘贴回调链接完成添加
3. 可选：设置备注名、签到时间、PushPlus Token
4. 之后每天自动签到，页面上可查看账号状态、积分余额和签到日志

## 数据与安全

- 所有数据（账号凭证、设备密钥、签到记录）保存在挂载目录的 `data/data.json`，登录凭证在 `data/auth.json`
- 密码加盐多轮哈希存储；API 未登录一律 401
- **公网部署请自行套反向代理加 HTTPS**，或仅监听 `127.0.0.1` 走 SSH 隧道

## 构建

```bash
# 前端
cd web && npm ci && npm run build

# 后端（含前端静态资源托管）
cd .. && go build -o trae-signin-web .
```

Docker 多架构镜像由 GitHub Actions 自动构建：推送 `v*` tag 或手动触发 workflow，镜像推送至 GHCR，同时产出可 `docker load` 的架构镜像文件并附到 Release。

## 说明

本项目仅供个人学习与账号管理用途，请遵守 TRAE 服务条款。
