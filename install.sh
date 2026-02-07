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
VERSION="1.1.0"

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
    echo "  --uninstall      Remove Aeterna installation"
    echo "  --backup         Create backup of current installation"
    echo "  --update         Update existing installation"
    echo "  --status         Check status of Aeterna services"
    echo "  --version, -v    Show version"
    echo ""
    echo "Examples:"
    echo "  $0               Run installation wizard"
    echo "  $0 --backup      Create backup before updating"
    echo "  $0 --uninstall   Remove Aeterna completely"
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
    
    # Check available ports
    echo ""
    info "Checking port availability..."
    
    if check_port 80; then
        warning "Port 80 is already in use!"
        echo -e "  ${DIM}Another service is using HTTP port. Stop it or use a different server.${NC}"
        requirements_met=false
    else
        success "Port 80 is available"
    fi
    
    if check_port 443; then
        warning "Port 443 is already in use!"
        echo -e "  ${DIM}Another service is using HTTPS port. Stop it or use a different server.${NC}"
        requirements_met=false
    else
        success "Port 443 is available"
    fi
    
    # Check available disk space (minimum 2GB)
    local available_space=$(df -BG / | awk 'NR==2 {print $4}' | tr -d 'G')
    if [ "$available_space" -lt 2 ]; then
        warning "Low disk space: ${available_space}GB available (minimum 2GB recommended)"
        requirements_met=false
    else
        success "Disk space: ${available_space}GB available"
    fi
    
    # Check available memory (minimum 1GB)
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
            local http_allowed=$(sudo ufw status | grep -E "80/tcp|80 " | grep -c "ALLOW")
            local https_allowed=$(sudo ufw status | grep -E "443/tcp|443 " | grep -c "ALLOW")
            
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
        else
            success "UFW firewall is inactive"
        fi
    elif check_command firewall-cmd; then
        if sudo firewall-cmd --state 2>/dev/null | grep -q "running"; then
            warning "firewalld is active. Ensure ports 80 and 443 are open."
        fi
    else
        success "No firewall detected"
    fi
}

# Collect configuration
collect_config() {
    echo ""
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}  Configuration${NC}"
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    # Domain Configuration
    echo ""
    echo -e "${CYAN}📌 Domain Configuration${NC}"
    echo -e "${DIM}Your domain must point to this server's IP address${NC}"
    echo ""
    
    prompt DOMAIN "Enter your domain (e.g., aeterna.example.com)" ""
    
    if [ -z "$DOMAIN" ]; then
        error "Domain is required!"
    fi
    
    # Check DNS
    check_dns "$DOMAIN"
    
    # Email for SSL
    prompt ACME_EMAIL "Email for SSL certificates" "admin@$DOMAIN"
    
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
    
    prompt OWNER_EMAIL "Owner Email (for reminders)" "$ACME_EMAIL"
    
    # Installation Directory
    echo ""
    echo -e "${CYAN}📁 Installation Directory${NC}"
    prompt INSTALL_DIR "Installation directory" "/opt/aeterna"
    
    # Check firewall
    check_firewall
}

# Show configuration summary and confirm
confirm_installation() {
    echo ""
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}  Configuration Summary${NC}"
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo -e "  ${CYAN}Domain:${NC}          $DOMAIN"
    echo -e "  ${CYAN}SSL Email:${NC}       $ACME_EMAIL"
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

# Create environment file
create_env_file() {
    step "Creating environment configuration..."
    
    cat > .env << EOF
# Aeterna Production Configuration
# Generated by install.sh v${VERSION} on $(date)

# Domain Configuration
DOMAIN=$DOMAIN
ACME_EMAIL=$ACME_EMAIL

# Database Configuration
DB_USER=$DB_USER
DB_PASS=$DB_PASS
DB_NAME=$DB_NAME

# Application Settings
ENV=production
ALLOWED_ORIGINS=https://$DOMAIN
VITE_API_URL=/api

# Owner Configuration
OWNER_EMAIL=$OWNER_EMAIL
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

# Start the application
start_application() {
    echo ""
    step "Building and starting Aeterna..."
    
    docker compose -f docker-compose.prod.yml pull 2>/dev/null || true
    docker compose -f docker-compose.prod.yml build --no-cache
    docker compose -f docker-compose.prod.yml up -d
    
    success "Aeterna containers started!"
}

# Wait for services to be ready
wait_for_services() {
    step "Waiting for services to be ready..."
    
    local max_attempts=60
    local attempt=0
    
    echo -n "  "
    while [ $attempt -lt $max_attempts ]; do
        if curl -s -o /dev/null -w "%{http_code}" "http://localhost:80" 2>/dev/null | grep -qE "301|200|302"; then
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
    info "Check logs with: docker compose -f docker-compose.prod.yml logs"
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
    
    # Backup database
    cd "${INSTALL_DIR:-/opt/aeterna}"
    if docker compose -f docker-compose.prod.yml ps db 2>/dev/null | grep -q "Up"; then
        docker compose -f docker-compose.prod.yml exec -T db pg_dump -U aeterna aeterna > "$backup_dir/database_backup.sql" 2>/dev/null || warning "Could not backup database"
    fi
    
    success "Backup created at $backup_dir"
}

# Uninstall
uninstall() {
    echo ""
    echo -e "${RED}${BOLD}WARNING: This will remove Aeterna completely!${NC}"
    echo ""
    
    local install_path="${INSTALL_DIR:-/opt/aeterna}"
    
    if [ ! -d "$install_path" ]; then
        error "No installation found at $install_path"
    fi
    
    if prompt_yn "Create backup before uninstalling?" "y"; then
        INSTALL_DIR="$install_path"
        create_backup
    fi
    
    if ! prompt_yn "Are you sure you want to uninstall Aeterna?" "n"; then
        info "Uninstall cancelled."
        exit 0
    fi
    
    info "Stopping containers..."
    cd "$install_path"
    docker compose -f docker-compose.prod.yml down -v 2>/dev/null || true
    
    info "Removing installation directory..."
    sudo rm -rf "$install_path"
    
    success "Aeterna has been uninstalled."
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
    docker compose -f docker-compose.prod.yml ps
    
    echo ""
    
    # Check if services are healthy
    local frontend_status=$(docker compose -f docker-compose.prod.yml ps frontend 2>/dev/null | grep -c "Up")
    local backend_status=$(docker compose -f docker-compose.prod.yml ps backend 2>/dev/null | grep -c "Up")
    local db_status=$(docker compose -f docker-compose.prod.yml ps db 2>/dev/null | grep -c "Up")
    
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
    
    docker compose -f docker-compose.prod.yml build --no-cache
    docker compose -f docker-compose.prod.yml up -d
    
    success "Aeterna has been updated!"
}

# Print completion message
print_completion() {
    local server_ip=$(get_server_ip)
    
    echo ""
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}  ✨ Installation Complete! ✨${NC}"
    echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo -e "  ${BOLD}Your Aeterna instance is now running!${NC}"
    echo ""
    echo -e "  ${CYAN}Access URL:${NC}     https://$DOMAIN"
    echo -e "  ${CYAN}Server IP:${NC}      $server_ip"
    echo -e "  ${CYAN}Install Dir:${NC}    $INSTALL_DIR"
    echo ""
    echo -e "  ${BOLD}📋 Next Steps:${NC}"
    echo "  1. Ensure DNS A record points $DOMAIN → $server_ip"
    echo "  2. Open https://$DOMAIN in your browser"
    echo "  3. Set up your master password"
    if [ "$CONFIGURE_SMTP" != true ]; then
        echo "  4. Configure SMTP in Settings for email delivery"
    fi
    echo ""
    echo -e "  ${BOLD}🔧 Useful Commands:${NC}"
    echo "  cd $INSTALL_DIR"
    echo "  docker compose -f docker-compose.prod.yml logs -f    # View logs"
    echo "  docker compose -f docker-compose.prod.yml restart    # Restart"
    echo "  docker compose -f docker-compose.prod.yml down       # Stop"
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
    case "${1:-}" in
        --help|-h)
            print_banner
            print_help
            exit 0
            ;;
        --version|-v)
            echo "Aeterna Installation Wizard v$VERSION"
            exit 0
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
    esac
    
    clear
    print_banner
    
    # Check if running as root
    if [ "$EUID" -eq 0 ]; then
        warning "Running as root. It's recommended to run as a normal user with sudo privileges."
        echo ""
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
