#!/bin/bash
# HubSync Installer
# This script downloads and installs the latest version of HubSync
set -e

# Configuration
REPO="yugasun/hubsync"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="hubsync"

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
    LATEST=$(curl -s https://api.github.com/repos/$REPO/releases/latest | grep tag_name | cut -d '"' -f4)
    if [ -z "$LATEST" ]; then
        echo -e "${RED}Error: Failed to determine latest version. Check your internet connection.${NC}"
        exit 1
    fi
    echo -e "${GREEN}Latest version: $LATEST${NC}"

    # Skip if already at latest version
    if [[ "$CURRENT_VERSION" == "$LATEST" ]]; then
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
    URL="https://github.com/$REPO/releases/download/$LATEST/$BINARY"

    # Create temporary directory
    TMP=$(mktemp -d)
    trap cleanup EXIT
    echo -e "${BLUE}Using temporary directory: $TMP${NC}"
    cd $TMP

    # Download binary
    echo -e "${BLUE}Downloading $BINARY...${NC}"
    if ! curl -fsSLO --retry 3 "$URL" & pid=$!; then
        show_spinner $pid "Downloading"
        wait $pid
    else
        echo -e "${RED}Download failed: $URL${NC}"
        echo -e "${RED}Please check your internet connection or if the release exists.${NC}"
        exit 1
    fi

    # Verify download
    if [ ! -f "$BINARY" ]; then
        echo -e "${RED}Downloaded file not found${NC}"
        exit 1
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
        echo -e "${GREEN}HubSync version: $($INSTALL_DIR/$BINARY_NAME version)${NC}"
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
            echo -e "${YELLOW}You may need to add it with: ${NC}${BLUE}export PATH=$PATH:$INSTALL_DIR${NC}"
        fi
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
