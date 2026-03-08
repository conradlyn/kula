#!/usr/bin/env bash

set -e

VERSION="0.7.2"
GITHUB_REPO="c0m4r/kula"
RELEASE_URL="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}"

# Define colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}===========================================${NC}"
echo -e "${CYAN}      kula - system monitoring daemon      ${NC}"
echo -e "${CYAN}===========================================${NC}"
echo -e "Version: ${VERSION}"
echo ""

# BETA WARNING
echo -e "${YELLOW}Warning: This script is in beta and might not work as expected.${NC}"
echo ""

read -p "Do you want to continue? [y/N] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${RED}Installation aborted.${NC}"
    exit 1
fi

# Hashes mapping
declare -A HASHES
HASHES["kula-0.7.2-amd64.deb"]="51be1330fff5262a6541f1c60b749593abfc4dc82cb322a41d1ae2b15c81f758"
HASHES["kula-0.7.2-amd64.tar.gz"]="8fd7ec391db8245d3988b3f50aa013d02df7b14d3de06d35b175099f7a52e064"
HASHES["kula-0.7.2-arm64.deb"]="f91694ae18c7523d5c3c54f8fbb272cbe42c24ae456dc0611eb1b2b6f093f5d9"
HASHES["kula-0.7.2-arm64.tar.gz"]="7e91291368de3445cc445a8153f7d31c93a61196b062ec1781d8c7219685fe36"
HASHES["kula-0.7.2-aur.tar.gz"]="5d3882fd0c4d46e08ae6d70e848960505eb0997cf816982af9e99480fad8f24e"
HASHES["kula-0.7.2-riscv64.deb"]="1c5a5fa3dab2eea8246735709ac6c98825d6432dad5bd73fe80d62000c51483d"
HASHES["kula-0.7.2-riscv64.tar.gz"]="f56d659bc07143d49af560a37b87ab7c1e1bb090188765e3fb3f09d1a267e0d2"

# Detect Architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64) HOST_ARCH="amd64" ;;
    aarch64) HOST_ARCH="arm64" ;;
    riscv64) HOST_ARCH="riscv64" ;;
    *)
        echo -e "${RED}Error: Unsupported architecture $ARCH${NC}"
        exit 1
        ;;
esac
echo -e "Detected Architecture: ${GREEN}${HOST_ARCH}${NC}"

# Detect OS
OS_FAMILY="unknown"
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS_ID=${ID}
    OS_LIKE=${ID_LIKE:-""}
    
    if [[ "$OS_ID" == "debian" || "$OS_ID" == "ubuntu" || "$OS_LIKE" == *"debian"* || "$OS_LIKE" == *"ubuntu"* ]]; then
        OS_FAMILY="debian"
    elif [[ "$OS_ID" == "arch" || "$OS_ID" == "manjaro" || "$OS_LIKE" == *"arch"* ]]; then
        OS_FAMILY="arch"
    elif [[ "$OS_ID" == "fedora" || "$OS_ID" == "rhel" || "$OS_ID" == "rocky" || "$OS_ID" == "alma" || "$OS_LIKE" == *"fedora"* || "$OS_LIKE" == *"rhel"* ]]; then
        OS_FAMILY="rpm"
    fi
fi
echo -e "Detected OS Family: ${GREEN}${OS_FAMILY}${NC}"

# Detect Init System
INIT_SYSTEM="unknown"
if command -v systemctl >/dev/null 2>&1 && systemctl --no-pager >/dev/null 2>&1 || [ -d /run/systemd/system ]; then
    INIT_SYSTEM="systemd"
fi
echo -e "Detected Init System: ${GREEN}${INIT_SYSTEM}${NC}"

# Download function
download_and_verify() {
    local filename=$1
    local expected_hash=${HASHES[$filename]}
    local target="/tmp/$filename"
    local url="${RELEASE_URL}/${filename}"

    if [ -z "$expected_hash" ]; then
        echo -e "${RED}Error: No hash found for $filename. This version might not support your platform yet.${NC}"
        exit 1
    fi

    echo -e "${BLUE}Downloading $filename...${NC}" >&2
    if command -v curl >/dev/null; then
        curl -sL "$url" -o "$target"
    elif command -v wget >/dev/null; then
        wget -qO "$target" "$url"
    else
        echo -e "${RED}Error: Neither curl nor wget is installed.${NC}"
        exit 1
    fi

    echo -e "${BLUE}Checking SHA256 sum...${NC}" >&2
    local actual_hash
    if command -v sha256sum >/dev/null; then
        actual_hash=$(sha256sum "$target" | awk '{print $1}')
    else
        actual_hash=$(shasum -a 256 "$target" | awk '{print $1}')
    fi

    if [ "$actual_hash" != "$expected_hash" ]; then
        echo -e "${RED}Error: Checksum mismatch for $filename!${NC}" >&2
        echo -e "${YELLOW}Expected: $expected_hash${NC}" >&2
        echo -e "${YELLOW}Got:      $actual_hash${NC}" >&2
        rm -f "$target"
        exit 1
    fi
    echo -e "${GREEN}Checksum verified successfully for $filename.${NC}" >&2
    echo "$target"
}

# Determine action
INSTALL_METHOD=""

if [ "$OS_FAMILY" == "debian" ]; then
    INSTALL_METHOD="deb"
elif [ "$OS_FAMILY" == "rpm" ]; then
    INSTALL_METHOD="rpm"
elif [ "$OS_FAMILY" == "arch" ] && command -v pacman >/dev/null; then
    INSTALL_METHOD="aur"
else
    # Fallback options
    if [ "$INIT_SYSTEM" == "systemd" ]; then
        INSTALL_METHOD="tarball_systemd"
    elif command -v docker >/dev/null; then
        INSTALL_METHOD="docker"
    else
        INSTALL_METHOD="tarball_opt"
    fi
fi

echo -e "\nProposed installation method: ${YELLOW}${INSTALL_METHOD}${NC}"
read -p "Do you want to continue with this installation method? [y/N] " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo -e "${RED}Installation aborted.${NC}"
    exit 1
fi

installer_sudo=""
if [ "$(id -u)" -ne 0 ]; then
    if command -v sudo >/dev/null; then
        installer_sudo="sudo"
    elif command -v doas >/dev/null; then
        installer_sudo="doas"
    elif command -v su >/dev/null; then
        installer_sudo="su -c"
    else
        echo -e "${YELLOW}Warning: You are not root and sudo is not available. Installation may fail.${NC}"
    fi
fi

echo ""

# Execute installation
if [ "$INSTALL_METHOD" == "deb" ]; then
    filename="kula-${VERSION}-${HOST_ARCH}.deb"
    target=$(download_and_verify "$filename")
    echo -e "${BLUE}Installing Debian package...${NC}"
    $installer_sudo dpkg -i "$target" || $installer_sudo apt-get install -f -y "$target"
    rm -f "$target"
    echo -e "${GREEN}Installation successful!${NC}"

elif [ "$INSTALL_METHOD" == "rpm" ]; then
    RPM_ARCH=$HOST_ARCH
    if [ "$HOST_ARCH" == "amd64" ]; then RPM_ARCH="x86_64"; fi
    if [ "$HOST_ARCH" == "arm64" ]; then RPM_ARCH="aarch64"; fi
    
    filename="kula-${VERSION}-${RPM_ARCH}.rpm"
    target=$(download_and_verify "$filename")
    echo -e "${BLUE}Installing RPM package...${NC}"
    if command -v dnf >/dev/null; then
        $installer_sudo dnf install -y "$target"
    elif command -v yum >/dev/null; then
        $installer_sudo yum install -y "$target"
    else
        $installer_sudo rpm -ivh "$target"
    fi
    rm -f "$target"
    echo -e "${GREEN}Installation successful!${NC}"

elif [ "$INSTALL_METHOD" == "aur" ]; then
    filename="kula-${VERSION}-aur.tar.gz"
    # For makepkg, we should NOT be root.
    if [ "$(id -u)" -eq 0 ]; then
        echo -e "${RED}Error: AUR installation should not be run as root.${NC}"
        echo -e "Please run this script as a normal user with sudo privileges."
        exit 1
    fi
    target=$(download_and_verify "$filename")
    
    echo -e "${BLUE}Extracting and building AUR package...${NC}"
    build_dir="/tmp/kula-aur-build"
    rm -rf "$build_dir"
    mkdir -p "$build_dir"
    tar -xzf "$target" -C "$build_dir"
    
    cd "$build_dir/kula-$VERSION-aur"
    makepkg -si
    cd - >/dev/null
    rm -rf "$build_dir" "$target"
    echo -e "${GREEN}Installation successful!${NC}"

elif [ "$INSTALL_METHOD" == "tarball_systemd" ]; then
    filename="kula-${VERSION}-${HOST_ARCH}.tar.gz"
    target=$(download_and_verify "$filename")
    
    echo -e "${BLUE}Installing from tarball to system directories...${NC}"
    extract_dir="/tmp/kula_extract_$$"
    mkdir -p "$extract_dir"
    tar -xzf "$target" -C "$extract_dir"
    
    cd "$extract_dir/kula"
    $installer_sudo install -Dm755 kula /usr/bin/kula
    $installer_sudo install -Dm644 addons/init/systemd/kula.service /etc/systemd/system/kula.service
    $installer_sudo install -Dm640 config.example.yaml /etc/kula/config.example.yaml
    $installer_sudo install -dm750 /var/lib/kula
    
    if ! getent group kula >/dev/null; then
        $installer_sudo groupadd --system kula
    fi
    if ! getent passwd kula >/dev/null; then
        $installer_sudo useradd --system -g kula -d /var/lib/kula -s /bin/false -c "Kula System Monitoring Daemon" kula
    fi
    $installer_sudo chown -R kula:kula /etc/kula /var/lib/kula
    
    echo -e "${BLUE}Reloading systemd and enabling service...${NC}"
    $installer_sudo systemctl daemon-reload
    $installer_sudo systemctl enable kula.service
    $installer_sudo systemctl start kula.service
    
    cd - >/dev/null
    rm -rf "$extract_dir" "$target"
    echo -e "${GREEN}Installation successful!${NC}"
    
elif [ "$INSTALL_METHOD" == "docker" ]; then
    echo -e "${BLUE}Docker is installed. You can run Kula via Docker container.${NC}"
    echo -e "Run the following command to start Kula:"
    echo -e "${CYAN}docker run -d --name kula --net host -v kula_data:/var/lib/kula c0m4r/kula:latest${NC}"
    echo -e "To persist configuration, use volume mounts and provide your config.yaml."

elif [ "$INSTALL_METHOD" == "tarball_opt" ]; then
    filename="kula-${VERSION}-${HOST_ARCH}.tar.gz"
    target=$(download_and_verify "$filename")
    
    echo -e "${BLUE}Installing to /opt/kula...${NC}"
    if [ ! -d "/opt/kula" ]; then
        $installer_sudo mkdir -p /opt/kula
    fi
    $installer_sudo tar -xzf "$target" -C /opt
    
    rm -f "$target"
    echo -e "${GREEN}Extracted to /opt/kula successfully.${NC}"
    echo -e "To run Kula manually:"
    echo -e "${CYAN}  cd /opt/kula${NC}"
    echo -e "${CYAN}  cp config.example.yaml config.yaml${NC}"
    echo -e "${CYAN}  ./kula serve${NC}"
fi

echo -e "\n${GREEN}Thank you for installing Kula!${NC}"
