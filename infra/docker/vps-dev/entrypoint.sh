#!/bin/bash
set -euo pipefail

mkdir -p /root/.ssh
chmod 700 /root/.ssh

if [ -f /mnt/authorized_keys ]; then
  cp /mnt/authorized_keys /root/.ssh/authorized_keys
  chmod 600 /root/.ssh/authorized_keys
fi

# Allow root login with pubkey only (dev only).
sed -i 's/^#*PermitRootLogin.*/PermitRootLogin prohibit-password/' /etc/ssh/sshd_config
sed -i 's/^#*PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config

mkdir -p /opt/qx/agent /opt/qx/server /etc/qx-agent

# Static unit in the image layer; config + binary live on persistent volumes (/opt/qx, /etc/qx-agent).
cat > /etc/systemd/system/qx-agent.service <<'UNIT'
[Unit]
Description=QX Agent
After=network-online.target

[Service]
ExecStart=/opt/qx/agent/qx-agent
WorkingDirectory=/opt/qx/server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
UNIT

mkdir -p /etc/systemd/system/multi-user.target.wants
ln -sf /etc/systemd/system/qx-agent.service /etc/systemd/system/multi-user.target.wants/qx-agent.service

exec /lib/systemd/systemd
