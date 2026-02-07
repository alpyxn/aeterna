# Aeterna - Dead Man's Switch

A secure, self-hosted "dead man's switch" that sends pre-written messages to designated recipients if you stop checking in.

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Docker](https://img.shields.io/badge/docker-ready-brightgreen.svg)

## 🚀 Quick Install (VPS/VDS)

One command to deploy on any Linux VPS:

```bash
curl -fsSL https://raw.githubusercontent.com/alpyxn/aeterna/main/install.sh | bash
```

The wizard will:
- ✅ Install Docker & Docker Compose if needed
- ✅ Configure your domain & SSL certificates automatically
- ✅ Set up PostgreSQL database
- ✅ Deploy the application

**Requirements:**
- Linux VPS (Ubuntu/Debian recommended)
- Domain pointed to your server's IP
- Ports 80 and 443 open

## 📦 Manual Installation

### 1. Clone the repository
```bash
git clone https://github.com/alpyxn/aeterna.git
cd aeterna
```

### 2. Configure environment
```bash
cp .env.production.example .env
# Edit .env with your settings
nano .env
```

### 3. Deploy
```bash
docker compose -f docker-compose.prod.yml up -d
```

## 💻 Development Setup

```bash
# Clone repository
git clone https://github.com/alpyxn/aeterna.git
cd aeterna

# Copy environment file
cp .env.example .env

# Start with Docker Compose
docker compose up --build

# Access at http://localhost:5173
```

## 🔧 Configuration

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `DOMAIN` | Your domain name | ✅ Production |
| `ACME_EMAIL` | Email for SSL certificates | ✅ Production |
| `DB_USER` | PostgreSQL username | ✅ |
| `DB_PASS` | PostgreSQL password | ✅ |
| `DB_NAME` | PostgreSQL database name | ✅ |
| `SMTP_HOST` | SMTP server host | ⚡ For email |
| `SMTP_PORT` | SMTP server port | ⚡ For email |
| `SMTP_USER` | SMTP username | ⚡ For email |
| `SMTP_PASS` | SMTP password | ⚡ For email |

### SMTP Configuration

Configure email in **Settings** after installation, or set environment variables:

**Gmail:**
```
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your@gmail.com
SMTP_PASS=your_app_password
```

**Yandex:**
```
SMTP_HOST=smtp.yandex.com
SMTP_PORT=465
SMTP_USER=your@yandex.com
SMTP_PASS=your_password
```

## 📚 How It Works

1. **Create a Switch** - Write a message and set a recipient
2. **Set Timer** - Choose how long before the switch triggers (1 hour to 1 year)
3. **Stay Alive** - Click "I'm Alive" before the timer runs out
4. **Auto-Delivery** - If you miss the deadline, your message is sent

## 🛡️ Security

- Master password protection for all management
- Encrypted message storage
- Rate-limited authentication
- HTTPS/TLS with Let's Encrypt
- No external tracking

## 📁 Project Structure

```
aeterna/
├── backend/           # Go API server
│   ├── cmd/server/    # Main entry point
│   ├── internal/      # Core logic
│   └── Dockerfile
├── frontend/          # React + Vite frontend
│   ├── src/
│   └── Dockerfile
├── docker-compose.yml          # Development
├── docker-compose.prod.yml     # Production with Traefik
└── install.sh                  # Auto-installer
```

## 🔄 Updates

```bash
cd /opt/aeterna  # or your install directory
git pull
docker compose -f docker-compose.prod.yml up -d --build
```

## 📝 License

MIT License - see [LICENSE](LICENSE) for details.

---

Made with ❤️ for peace of mind.
