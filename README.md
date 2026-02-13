# Aeterna

<p align="center">
  <img src="https://img.shields.io/badge/Go-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/React-61DAFB?style=flat-square&logo=react&logoColor=black" alt="React">
  <img src="https://img.shields.io/badge/SQLite-003B57?style=flat-square&logo=sqlite&logoColor=white" alt="SQLite">
  <img src="https://img.shields.io/badge/Docker-2496ED?style=flat-square&logo=docker&logoColor=white" alt="Docker">
  <img src="https://img.shields.io/badge/License-GPL--3.0-blue?style=flat-square" alt="GPL-3.0 License">
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
git clone https://github.com/alpyxn/aeterna.git
cd aeterna
./install.sh
```

### Installation Modes

During installation, you will be prompted to choose a mode:

1. **Production (Nginx + SSL)** - *Recommended*
   - Automatic HTTPS with Let's Encrypt
   - Nginx reverse proxy
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

The installer guides you through interactive configuration:
- **Domain**: Your domain name (required for SSL)
- **SMTP**: Email settings for notifications
- **Database**: Location for SQLite data (default: `./data`)

Post-installation, settings can be found in the `.env` file.

## Security

Aeterna handles security automatically:
- **Encryption**: Messages are encrypted at rest (AES-256-GCM).
- **Key Management**: The encryption key is generated securely and stored in `secrets/encryption_key`. It is **never** exposed in environment variables.
- **SSL**: Automatic certificate management via Let's Encrypt (in Production mode).

## Architecture

```
backend/     Go API server
frontend/    React application  
```

Both containerized. SQLite for storage (single file database). nginx for reverse proxy and SSL.

## License

GPL-3.0

---

*Named for the Latin word meaning "eternal" — because some messages are meant to outlast us.*
