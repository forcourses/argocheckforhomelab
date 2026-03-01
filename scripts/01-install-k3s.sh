#!/bin/bash
set -euo pipefail

# ============================================================
# 01-install-k3s.sh — Install k3s on Mac Mini
# ============================================================
# Prerequisites:
#   - macOS with Lima/Colima OR a Linux VM on the Mac Mini
#   - curl installed
#
# NOTE: k3s runs natively on Linux. On macOS you need a Linux VM.
#       Recommended: Use Colima, Lima, or UTM to run an Ubuntu VM,
#       then install k3s inside that VM.
# ============================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_DIR="$SCRIPT_DIR/../cluster-setup"

echo "========================================="
echo "  k3s Installation for Mac Mini Homelab"
echo "========================================="

# --- Check if running inside Linux (VM or native) ---
if [[ "$(uname)" == "Darwin" ]]; then
    echo ""
    echo "⚠️  k3s requires Linux. You're on macOS."
    echo ""
    echo "Option A — Use Colima (recommended for simplicity):"
    echo "  brew install colima"
    echo "  colima start --runtime containerd --cpu 4 --memory 10 --disk 100 --network-address"
    echo "  colima ssh"
    echo "  # Then run this script inside the VM"
    echo ""
    echo "Option B — Use Lima:"
    echo "  brew install lima"
    echo "  limactl start --name=k3s template://k3s"
    echo ""
    echo "Option C — Use UTM/Multipass for a full Ubuntu VM:"
    echo "  multipass launch --name k3s-node --cpus 4 --memory 10G --disk 100G"
    echo "  multipass shell k3s-node"
    echo "  # Then run this script inside the VM"
    echo ""
    exit 1
fi

# --- Pre-flight checks ---
echo "[1/4] Pre-flight checks..."

# Check available memory
TOTAL_MEM_KB=$(grep MemTotal /proc/meminfo | awk '{print $2}')
TOTAL_MEM_GB=$((TOTAL_MEM_KB / 1024 / 1024))
echo "  Total memory: ${TOTAL_MEM_GB}GB"

if [ "$TOTAL_MEM_GB" -lt 6 ]; then
    echo "  ⚠️  Warning: Less than 6GB RAM available. Consider allocating more to the VM."
fi

# --- Copy k3s config ---
echo "[2/4] Setting up k3s configuration..."
sudo mkdir -p /etc/rancher/k3s
sudo cp "$CONFIG_DIR/k3s-config.yaml" /etc/rancher/k3s/config.yaml
echo "  ✅ Config copied to /etc/rancher/k3s/config.yaml"

# --- Install k3s ---
echo "[3/4] Installing k3s..."
curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="server" sh -

# --- Verify ---
echo "[4/4] Verifying installation..."
sleep 5

# Wait for node to be ready
echo "  Waiting for node to be ready..."
for i in $(seq 1 60); do
    if sudo k3s kubectl get nodes | grep -q " Ready"; then
        break
    fi
    sleep 2
done

sudo k3s kubectl get nodes
echo ""

# --- Setup kubeconfig ---
echo "Setting up kubeconfig..."
mkdir -p ~/.kube
sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
sudo chown "$(id -u):$(id -g)" ~/.kube/config
echo "  ✅ Kubeconfig ready at ~/.kube/config"

echo ""
echo "========================================="
echo "  ✅ k3s installed successfully!"
echo "  Next: Run ./02-install-cilium.sh"
echo "========================================="
