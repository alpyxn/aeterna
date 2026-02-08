# Aeterna

<p align="center">
  <img src="https://img.shields.io/badge/Go-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/React-61DAFB?style=flat-square&logo=react&logoColor=black" alt="React">
  <img src="https://img.shields.io/badge/PostgreSQL-4169E1?style=flat-square&logo=postgresql&logoColor=white" alt="PostgreSQL">
  <img src="https://img.shields.io/badge/Docker-2496ED?style=flat-square&logo=docker&logoColor=white" alt="Docker">
  <img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="MIT License">
</p>

*"What words would you leave behind?"*

---

Aeterna is a dead man's switch. You write messages. You check in regularly. If you stop checking in, your messages are delivered.

It's that simple. And that important.

## The Concept

Some things need to be said, but only at the right time. A password that should reach your partner. A letter that waits for the right moment. Instructions that matter only when you're not there to give them.

Aeterna holds these words. It watches. It waits. And when the time comes, it delivers.

## Quick Start

```bash
curl -fsSL https://raw.githubusercontent.com/alpyxn/aeterna/main/install.sh | bash
```

Requires: Linux server, domain name, ports 80/443 open.

## Manual Setup

```bash
git clone https://github.com/alpyxn/aeterna.git
cd aeterna
cp .env.production.example .env
# Configure your settings
docker compose -f docker-compose.prod.yml up -d
```

## How It Works

1. Write a message, set a recipient
2. Choose your check-in interval (1 hour to 1 year)
3. Check in before the timer expires
4. Miss a check-in, and your message is delivered

## Configuration

| Variable | Purpose |
|----------|---------|
| `DOMAIN` | Your domain name |
| `ACME_EMAIL` | For SSL certificates |
| `DB_USER`, `DB_PASS`, `DB_NAME` | Database credentials |
| `ENCRYPTION_KEY` | Message encryption key |

SMTP settings are configured through the web interface after installation.

## Security

All messages are encrypted at rest using AES-256. Authentication is rate-limited. HTTPS is automatic via Let's Encrypt. Your words stay private until they're meant to be read.

## Architecture

```
backend/     Go API server
frontend/    React application  
```

Both containerized. PostgreSQL for storage. Traefik for routing.

## Updates

```bash
git pull && docker compose -f docker-compose.prod.yml up -d --build
```

## License

MIT

---

*Named for the Latin word meaning "eternal" — because some messages are meant to outlast us.*
