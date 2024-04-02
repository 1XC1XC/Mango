#!/usr/bin/env bash

set -euo pipefail

mango_dir="$HOME/.mango"
bin_dir="$mango_dir/bin"
cache_dir="$mango_dir/cache"
version_dir="$mango_dir/version"
exe="$bin_dir/mango"

quietly() {
  "$@" >/dev/null 2>&1
}

shell=$(basename "$SHELL")

case $(uname -m) in
x86_64) arch="amd64" ;;
i386 | i686) arch="386" ;;
arm64 | aarch64) arch="arm64" ;;
arm*) arch="arm" ;;
*) error "Unsupported architecture: $(uname -m)" ;;
esac

fetch_latest_release() {
    curl -s "https://api.github.com/repos/1XC1XC/Mango/releases/latest"
}

extract_download_url() {
    grep "browser_download_url" | grep "linux" | grep "$arch" | cut -d '"' -f 4
}

quietly mkdir -p "$bin_dir" "$cache_dir" "$version_dir"

release_info=$(fetch_latest_release)

download_url=$(echo "$release_info" | extract_download_url)
if [[ -z $download_url ]]; then
    error "Failed to find a suitable Mango release for your architecture"
fi

if command -v curl &>/dev/null; then
  curl -fsSL "$download_url" -o "$exe.tar.gz" || error "Failed to download Mango"
else 
  error "curl is required to download Mango. Please install it and run this script again."
fi

tar -xzf "$exe.tar.gz" -C "$bin_dir" || error "Failed to extract Mango archive"
rm "$exe.tar.gz"

chmod +x "$exe" || error "Failed to set execute permission on Mango binary"

update_shell_config() {
    config_file="$1"

    if grep -q "^export PATH=.*$bin_dir" "$config_file"; then
        echo "PATH already includes $bin_dir in ~/${config_file##*/}"
    else
        read -p "Do you want to update your shell configuration file (~/${config_file##*/}) to include Mango in PATH? [y/N]: " choice
        case "$choice" in
            y|Y)
                echo -e "\n# Add Mango to PATH\nexport PATH=\"$bin_dir:\$PATH\"" >>"$config_file"
                echo "Updated ~/${config_file##*/} with Mango PATH"
                updated=true
                ;;
            *)
                echo "Skipping shell configuration update."
                updated=false
                ;;
        esac
    fi
}

case $shell in
fish)
    config_file="$HOME/.config/fish/config.fish"
    mkdir -p "$(dirname "$config_file")"
    update_shell_config "$config_file" "set -gx PATH \"$bin_dir\" \$PATH"
    ;;
zsh | bash) 
    config_file="$HOME/.zshrc" 
    update_shell_config "$config_file" "source $config_file"
    ;;
*)
    error "Unsupported shell: $shell"
    ;;
esac

echo "Mango installation completed successfully!"
if [[ $updated = true ]]; then
    echo "Please restart your terminal or run 'source ~/${config_file##*/}' to start using Mango."
else
    echo "You'll need to manually add Mango to your PATH by adding the following line to your shell configuration file (~/${config_file##*/}):"
    echo "export PATH=\"$bin_dir:\$PATH\""
fi
echo "You can then run 'mango --help' to get started."
