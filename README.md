# 🏠 Kube Homelab — Mac Mini Edition

A production-like Kubernetes homelab running on a Mac Mini with resource-conscious defaults.

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│                      Mac Mini (host)                     │
│  RAM: 16GB+ │ CPU: Apple Silicon │ Disk: 256GB+         │
│                                                          │
│  ┌────────────────────────────────────────────────────┐  │
│  │              k3s Cluster (lightweight)              │  │
│  │                                                    │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────────────┐ │  │
│  │  │  Cilium   │  │ Longhorn │  │   Cert-Manager   │ │  │
│  │  │  (CNI)    │  │  (CSI)   │  │                  │ │  │
│  │  └──────────┘  └──────────┘  └──────────────────┘ │  │
│  │                                                    │  │
│  │  ┌──────────────────────────────────────────────┐  │  │
│  │  │              ArgoCD (GitOps)                  │  │  │
│  │  │  ├─ App-of-Apps (bootstrap)                  │  │  │
│  │  │  ├─ Infrastructure apps                      │  │  │
│  │  │  └─ User apps                                │  │  │
│  │  └──────────────────────────────────────────────┘  │  │
│  │                                                    │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────────────┐ │  │
│  │  │Monitoring │  │ Logging  │  │ Portfolio Tracker│ │  │
│  │  │Prometheus │  │  Loki +  │  │   (Go + Postgres)│ │  │
│  │  │+ Grafana  │  │Promtail  │  │   Investment app │ │  │
│  │  └──────────┘  └──────────┘  └──────────────────┘ │  │
│  └────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

## What's Included

| Component | Tool | Purpose | Resource Profile |
|-----------|------|---------|-----------------|
| Kubernetes | k3s | Lightweight K8s | ~512MB RAM |
| CNI | Cilium | Networking + Security | ~256MB RAM |
| CSI / Storage | Longhorn | Distributed block storage | ~256MB RAM |
| GitOps | ArgoCD | Declarative app delivery | ~256MB RAM |
| Monitoring | Prometheus + Grafana | Metrics & dashboards | ~512MB RAM |
| Logging | Loki + Promtail | Log aggregation | ~256MB RAM |
| Certs | cert-manager | TLS automation | ~64MB RAM |
| App | Portfolio Tracker | Investment tracking API | ~128MB RAM |
| Database | PostgreSQL | App database | ~256MB RAM |
| **Total** | | | **~2.5GB RAM** |

## Quick Start

```bash
# 1. Install k3s (without default CNI — we use Cilium)
./scripts/01-install-k3s.sh

# 2. Install Cilium CNI
./scripts/02-install-cilium.sh

# 3. Install ArgoCD
./scripts/03-install-argocd.sh

# 4. Bootstrap everything via App-of-Apps
kubectl apply -f argocd/bootstrap/app-of-apps.yaml

# 5. Get ArgoCD admin password
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d
```

## Accessing Services

| Service | URL | Notes |
|---------|-----|-------|
| ArgoCD | https://argocd.local | admin / (initial secret) |
| Grafana | https://grafana.local | admin / prom-operator |
| Portfolio Tracker | https://portfolio.local | API docs at /swagger |

## Project Structure

```
kube-homelab/
├── README.md
├── cluster-setup/            # k3s config
│   └── k3s-config.yaml
├── argocd/
│   ├── bootstrap/            # Initial ArgoCD + App-of-Apps
│   │   └── app-of-apps.yaml
│   ├── applications/         # Individual ArgoCD Application manifests
│   │   ├── cilium.yaml
│   │   ├── longhorn.yaml
│   │   ├── monitoring.yaml
│   │   ├── logging.yaml
│   │   ├── cert-manager.yaml
│   │   └── portfolio-tracker.yaml
│   └── app-of-apps/          # Kustomize overlay for all apps
│       └── kustomization.yaml
├── infrastructure/
│   ├── cilium/               # Cilium Helm values
│   ├── longhorn/             # Longhorn Helm values
│   ├── monitoring/           # kube-prometheus-stack values
│   ├── logging/              # Loki + Promtail values
│   └── cert-manager/         # cert-manager values
├── apps/
│   └── portfolio-tracker/    # Investment tracking app
│       ├── helm/             # Helm chart
│       └── src/              # Application source code
└── scripts/                  # Setup & utility scripts
```

## Resource Management

This setup is designed for a Mac Mini and won't eat all your resources:

- **k3s** instead of full k8s (saves ~1-2GB RAM)
- **Loki** instead of Elasticsearch (saves ~4GB RAM)
- All components have explicit resource limits
- Longhorn configured with single replica (homelab, not prod)
- Prometheus retention set to 7 days
- Node reservations configured to protect the host OS

## Customization

Edit `cluster-setup/k3s-config.yaml` to adjust:
- `system-reserved`: RAM/CPU reserved for host OS
- `kube-reserved`: RAM/CPU reserved for k8s system

Edit individual `values.yaml` files under `infrastructure/` to tune each component.
# argocheckforhomelab
