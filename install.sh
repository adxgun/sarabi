#!/bin/bash

# Function to get the server's public IP address
get_public_ip() {
  PUBLIC_IP=$(curl -s ifconfig.me)
  echo "Server Public IP: $PUBLIC_IP"
}

# Function to prompt for a domain name and obtain Let's Encrypt certificate
setup_domain() {
  read -p "Enter your domain name (leave blank to use IP address): " DOMAIN_NAME

  if [ -n "$DOMAIN_NAME" ]; then
    echo "Domain name provided: $DOMAIN_NAME"

    # Install Certbot and obtain Let's Encrypt certificate
    sudo apt update
    sudo apt install -y certbot
    sudo certbot certonly --standalone -d "$DOMAIN_NAME" --non-interactive --agree-tos -m admin@$DOMAIN_NAME

    # Set environment variables for certificate paths
    export SERVER_SSL_CERT_FILE="/etc/letsencrypt/live/$DOMAIN_NAME/fullchain.pem"
    export SERVER_SSL_KEY_FILE="/etc/letsencrypt/live/$DOMAIN_NAME/privkey.pem"

    echo "SSL_CERT_FILE=$SSL_CERT_FILE"
    echo "SSL_KEY_FILE=$SSL_KEY_FILE"
  else
    echo "No domain name provided. Using IP address: $PUBLIC_IP"
  fi
}

# Function to install Docker
install_docker() {
  sudo apt update
  sudo apt install -y docker.io
  sudo systemctl enable docker
  sudo systemctl start docker
}

# Function to download the latest Sarabi binary from GitHub releases
download_sarabi() {
  LATEST_LINUX_URL=$(
    curl -s https://api.github.com/repos/adxgun/sarabi/releases/latest \
    | grep "browser_download_url" \
    | grep "linux_amd64" \
    | cut -d '"' -f 4
  )

  if [ -z "$LATEST_LINUX_URL" ]; then
    echo "Failed to fetch sarabi Linux release URL. Exiting."
    exit 1
  fi

  echo "Downloading sarabi (Linux) binary from $LATEST_LINUX_URL"
  wget -O sarabi "$LATEST_LINUX_URL"
  chmod +x sarabi
}

# Function to generate secrets
generate_access_secret() {
  ACCESS_SECRET=$(openssl rand -hex 16)
  export ACCESS_SECRET
}

generate_or_recover_encryption_key() {
  sudo mkdir -p /etc/sarabi

  # Check if the encryption key file exists
  if [ -f "$ENCRYPTION_KEY_FILE" ]; then
    echo "🔑 Recovering existing encryption key from $ENCRYPTION_KEY_FILE..."
    ENCRYPTION_KEY=$(sudo cat "$ENCRYPTION_KEY_FILE")
  else
    echo "🔑 Generating a new encryption key..."
    ENCRYPTION_KEY=$(openssl rand -hex 32)
    echo "$ENCRYPTION_KEY" | sudo tee "$ENCRYPTION_KEY_FILE" > /dev/null
    sudo chmod 600 "$ENCRYPTION_KEY_FILE" # Restrict access to the key file
  fi

  export ENCRYPTION_KEY
}

# Function to start Sarabi as a service
start_sarabi_service() {
  cat <<EOF | sudo tee /etc/systemd/system/sarabi.service > /dev/null
[Unit]
Description=Sarabi Service
After=network.target

[Service]
ExecStart=$(pwd)/sarabi
Environment=ACCESS_SECRET=$ACCESS_SECRET
Environment=ENCRYPTION_KEY=$ENCRYPTION_KEY
$(if [ -n "$DOMAIN_NAME" ]; then
  echo "Environment=SSL_CERT_FILE=$SSL_CERT_FILE"
  echo "Environment=SSL_KEY_FILE=$SSL_KEY_FILE"
fi)
Restart=always
User=root

[Install]
WantedBy=multi-user.target
EOF

  sudo systemctl daemon-reload
  sudo systemctl enable sarabi
  sudo systemctl start sarabi
}

# Function to output success message and secrets
output_success_message() {
  echo -e "\n🎉🎉🎉 \033[1;32mCongratulations! Sarabi is now up and running!\033[0m 🎉🎉🎉"
  echo -e "\n🌟 \033[1;36mHere are the details:\033[0m 🌟"

  if [ -n "$DOMAIN_NAME" ]; then
    echo -e "\n🌐 \033[1;33mAccess URL:\033[0m \033[1;32mhttps://$DOMAIN_NAME:3646/\033[0m"
  else
    echo -e "\n🌐 \033[1;33mAccess URL:\033[0m \033[1;32mhttp://$PUBLIC_IP:3646/\033[0m"
  fi

  echo -e "\n🔑 \033[1;33mGenerated Secrets:\033[0m"
  echo -e "   \033[1;34mACCESS_SECRET:\033[0m \033[1;35m$ACCESS_SECRET\033[0m"
  echo -e "   \033[1;34mENCRYPTION_KEY:\033[0m \033[1;35m$ENCRYPTION_KEY\033[0m"

  echo -e "\n🚀 \033[1;36mYour Sarabi service is ready to rock!\033[0m 🚀"
  echo -e "\n💡 \033[1;33mPro Tip:\033[0m Keep your secrets safe and secure! 🔒"
  echo -e "\n🙌 \033[1;32mThank you for using this setup script! Have a great day!\033[0m 🙌\n"
}

main() {
  get_public_ip
  setup_domain
  install_docker
  download_sarabi
  generate_access_secret
  generate_or_recover_encryption_key
  start_sarabi_service
  output_success_message
}

main