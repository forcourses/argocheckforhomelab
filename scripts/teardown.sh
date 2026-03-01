#!/bin/bash
set -euo pipefail

# ============================================================
# teardown.sh — Remove everything cleanly
# ============================================================

echo "⚠️  This will destroy the entire k3s cluster and all data!"
read -p "Are you sure? (yes/no): " confirm
if [ "$confirm" != "yes" ]; then
    echo "Aborted."
    exit 0
fi

echo ""
echo "[1/2] Uninstalling k3s..."
if command -v /usr/local/bin/k3s-uninstall.sh &>/dev/null; then
    /usr/local/bin/k3s-uninstall.sh
else
    echo "  k3s not found, skipping."
fi

echo "[2/2] Cleaning up config files..."
rm -rf ~/.kube/config
sudo rm -rf /etc/rancher/k3s

echo ""
echo "✅ Teardown complete. Your Mac Mini is clean."
