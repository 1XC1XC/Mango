#!/bin/bash

set -e

mango_dir="$HOME/.mango"

echo "Installing Mango..."

if [[ ! -d $mango_dir ]]; then
    echo "Creating Mango directories..."
    mkdir -p $mango_dir/bin $mango_dir/cache $mango_dir/version
fi

fetch_latest_release_url() {
    GitURL="https://api.github.com/repos/1XC1XC/Mango/releases/latest"
    arch=$(uname -m)

    case $arch in
        x86_64)
            grep_arch="amd64"
            ;;
        i386|i686)
            grep_arch="386"
            ;;
        arm64|aarch64)
            grep_arch="arm64"
            ;;
        arm*)
            grep_arch="arm"
            ;;
        *)
            echo "Unsupported architecture: $arch"
            exit 1
            ;;
    esac

    download_url=$(curl -s $GitURL | grep "browser_download_url" | grep "linux" | grep "$grep_arch" | cut -d '"' -f 4)
    echo $download_url
}

echo "Fetching the latest Mango release..."
download_url=$(fetch_latest_release_url)

echo "Downloading Mango..."
curl -L $download_url -o mango_latest.tar.gz

echo "Extracting Mango..."
tar -xzf mango_latest.tar.gz -C $mango_dir/bin

echo "Cleaning up..."
rm mango_latest.tar.gz

if [[ ":$PATH:" == *":$mango_dir/bin:"* ]]; then
    echo "PATH already includes '$mango_dir/bin'"
else
    echo "Updating PATH and setting up autocompletion..."

    shell=$(basename $SHELL)
    case $shell in
        bash)
            echo "export PATH=\"$mango_dir/bin:\$PATH\"" >> ~/.bashrc
            $mango_dir/bin/mango completion bash > /usr/local/etc/bash_completion.d/mango
            source ~/.bashrc
            echo "PATH modification and autocompletion added to .bashrc"
            ;;
        zsh)
            echo "export PATH=\"$mango_dir/bin:\$PATH\"" >> ~/.zshrc
            $mango_dir/bin/mango completion zsh > /usr/local/share/zsh/site-functions/_mango
            source ~/.zshrc
            echo "PATH modification and autocompletion added to .zshrc"
            ;;
        fish) 
            echo "set -gx PATH \"$mango_dir/bin\" \$PATH" >> ~/.config/fish/config.fish
            $mango_dir/bin/mango completion fish > ~/.config/fish/completions/mango.fish
            source ~/.config/fish/config.fish
            echo "PATH modification and autocompletion added to fish config"
            ;;
        *)
            echo "Unsupported shell. Please add '$mango_dir/bin' to your PATH manually."
            ;;
    esac
fi

echo "Mango installation completed successfully!"
