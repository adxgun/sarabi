To secure access to your **OVHcloud VPS**, you can use either **password-based authentication** or **SSH key-based authentication**. Below are detailed instructions for setting up both.


## 🔑 Option 2: **Use SSH Key Authentication (Recommended)**

> 🔒 More secure than password auth. You generate a key on your local machine and upload the public part to your VPS.

### Step 1: Generate SSH Key (on your computer)

```bash
ssh-keygen -t ed25519 -C "your_email@example.com"
```

* Saves to `~/.ssh/id_ed25519` by default
* You'll get:

    * `~/.ssh/id_ed25519` (private key — **keep this safe**)
    * `~/.ssh/id_ed25519.pub` (public key)

### Step 2: Add Public Key to VPS

#### 🔹 Option A: Use OVHcloud Control Panel (before first deployment)

* When creating the VPS, add your public key under **SSH Key** section

#### 🔹 Option B: Manually add it (if VPS is already running)

1. SSH into your VPS (with the password you received from OVH):

   ```bash
   ssh root@<your-vps-ip>
   ```
2. Create the `.ssh` directory:

   ```bash
   mkdir -p ~/.ssh && chmod 700 ~/.ssh
   ```
3. Paste your public key into `authorized_keys`:

   ```bash
   echo "ssh-ed25519 AAAAC3NzaC1..." >> ~/.ssh/authorized_keys
   chmod 600 ~/.ssh/authorized_keys
   ```

### Step 3: SSH with your private key

```bash
ssh -i ~/.ssh/id_ed25519 root@<your-vps-ip>
```

---

## 🛡️ Disable Password Authentication (Optional but Recommended)

To force SSH key usage only:

1. Edit SSH config:

   ```bash
   sudo nano /etc/ssh/sshd_config
   ```
2. Change or add the following:

   ```
   PasswordAuthentication no
   PermitRootLogin prohibit-password
   ```
3. Restart SSH:

   ```bash
   sudo systemctl restart ssh
   ```

---

## Need Help?

Let me know:

* Your VPS OS (e.g., Ubuntu, Debian, CentOS)
* Whether you're trying to **set a password**, **reset it**, or **use SSH keys**
* Whether your VPS is **newly created** or **already running**

I’ll guide you accordingly.
