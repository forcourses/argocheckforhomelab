#!/bin/bash
set -euo pipefail

# ============================================================
# 03-install-argocd.sh — Install ArgoCD + Bootstrap App-of-Apps
# ============================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$SCRIPT_DIR/.."

echo "========================================="
echo "  ArgoCD Installation + Bootstrap"
echo "========================================="

# --- Install ArgoCD ---
echo "[1/4] Creating argocd namespace..."
kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f -

echo "[2/4] Installing ArgoCD (resource-optimized)..."
helm repo add argo https://argoproj.github.io/argo-helm
helm repo update

helm install argocd argo/argo-cd \
    --namespace argocd \
    --version 7.7.5 \
    --set server.resources.limits.memory=256Mi \
    --set server.resources.limits.cpu=250m \
    --set server.resources.requests.memory=128Mi \
    --set server.resources.requests.cpu=50m \
    --set controller.resources.limits.memory=512Mi \
    --set controller.resources.limits.cpu=500m \
    --set controller.resources.requests.memory=256Mi \
    --set controller.resources.requests.cpu=100m \
    --set repoServer.resources.limits.memory=256Mi \
    --set repoServer.resources.limits.cpu=250m \
    --set repoServer.resources.requests.memory=128Mi \
    --set repoServer.resources.requests.cpu=50m \
    --set redis.resources.limits.memory=128Mi \
    --set redis.resources.limits.cpu=200m \
    --set redis.resources.requests.memory=64Mi \
    --set redis.resources.requests.cpu=50m \
    --set dex.enabled=false \
    --set notifications.enabled=false \
    --set applicationSet.enabled=true \
    --set server.insecure=true  # Behind Cilium ingress, TLS terminated there

echo "[3/4] Waiting for ArgoCD to be ready..."
kubectl -n argocd rollout status deployment/argocd-server --timeout=300s
kubectl -n argocd rollout status deployment/argocd-repo-server --timeout=300s

# --- Get initial password ---
echo "[4/4] Retrieving admin credentials..."
echo ""
ARGOCD_PASS=$(kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d)
echo "  ArgoCD Admin Password: $ARGOCD_PASS"
echo "  URL: https://localhost:8080 (after port-forward)"
echo ""
echo "  Quick access:"
echo "    kubectl port-forward svc/argocd-server -n argocd 8080:443"
echo ""

# --- Bootstrap App-of-Apps ---
echo "Bootstrapping App-of-Apps pattern..."
echo ""
echo "  ⚠️  Before applying, update the 'repoURL' in argocd/bootstrap/app-of-apps.yaml"
echo "     to point to YOUR Git repository."
echo ""
echo "  When ready, run:"
echo "    kubectl apply -f argocd/bootstrap/app-of-apps.yaml"
echo ""
echo "========================================="
echo "  ✅ ArgoCD installed!"
echo "  Access: kubectl port-forward svc/argocd-server -n argocd 8080:443"
echo "========================================="
