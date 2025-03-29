---
title: Getting Started
description: Definition and description of sarabi
slug: getting-started
---

import { Steps } from '@astrojs/starlight/components';

Its simple to get started! One of the great thing about sarabi is how easy it is to setup and start using. You can deploy your first app in minutes!

## Prerequisites

#### **Operating System**
- **Linux** is the only fully supported operating system for Sarabi. This is because Sarabi utilizes Linux-specific features such as `ufw` and other system tools that are not available on macOS or Windows.
- **macOS and Windows**: While Sarabi can be installed on macOS and Windows for local testing and exploration of its core features, these platforms are not recommended for production use. For proper deployment, a Linux-based VPS (Virtual Private Server) is strongly advised.

#### **System Requirements**
- There are no strict system requirements for Sarabi, as resource usage largely depends on the applications you intend to manage or run with it.
- For getting started, we recommend a minimum of **2GB of RAM** and **40GB of disk space**.

#### **Docker**
- Sarabi relies on **Docker** to deploy and manage applications.
- You do not need to manually install Docker beforehand, as the Sarabi installation script will automatically handle Docker installation for you.
## Installation
You will need to install sarabi server on your VPS, it is responsible for processing application deployment and management requests. The server is a single Golang binary which uses `sqlite` to persist application data e.g environment properties, configurations etc. 
<Steps>
1. **Download the installation script using the command below**
```shell
curl -sSL https://raw.githubusercontent.com/adxgun/sarabi/refs/heads/master/install.sh -o install.sh
```
2. **Run the installation script**
```shell
chmod +x install.sh && ./install.sh
```
This command will prompt you to enter your server domain name, this is the domain name you will use to access the sarabi server. Enter the domain name and wait for the script to finish the installation process.


3. **Verify Installation:**
  Copy the Access key output from the command above and let's use it to verify the installation.
```shell
curl -X GET "${SERVER_URL}/v1/ping" -H "X-Access-Key: ${YOUR_ACCESS_KEY}"
```
Replace the `${SERVER_URL}` and `${YOUR_ACCESS_KEY}` with the installation output. You should see a JSON response like below
```json
{
  "success": true, 
  "data": {
    "version": "v1.0.18"
  }
}
```
This confirms sarabi is installed and ready to go!
</Steps>