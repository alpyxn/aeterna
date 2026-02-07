#!/bin/bash

#######################################
# Aeterna - Dead Man's Switch
# One-Click Installation Script
#######################################

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
NC='\033[0m' # No Color
BOLD='\033[1m'
DIM='\033[2m'

# Script version
VERSION="1.3.0"

# Default values
PROXY_MODE=""  # traefik, nginx, or simple
PROXY_MODE_SET=false

# ASCII Art
print_banner() {
    echo -e "${CYAN}"
    echo "    _         _                        "
    echo "   / \   ___| |_ ___ _ __ _ __   __ _ "
    echo "  / _ \ / _ \ __/ _ \ '__| '_ \ / _\` |"
    echo " / ___ \  __/ ||  __/ |  | | | | (_| |"
    echo "/_/   \_\___|\__\___|_|  |_| |_|\__,_|"
    echo ""
    echo -e "${NC}${BOLD}Dead Man's Switch - Installation Wizard v${VERSION}${NC}"
    echo ""
}

# Print colored messages
info() { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[✓]${NC} $1"; }
warning() { echo -e "${YELLOW}[!]${NC} $1"; }
error() { echo -e "${RED}[✗]${NC} $1"; exit 1; }
step() { echo -e "${MAGENTA}[→]${NC} $1"; }

# Print help message
print_help() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --help, -h       Show this help message"
    echo "  --traefik        Use Traefik reverse proxy (default, standalone with SSL)"
    echo "  --nginx          Use nginx reverse proxy (for existing nginx servers)"
    echo "  --simple         Simple mode: IP-only on port 5000 (no SSL, for testing)"
    echo "  --uninstall      Remove Aeterna installation"
    echo "  --backup         Create backup of current installation"
    echo "  --update         Update existing installation"
    echo "  --status         Check status of Aeterna services"
    echo "  --version, -v    Show version"
    echo ""
    echo "Examples:"
    echo "  $0               Run installation wizard with interactive mode selection"
    echo "  $0 --traefik     Install with Traefik (automatic SSL)"
    echo "  $0 --nginx       Install behind existing nginx"
    echo "  $0 --simple      Install on port 5000 without SSL (testing/dev)"
    echo ""
}

# Check if command exists
check_command() {
    if ! command -v "$1" &> /dev/null; then
        return 1
    fi
    return 0
}

# Get server's public IP
get_server_ip() {
    curl -s --connect-timeout 5 ifconfig.me 2>/dev/null || \
    curl -s --connect-timeout 5 icanhazip.com 2>/dev/null || \
    curl -s --connect-timeout 5 ipecho.net/plain 2>/dev/null || \
    echo "unknown"
}

# Check if port is in use
check_port() {
    local port=$1
    if command -v ss &> /dev/null; then
        ss -tuln | grep -q ":$port " && return 0
    elif command -v netstat &> /dev/null; then
        netstat -tuln | grep -q ":$port " && return 0
    elif command -v lsof &> /dev/null; then
        lsof -i:$port &>/dev/null && return 0
    fi
    return 1
}

# Detect if nginx is running
detect_nginx() {
    if check_port 80 || check_port 443; then
        if pgrep -x "nginx" > /dev/null 2>&1; then
            return 0
        fi
    fi
    return 1
}

# Check system requirements
check_requirements() {
    echo ""
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}  System Requirements Check${NC}"
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    
    local requirements_met=true
    
    # Check curl
    if ! check_command curl; then
        warning "curl not found. Installing..."
        sudo apt-get update && sudo apt-get install -y curl
        success "curl installed"
    else
        success "curl found"
    fi
    
    # Check openssl
    if ! check_command openssl; then
        warning "openssl not found. Installing..."
        sudo apt-get update && sudo apt-get install -y openssl
        success "openssl installed"
    else
        success "openssl found"
    fi
    
    # Check Docker
    if ! check_command docker; then
        warning "Docker not found. Installing..."
        curl -fsSL https://get.docker.com | sh
        sudo usermod -aG docker $USER
        success "Docker installed"
        warning "You may need to log out and back in for Docker group changes to take effect"
    else
        success "Docker found: $(docker --version | cut -d' ' -f3 | tr -d ',')"
    fi
    
    # Check Docker Compose
    if ! docker compose version &> /dev/null; then
        warning "Docker Compose v2 not found. Installing..."
        sudo apt-get update && sudo apt-get install -y docker-compose-plugin
        success "Docker Compose installed"
    else
        success "Docker Compose found: $(docker compose version --short)"
    fi
    
    # Check Git
    if ! check_command git; then
        warning "Git not found. Installing..."
        sudo apt-get update && sudo apt-get install -y git
        success "Git installed"
    else
        success "Git found"
    fi
    
    # Check available ports based on mode
    echo ""
    info "Checking port availability..."
    
    case "$PROXY_MODE" in
        traefik)
            if check_port 80; then
                warning "Port 80 is already in use!"
                if detect_nginx; then
                    echo -e "  ${CYAN}nginx detected!${NC} Consider using ${BOLD}--nginx${NC} or ${BOLD}--simple${NC} mode."
                fi
                requirements_met=false
            else
                success "Port 80 is available"
            fi
            
            if check_port 443; then
                warning "Port 443 is already in use!"
                requirements_met=false
            else
                success "Port 443 is available"
            fi
            ;;
        nginx)
            if check_port 8080; then
                warning "Port 8080 is in use (needed for backend)"
                requirements_met=false
            else
                success "Port 8080 is available (backend)"
            fi
            
            if check_port 8081; then
                warning "Port 8081 is in use (needed for frontend)"
                requirements_met=false
            else
                success "Port 8081 is available (frontend)"
            fi
            ;;
        simple)
            if check_port 5000; then
                warning "Port 5000 is already in use!"
                requirements_met=false
            else
                success "Port 5000 is available"
            fi
            ;;
    esac
    
    # Check available disk space (minimum 2GB)
    local available_space=$(df -BG / | awk 'NR==2 {print $4}' | tr -d 'G')
    if [ "$available_space" -lt 2 ]; then
        warning "Low disk space: ${available_space}GB available (minimum 2GB recommended)"
        requirements_met=false
    else
        success "Disk space: ${available_space}GB available"
    fi
    
    # Check available memory (minimum 512MB)
    local available_mem=$(free -m | awk 'NR==2 {print $7}')
    if [ "$available_mem" -lt 512 ]; then
        warning "Low memory: ${available_mem}MB available (minimum 512MB recommended)"
    else
        success "Memory: ${available_mem}MB available"
    fi
    
    if [ "$requirements_met" = false ]; then
        echo ""
        read -p "Some requirements are not met. Continue anyway? [y/N]: " continue_choice
        if [ "$continue_choice" != "y" ] && [ "$continue_choice" != "Y" ]; then
            error "Installation cancelled due to unmet requirements."
        fi
    fi
}

# Get user input with default value
prompt() {
    local var_name=$1
    local prompt_text=$2
    local default_value=$3
    local is_secret=$4
    
    if [ "$is_secret" = "true" ]; then
        read -sp "$prompt_text [$default_value]: " value
        echo ""
    else
        read -p "$prompt_text [$default_value]: " value
    fi
    
    if [ -z "$value" ]; then
        value=$default_value
    fi
    
    eval "$var_name='$value'"
}

# Prompt yes/no question
prompt_yn() {
    local prompt_text=$1
    local default=$2
    local result
    
    if [ "$default" = "y" ]; then
        read -p "$prompt_text [Y/n]: " result
        if [ -z "$result" ] || [ "$result" = "y" ] || [ "$result" = "Y" ]; then
            return 0
        fi
    else
        read -p "$prompt_text [y/N]: " result
        if [ "$result" = "y" ] || [ "$result" = "Y" ]; then
            return 0
        fi
    fi
    return 1
}

# Generate random password
generate_password() {
    openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 24
}

# Check DNS configuration
check_dns() {
    local domain=$1
    local server_ip=$(get_server_ip)
    
    info "Checking DNS configuration for $domain..."
    
    if ! check_command dig; then
        warning "dig not installed, skipping DNS check"
        return
    fi
    
    local domain_ip=$(dig +short "$domain" 2>/dev/null | head -n1)
    
    if [ -z "$domain_ip" ]; then
        warning "Could not resolve $domain"
        echo -e "  ${DIM}Make sure DNS A record points to this server${NC}"
    elif [ "$server_ip" = "$domain_ip" ]; then
        success "DNS correctly configured ($domain → $server_ip)"
    else
        warning "DNS mismatch detected!"
        echo -e "  ${DIM}Server IP:  $server_ip${NC}"
        echo -e "  ${DIM}Domain IP:  $domain_ip${NC}"
        echo -e "  ${DIM}SSL certificate may fail if DNS is not pointing to this server${NC}"
    fi
}

# Check firewall status
check_firewall() {
    echo ""
    info "Checking firewall configuration..."
    
    if check_command ufw; then
        if sudo ufw status | grep -q "Status: active"; then
            case "$PROXY_MODE" in
                traefik)
                    local http_allowed=$(sudo ufw status | grep -E "80/tcp|80 " | grep -c "ALLOW" || echo "0")
                    local https_allowed=$(sudo ufw status | grep -E "443/tcp|443 " | grep -c "ALLOW" || echo "0")
                    
                    if [ "$http_allowed" -eq 0 ] || [ "$https_allowed" -eq 0 ]; then
                        warning "UFW firewall is active but ports 80/443 may not be open"
                        if prompt_yn "Open ports 80 and 443 in firewall?" "y"; then
                            sudo ufw allow 80/tcp
                            sudo ufw allow 443/tcp
                            success "Firewall ports opened"
                        fi
                    else
                        success "Firewall configured correctly (ports 80, 443 open)"
                    fi
                    ;;
                nginx)
                    success "nginx mode: No additional firewall configuration needed"
                    ;;
                simple)
                    local port_allowed=$(sudo ufw status | grep -E "5000/tcp|5000 " | grep -c "ALLOW" || echo "0")
                    if [ "$port_allowed" -eq 0 ]; then
                        warning "UFW firewall is active but port 5000 may not be open"
                        if prompt_yn "Open port 5000 in firewall?" "y"; then
                            sudo ufw allow 5000/tcp
                            success "Firewall port 5000 opened"
                        fi
                    else
                        success "Firewall configured correctly (port 5000 open)"
                    fi
                    ;;
            esac
        else
            success "UFW firewall is inactive"
        fi
    elif check_command firewall-cmd; then
        if sudo firewall-cmd --state 2>/dev/null | grep -q "running"; then
            warning "firewalld is active. Ensure required ports are open."
        fi
    else
        success "No firewall detected"
    fi
}

# Select proxy mode
select_proxy_mode() {
    echo ""
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}  Installation Mode${NC}"
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    
    # Auto-detect nginx
    if detect_nginx; then
        echo -e "  ${YELLOW}⚠ nginx detected on this server!${NC}"
        echo ""
    fi
    
    echo -e "  ${CYAN}1)${NC} ${BOLD}Traefik${NC} - Standalone with automatic SSL"
    echo -e "     ${DIM}• For dedicated servers with a domain${NC}"
    echo -e "     ${DIM}• Automatic Let's Encrypt certificates${NC}"
    echo -e "     ${DIM}• Uses ports 80 and 443${NC}"
    echo ""
    echo -e "  ${CYAN}2)${NC} ${BOLD}nginx${NC} - Behind existing nginx"
    echo -e "     ${DIM}• For servers with existing websites${NC}"
    echo -e "     ${DIM}• You manage SSL via nginx/certbot${NC}"
    echo -e "     ${DIM}• Uses internal ports 8080/8081${NC}"
    echo ""
    echo -e "  ${CYAN}3)${NC} ${BOLD}Simple${NC} - IP only, no SSL (testing/dev)"
    echo -e "     ${DIM}• No domain required, access via IP${NC}"
    echo -e "     ${DIM}• No SSL/HTTPS (not for production!)${NC}"
    echo -e "     ${DIM}• Runs on port 5000${NC}"
    echo ""
    
    local default_choice="1"
    if detect_nginx; then
        default_choice="2"
    fi
    
    read -p "Select installation mode [${default_choice}]: " mode_choice
    mode_choice=${mode_choice:-$default_choice}
    
    case $mode_choice in
        1)
            PROXY_MODE="traefik"
            success "Using Traefik mode (standalone with SSL)"
            ;;
        2)
            PROXY_MODE="nginx"
            success "Using nginx mode (behind existing nginx)"
            ;;
        3)
            PROXY_MODE="simple"
            success "Using Simple mode (IP only, port 5000)"
            warning "This mode has no SSL - not recommended for production!"
            ;;
        *)
            PROXY_MODE="traefik"
            success "Using Traefik mode (standalone with SSL)"
            ;;
    esac
}

# Collect configuration
collect_config() {
    echo ""
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}  Configuration${NC}"
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    local server_ip=$(get_server_ip)
    
    # Domain/IP Configuration
    echo ""
    if [ "$PROXY_MODE" = "simple" ]; then
        echo -e "${CYAN}📌 Access Configuration${NC}"
        echo -e "${DIM}Your server IP: ${BOLD}$server_ip${NC}"
        echo ""
        
        DOMAIN="$server_ip"
        ACME_EMAIL=""
        ALLOWED_ORIGINS="http://$server_ip:5000"
        
        success "Will be accessible at: http://$server_ip:5000"
    else
        echo -e "${CYAN}📌 Domain Configuration${NC}"
        echo -e "${DIM}Your domain must point to this server's IP address${NC}"
        echo ""
        
        prompt DOMAIN "Enter your domain (e.g., aeterna.example.com)" ""
        
        if [ -z "$DOMAIN" ]; then
            error "Domain is required!"
        fi
        
        # Check DNS
        check_dns "$DOMAIN"
        
        ALLOWED_ORIGINS="https://$DOMAIN"
        
        # Email for SSL (only for Traefik mode)
        if [ "$PROXY_MODE" = "traefik" ]; then
            prompt ACME_EMAIL "Email for SSL certificates" "admin@$DOMAIN"
        else
            ACME_EMAIL="admin@$DOMAIN"
        fi
    fi
    
    # Database Configuration
    echo ""
    echo -e "${CYAN}🗄️  Database Configuration${NC}"
    echo -e "${DIM}PostgreSQL credentials (auto-generated passwords recommended)${NC}"
    echo ""
    
    local default_db_pass=$(generate_password)
    prompt DB_USER "Database username" "aeterna"
    prompt DB_PASS "Database password" "$default_db_pass" "true"
    prompt DB_NAME "Database name" "aeterna"
    
    # SMTP Configuration
    echo ""
    echo -e "${CYAN}📧 Email (SMTP) Configuration${NC}"
    echo -e "${DIM}Required for sending Dead Man's Switch notifications${NC}"
    echo ""
    
    if prompt_yn "Configure SMTP now? (can be done later in Settings)" "y"; then
        CONFIGURE_SMTP=true
        
        echo ""
        echo -e "${DIM}Common SMTP providers:${NC}"
        echo -e "${DIM}  Gmail:     smtp.gmail.com:587${NC}"
        echo -e "${DIM}  Outlook:   smtp-mail.outlook.com:587${NC}"
        echo -e "${DIM}  SendGrid:  smtp.sendgrid.net:587${NC}"
        echo ""
        
        prompt SMTP_HOST "SMTP Server" "smtp.gmail.com"
        prompt SMTP_PORT "SMTP Port" "587"
        prompt SMTP_USER "SMTP Username (email)" ""
        prompt SMTP_PASS "SMTP Password (app password)" "" "true"
        prompt SMTP_FROM "From Email Address" "$SMTP_USER"
        prompt SMTP_FROM_NAME "From Name" "Aeterna"
    else
        CONFIGURE_SMTP=false
        warning "SMTP not configured. You can set it up later in the application settings."
    fi
    
    # Owner Email for Reminders
    echo ""
    echo -e "${CYAN}👤 Owner Configuration${NC}"
    echo -e "${DIM}Your email for receiving reminder notifications${NC}"
    echo ""
    
    local default_owner_email="${ACME_EMAIL:-admin@example.com}"
    prompt OWNER_EMAIL "Owner Email (for reminders)" "$default_owner_email"
    
    # Installation Directory
    echo ""
    echo -e "${CYAN}📁 Installation Directory${NC}"
    prompt INSTALL_DIR "Installation directory" "/opt/aeterna"
    
    # Check firewall
    check_firewall
}

# Show configuration summary and confirm
confirm_installation() {
    local server_ip=$(get_server_ip)
    
    echo ""
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}  Configuration Summary${NC}"
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    
    # Mode display
    case "$PROXY_MODE" in
        traefik)
            echo -e "  ${CYAN}Mode:${NC}            ${BOLD}Traefik${NC} (SSL enabled)"
            echo -e "  ${CYAN}Access URL:${NC}      https://$DOMAIN"
            echo -e "  ${CYAN}SSL Email:${NC}       $ACME_EMAIL"
            ;;
        nginx)
            echo -e "  ${CYAN}Mode:${NC}            ${BOLD}nginx${NC} (behind existing nginx)"
            echo -e "  ${CYAN}Access URL:${NC}      https://$DOMAIN"
            ;;
        simple)
            echo -e "  ${CYAN}Mode:${NC}            ${BOLD}Simple${NC} (IP only, no SSL)"
            echo -e "  ${CYAN}Access URL:${NC}      http://$server_ip:5000"
            echo -e "  ${YELLOW}⚠ Warning:${NC}       No SSL - not for production use!"
            ;;
    esac
    
    echo -e "  ${CYAN}Owner Email:${NC}     $OWNER_EMAIL"
    echo ""
    echo -e "  ${CYAN}Database User:${NC}   $DB_USER"
    echo -e "  ${CYAN}Database Name:${NC}   $DB_NAME"
    echo -e "  ${CYAN}Database Pass:${NC}   ****${DB_PASS: -4}"
    echo ""
    if [ "$CONFIGURE_SMTP" = true ]; then
        echo -e "  ${CYAN}SMTP Server:${NC}     $SMTP_HOST:$SMTP_PORT"
        echo -e "  ${CYAN}SMTP User:${NC}       $SMTP_USER"
        echo -e "  ${CYAN}SMTP From:${NC}       $SMTP_FROM_NAME <$SMTP_FROM>"
    else
        echo -e "  ${CYAN}SMTP:${NC}            Not configured"
    fi
    echo ""
    echo -e "  ${CYAN}Install Dir:${NC}     $INSTALL_DIR"
    echo ""
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    
    if ! prompt_yn "Proceed with installation?" "y"; then
        error "Installation cancelled by user."
    fi
}

# Clone or update repository
setup_repository() {
    echo ""
    step "Setting up Aeterna repository..."
    
    if [ -d "$INSTALL_DIR" ]; then
        warning "Directory $INSTALL_DIR already exists."
        if prompt_yn "Update existing installation?" "n"; then
            cd "$INSTALL_DIR"
            git fetch origin
            git pull origin main
            success "Repository updated"
        else
            error "Installation cancelled. Remove existing directory first: rm -rf $INSTALL_DIR"
        fi
    else
        sudo mkdir -p "$INSTALL_DIR"
        sudo chown $USER:$USER "$INSTALL_DIR"
        git clone https://github.com/alpyxn/aeterna.git "$INSTALL_DIR"
        success "Repository cloned"
    fi
    
    cd "$INSTALL_DIR"
}

# Get compose file for current mode
get_compose_file() {
    case "$PROXY_MODE" in
        traefik) echo "docker-compose.prod.yml" ;;
        nginx) echo "docker-compose.nginx.yml" ;;
        simple) echo "docker-compose.simple.yml" ;;
        *) echo "docker-compose.prod.yml" ;;
    esac
}

# Create environment file
create_env_file() {
    step "Creating environment configuration..."
    
    cat > .env << EOF
# Aeterna Production Configuration
# Generated by install.sh v${VERSION} on $(date)
# Mode: $PROXY_MODE

# Domain Configuration
DOMAIN=$DOMAIN
ACME_EMAIL=${ACME_EMAIL:-}

# Database Configuration
DB_USER=$DB_USER
DB_PASS=$DB_PASS
DB_NAME=$DB_NAME

# Application Settings
ENV=production
ALLOWED_ORIGINS=$ALLOWED_ORIGINS
VITE_API_URL=/api

# Owner Configuration
OWNER_EMAIL=$OWNER_EMAIL

# Installation Mode
PROXY_MODE=$PROXY_MODE
EOF

    # Add SMTP configuration if provided
    if [ "$CONFIGURE_SMTP" = true ]; then
        cat >> .env << EOF

# SMTP Configuration
SMTP_HOST=$SMTP_HOST
SMTP_PORT=$SMTP_PORT
SMTP_USER=$SMTP_USER
SMTP_PASS=$SMTP_PASS
SMTP_FROM=$SMTP_FROM
SMTP_FROM_NAME=$SMTP_FROM_NAME
EOF
    fi

    chmod 600 .env
    success "Environment file created"
}

# Generate nginx configuration
generate_nginx_config() {
    local nginx_config="$INSTALL_DIR/nginx-aeterna.conf"
    
    cat > "$nginx_config" << EOF
# Aeterna nginx configuration
# Copy this to /etc/nginx/sites-available/aeterna
# Then: sudo ln -s /etc/nginx/sites-available/aeterna /etc/nginx/sites-enabled/
# And: sudo nginx -t && sudo systemctl reload nginx

server {
    listen 80;
    server_name $DOMAIN;
    
    # Redirect HTTP to HTTPS
    return 301 https://\$server_name\$request_uri;
}

server {
    listen 443 ssl http2;
    server_name $DOMAIN;
    
    # SSL Configuration - Update paths to your certificates
    ssl_certificate /etc/letsencrypt/live/$DOMAIN/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/$DOMAIN/privkey.pem;
    
    # SSL Security Settings
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_prefer_server_ciphers on;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
    
    # Security Headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    
    # API Backend
    location /api {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_cache_bypass \$http_upgrade;
    }
    
    # Frontend
    location / {
        proxy_pass http://127.0.0.1:8081;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_cache_bypass \$http_upgrade;
    }
}
EOF

    success "nginx configuration generated: $nginx_config"
}

# Start the application
start_application() {
    echo ""
    step "Building and starting Aeterna..."
    
    local compose_file=$(get_compose_file)
    
    docker compose -f "$compose_file" pull 2>/dev/null || true
    docker compose -f "$compose_file" build --no-cache
    docker compose -f "$compose_file" up -d
    
    success "Aeterna containers started!"
    
    # Generate nginx config if in nginx mode
    if [ "$PROXY_MODE" = "nginx" ]; then
        generate_nginx_config
    fi
}

# Wait for services to be ready
wait_for_services() {
    step "Waiting for services to be ready..."
    
    local max_attempts=60
    local attempt=0
    local check_url="http://localhost"
    
    case "$PROXY_MODE" in
        traefik) check_url="http://localhost:80" ;;
        nginx) check_url="http://localhost:8081" ;;
        simple) check_url="http://localhost:5000" ;;
    esac
    
    echo -n "  "
    while [ $attempt -lt $max_attempts ]; do
        if curl -s -o /dev/null -w "%{http_code}" "$check_url" 2>/dev/null | grep -qE "301|200|302"; then
            echo ""
            success "All services are ready!"
            return 0
        fi
        attempt=$((attempt + 1))
        echo -n "."
        sleep 2
    done
    
    echo ""
    warning "Services are taking longer than expected to start."
    
    local compose_file=$(get_compose_file)
    info "Check logs with: docker compose -f $compose_file logs"
}

# Create backup
create_backup() {
    local backup_dir="${INSTALL_DIR:-/opt/aeterna}_backup_$(date +%Y%m%d_%H%M%S)"
    
    if [ ! -d "${INSTALL_DIR:-/opt/aeterna}" ]; then
        error "No installation found at ${INSTALL_DIR:-/opt/aeterna}"
    fi
    
    info "Creating backup at $backup_dir..."
    
    # Backup files
    sudo cp -r "${INSTALL_DIR:-/opt/aeterna}" "$backup_dir"
    
    # Determine compose file
    cd "${INSTALL_DIR:-/opt/aeterna}"
    local compose_file="docker-compose.prod.yml"
    if [ -f ".env" ]; then
        local mode=$(grep "PROXY_MODE=" .env 2>/dev/null | cut -d'=' -f2)
        case "$mode" in
            nginx) compose_file="docker-compose.nginx.yml" ;;
            simple) compose_file="docker-compose.simple.yml" ;;
        esac
    fi
    
    # Backup database
    if docker compose -f "$compose_file" ps db 2>/dev/null | grep -q "Up"; then
        docker compose -f "$compose_file" exec -T db pg_dump -U aeterna aeterna > "$backup_dir/database_backup.sql" 2>/dev/null || warning "Could not backup database"
    fi
    
    success "Backup created at $backup_dir"
}

# Uninstall
uninstall() {
    echo ""
    echo -e "${RED}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${RED}${BOLD}  WARNING: This will remove Aeterna completely!${NC}"
    echo -e "${RED}${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    
    local install_path="${INSTALL_DIR:-/opt/aeterna}"
    
    if [ ! -d "$install_path" ]; then
        error "No installation found at $install_path"
    fi
    
    # Show what will be removed
    echo -e "  ${CYAN}The following will be removed:${NC}"
    echo "  • Installation directory: $install_path"
    echo "  • Docker containers and volumes"
    echo "  • Docker images (aeterna-related)"
    
    # Check for nginx config
    local nginx_available="/etc/nginx/sites-available/aeterna"
    local nginx_enabled="/etc/nginx/sites-enabled/aeterna"
    if [ -f "$nginx_available" ] || [ -L "$nginx_enabled" ]; then
        echo "  • nginx configuration files"
    fi
    echo ""
    
    if prompt_yn "Create backup before uninstalling?" "y"; then
        INSTALL_DIR="$install_path"
        create_backup
    fi
    
    if ! prompt_yn "Are you sure you want to uninstall Aeterna?" "n"; then
        info "Uninstall cancelled."
        exit 0
    fi
    
    echo ""
    step "Stopping and removing Docker containers..."
    cd "$install_path"
    
    # Stop and remove containers with volumes
    docker compose -f docker-compose.prod.yml down -v --remove-orphans 2>/dev/null || true
    docker compose -f docker-compose.nginx.yml down -v --remove-orphans 2>/dev/null || true
    docker compose -f docker-compose.simple.yml down -v --remove-orphans 2>/dev/null || true
    
    # Remove Docker images
    step "Removing Docker images..."
    local project_name=$(basename "$install_path")
    docker images --format "{{.Repository}}:{{.Tag}}" | grep -E "^${project_name}[-_]" | xargs -r docker rmi 2>/dev/null || true
    docker images --format "{{.Repository}}:{{.Tag}}" | grep -E "aeterna[-_]" | xargs -r docker rmi 2>/dev/null || true
    
    # Prune dangling images from this project
    docker image prune -f 2>/dev/null || true
    success "Docker cleanup complete"
    
    # Remove nginx configuration if exists
    if [ -f "$nginx_available" ] || [ -L "$nginx_enabled" ]; then
        step "Removing nginx configuration..."
        
        if [ -L "$nginx_enabled" ]; then
            sudo rm -f "$nginx_enabled"
            success "Removed $nginx_enabled"
        fi
        
        if [ -f "$nginx_available" ]; then
            sudo rm -f "$nginx_available"
            success "Removed $nginx_available"
        fi
        
        # Test and reload nginx if it's running
        if pgrep -x "nginx" > /dev/null 2>&1; then
            if sudo nginx -t 2>/dev/null; then
                sudo systemctl reload nginx 2>/dev/null || sudo nginx -s reload 2>/dev/null || true
                success "nginx reloaded"
            fi
        fi
    fi
    
    # Remove installation directory
    step "Removing installation directory..."
    sudo rm -rf "$install_path"
    success "Removed $install_path"
    
    # Summary
    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}  Aeterna has been completely uninstalled!${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo -e "  ${CYAN}Removed:${NC}"
    echo "  ✓ Docker containers and volumes"
    echo "  ✓ Docker images"
    echo "  ✓ Installation directory ($install_path)"
    if [ -f "$nginx_available" ] || [ -L "$nginx_enabled" ]; then
        echo "  ✓ nginx configuration"
    fi
    echo ""
    echo -e "  ${DIM}Note: Database backups (if created) are preserved.${NC}"
    echo ""
}

# Check status
check_status() {
    local install_path="${INSTALL_DIR:-/opt/aeterna}"
    
    if [ ! -d "$install_path" ]; then
        error "No installation found at $install_path"
    fi
    
    echo ""
    echo -e "${BOLD}Aeterna Service Status${NC}"
    echo ""
    
    cd "$install_path"
    
    # Determine compose file and mode
    local compose_file="docker-compose.prod.yml"
    local mode="traefik"
    if [ -f ".env" ]; then
        mode=$(grep "PROXY_MODE=" .env 2>/dev/null | cut -d'=' -f2 || echo "traefik")
        case "$mode" in
            nginx) compose_file="docker-compose.nginx.yml" ;;
            simple) compose_file="docker-compose.simple.yml" ;;
        esac
    fi
    
    echo -e "  ${CYAN}Mode:${NC} $mode"
    echo ""
    
    docker compose -f "$compose_file" ps
    
    echo ""
    
    # Check if services are healthy
    local frontend_status=$(docker compose -f "$compose_file" ps frontend 2>/dev/null | grep -c "Up")
    local backend_status=$(docker compose -f "$compose_file" ps backend 2>/dev/null | grep -c "Up")
    local db_status=$(docker compose -f "$compose_file" ps db 2>/dev/null | grep -c "Up")
    
    if [ "$frontend_status" -eq 1 ] && [ "$backend_status" -eq 1 ] && [ "$db_status" -eq 1 ]; then
        success "All services are running"
    else
        warning "Some services may not be running properly"
    fi
}

# Update existing installation
update_installation() {
    local install_path="${INSTALL_DIR:-/opt/aeterna}"
    
    if [ ! -d "$install_path" ]; then
        error "No installation found at $install_path"
    fi
    
    if prompt_yn "Create backup before updating?" "y"; then
        INSTALL_DIR="$install_path"
        create_backup
    fi
    
    info "Updating Aeterna..."
    cd "$install_path"
    
    git fetch origin
    git pull origin main
    
    # Determine compose file
    local compose_file="docker-compose.prod.yml"
    if [ -f ".env" ]; then
        local mode=$(grep "PROXY_MODE=" .env 2>/dev/null | cut -d'=' -f2)
        case "$mode" in
            nginx) compose_file="docker-compose.nginx.yml" ;;
            simple) compose_file="docker-compose.simple.yml" ;;
        esac
    fi
    
    docker compose -f "$compose_file" build --no-cache
    docker compose -f "$compose_file" up -d
    
    success "Aeterna has been updated!"
}

# Print completion message
print_completion() {
    local server_ip=$(get_server_ip)
    local compose_file=$(get_compose_file)
    
    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}  ✨ Installation Complete! ✨${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo -e "  ${BOLD}Your Aeterna instance is now running!${NC}"
    echo ""
    
    case "$PROXY_MODE" in
        traefik)
            echo -e "  ${CYAN}Mode:${NC}           ${BOLD}Traefik${NC} (standalone with SSL)"
            echo -e "  ${CYAN}Access URL:${NC}     https://$DOMAIN"
            echo -e "  ${CYAN}Server IP:${NC}      $server_ip"
            echo -e "  ${CYAN}Install Dir:${NC}    $INSTALL_DIR"
            echo ""
            echo -e "  ${BOLD}📋 Next Steps:${NC}"
            echo "  1. Ensure DNS A record points $DOMAIN → $server_ip"
            echo "  2. Open https://$DOMAIN in your browser"
            echo "  3. Set up your master password"
            ;;
        nginx)
            echo -e "  ${CYAN}Mode:${NC}           ${BOLD}nginx${NC} (behind existing nginx)"
            echo -e "  ${CYAN}Access URL:${NC}     https://$DOMAIN"
            echo -e "  ${CYAN}Server IP:${NC}      $server_ip"
            echo -e "  ${CYAN}Install Dir:${NC}    $INSTALL_DIR"
            echo ""
            echo -e "  ${BOLD}📋 IMPORTANT: nginx Configuration Required${NC}"
            echo ""
            echo -e "  ${YELLOW}1. Get SSL certificate:${NC}"
            echo "     sudo certbot certonly --nginx -d $DOMAIN"
            echo ""
            echo -e "  ${YELLOW}2. Copy nginx config:${NC}"
            echo "     sudo cp $INSTALL_DIR/nginx-aeterna.conf /etc/nginx/sites-available/aeterna"
            echo "     sudo ln -s /etc/nginx/sites-available/aeterna /etc/nginx/sites-enabled/"
            echo ""
            echo -e "  ${YELLOW}3. Test and reload nginx:${NC}"
            echo "     sudo nginx -t && sudo systemctl reload nginx"
            ;;
        simple)
            echo -e "  ${CYAN}Mode:${NC}           ${BOLD}Simple${NC} (IP only, no SSL)"
            echo -e "  ${CYAN}Access URL:${NC}     ${BOLD}http://$server_ip:5000${NC}"
            echo -e "  ${CYAN}Install Dir:${NC}    $INSTALL_DIR"
            echo ""
            echo -e "  ${YELLOW}⚠ Security Warning:${NC}"
            echo "  This installation has NO SSL encryption."
            echo "  Do not use for sensitive data or production."
            echo ""
            echo -e "  ${BOLD}📋 Next Steps:${NC}"
            echo "  1. Open http://$server_ip:5000 in your browser"
            echo "  2. Set up your master password"
            ;;
    esac
    
    if [ "$CONFIGURE_SMTP" != true ]; then
        echo "  • Configure SMTP in Settings for email delivery"
    fi
    echo ""
    
    echo -e "  ${BOLD}🔧 Useful Commands:${NC}"
    echo "  cd $INSTALL_DIR"
    echo "  docker compose -f $compose_file logs -f    # View logs"
    echo "  docker compose -f $compose_file restart    # Restart"
    echo "  docker compose -f $compose_file down       # Stop"
    echo ""
    echo -e "  ${BOLD}📦 Maintenance:${NC}"
    echo "  ./install.sh --backup      # Create backup"
    echo "  ./install.sh --update      # Update to latest version"
    echo "  ./install.sh --status      # Check service status"
    echo "  ./install.sh --uninstall   # Remove installation"
    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# Main installation flow
main() {
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --help|-h)
                print_banner
                print_help
                exit 0
                ;;
            --version|-v)
                echo "Aeterna Installation Wizard v$VERSION"
                exit 0
                ;;
            --nginx)
                PROXY_MODE="nginx"
                PROXY_MODE_SET=true
                shift
                ;;
            --traefik)
                PROXY_MODE="traefik"
                PROXY_MODE_SET=true
                shift
                ;;
            --simple)
                PROXY_MODE="simple"
                PROXY_MODE_SET=true
                shift
                ;;
            --uninstall)
                print_banner
                uninstall
                exit 0
                ;;
            --backup)
                print_banner
                create_backup
                exit 0
                ;;
            --update)
                print_banner
                update_installation
                exit 0
                ;;
            --status)
                print_banner
                check_status
                exit 0
                ;;
            *)
                error "Unknown option: $1. Use --help for usage."
                ;;
        esac
    done
    
    clear
    print_banner
    
    # Check if running as root
    if [ "$EUID" -eq 0 ]; then
        warning "Running as root. It's recommended to run as a normal user with sudo privileges."
        echo ""
    fi
    
    # Select proxy mode if not specified via command line
    if [ "$PROXY_MODE_SET" = false ]; then
        select_proxy_mode
    fi
    
    check_requirements
    collect_config
    confirm_installation
    setup_repository
    create_env_file
    start_application
    wait_for_services
    print_completion
}

# Run main function
main "$@"
