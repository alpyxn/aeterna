#!/bin/bash

#######################################
# Aeterna - Dead Man's Switch
# One-Click Installation Script
#
# Security & Safety Features:
# - Atomic file operations (temp files + atomic moves)
# - Encryption key validation (format, length, permissions)
# - Comprehensive error handling with cleanup traps
# - Input validation and sanitization
# - Disk space checks before operations
# - Docker Compose file validation
# - Container startup verification
# - Graceful error recovery where possible
# - Detailed error messages for troubleshooting
#######################################

# Exit on error, but allow controlled error handling
# -e: Exit immediately if a command exits with non-zero status
# -u: Treat unset variables as an error
# -o pipefail: Return value of pipeline is last command to exit with non-zero
set -euo pipefail

# Trap errors and provide cleanup
cleanup_on_error() {
    local exit_code=$?
    # Disable trap to prevent recursive calls (error() calls exit which re-triggers ERR)
    trap - ERR
    if [ $exit_code -ne 0 ]; then
        echo ""
        echo -e "${RED}[✗]${NC} Installation failed (exit code $exit_code)"
        echo "Check the error messages above for details."
        echo "You may need to clean up manually: rm -rf ${INSTALL_DIR:-/opt/aeterna}"
    fi
    exit $exit_code
}
trap cleanup_on_error ERR

# Cleanup function for interrupted installations
cleanup_interrupt() {
    echo ""
    warning "Installation interrupted by user"
    exit 130
}
trap cleanup_interrupt INT TERM

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
BRANCH="feat/recovery-key" # Branch to checkout during installation

# Default values
PROXY_MODE=""  # nginx or simple
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
    echo "  --nginx          Install with nginx + SSL (default, recommended)"
    echo "  --simple         Simple mode: IP-only on port 5000 (no SSL, for testing)"
    echo "  --uninstall      Remove Aeterna installation"
    echo "  --backup         Create backup of current installation"
    echo "  --update         Update existing installation"
    echo "  --status         Check status of Aeterna services"
    echo "  --version, -v    Show version"
    echo ""
    echo "Examples:"
    echo "  $0               Run installation wizard"
    echo "  $0 --nginx       Install with nginx + automatic SSL (recommended)"
    echo "  $0 --simple      Install on port 5000 without SSL (testing/dev)"
    echo ""
}

# Detect OS type
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS_ID="${ID}"
        OS_ID_LIKE="${ID_LIKE:-}"
    elif [ -f /etc/redhat-release ]; then
        OS_ID="rhel"
        OS_ID_LIKE="rhel fedora"
    elif [ -f /etc/debian_version ]; then
        OS_ID="debian"
        OS_ID_LIKE="debian"
    else
        OS_ID="unknown"
        OS_ID_LIKE=""
    fi
}

# Check if system is apt-based (Debian/Ubuntu)
is_apt_based() {
    [[ "$OS_ID" == "debian" || "$OS_ID" == "ubuntu" || "$OS_ID_LIKE" == *"debian"* ]]
}

# Check if system is dnf-based (Fedora)
is_dnf_based() {
    [[ "$OS_ID" == "fedora" || ("$OS_ID_LIKE" == *"fedora"* && -x "$(command -v dnf)") ]]
}

# Check if system is yum-based (RHEL/CentOS)
is_yum_based() {
    [[ "$OS_ID" == "rhel" || "$OS_ID" == "centos" || "$OS_ID" == "rocky" || "$OS_ID" == "almalinux" || "$OS_ID_LIKE" == *"rhel"* ]]
}

# Install system packages based on detected package manager
install_packages() {
    local packages=("$@")
    detect_os
    
    if is_apt_based; then
        sudo apt-get update -qq && sudo apt-get install -y -qq "${packages[@]}"
    elif is_dnf_based; then
        sudo dnf install -y -q "${packages[@]}"
    elif is_yum_based; then
        sudo yum install -y -q "${packages[@]}"
    else
        return 1
    fi
}

# Install Docker Compose v2 (cross-platform)
install_docker_compose() {
    detect_os
    
    local installed=false
    
    # Try package manager first
    if is_apt_based || is_dnf_based || is_yum_based; then
        if install_packages docker-compose-plugin; then
            installed=true
        fi
    fi
    
    # Fallback: Download binary directly from GitHub
    if [ "$installed" = false ]; then
        warning "Package manager installation failed, downloading binary..."
        
        local arch=$(uname -m)
        case "$arch" in
            x86_64) arch="x86_64" ;;
            aarch64|arm64) arch="aarch64" ;;
            armv7l) arch="armv7" ;;
            *) error "Unsupported architecture: $arch" ;;
        esac
        
        local compose_version=$(curl -s https://api.github.com/repos/docker/compose/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        if [ -z "$compose_version" ]; then
            compose_version="v2.24.5"  # Fallback version
        fi
        
        info "Downloading Docker Compose ${compose_version} for ${arch}..."
        
        # Create Docker CLI plugins directory
        local plugin_dir="/usr/local/lib/docker/cli-plugins"
        sudo mkdir -p "$plugin_dir"
        
        # Download and install
        if sudo curl -SL "https://github.com/docker/compose/releases/download/${compose_version}/docker-compose-linux-${arch}" -o "${plugin_dir}/docker-compose"; then
            sudo chmod +x "${plugin_dir}/docker-compose"
            installed=true
        fi
        
        # Also try user-level plugin directory if system-level failed
        if [ "$installed" = false ]; then
            plugin_dir="${HOME}/.docker/cli-plugins"
            mkdir -p "$plugin_dir"
            if curl -SL "https://github.com/docker/compose/releases/download/${compose_version}/docker-compose-linux-${arch}" -o "${plugin_dir}/docker-compose"; then
                chmod +x "${plugin_dir}/docker-compose"
                installed=true
            fi
        fi
    fi
    
    return $([ "$installed" = true ] && echo 0 || echo 1)
}

# Check if command exists
check_command() {
    if ! command -v "$1" &> /dev/null; then
        return 1
    fi
    return 0
}

# Get server's public IPv4 address
get_server_ip() {
    curl -4 -s --connect-timeout 5 ifconfig.me 2>/dev/null || \
    curl -4 -s --connect-timeout 5 icanhazip.com 2>/dev/null || \
    curl -4 -s --connect-timeout 5 ipecho.net/plain 2>/dev/null || \
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

# Find a free port starting from a given port
find_free_port() {
    local port=$1
    while check_port "$port"; do
        port=$((port + 1))
    done
    echo "$port"
}

# Check system requirements
check_requirements() {
    echo ""
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}  System Requirements Check${NC}"
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    
    local requirements_met=true
    
    export DEBIAN_FRONTEND=noninteractive
    
    # Check curl
    if ! check_command curl; then
        warning "curl not found. Installing..."
        if ! install_packages curl; then
            error "Failed to install curl"
        fi
        success "curl installed"
    else
        success "curl found"
    fi
    
    # Check openssl
    if ! check_command openssl; then
        warning "openssl not found. Installing..."
        if ! install_packages openssl; then
            error "Failed to install openssl"
        fi
        success "openssl installed"
    else
        success "openssl found"
    fi
    
    # Verify openssl can generate random data
    if ! openssl rand -base64 16 > /dev/null 2>&1; then
        error "openssl is installed but cannot generate random data. Check your system's entropy source."
    fi
    
    # Check Docker
    if ! check_command docker; then
        warning "Docker not found. Installing..."
        curl -fsSL https://get.docker.com | sh
        sudo usermod -aG docker "$USER"
        success "Docker installed"
        warning "You may need to log out and back in for Docker group changes to take effect"
    else
        success "Docker found: $(docker --version | cut -d' ' -f3 | tr -d ',')"
    fi
    
    # Check Docker Compose
    if ! docker compose version &> /dev/null; then
        warning "Docker Compose v2 not found. Installing..."
        install_docker_compose
        if docker compose version &> /dev/null; then
            success "Docker Compose installed: $(docker compose version --short)"
        else
            error "Failed to install Docker Compose. Please install it manually."
        fi
    else
        success "Docker Compose found: $(docker compose version --short)"
    fi
    
    # Check Git
    if ! check_command git; then
        warning "Git not found. Installing..."
        if ! install_packages git; then
            error "Failed to install git"
        fi
        success "Git installed"
    else
        success "Git found"
    fi
    
    # Install nginx and certbot for nginx mode
    if [ "$PROXY_MODE" = "nginx" ]; then
        if ! check_command nginx; then
            warning "nginx not found. Installing..."
            if ! install_packages nginx; then
                error "Failed to install nginx"
            fi
            sudo systemctl enable nginx
            sudo systemctl start nginx
            success "nginx installed and started"
        else
            success "nginx found"
        fi
        
        if ! check_command certbot; then
            warning "certbot not found. Installing..."
            if ! install_packages certbot python3-certbot-nginx; then
                error "Failed to install certbot"
            fi
            success "certbot installed"
        else
            success "certbot found"
        fi
    fi
    
    # Check available ports based on mode
    echo ""
    info "Checking port availability..."
    
    case "$PROXY_MODE" in
        nginx)
            BACKEND_PORT=$(find_free_port 8080)
            success "Port $BACKEND_PORT is available (backend)"
            
            FRONTEND_PORT=$(find_free_port $((BACKEND_PORT + 1)))
            success "Port $FRONTEND_PORT is available (frontend)"
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
    local is_secret=${4:-""}  # Optional parameter, default to empty string
    
    local value=""
    if [ "$is_secret" = "true" ]; then
        read -sp "$prompt_text [$default_value]: " value
        echo ""
    else
        read -p "$prompt_text [$default_value]: " value
    fi
    
    if [ -z "$value" ]; then
        value="$default_value"
    fi
    
    printf -v "$var_name" '%s' "$value"
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
    
    # Resolve domain A record (IPv4 only)
    local domain_ip
    domain_ip=$(dig +short A "$domain" 2>/dev/null | head -n1)
    
    if [ -z "$domain_ip" ]; then
        warning "Could not resolve $domain (no A record found)"
        echo -e "  ${DIM}Make sure DNS A record points to this server: $server_ip${NC}"
        if [ "$PROXY_MODE" != "simple" ]; then
            if ! prompt_yn "Continue without DNS? (SSL will fail until DNS is configured)" "n"; then
                error "Installation cancelled. Configure DNS first: $domain → $server_ip"
            fi
        fi
    elif [ "$server_ip" = "$domain_ip" ]; then
        success "DNS correctly configured ($domain → $server_ip)"
    else
        echo ""
        warning "DNS mismatch detected!"
        echo -e "  ${DIM}Server IPv4:  $server_ip${NC}"
        echo -e "  ${DIM}Domain A record:  $domain_ip${NC}"
        echo ""
        warning "SSL certificate will fail until DNS points to this server."
        warning "Update your DNS A record: $domain → $server_ip"
        if [ "$PROXY_MODE" != "simple" ]; then
            if ! prompt_yn "Continue anyway? (SSL will fail until DNS is fixed)" "n"; then
                error "Installation cancelled. Fix DNS first: $domain → $server_ip"
            fi
        fi
    fi
}

# Check firewall status
check_firewall() {
    echo ""
    info "Checking firewall configuration..."
    
    if check_command ufw; then
        if sudo ufw status | grep -q "Status: active"; then
            case "$PROXY_MODE" in
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
    
    echo -e "  ${CYAN}1)${NC} ${BOLD}nginx + SSL${NC} (recommended)"
    echo -e "     ${DIM}• Automatic nginx + certbot SSL setup${NC}"
    echo -e "     ${DIM}• Requires a domain name${NC}"
    echo -e "     ${DIM}• Dynamically assigns internal ports${NC}"
    echo ""
    echo -e "  ${CYAN}2)${NC} ${BOLD}Simple${NC} - IP only, no SSL (testing/dev)"
    echo -e "     ${DIM}• No domain required, access via IP${NC}"
    echo -e "     ${DIM}• No SSL/HTTPS (not for production!)${NC}"
    echo -e "     ${DIM}• Runs on port 5000${NC}"
    echo ""
    
    read -p "Select installation mode [1]: " mode_choice
    mode_choice=${mode_choice:-1}
    
    case $mode_choice in
        1)
            PROXY_MODE="nginx"
            success "Using nginx mode (with SSL)"
            ;;
        2)
            PROXY_MODE="simple"
            success "Using Simple mode (IP only, port 5000)"
            warning "This mode has no SSL - not recommended for production!"
            ;;
        *)
            PROXY_MODE="nginx"
            success "Using nginx mode (with SSL)"
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
        ACME_EMAIL="admin@$DOMAIN"
    fi
    
    # Database Configuration
    echo ""
    echo -e "${CYAN}🗄️  Database Configuration${NC}"
    echo -e "${DIM}SQLite database will be created automatically${NC}"
    echo -e "${DIM}Database file location: ${INSTALL_DIR:-/opt/aeterna}/data/aeterna.db${NC}"
    echo ""
    
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
        nginx)
            echo -e "  ${CYAN}Mode:${NC}            ${BOLD}nginx + SSL${NC}"
            echo -e "  ${CYAN}Access URL:${NC}      https://$DOMAIN"
            ;;
        simple)
            echo -e "  ${CYAN}Mode:${NC}            ${BOLD}Simple${NC} (IP only, no SSL)"
            echo -e "  ${CYAN}Access URL:${NC}      http://$server_ip:5000"
            echo -e "  ${YELLOW}⚠ Warning:${NC}       No SSL - not for production use!"
            ;;
    esac
    
    echo ""
    echo -e "  ${CYAN}Database:${NC}         SQLite (file: $INSTALL_DIR/data/aeterna.db)"
    echo ""
    echo ""
    echo ""
    echo -e "  ${CYAN}Install Dir:${NC}     $INSTALL_DIR"
    echo ""
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    
    # Validate critical inputs before proceeding
    if [ -z "${DOMAIN:-}" ]; then
        error "DOMAIN is not set"
    fi
    
    # SQLite mode uses file-based database and doesn't need credentials
    
    if [ -z "${INSTALL_DIR:-}" ]; then
        error "INSTALL_DIR is not set"
    fi
    
    # Validate domain format (basic check)
    if ! echo "$DOMAIN" | grep -qE '^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$'; then
        warning "Domain format may be invalid: $DOMAIN"
        if ! prompt_yn "Continue anyway?" "n"; then
            error "Installation cancelled"
        fi
    fi
    
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
            git pull origin "$BRANCH"
            success "Repository updated"
        else
            error "Installation cancelled. Remove existing directory first: rm -rf $INSTALL_DIR"
        fi
    else
        sudo mkdir -p "$INSTALL_DIR"
        sudo chown "$USER":"$USER" "$INSTALL_DIR"
        git clone -b "$BRANCH" https://github.com/alpyxn/aeterna.git "$INSTALL_DIR"
        success "Repository cloned"
    fi
    
    cd "$INSTALL_DIR"
}

# Get compose file for current mode
get_compose_file() {
    case "$PROXY_MODE" in
        nginx) echo "docker-compose.nginx.yml" ;;
        simple) echo "docker-compose.simple.yml" ;;
        *) echo "docker-compose.nginx.yml" ;;
    esac
}

# Validate encryption key format
validate_encryption_key() {
    local key_file="$1"
    if [ ! -f "$key_file" ]; then
        return 1
    fi
    
    local key_content
    key_content=$(cat "$key_file" | tr -d '[:space:]')
    
    # Check key is not empty
    if [ -z "$key_content" ]; then
        return 1
    fi
    
    # Validate base64 format and length (should decode to exactly 32 bytes)
    local decoded_length
    decoded_length=$(echo "$key_content" | base64 -d 2>/dev/null | wc -c)
    
    if [ "$decoded_length" -ne 32 ]; then
        return 1
    fi
    
    return 0
}

# Generate encryption key with validation
generate_encryption_key() {
    local key_file="$1"
    local max_attempts=5
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        attempt=$((attempt + 1))
        
        # Generate key
        if ! ENCRYPTION_KEY=$(openssl rand -base64 32 2>/dev/null); then
            error "Failed to generate encryption key: openssl command failed"
        fi
        
        # Remove whitespace
        ENCRYPTION_KEY=$(echo "$ENCRYPTION_KEY" | tr -d '[:space:]')
        
        # Validate key format
        if ! echo "$ENCRYPTION_KEY" | base64 -d > /dev/null 2>&1; then
            warning "Generated invalid base64 key, retrying... (attempt $attempt/$max_attempts)"
            continue
        fi
        
        local decoded_length
        decoded_length=$(echo "$ENCRYPTION_KEY" | base64 -d | wc -c)
        if [ "$decoded_length" -ne 32 ]; then
            warning "Generated key has wrong length ($decoded_length bytes), retrying... (attempt $attempt/$max_attempts)"
            continue
        fi
        
        # Write key file atomically (write to temp, then move)
        local temp_file="${key_file}.tmp"
        if ! echo "$ENCRYPTION_KEY" > "$temp_file" 2>/dev/null; then
            error "Failed to write encryption key to temporary file: $temp_file"
        fi
        
        # Set secure permissions before moving
        if ! chmod 600 "$temp_file" 2>/dev/null; then
            rm -f "$temp_file"
            error "Failed to set permissions on encryption key file"
        fi
        
        # Atomic move
        if ! mv "$temp_file" "$key_file" 2>/dev/null; then
            rm -f "$temp_file"
            error "Failed to create encryption key file: $key_file"
        fi
        
        # Verify file was created and has correct permissions
        if [ ! -f "$key_file" ]; then
            error "Encryption key file was not created: $key_file"
        fi
        
        local file_perms
        file_perms=$(stat -c "%a" "$key_file" 2>/dev/null || stat -f "%OLp" "$key_file" 2>/dev/null || echo "unknown")
        if [ "$file_perms" != "600" ] && [ "$file_perms" != "0600" ]; then
            warning "Encryption key file has incorrect permissions ($file_perms), fixing..."
            chmod 600 "$key_file" || error "Failed to fix encryption key file permissions"
        fi
        
        # Final validation
        if ! validate_encryption_key "$key_file"; then
            warning "Generated key failed validation, retrying... (attempt $attempt/$max_attempts)"
            rm -f "$key_file"
            continue
        fi
        
        return 0
    done
    
    error "Failed to generate valid encryption key after $max_attempts attempts"
}

# Create environment file
create_env_file() {
    step "Creating environment configuration..."
    
    # Validate INSTALL_DIR is set and valid
    if [ -z "${INSTALL_DIR:-}" ]; then
        error "INSTALL_DIR is not set"
    fi
    
    if [ ! -d "$INSTALL_DIR" ]; then
        error "Installation directory does not exist: $INSTALL_DIR"
    fi
    
    # Create data directory for SQLite database
    step "Creating data directory for SQLite database..."
    if ! mkdir -p "$INSTALL_DIR/data" 2>/dev/null; then
        error "Failed to create data directory: $INSTALL_DIR/data"
    fi
    
    # Verify directory was created
    if [ ! -d "$INSTALL_DIR/data" ]; then
        error "Data directory was not created: $INSTALL_DIR/data"
    fi
    
    # Generate encryption key file BEFORE creating .env (security: key never goes in .env)
    step "Generating encryption key..."
    
    # Create secrets directory with error handling
    if ! mkdir -p "$INSTALL_DIR/secrets" 2>/dev/null; then
        error "Failed to create secrets directory: $INSTALL_DIR/secrets"
    fi
    
    # Verify directory was created
    if [ ! -d "$INSTALL_DIR/secrets" ]; then
        error "Secrets directory was not created: $INSTALL_DIR/secrets"
    fi
    
    local key_file="$INSTALL_DIR/secrets/encryption_key"
    
    if [ -f "$key_file" ]; then
        info "Encryption key file already exists, validating..."
        if validate_encryption_key "$key_file"; then
            info "Existing encryption key is valid, preserving it"
            info "To regenerate, delete: $key_file"
        else
            warning "Existing encryption key file is invalid or corrupted"
            if prompt_yn "Regenerate encryption key? (WARNING: This will make existing encrypted data unreadable)" "n"; then
                rm -f "$key_file"
                generate_encryption_key "$key_file"
                success "Encryption key regenerated"
            else
                error "Cannot proceed with invalid encryption key. Please fix or remove: $key_file"
            fi
        fi
    else
        generate_encryption_key "$key_file"
        success "Encryption key generated and stored securely in secrets/encryption_key"
    fi
    
    # Final verification
    if ! validate_encryption_key "$key_file"; then
        error "Encryption key validation failed after generation"
    fi
    
    
    # Create .env file atomically
    local env_temp=".env.tmp"
    cat > "$env_temp" << EOF
# Aeterna Production Configuration
# Generated by install.sh v${VERSION} on $(date)
# Mode: $PROXY_MODE

# Domain Configuration
DOMAIN=$DOMAIN
ACME_EMAIL=${ACME_EMAIL:-}

# Database Configuration
# SQLite database will be created automatically at ./data/aeterna.db
# No database credentials needed for SQLite

# Application Settings
ENV=production
ALLOWED_ORIGINS=$ALLOWED_ORIGINS
VITE_API_URL=/api

# Security Note:
# Encryption key is stored in secrets/encryption_key (NOT in this file)
# The key file is mounted as a Docker secret at /run/secrets/encryption_key

# Installation Mode
PROXY_MODE=$PROXY_MODE

# Internal Ports
BACKEND_PORT=${BACKEND_PORT:-8080}
FRONTEND_PORT=${FRONTEND_PORT:-8081}
EOF

    # Set permissions before moving
    if ! chmod 600 "$env_temp" 2>/dev/null; then
        rm -f "$env_temp"
        error "Failed to set permissions on .env file"
    fi
    
    # Atomic move
    if ! mv "$env_temp" .env 2>/dev/null; then
        rm -f "$env_temp"
        error "Failed to create .env file"
    fi
    
    # Verify .env was created
    if [ ! -f .env ]; then
        error ".env file was not created"
    fi
    
    success "Environment file created"
}

# Setup nginx with SSL certificate
setup_nginx() {
    local nginx_available="/etc/nginx/sites-available/aeterna"
    local nginx_enabled="/etc/nginx/sites-enabled/aeterna"
    
    step "Configuring nginx..."
    
    # Step 1: Create initial HTTP-only config (needed for certbot to verify domain)
    sudo tee "$nginx_available" > /dev/null << EOF
server {
    listen 80;
    server_name $DOMAIN;
    client_max_body_size 12m;

    # API Backend
    location /api {
        proxy_pass http://127.0.0.1:${BACKEND_PORT:-8080};
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
        proxy_pass http://127.0.0.1:${FRONTEND_PORT:-8081};
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
    
    # Step 2: Enable site and remove default if it conflicts
    if [ ! -L "$nginx_enabled" ]; then
        sudo ln -sf "$nginx_available" "$nginx_enabled"
    fi
    
    # Remove default site if it exists (it listens on port 80 and conflicts)
    if [ -L "/etc/nginx/sites-enabled/default" ]; then
        sudo rm -f /etc/nginx/sites-enabled/default
        info "Removed default nginx site to avoid port 80 conflict"
    fi
    
    # Step 3: Test and reload nginx with HTTP config
    if ! sudo nginx -t 2>/dev/null; then
        error "nginx configuration test failed. Check: sudo nginx -t"
    fi
    sudo systemctl reload nginx
    success "nginx configured with HTTP"
    
    # Step 4: Obtain SSL certificate
    step "Obtaining SSL certificate for $DOMAIN..."
    
    local certbot_email="${ACME_EMAIL:-admin@$DOMAIN}"
    
    if sudo certbot --nginx -d "$DOMAIN" --non-interactive --agree-tos -m "$certbot_email" --redirect; then
        success "SSL certificate obtained and nginx configured with HTTPS"
    else
        warning "Failed to obtain SSL certificate automatically"
        warning "This usually means DNS is not pointing to this server yet"
        echo ""
        info "You can obtain the certificate later by running:"
        echo "  sudo certbot --nginx -d $DOMAIN"
        echo ""
        info "Aeterna is accessible via HTTP in the meantime: http://$DOMAIN"
    fi
    
    # Save a backup copy of the config in the install dir
    cp "$nginx_available" "$INSTALL_DIR/nginx-aeterna.conf" 2>/dev/null || true
}

# Start the application
start_application() {
    echo ""
    step "Building and starting Aeterna..."
    
    # Validate INSTALL_DIR
    if [ -z "${INSTALL_DIR:-}" ]; then
        error "INSTALL_DIR is not set"
    fi
    
    # Ensure we're in the installation directory
    if ! cd "$INSTALL_DIR" 2>/dev/null; then
        error "Failed to change to installation directory: $INSTALL_DIR"
    fi
    
    # Verify we're in the right directory (check for docker-compose files)
    if [ ! -f "docker-compose.nginx.yml" ] && [ ! -f "docker-compose.simple.yml" ]; then
        error "Not in Aeterna installation directory. Missing docker-compose files."
    fi
    
    # Verify encryption key file exists and is valid
    local key_file="$INSTALL_DIR/secrets/encryption_key"
    if [ ! -f "$key_file" ]; then
        error "Encryption key file not found at $key_file"
        error "Please run create_env_file() first or create the key file manually"
    fi
    
    if ! validate_encryption_key "$key_file"; then
        error "Encryption key file is invalid or corrupted: $key_file"
        error "Please regenerate it or fix the file"
    fi
    
    # Verify key file permissions
    local file_perms
    file_perms=$(stat -c "%a" "$key_file" 2>/dev/null || stat -f "%OLp" "$key_file" 2>/dev/null || echo "unknown")
    if [ "$file_perms" != "600" ] && [ "$file_perms" != "0600" ]; then
        warning "Encryption key file has insecure permissions ($file_perms), fixing..."
        if ! chmod 600 "$key_file" 2>/dev/null; then
            error "Failed to fix encryption key file permissions. Please run: chmod 600 $key_file"
        fi
    fi
    
    # Get compose file
    local compose_file
    compose_file=$(get_compose_file)
    
    # Verify compose file exists
    if [ ! -f "$compose_file" ]; then
        error "Docker Compose file not found: $compose_file"
    fi
    
    # Verify docker-compose can read the file
    if ! docker compose -f "$compose_file" config > /dev/null 2>&1; then
        error "Docker Compose file is invalid or has errors: $compose_file"
    fi
    
    # Remove docker-compose.yml (dev-only) to prevent Docker Compose from merging it
    # Production uses docker-compose.{nginx,simple}.yml exclusively
    if [ -f "docker-compose.yml" ] && [ "$compose_file" != "docker-compose.yml" ]; then
        info "Removing development docker-compose.yml to prevent conflicts"
        rm -f docker-compose.yml
    fi
    
    # Also remove any override file that could inject unwanted services
    if [ -f "docker-compose.override.yml" ]; then
        info "Removing docker-compose.override.yml to prevent conflicts"
        rm -f docker-compose.override.yml
    fi
    
    # Check disk space (at least 2GB free recommended)
    local available_space
    available_space=$(df -BG "$INSTALL_DIR" | tail -1 | awk '{print $4}' | sed 's/G//')
    if [ "$available_space" -lt 2 ]; then
        warning "Low disk space: ${available_space}GB available (2GB+ recommended)"
        if ! prompt_yn "Continue anyway?" "n"; then
            error "Installation cancelled due to low disk space"
        fi
    fi
    
    # Pull images (non-critical, continue on failure)
    info "Pulling Docker images..."
    if ! docker compose -f "$compose_file" pull 2>/dev/null; then
        warning "Failed to pull some images, will build from source"
    fi
    
    # Build images
    info "Building Docker images..."
    if ! docker compose -f "$compose_file" build --no-cache; then
        error "Failed to build Docker images. Check the error messages above."
    fi
    
    # Start containers
    info "Starting containers..."
    if ! docker compose -f "$compose_file" up -d; then
        error "Failed to start containers. Check the error messages above."
    fi
    
    # Verify containers started
    sleep 2
    local containers_up
    containers_up=$(docker compose -f "$compose_file" ps --format "{{.Status}}" 2>/dev/null | grep -c "Up" || echo "0")
    if [ "$containers_up" = "0" ]; then
        warning "Some containers may not have started properly"
        info "Check logs with: docker compose -f $compose_file logs"
    fi
    
    # Verify backend is actually running
    local backend_status
    backend_status=$(docker compose -f "$compose_file" ps backend --format "{{.Status}}" 2>/dev/null || echo "")
    if ! echo "$backend_status" | grep -q "Up" 2>/dev/null; then
        error "Backend container failed to start. Check logs: docker compose -f $compose_file logs backend"
    fi
    
    success "Aeterna containers started!"
    
    # Wait a bit for containers to fully initialize
    info "Waiting for containers to initialize..."
    sleep 5
    
    # Check backend health
    step "Verifying backend health..."
    local health_attempts=0
    local max_health_attempts=30
    local backend_healthy=false
    
    # Determine how to reach the backend
    # - nginx: backend is on localhost:$BACKEND_PORT
    # - simple: backend is behind proxy on localhost:5000
    local health_check_url=""
    case "$PROXY_MODE" in
        nginx)   health_check_url="http://localhost:${BACKEND_PORT:-8080}/api/setup/status" ;;
        simple)  health_check_url="http://localhost:5000/api/setup/status" ;;
        *)       health_check_url="http://localhost:${BACKEND_PORT:-8080}/api/setup/status" ;;
    esac
    
    while [ $health_attempts -lt $max_health_attempts ]; do
        local http_code="000"
        http_code=$(curl -s -o /dev/null -w "%{http_code}" "$health_check_url" 2>/dev/null || echo "000")
        
        if [ "$http_code" = "200" ] || [ "$http_code" = "401" ] || [ "$http_code" = "404" ]; then
            backend_healthy=true
            break
        fi
        
        health_attempts=$((health_attempts + 1))
        if [ $health_attempts -lt $max_health_attempts ]; then
            sleep 2
        fi
    done
    
    if [ "$backend_healthy" = false ]; then
        warning "Backend health check failed after ${max_health_attempts} attempts"
        
        # Check backend logs for actual errors
        local backend_logs
        backend_logs=$(docker compose -f "$compose_file" logs --tail=50 backend 2>/dev/null || echo "")
        
        if echo "$backend_logs" | grep -qi "FATAL.*encryption\|failed to initialize encryption"; then
            error "Backend failed to start: Encryption key issue detected."
            error "Verify key file exists: $INSTALL_DIR/secrets/encryption_key"
            error "Check logs: docker compose -f $compose_file logs backend"
        elif echo "$backend_logs" | grep -qi "FATAL.*database\|database.*failed"; then
            error "Backend failed to start: Database error detected."
            error "Check logs: docker compose -f $compose_file logs backend"
        elif echo "$backend_logs" | grep -qi "FATAL\|panic"; then
            error "Backend crashed. Check logs: docker compose -f $compose_file logs backend"
        else
            warning "Backend may still be starting. Check logs:"
            info "  docker compose -f $compose_file logs backend"
        fi
    else
        success "Backend is healthy and responding"
    fi
    
    # Setup nginx with SSL if in nginx mode
    if [ "$PROXY_MODE" = "nginx" ]; then
        setup_nginx
    fi
}

# Wait for services to be ready
wait_for_services() {
    step "Waiting for services to be ready..."
    
    local max_attempts=60
    local attempt=0
    local check_url="http://localhost"
    
    case "$PROXY_MODE" in
        nginx) check_url="http://localhost:${FRONTEND_PORT:-8081}" ;;
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
    local compose_file="docker-compose.nginx.yml"
    if [ -f ".env" ]; then
        local mode=$(grep "PROXY_MODE=" .env 2>/dev/null | cut -d'=' -f2)
        case "$mode" in
            simple) compose_file="docker-compose.simple.yml" ;;
            *) compose_file="docker-compose.nginx.yml" ;;
        esac
    fi
    
    # Backup SQLite database file
    local db_file="${INSTALL_DIR:-/opt/aeterna}/data/aeterna.db"
    if [ -f "$db_file" ]; then
        # Create data directory in backup if it doesn't exist
        mkdir -p "$backup_dir/data" 2>/dev/null || warning "Could not create data directory in backup"
        cp "$db_file" "$backup_dir/data/aeterna.db" 2>/dev/null || warning "Could not backup SQLite database"
        info "SQLite database backed up"
    else
        info "No SQLite database file found (new installation)"
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
    local compose_file="docker-compose.nginx.yml"
    local mode="nginx"
    if [ -f ".env" ]; then
        mode=$(grep "PROXY_MODE=" .env 2>/dev/null | cut -d'=' -f2 || echo "nginx")
        case "$mode" in
            simple) compose_file="docker-compose.simple.yml" ;;
            *) compose_file="docker-compose.nginx.yml" ;;
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
    git pull origin "$BRANCH"
    
    # Ensure secrets directory exists (for encryption key)
    if ! mkdir -p "$install_path/secrets" 2>/dev/null; then
        error "Failed to create secrets directory: $install_path/secrets"
    fi
    
    local key_file="$install_path/secrets/encryption_key"
    if [ ! -f "$key_file" ]; then
        warning "Encryption key file not found. Generating new one..."
        warning "WARNING: This will make existing encrypted data unreadable!"
        if ! prompt_yn "Generate new encryption key? (You will lose access to existing encrypted messages)" "n"; then
            error "Update cancelled. Please restore the original encryption key file first."
        fi
        generate_encryption_key "$key_file"
        warning "NEW encryption key generated. Existing encrypted data may not be decryptable."
        warning "If you have existing encrypted messages, restore the original key file."
    else
        info "Encryption key file found, validating..."
        if validate_encryption_key "$key_file"; then
            info "Existing encryption key is valid, preserving it"
        else
            warning "Existing encryption key is invalid or corrupted"
            if prompt_yn "Regenerate encryption key? (WARNING: This will make existing encrypted data unreadable)" "n"; then
                generate_encryption_key "$key_file"
                warning "Encryption key regenerated"
            else
                error "Cannot proceed with invalid encryption key. Please fix or restore the original key file."
            fi
        fi
    fi
    
    # Ensure data directory exists for SQLite database
    if ! mkdir -p "$install_path/data" 2>/dev/null; then
        error "Failed to create data directory: $install_path/data"
    fi
    
    # Determine compose file
    local compose_file="docker-compose.nginx.yml"
    if [ -f ".env" ]; then
        local mode=$(grep "PROXY_MODE=" .env 2>/dev/null | cut -d'=' -f2)
        case "$mode" in
            simple) compose_file="docker-compose.simple.yml" ;;
            *) compose_file="docker-compose.nginx.yml" ;;
        esac
    fi
    
    # Build images
    info "Building Docker images..."
    if ! docker compose -f "$compose_file" build --no-cache; then
        error "Failed to build Docker images. Check the error messages above."
    fi
    
    # Start containers
    info "Starting containers..."
    if ! docker compose -f "$compose_file" up -d; then
        error "Failed to start containers. Check the error messages above."
    fi
    
    # Verify containers started
    sleep 2
    if ! docker compose -f "$compose_file" ps | grep -q "Up"; then
        warning "Some containers may not have started properly"
        info "Check logs with: docker compose -f $compose_file logs"
    fi
    
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
        nginx)
            echo -e "  ${CYAN}Mode:${NC}           ${BOLD}nginx + SSL${NC}"
            echo -e "  ${CYAN}Access URL:${NC}     https://$DOMAIN"
            echo -e "  ${CYAN}Server IP:${NC}      $server_ip"
            echo -e "  ${CYAN}Install Dir:${NC}    $INSTALL_DIR"
            echo ""
            echo -e "  ${BOLD}📋 Next Steps:${NC}"
            echo "  1. Ensure DNS A record points $DOMAIN → $server_ip"
            echo "  2. Open https://$DOMAIN in your browser"
            echo "  3. Set up your master password"
            echo ""
            if [ ! -f "/etc/letsencrypt/live/$DOMAIN/fullchain.pem" ]; then
                echo -e "  ${YELLOW}⚠ SSL certificate was not obtained during install.${NC}"
                echo "  Run this after DNS is configured:"
                echo "     sudo certbot --nginx -d $DOMAIN"
                echo ""
            fi
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
    
    echo ""
    echo -e "  ${BOLD}🔐 Security:${NC}"
    echo "  • Encryption key stored in: $INSTALL_DIR/secrets/encryption_key"
    echo "  • Keep this file secure! It's required to decrypt your messages."
    echo ""
    
    echo "  • Configure SMTP in Settings for email delivery"
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
    if [ "$(id -u)" -eq 0 ]; then
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
