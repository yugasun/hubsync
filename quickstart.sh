#!/bin/bash
# HubSync Quick Start Script
# This script helps first-time users set up and run HubSync

# Colors for terminal output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}"
echo "██╗  ██╗██╗   ██╗██████╗ ███████╗██╗   ██╗███╗   ██╗ ██████╗"
echo "██║  ██║██║   ██║██╔══██╗██╔════╝╚██╗ ██╔╝████╗  ██║██╔════╝"
echo "███████║██║   ██║██████╔╝███████╗ ╚████╔╝ ██╔██╗ ██║██║     "
echo "██╔══██║██║   ██║██╔══██╗╚════██║  ╚██╔╝  ██║╚██╗██║██║     "
echo "██║  ██║╚██████╔╝██████╔╝███████║   ██║   ██║ ╚████║╚██████╗"
echo "╚═╝  ╚═╝ ╚═════╝ ╚═════╝ ╚══════╝   ╚═╝   ╚═╝  ╚═══╝ ╚═════╝"
echo -e "${NC}"
echo "Docker Hub Image Synchronization Tool - Quick Start Guide"
echo "This script will help you set up HubSync for first use."
echo ""

# Check if HubSync is installed
if ! command -v hubsync &> /dev/null; then
    echo -e "${YELLOW}HubSync is not installed. Would you like to install it now? (y/n)${NC}"
    read -r install_answer
    if [[ "$install_answer" =~ ^[Yy]$ ]]; then
        echo -e "${BLUE}Installing HubSync...${NC}"
        if [[ "$(uname)" == "Darwin" ]] && command -v brew &> /dev/null; then
            echo -e "${BLUE}Detected Homebrew, installing via brew...${NC}"
            brew tap yugasun/hubsync https://github.com/yugasun/hubsync
            brew install hubsync
        else
            echo -e "${BLUE}Installing via shell script...${NC}"
            bash -c "$(curl -fsSL https://raw.githubusercontent.com/yugasun/hubsync/main/install.sh)"
        fi
    else
        echo -e "${RED}HubSync installation skipped. Please install HubSync and run this script again.${NC}"
        exit 1
    fi
fi

# Check if .env file exists
if [ -f .env ]; then
    echo -e "${BLUE}Found existing .env file. Would you like to use these settings? (y/n)${NC}"
    read -r env_answer
    if [[ "$env_answer" =~ ^[Yy]$ ]]; then
        echo -e "${GREEN}Using existing .env file.${NC}"
    else
        setup_env=true
    fi
else
    setup_env=true
fi

# Set up .env file if needed
if [[ "$setup_env" == true ]]; then
    echo -e "${BLUE}Setting up your Docker credentials...${NC}"
    
    echo -e "${YELLOW}Enter your Docker username:${NC}"
    read -r docker_username
    
    echo -e "${YELLOW}Enter your Docker password/token:${NC}"
    read -r -s docker_password
    echo ""
    
    echo -e "${YELLOW}Enter your Docker repository (leave blank for Docker Hub):${NC}"
    read -r docker_repository
    
    echo -e "${YELLOW}Enter your Docker namespace (leave blank for 'yugasun'):${NC}"
    read -r docker_namespace
    
    # Create .env file
    echo -e "${BLUE}Creating .env file...${NC}"
    cat > .env << EOF
DOCKER_USERNAME=${docker_username}
DOCKER_PASSWORD=${docker_password}
EOF

    if [[ -n "$docker_repository" ]]; then
        echo "DOCKER_REPOSITORY=${docker_repository}" >> .env
    fi
    
    if [[ -n "$docker_namespace" ]]; then
        echo "DOCKER_NAMESPACE=${docker_namespace}" >> .env
    fi
    
    echo -e "${GREEN}.env file created successfully.${NC}"
fi

# Ask for images to sync
echo -e "${BLUE}Let's set up your first sync job.${NC}"
echo -e "${YELLOW}Enter the images to sync (comma-separated, e.g., nginx:latest,redis:6):${NC}"
read -r images

# Convert input to proper JSON format
IFS=',' read -ra IMAGE_ARRAY <<< "$images"
json_content='{"hubsync":['
for i in "${!IMAGE_ARRAY[@]}"; do
    if [ $i -gt 0 ]; then
        json_content+=','
    fi
    json_content+="\"${IMAGE_ARRAY[$i]}\""
done
json_content+=']}'

# Run HubSync
echo -e "${BLUE}Starting HubSync with the following images:${NC}"
for img in "${IMAGE_ARRAY[@]}"; do
    echo -e "  - ${YELLOW}$img${NC}"
done

echo -e "${BLUE}Running HubSync...${NC}"
source .env
hubsync --content="$json_content"

# Check the result
if [ $? -eq 0 ]; then
    echo -e "${GREEN}Sync completed successfully! Check output.log for details.${NC}"
    echo -e "${BLUE}You can run future syncs with:${NC}"
    echo -e "  ${YELLOW}hubsync --content='$json_content'${NC}"
    echo -e "${BLUE}Or add more configuration options:${NC}"
    echo -e "  ${YELLOW}hubsync --outputPath=custom.log --concurrency=5 --content='$json_content'${NC}"
else
    echo -e "${RED}Sync failed. Please check the error messages above.${NC}"
fi

echo ""
echo -e "${GREEN}Thank you for using HubSync!${NC}"