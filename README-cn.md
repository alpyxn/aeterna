# Aeterna（永恒）
**中文丨[English](README.md)**

***中文内容翻译进行中***
<p align="center">
  <img src="assets/hero.png" alt="Aeterna 永恒" width="600">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/React-61DAFB?style=flat-square&logo=react&logoColor=black" alt="React">
  <img src="https://img.shields.io/badge/SQLite-003B57?style=flat-square&logo=sqlite&logoColor=white" alt="SQLite">
  <img src="https://img.shields.io/badge/Docker-2496ED?style=flat-square&logo=docker&logoColor=white" alt="Docker">
  <img src="https://img.shields.io/badge/许可证-GPL--3.0-blue?style=flat-square" alt="GPL-3.0 许可证">
</p>

## 目录
- [核心功能](#核心功能)
- [界面截图](#界面截图)
- [快速开始](#快速开始)
- [管理与维护](#管理与维护)
- [配置说明](#配置说明)
- [反向代理模板](#反向代理模板)
- [安全性](#安全性)
- [架构设计](#架构设计)
- [项目结构](#项目结构)
- [免责声明](#免责声明)
- [支持项目](#支持项目)
- [许可证](#许可证)


*"你会为这个世界留下什么话？"*

---

'Aeterna 永恒'是一个"赛博遗嘱"。你留下消息，定期"签到"证明你还在。若你停止签到，你的消息便会被递送出去。

如此简单，却又如此重要。

永恒替你保管这些话语。它静候、它等待。当那一刻来临时，它负责递送。

## 核心功能

- **邮件递送**：若你未能按时签到，系统会自动将你的消息与文件递送给指定收件人。
- **Webhook集成**：当开关触发时，可联动外部服务（如家庭自动化、自定义脚本等）。
- **文件附件**：可为开关安全地附加敏感文档、照片或说明。
- **自动清理**：为最大限度保护隐私，附件在成功递送后会立即从服务器上删除。
- **一键安装**：提供全面的安装向导。
- **心跳系统**：可通过网页界面或邮件中的快捷链接轻松完成签到。
- **隐私优先架构**：消息与附件在你的私有服务器上以静态加密（AES-256-GCM）形式存储，仅在递送时刻解密。

## 界面截图

<p align="center">
  <table>
    <tr>
      <td align="center" style="padding: 12px;">
        <a href="assets/screenshots/dashboard.png" target="_blank">
          <img src="assets/screenshots/dashboard.png" alt="仪表盘" width="280">
        </a><br><sub><b>仪表盘</b></sub>
      </td>
      <td align="center" style="padding: 12px;">
        <a href="assets/screenshots/creatingswitch.png" target="_blank">
          <img src="assets/screenshots/creatingswitch.png" alt="创建开关" width="280">
        </a><br><sub><b>创建开关</b></sub>
      </td>
      <td align="center" style="padding: 12px;">
        <a href="assets/screenshots/settings.png" target="_blank">
          <img src="assets/screenshots/settings.png" alt="设置" width="280">
        </a><br><sub><b>设置</b></sub>
      </td>
    </tr>
  </table>
</p>



## 快速开始

```bash
git clone https://github.com/alpyxn/aeterna.git
cd aeterna
./install.sh
```

### 手动安装

如果你不希望使用自动化安装脚本，也可以使用我们已发布的 Docker 镜像直接部署 Aeterna。

#### Docker Compose（使用已发布镜像）

1. **新建目录并进入：**
   ```bash
   mkdir -p aeterna && cd aeterna
   ```

2. **创建所需目录及加密密钥：**
   ```bash
   mkdir -p data secrets
   openssl rand -base64 32 | tr -d '\n' > secrets/encryption_key
   chmod 600 secrets/encryption_key
   ```

3. **创建 .env 文件**
   ```bash
   SERVER_IP="$(curl -4fsS ifconfig.me || curl -4fsS icanhazip.com || true)"
   if [ -z "$SERVER_IP" ]; then
     echo "无法自动检测公网 IPv4 地址。请手动设置 SERVER_IP" >&2
     exit 1
   fi

   cat > .env <<EOF
   # 推荐使用你的域名，或使用服务器 IP
   DOMAIN=${SERVER_IP}
   ENV=production
   VITE_API_URL=/api
   # 必须与浏览器中访问的地址完全一致
   ALLOWED_ORIGINS=http://${SERVER_IP}:5000,http://localhost:5000,http://127.0.0.1:5000
   BASE_URL=http://${SERVER_IP}:5000
   PROXY_MODE=simple
   EOF
   ```

 注意事项:
  - 如果你使用域名，请相应设置：
    - `ALLOWED_ORIGINS=https://你的域名`
    - `BASE_URL=https://你的域名`
  - `ALLOWED_ORIGINS` 必须包含浏览器地址栏中显示的完整源（协议 + 主机 + 端口）。


4. **创建使用镜像仓库的 docker-compose.yml：**
   ```yaml
   services:
     backend:
       image: ghcr.io/alpyxn/aeterna-backend:main
       env_file:
         - .env
       environment:
         - DATABASE_PATH=/app/data/aeterna.db
         - ENV=production
         - ALLOWED_ORIGINS=${ALLOWED_ORIGINS:-*}
         - BASE_URL=${BASE_URL:-http://${DOMAIN}:5000}
       command: ["./main", "--encryption-key-file=/run/secrets/encryption_key"]
       secrets:
         - encryption_key
       volumes:
         - ./data:/app/data
       restart: always
       networks:
         - aeterna-net

     frontend:
       image: ghcr.io/alpyxn/aeterna-frontend:main
       depends_on:
         - backend
       restart: always
       networks:
         - aeterna-net

     proxy:
       image: nginx:alpine
       ports:
         - "5000:80"
       volumes:
         - ./proxy-simple.conf:/etc/nginx/conf.d/default.conf:ro
       depends_on:
         - backend
         - frontend
       restart: always
       networks:
         - aeterna-net

   secrets:
     encryption_key:
       file: ./secrets/encryption_key

   networks:
     aeterna-net:
       driver: bridge
   ```

5. **创建 proxy-simple.conf：**
   ```nginx
   server {
       listen 80;
       server_name localhost;

       resolver 127.0.0.11 valid=30s;
       set $backend_upstream http://backend:3000;

       location / {
           proxy_pass http://frontend:80;
           proxy_http_version 1.1;
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
           proxy_set_header X-Forwarded-Proto $scheme;
       }

       location /api/ {
           proxy_pass $backend_upstream;
           proxy_http_version 1.1;
           proxy_set_header Host $host;
           proxy_set_header X-Real-IP $remote_addr;
           proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
           proxy_set_header X-Forwarded-Proto $scheme;
       }
   }
   ```

6. **启动服务：**
   ```bash
   docker compose up -d
   ```

7. **打开 Aeterna：**
   - http://localhost:5000

如果你更倾向于使用 Docker Hub，请将镜像名称替换为：
- `docker.io/alpyxn/aeterna-backend:main`
- `docker.io/alpyxn/aeterna-frontend:main`

### 安装模式

在安装过程中，系统会提示你选择一种模式：

1. **生产模式（反向代理 + SSL）** - *推荐*
   - 脚本已针对 Nginx 与 Let's Encrypt 的自动配置进行了专门适配。
   - 你也可以根据需要调整为 Caddy、Apache 或 Traefik。
   - 包含安全标头与配置。

2. **开发模式（简单模式）** - *不推荐用于生产环境*
   - 直接在端口 5000 上运行（仅限 IP 地址访问）。
   - 无加密/SSL - 对敏感数据而言不安全。
   - 仅适用于本地测试或开发用途。

## 管理与维护

install.sh 脚本内置了管理命令：

| 命令     | 描述        |
|---------|-------------|
| `./install.sh --update` | 更新至最新版本 |
| `./install.sh --backup` | 创建数据与配置的完整备份 |
| `./install.sh --status` | 检查服务运行状态 |
| `./install.sh --uninstall` | 移除容器及相关安装文件 |

## 配置说明

安装向导会引导你完成基础配置：
   - 域名：你的域名（申请 SSL 证书所必需）。
   - 加密：自动生成唯一的 AES-256 密钥。
     
   SMTP 设置（发送邮件所必需）需在安装完成后，通过应用程序的设置菜单进行配置。这样便于进行实时测试与更灵活的管理。

## 反向代理模板

以下文档中提供了适用于  **Nginx**, **Traefik**,  **Caddy** 的现成示例：

- [`docs/proxy-templates.md`](docs/proxy-templates.md)

该文档还包含必需的 .env 变量（ALLOWED_ORIGINS、BASE_URL）配置说明与部署注意事项。



## 安全性

Aeterna 已自动处理相关安全事项：
- **加密**：消息与文件附件使用 AES-256-GCM 算法进行静态加密。
- **密钥管理**：加密密钥以安全方式生成，并存储在 secrets/encryption_key 文件中。它绝不会暴露在环境变量或配置文件中。
- **数据清理**：文件附件在成功递送给收件人后，会从磁盘上永久删除。
- **SSL**：通过 Let's Encrypt 自动管理证书（生产模式下）。

## 架构设计

```
backend/     Go 语言 API 服务端
frontend/    React 前端应用  
```

两个组件均可在 Docker 容器中运行，也可以原生运行。存储层使用 SQLite（单文件数据库）。你可以使用任意反向代理（如 Nginx、Caddy、Apache）将它们整合在一起并提供 SSL 服务。

## 项目结构

```bash
.
├── assets/             # 图片与设计资源
├── backend/            # Go 源代码
│   ├── cmd/            # 程序入口 (main.go)
│   └── internal/       # 核心业务逻辑、处理器与服务
├── frontend/           # React 前端源码
│   ├── src/            # 组件、页面与钩子函数
│   └── public/         # Web 静态资源
├── secrets/            # 加密密钥文件（Git 忽略）
├── docker-compose.*    # 适用于不同场景的部署配置
└── install.sh          # 自动化安装脚本
```


## 免责声明

Aeterna 处理的是敏感数据，且涉及高风险后果（若你停止签到则自动递送）。 **请阅读完整的 [免责声明](disclaimer.md)** ，了解责任限制、你应承担的部署与合规责任，以及本软件不提供的保证。

## 支持项目

💖 支持本项目

如果你觉得这个项目有用，不妨考虑支持它的开发。每一份贡献都弥足珍贵！


****请参考原项目****
https://github.com/alpyxn/aeterna
| Asset | Network | Address |
| --- | --- | --- |
| **Bitcoin (BTC)** | Bitcoin |
| **Solana (SOL)** | Solana |
| **USDT** | ERC-20 / BEP-20 |
| **Monero (XMR)** | Monero |


## 许可证

GPL-3.0

---

*以拉丁语中意为"永恒"的词汇命名 —— 因为有些话语，注定要比我们的生命更长久。*
