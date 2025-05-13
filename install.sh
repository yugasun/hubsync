#!/bin/bash
# HubSync Installer
# This script downloads and installs the latest version of HubSync
set -e

# Configuration
REPO="yugasun/hubsync"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="hubsync"
# This value will be auto-updated by the release workflow
LATEST_VERSION="0.2.4"
# This timestamp helps track when the script was last updated
LAST_UPDATED="2025-05-13"

# Colors for terminal output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Print banner
echo -e "${BLUE}"
echo "██╗  ██╗██╗   ██╗██████╗ ███████╗██╗   ██╗███╗   ██╗ ██████╗"
echo "██║  ██║██║   ██║██╔══██╗██╔════╝╚██╗ ██╔╝████╗  ██║██╔════╝"
echo "███████║██║   ██║██████╔╝███████╗ ╚████╔╝ ██╔██╗ ██║██║     "
echo "██╔══██║██║   ██║██╔══██╗╚════██║  ╚██╔╝  ██║╚██╗██║██║     "
echo "██║  ██║╚██████╔╝██████╔╝███████║   ██║   ██║ ╚████║╚██████╗"
echo "╚═╝  ╚═╝ ╚═════╝ ╚═════╝ ╚══════╝   ╚═╝   ╚═╝  ╚═══╝ ╚═════╝"
echo -e "${NC}"
echo "Docker Hub Image Synchronization Tool"
echo "Version: 2.0"

# Function to show spinner while waiting
show_spinner() {
  local pid=$1
  local message=$2
  local i=1
  local sp='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'
  echo -ne "${BLUE}${message}... ${NC}"
  while ps -p $pid > /dev/null; do
    printf "\r${BLUE}${message}... ${CYAN}${sp:i++%${#sp}:1}${NC}"
    sleep 0.1
  done
  printf "\r${GREEN}${message}... Done!${NC}\n"
}

# Check for required commands
check_cmd() {
    if ! command -v "$1" &> /dev/null; then
        echo -e "${RED}Error: $1 is required but not installed.${NC}"
        return 1
    fi
    return 0
}

# Check if all dependencies are installed
check_dependencies() {
    local missing_deps=()
    
    for cmd in curl grep cut uname tar gzip; do
        if ! check_cmd "$cmd" > /dev/null; then
            missing_deps+=("$cmd")
        fi
    done
    
    if [ ${#missing_deps[@]} -gt 0 ]; then
        echo -e "${RED}Error: Missing dependencies: ${missing_deps[*]}${NC}"
        echo -e "${YELLOW}Please install them and try again.${NC}"
        exit 1
    fi
}

# Cleanup function to run on exit
cleanup() {
    if [ -d "$TMP" ]; then
        rm -rf "$TMP"
    fi
}

# Check current version if already installed
check_current_version() {
    if command -v "$BINARY_NAME" &> /dev/null; then
        CURRENT_VERSION=$($BINARY_NAME version 2>/dev/null || echo "unknown")
        echo -e "${YELLOW}Current version: ${CURRENT_VERSION}${NC}"
    else
        echo -e "${YELLOW}No existing installation found${NC}"
        CURRENT_VERSION="not_installed"
    fi
}

# Detect OS and architecture
detect_system() {
    echo -e "${BLUE}Detecting system architecture...${NC}"
    OS=$(uname | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    if [[ "$ARCH" == "x86_64" ]]; then
        ARCH=amd64
    elif [[ "$ARCH" == "arm64" || "$ARCH" == "aarch64" ]]; then
        ARCH=arm64
    elif [[ "$ARCH" == "armv7l" ]]; then
        ARCH=arm
    else
        echo -e "${RED}Unsupported architecture: $ARCH${NC}"
        exit 1
    fi

    if [[ "$OS" == "darwin" ]]; then
        echo -e "${YELLOW}Detected: macOS $ARCH${NC}"
    elif [[ "$OS" == "linux" ]]; then
        echo -e "${YELLOW}Detected: Linux $ARCH${NC}"
    else
        echo -e "${RED}Unsupported OS: $OS${NC}"
        exit 1
    fi
}

# Download the latest version
download_release() {
    # Determine the latest version
    echo -e "${BLUE}Fetching latest release...${NC}"
    
    # Use hardcoded version if available (from auto-updated script)
    if [ -n "$LATEST_VERSION" ]; then
        LATEST="v$LATEST_VERSION"
        echo -e "${GREEN}Using latest version from script: $LATEST${NC}"
    else
        # Otherwise fetch from GitHub API
        LATEST=$(curl -s https://api.github.com/repos/$REPO/releases/latest | grep tag_name | cut -d '"' -f4)
        if [ -z "$LATEST" ]; then
            echo -e "${RED}Error: Failed to determine latest version. Check your internet connection.${NC}"
            exit 1
        fi
        echo -e "${GREEN}Latest version from GitHub: $LATEST${NC}"
    fi

    # Skip if already at latest version
    if [[ "$CURRENT_VERSION" == "$LATEST" || "$CURRENT_VERSION" == "${LATEST#v}" ]]; then
        echo -e "${GREEN}You already have the latest version installed!${NC}"
        if [ "$1" != "force" ]; then
            echo -e "${YELLOW}Use --force to reinstall anyway.${NC}"
            exit 0
        else
            echo -e "${YELLOW}Force flag detected, reinstalling...${NC}"
        fi
    fi

    # Define download URL
    BINARY="hubsync-${OS}-${ARCH}"
    if [[ "$OS" == "windows" ]]; then
        BINARY="${BINARY}.exe"
    fi
    URL="https://github.com/$REPO/releases/download/$LATEST/$BINARY"

    # Create temporary directory
    TMP=$(mktemp -d)
    trap cleanup EXIT
    echo -e "${BLUE}Using temporary directory: $TMP${NC}"
    cd $TMP

    # Download binary with progress indicator
    echo -e "${BLUE}Downloading $BINARY from $URL...${NC}"
    download_with_progress() {
        local url=$1
        local output=$2
        local max_retries=3
        local retry=0
        
        while [ $retry -lt $max_retries ]; do
            if curl -fSL --progress-bar "$url" -o "$output"; then
                return 0
            else
                retry=$((retry + 1))
                if [ $retry -lt $max_retries ]; then
                    echo -e "${YELLOW}Download failed. Retrying ($retry/$max_retries)...${NC}"
                    sleep 2
                fi
            fi
        done
        return 1
    }
    
    if download_with_progress "$URL" "$BINARY"; then
        echo -e "${GREEN}Download successful!${NC}"
    else
        echo -e "${RED}Download failed after multiple attempts: $URL${NC}"
        echo -e "${YELLOW}Trying fallback to latest tagged binary...${NC}"
        
        # Try the -latest- prefixed version as fallback
        FALLBACK_BINARY="hubsync-latest-${OS}-${ARCH}"
        if [[ "$OS" == "windows" ]]; then
            FALLBACK_BINARY="${FALLBACK_BINARY}.exe"
        fi
        
        FALLBACK_URL="https://github.com/$REPO/releases/download/$LATEST/$FALLBACK_BINARY"
        echo -e "${BLUE}Trying $FALLBACK_URL${NC}"
        
        if download_with_progress "$FALLBACK_URL" "$BINARY"; then
            echo -e "${GREEN}Fallback download successful!${NC}"
        else 
            echo -e "${RED}All download attempts failed.${NC}"
            echo -e "${YELLOW}Trying direct download from the main branch...${NC}"
            
            # Final fallback: try to download from the main branch's latest build
            MAIN_URL="https://raw.githubusercontent.com/$REPO/main/bin/hubsync-${OS}-${ARCH}"
            if [[ "$OS" == "windows" ]]; then
                MAIN_URL="${MAIN_URL}.exe"
            fi
            
            if download_with_progress "$MAIN_URL" "$BINARY"; then
                echo -e "${GREEN}Downloaded development version from main branch${NC}"
                echo -e "${YELLOW}Warning: This may not be a stable release${NC}"
            else
                echo -e "${RED}All download attempts failed.${NC}"
                echo -e "${RED}Please check your internet connection or if the release exists.${NC}"
                exit 1
            fi
        fi
    fi

    # Verify download
    if [ ! -f "$BINARY" ]; then
        echo -e "${RED}Downloaded file not found${NC}"
        exit 1
    fi

    # Check file size to ensure it's not empty or too small
    FILE_SIZE=$(wc -c < "$BINARY")
    if [ "$FILE_SIZE" -lt 1000000 ]; then  # Less than 1MB is suspicious for a Go binary
        echo -e "${YELLOW}Warning: The downloaded binary seems unusually small (${FILE_SIZE} bytes)${NC}"
        echo -e "${YELLOW}It may be incomplete or corrupted.${NC}"
        if [ "$1" != "force" ]; then
            echo -e "${YELLOW}Use --force to install anyway.${NC}"
            exit 1
        fi
    fi

    # Make binary executable
    chmod +x "$BINARY"
}

# Install binary to system
install_binary() {
    # Check if install directory exists and is writable
    if [ ! -d "$INSTALL_DIR" ]; then
        echo -e "${YELLOW}Install directory $INSTALL_DIR does not exist. Creating...${NC}"
        if ! mkdir -p "$INSTALL_DIR" 2>/dev/null; then
            echo -e "${RED}Failed to create install directory. Trying with sudo...${NC}"
            sudo mkdir -p "$INSTALL_DIR" || {
                echo -e "${RED}Failed to create install directory even with sudo.${NC}"
                exit 1
            }
        fi
    fi

    # Backup existing binary if present
    if [ -f "$INSTALL_DIR/$BINARY_NAME" ]; then
        echo -e "${YELLOW}Backing up existing binary...${NC}"
        if ! cp "$INSTALL_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME.backup" 2>/dev/null; then
            echo -e "${YELLOW}Using sudo to create backup...${NC}"
            sudo cp "$INSTALL_DIR/$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME.backup" || {
                echo -e "${YELLOW}Couldn't create backup. Continuing anyway...${NC}"
            }
        fi
    fi

    # Install binary
    echo -e "${BLUE}Installing to $INSTALL_DIR/$BINARY_NAME...${NC}"
    if ! mv "$BINARY" "$INSTALL_DIR/$BINARY_NAME" 2>/dev/null; then
        echo -e "${YELLOW}Insufficient permissions. Using sudo...${NC}"
        sudo mv "$BINARY" "$INSTALL_DIR/$BINARY_NAME" || {
            echo -e "${RED}Installation failed! Could not move binary to $INSTALL_DIR${NC}"
            echo -e "${RED}Try running this script with elevated privileges.${NC}"
            exit 1
        }
    fi

    # Verify installation
    echo -e "${BLUE}Verifying installation...${NC}"
    if command -v "$INSTALL_DIR/$BINARY_NAME" &> /dev/null; then
        echo -e "${GREEN}Installation verified!${NC}"
        VERSION_OUTPUT=$("$INSTALL_DIR/$BINARY_NAME" version 2>/dev/null || echo "unknown")
        echo -e "${GREEN}HubSync version: $VERSION_OUTPUT${NC}"
    else
        echo -e "${RED}Verification failed. Please check your PATH settings.${NC}"
    fi

    # Show usage hint
    echo -e "\n${CYAN}To get started with HubSync, run:${NC}"
    echo -e "  ${YELLOW}$BINARY_NAME --help${NC}"
}

# Verify installation
verify_installation() {
    if command -v "$BINARY_NAME" &> /dev/null; then
        INSTALLED_VERSION=$($BINARY_NAME version 2>/dev/null || echo "unknown")
        echo -e "${GREEN}HubSync ${INSTALLED_VERSION} successfully installed to $INSTALL_DIR/$BINARY_NAME${NC}"
        
        # Check if the installed binary is in PATH
        if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
            echo -e "${YELLOW}Warning: $INSTALL_DIR is not in your PATH.${NC}"
            case "$SHELL" in
                */bash)
                    echo -e "${YELLOW}Add to your ~/.bashrc or ~/.bash_profile:${NC}"
                    echo -e "${BLUE}export PATH=\"\$PATH:$INSTALL_DIR\"${NC}"
                    ;;
                */zsh)
                    echo -e "${YELLOW}Add to your ~/.zshrc:${NC}"
                    echo -e "${BLUE}export PATH=\"\$PATH:$INSTALL_DIR\"${NC}"
                    ;;
                */fish)
                    echo -e "${YELLOW}For fish shell, run:${NC}"
                    echo -e "${BLUE}fish_add_path $INSTALL_DIR${NC}"
                    ;;
                *)
                    echo -e "${YELLOW}Add this directory to your PATH:${NC}"
                    echo -e "${BLUE}export PATH=\"\$PATH:$INSTALL_DIR\"${NC}"
                    ;;
            esac
        fi
        
        # Add shell completion suggestion
        echo -e "${CYAN}To enable shell completion, run:${NC}"
        echo -e "  ${YELLOW}$BINARY_NAME completion [bash|zsh|fish] > /path/to/completion/file${NC}"
    else
        echo -e "${RED}Installation verification failed.${NC}"
        echo -e "${RED}Please check if $INSTALL_DIR is in your PATH.${NC}"
        exit 1
    fi
}

# Print usage instructions
print_usage() {
    echo ""
    echo -e "${CYAN}=== USAGE INSTRUCTIONS ===${NC}"
    echo ""
    echo -e "${BLUE}Create a .env file with your Docker credentials:${NC}"
    echo -e "${YELLOW}DOCKER_USERNAME=your_username"
    echo -e "DOCKER_PASSWORD=your_token_or_password"
    echo -e "DOCKER_REPOSITORY=your_repository # Optional"
    echo -e "DOCKER_NAMESPACE=your_namespace # Optional${NC}"
    echo ""
    echo -e "Run ${BLUE}hubsync --help${NC} for a list of commands"
    echo -e "Example: ${BLUE}hubsync --content='{\"hubsync\":[\"nginx:latest\"]}'${NC}"
    echo ""
    echo -e "${GREEN}For a guided setup, run our quickstart script:${NC}"
    echo -e "${BLUE}curl -fsSL https://raw.githubusercontent.com/$REPO/main/quickstart.sh | bash${NC}"
    echo ""
    echo -e "${GREEN}Thank you for installing HubSync!${NC}"
    
    # Check for updates to this script
    if [ -n "$LATEST_VERSION" ] && [ -n "$LAST_UPDATED" ]; then
        echo ""
        echo -e "${CYAN}This installer script was last updated: ${LAST_UPDATED}${NC}"
        TODAY=$(date +%Y-%m-%d)
        # Convert dates to seconds since epoch for comparison
        LAST_SEC=$(date -d "$LAST_UPDATED" +%s 2>/dev/null || date -j -f "%Y-%m-%d" "$LAST_UPDATED" +%s 2>/dev/null)
        TODAY_SEC=$(date -d "$TODAY" +%s 2>/dev/null || date -j -f "%Y-%m-%d" "$TODAY" +%s 2>/dev/null)
        
        # If the dates can be compared and it's been more than 30 days
        if [ -n "$LAST_SEC" ] && [ -n "$TODAY_SEC" ]; then
            DAYS_DIFF=$(( (TODAY_SEC - LAST_SEC) / 86400 ))
            if [ $DAYS_DIFF -gt 30 ]; then
                echo -e "${YELLOW}This installer script is over 30 days old. Consider getting a fresh copy:${NC}"
                echo -e "${BLUE}curl -fsSL https://raw.githubusercontent.com/$REPO/main/install.sh | bash${NC}"
            fi
        fi
    fi
}

# Main execution
main() {
    # Process arguments
    FORCE=false
    if [[ "$1" == "--force" || "$1" == "-f" ]]; then
        FORCE=true
    fi
    
    # Run the installation steps
    echo -e "${BLUE}Starting HubSync installation...${NC}"
    check_dependencies
    check_current_version
    detect_system
    if $FORCE; then
        download_release "force"
    else
        download_release
    fi
    install_binary
    verify_installation
    print_usage
}

# Execute main function with all arguments
main "$@"
