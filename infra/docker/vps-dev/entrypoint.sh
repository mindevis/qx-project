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

mkdir -p /opt/qxsystem/agent /opt/qxsystem/server /etc/qxsystem/agent

# Static unit in the image layer; config + binary live on persistent volumes (/opt/qxsystem, /etc/qxsystem/agent).
cat > /etc/systemd/system/qx-agent.service <<'UNIT'
[Unit]
Description=QX Agent
After=network-online.target

[Service]
ExecStart=/opt/qxsystem/agent/qx-agent
WorkingDirectory=/opt/qxsystem/server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
UNIT

mkdir -p /etc/systemd/system/multi-user.target.wants
ln -sf /etc/systemd/system/qx-agent.service /etc/systemd/system/multi-user.target.wants/qx-agent.service

exec /lib/systemd/systemd
