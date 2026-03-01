#!/bin/bash
set -euo pipefail

# ============================================================
# 02-install-cilium.sh — Install Cilium CNI
# ============================================================

echo "========================================="
echo "  Cilium CNI Installation"
echo "========================================="

# --- Install Helm if not present ---
if ! command -v helm &>/dev/null; then
    echo "[0/3] Installing Helm..."
    curl -fsSL https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
fi

# --- Add Cilium Helm repo ---
echo "[1/3] Adding Cilium Helm repo..."
helm repo add cilium https://helm.cilium.io/
helm repo update

# --- Install Cilium ---
echo "[2/3] Installing Cilium..."
helm install cilium cilium/cilium \
    --version 1.16.5 \
    --namespace kube-system \
    --set operator.replicas=1 \
    --set operator.resources.limits.memory=256Mi \
    --set operator.resources.limits.cpu=500m \
    --set operator.resources.requests.memory=128Mi \
    --set operator.resources.requests.cpu=100m \
    --set resources.limits.memory=256Mi \
    --set resources.limits.cpu=500m \
    --set resources.requests.memory=128Mi \
    --set resources.requests.cpu=100m \
    --set hubble.enabled=true \
    --set hubble.relay.enabled=true \
    --set hubble.ui.enabled=true \
    --set hubble.relay.resources.limits.memory=128Mi \
    --set hubble.relay.resources.limits.cpu=200m \
    --set hubble.ui.resources.limits.memory=128Mi \
    --set hubble.ui.resources.limits.cpu=200m \
    --set ipam.mode=kubernetes \
    --set kubeProxyReplacement=true \
    --set k8sServiceHost=127.0.0.1 \
    --set k8sServicePort=6443 \
    --set ingressController.enabled=true \
    --set ingressController.default=true \
    --set ingressController.loadbalancerMode=shared

# --- Wait for Cilium to be ready ---
echo "[3/3] Waiting for Cilium to be ready..."
kubectl -n kube-system rollout status daemonset/cilium --timeout=300s
kubectl -n kube-system rollout status deployment/cilium-operator --timeout=300s

echo ""
echo "  ✅ Cilium installed!"
echo ""

# --- Optional: Install Cilium CLI for status checks ---
if ! command -v cilium &>/dev/null; then
    echo "Installing Cilium CLI..."
    CILIUM_CLI_VERSION=$(curl -s https://raw.githubusercontent.com/cilium/cilium-cli/main/stable.txt)
    CLI_ARCH=amd64
    if [ "$(uname -m)" = "aarch64" ] || [ "$(uname -m)" = "arm64" ]; then CLI_ARCH=arm64; fi
    curl -L --fail --remote-name-all "https://github.com/cilium/cilium-cli/releases/download/${CILIUM_CLI_VERSION}/cilium-linux-${CLI_ARCH}.tar.gz"
    sudo tar xzvfC "cilium-linux-${CLI_ARCH}.tar.gz" /usr/local/bin
    rm -f "cilium-linux-${CLI_ARCH}.tar.gz"
fi

echo ""
cilium status --wait || true
echo ""
echo "========================================="
echo "  ✅ Cilium CNI ready!"
echo "  Next: Run ./03-install-argocd.sh"
echo "========================================="
