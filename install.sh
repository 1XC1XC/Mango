#!/usr/bin/env bash

set -euo pipefail

mango_dir="$HOME/.mango"
bin_dir="$mango_dir/bin"
exe="$bin_dir/mango"

info() {
    echo "INFO: $*"
}

error() {
    echo "ERROR: $*" >&2
    exit 1
}

# Detect the user's shell
shell=$(basename "$SHELL")

# Detect the system architecture
case $(uname -m) in
x86_64) arch="amd64" ;;
i386 | i686) arch="386" ;;
arm64 | aarch64) arch="arm64" ;;
arm*) arch="arm" ;;
*) error "Unsupported architecture: $(uname -m)" ;;
esac

# Function to fetch the latest release information from the GitHub API
fetch_latest_release() {
    curl -s "https://api.github.com/repos/1XC1XC/Mango/releases/latest"
}

# Function to extract the download URL for the current architecture
extract_download_url() {
    grep "browser_download_url" |
        grep "linux" |
        grep "$arch" |
        cut -d '"' -f 4
}

# Create the necessary directories
mkdir -p "$bin_dir" || error "Failed to create $bin_dir directory"

# Fetch the latest release information
info "Fetching the latest Mango release"
release_info=$(fetch_latest_release)

# Extract the download URL for the current architecture
download_url=$(echo "$release_info" | extract_download_url)
if [[ -z $download_url ]]; then
    error "Failed to find a suitable Mango release for your architecture"
fi

# Download the latest Mango release
info "Downloading Mango from $download_url"
curl -fsSL "$download_url" -o "$exe.tar.gz" || error "Failed to download Mango"

# Extract the downloaded archive
info "Extracting Mango archive"
tar -xzf "$exe.tar.gz" -C "$bin_dir" || error "Failed to extract Mango archive"
rm "$exe.tar.gz"

# Make the Mango binary executable
chmod +x "$exe" || error "Failed to set execute permission on Mango binary"

# Function to update the shell configuration
update_shell_config() {
    local config_file="$1"
    local source_command="$2"

    if [[ -w $config_file ]]; then
        echo "$source_command" >>"$config_file"
        info "Updated $config_file with Mango PATH"
        echo "source $config_file"
    else
        info "Manually add the following line to $config_file:"
        info "$source_command"
    fi
}

# Update the shell configuration based on the detected shell
case $shell in
fish)
    config_file="$HOME/.config/fish/config.fish"
    source_command="set -gx PATH \"$bin_dir\" \$PATH"
    update_shell_config "$config_file" "$source_command"
    ;;
zsh)
    config_file="$HOME/.zshrc"
    source_command="export PATH=\"$bin_dir:\$PATH\""
    update_shell_config "$config_file" "$source_command"
    ;;
bash)
    config_file="$HOME/.bashrc"
    source_command="export PATH=\"$bin_dir:\$PATH\""
    update_shell_config "$config_file" "$source_command"
    ;;
*)
    error "Unsupported shell: $shell"
    ;;
esac

info "Mango installation completed successfully!"
info "Run 'mango --help' to get started."
