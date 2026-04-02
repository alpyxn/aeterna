# Aeterna

<p align="center">
  <img src="assets/hero.png" alt="Aeterna Logo" width="600">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/React-61DAFB?style=flat-square&logo=react&logoColor=black" alt="React">
  <img src="https://img.shields.io/badge/SQLite-003B57?style=flat-square&logo=sqlite&logoColor=white" alt="SQLite">
  <img src="https://img.shields.io/badge/Docker-2496ED?style=flat-square&logo=docker&logoColor=white" alt="Docker">
  <img src="https://img.shields.io/badge/License-GPL--3.0-blue?style=flat-square" alt="GPL-3.0 License">
</p>

## Table of Contents
- [Key Features](#key-features)
- [Screenshots](#screenshots)
- [Quick Start](#quick-start)
- [Management](#management)
- [Configuration](#configuration)
- [Reverse Proxy Templates](#reverse-proxy-templates)
- [Security](#security)
- [Architecture](#architecture)
- [Project Structure](#project-structure)
- [License](#license)


*"What words would you leave behind?"*

---

Aeterna is a dead man's switch. You write messages. You check in regularly. If you stop checking in, your messages are delivered.

It's that simple. And that important.

Aeterna holds these words. It watches. It waits. And when the time comes, it delivers.

## Key Features

- **Email Delivery**: Automatic delivery of your messages and files to your loved ones if you fail to check in.
- **Webhook Integration**: Trigger external services (home automation, custom scripts, etc.) when your switch is activated.
- **File Attachments**: Securely attach sensitive documents, photos, or instructions to your switches.
- **Auto-Cleanup**: Attachments are automatically deleted from the server immediately after delivery for maximum privacy.
- **One-Click Install**: Comprehensive installation wizard.
- **Heartbeat System**: Simple check-in mechanism via web UI or a quick-link from your email.
- **Privacy-Focused Architecture**: Messages and attachments are encrypted at rest (AES-256-GCM) on your private server, ensuring they are only decrypted at the moment of delivery.

## Screenshots

<p align="center">
  <table>
    <tr>
      <td align="center" style="padding: 12px;">
        <a href="assets/screenshots/dashboard.png" target="_blank">
          <img src="assets/screenshots/dashboard.png" alt="Dashboard" width="280">
        </a><br><sub><b>Dashboard</b></sub>
      </td>
      <td align="center" style="padding: 12px;">
        <a href="assets/screenshots/creatingswitch.png" target="_blank">
          <img src="assets/screenshots/creatingswitch.png" alt="Creating a Switch" width="280">
        </a><br><sub><b>Creating a Switch</b></sub>
      </td>
      <td align="center" style="padding: 12px;">
        <a href="assets/screenshots/settings.png" target="_blank">
          <img src="assets/screenshots/settings.png" alt="Settings" width="280">
        </a><br><sub><b>Settings</b></sub>
      </td>
    </tr>
  </table>
</p>



## Quick Start

```bash
git clone https://github.com/alpyxn/aeterna.git
cd aeterna
./install.sh
```

### Manual Installation

If you prefer not to use the automated installation script, you can deploy Aeterna directly with our published Docker images.

#### Docker Compose (Published Images)

1. **Create a new folder and move into it:**
   ```bash
   mkdir -p aeterna && cd aeterna
   ```

2. **Create required directories and encryption key:**
   ```bash
   mkdir -p data secrets
   openssl rand -base64 32 | tr -d '\n' > secrets/encryption_key
   chmod 600 secrets/encryption_key
   ```

3. **Create `.env`:**
   ```bash
   SERVER_IP="$(curl -4fsS ifconfig.me || curl -4fsS icanhazip.com || true)"
   if [ -z "$SERVER_IP" ]; then
     echo "Could not detect public IPv4 automatically. Set SERVER_IP manually." >&2
     exit 1
   fi

   cat > .env <<EOF
   # Use your public domain (recommended) or server IP
   DOMAIN=${SERVER_IP}
   ENV=production
   VITE_API_URL=/api
   # Must match exactly what you open in the browser
   ALLOWED_ORIGINS=http://${SERVER_IP}:5000,http://localhost:5000,http://127.0.0.1:5000
   BASE_URL=http://${SERVER_IP}:5000
   PROXY_MODE=simple
   EOF
   ```

  Notes:
  - If you use a domain, set:
    - `ALLOWED_ORIGINS=https://your-domain`
    - `BASE_URL=https://your-domain`
  - `ALLOWED_ORIGINS` must include the exact origin shown in your browser address bar (scheme + host + port).


4. **Create `docker-compose.yml` using package images:**
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

5. **Create `proxy-simple.conf`:**
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

6. **Start the stack:**
   ```bash
   docker compose up -d
   ```

7. **Open Aeterna:**
   - http://localhost:5000

If you prefer Docker Hub, replace image names with:
- `docker.io/alpyxn/aeterna-backend:main`
- `docker.io/alpyxn/aeterna-frontend:main`

### Installation Modes

During installation, you will be prompted to choose a mode:

1. **Production (Reverse Proxy + SSL)** - *Recommended*
   - Specifically configured to work with Nginx and Let's Encrypt automatically via the script.
   - You can also adapt this for Caddy, Apache, or Traefik.
   - Secure headers and configuration

2. **Development (Simple)** - *Not Recommended for Production*
   - Runs directly on port 5000 (IP address only)
   - **No encryption/SSL** - insecure for sensitive data
   - Useful only for local testing or development

## Management

The `install.sh` script includes management commands:

| Command | Description |
|---------|-------------|
| `./install.sh --update` | Update to the latest version |
| `./install.sh --backup` | Create a full backup of data and config |
| `./install.sh --status` | Check service health and status |
| `./install.sh --uninstall` | Remove containers and installation |

## Configuration

The installer guides you through basic configuration:
- **Domain**: Your domain name (required for SSL)
- **Encryption**: Automatically generates a unique AES-256 key

**SMTP Settings** (required for sending emails) are configured post-installation through the application's **Settings** menu. This allows for live testing and easier management.

## Reverse Proxy Templates

Ready-to-use examples for **Nginx**, **Traefik**, and **Caddy** are available in:

- [`docs/proxy-templates.md`](docs/proxy-templates.md)

This document also includes required `.env` values (`ALLOWED_ORIGINS`, `BASE_URL`) and deployment notes.



## Security

Aeterna handles security automatically:
- **Encryption**: Messages and file attachments are encrypted at rest using AES-256-GCM.
- **Key Management**: The encryption key is generated securely and stored in `secrets/encryption_key`. It is **never** exposed in environment variables or configuration files.
- **Data Pruning**: File attachments are permanently deleted from the disk after successful delivery to the recipient.
- **SSL**: Automatic certificate management via Let's Encrypt (in Production mode).

## Architecture

```
backend/     Go API server
frontend/    React application  
```

Both components can run in Docker containers or natively. SQLite is used for storage (single file database). You can use **any reverse proxy** (proxy, caddy, apache) to serve them together and provide SSL.

## Project Structure

```bash
.
├── assets/             # Images and design assets
├── backend/            # Go source code
│   ├── cmd/            # Entry points (main.go)
│   └── internal/       # Core business logic, handlers, and services
├── frontend/           # React frontend source
│   ├── src/            # Components, pages, and hooks
│   └── public/         # Static assets for the web
├── secrets/            # Encryption keys (ignored by git)
├── docker-compose.*    # Deployment various configurations
└── install.sh          # Automated installation script
```


## License

GPL-3.0

---

*Named for the Latin word meaning "eternal" — because some messages are meant to outlast us.*
