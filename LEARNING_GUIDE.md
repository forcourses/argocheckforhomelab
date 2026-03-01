# 🎓 Kubernetes Homelab — The Learning Guide

> A step-by-step walkthrough of building a production-like Kubernetes platform on a Mac Mini.
> Every decision is explained. Every concept is taught as we go.

---

## Table of Contents

1. [Why This Project Exists](#1-why-this-project-exists)
2. [The Big Picture — What Are We Building?](#2-the-big-picture)
3. [Prerequisites](#3-prerequisites)
4. [Step 1 — Linux VM on macOS (Why You Need One)](#4-step-1--linux-vm)
5. [Step 2 — k3s: Your Lightweight Kubernetes](#5-step-2--k3s)
6. [Step 3 — Cilium: Container Networking (CNI)](#6-step-3--cilium-cni)
7. [Step 4 — Longhorn: Persistent Storage (CSI)](#7-step-4--longhorn-csi)
8. [Step 5 — ArgoCD: GitOps Delivery](#8-step-5--argocd-gitops)
9. [Step 6 — App-of-Apps: Bootstrapping Everything](#9-step-6--app-of-apps)
10. [Step 7 — Monitoring: Prometheus + Grafana](#10-step-7--monitoring)
11. [Step 8 — Logging: Loki + Promtail](#11-step-8--logging)
12. [Step 9 — The Application: Portfolio Tracker](#12-step-9--the-application)
13. [Step 10 — Connecting the Dots](#13-step-10--connecting-the-dots)
14. [Troubleshooting Guide](#14-troubleshooting)
15. [What to Learn Next](#15-what-to-learn-next)
16. [Glossary](#16-glossary)

---

## 1. Why This Project Exists

If you're learning Kubernetes, reading docs only gets you so far. You need a cluster you can break, fix, and experiment with. Cloud clusters cost money and disappear when you stop paying. A homelab is yours forever.

This project teaches you the **real stack** that production teams use — not just "hello world in a pod" but the full platform: networking, storage, GitOps, monitoring, logging, and an actual application with a database. The difference between "I know Kubernetes" and "I've run Kubernetes" is enormous, and this project bridges that gap.

**What you'll understand by the end:**

- Why Kubernetes needs a separate CNI, CSI, and ingress controller
- How GitOps works and why teams prefer it over `kubectl apply`
- How pods get IP addresses and find each other
- How data survives pod restarts (persistent storage)
- How to see what's happening inside your cluster (monitoring + logging)
- How a real app with a database runs in Kubernetes

---

## 2. The Big Picture

Here's what we're building and how every piece connects:

```
┌─────────────────────────────────────────────────────────────┐
│                       Mac Mini (Host)                        │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐  │
│  │          Linux VM (Colima / Lima / Multipass)           │  │
│  │          CPU: 4 cores │ RAM: 10GB │ Disk: 100GB        │  │
│  │                                                        │  │
│  │  ┌──────────────────────────────────────────────────┐  │  │
│  │  │                k3s (Kubernetes)                   │  │  │
│  │  │                                                  │  │  │
│  │  │  LAYER 1: Networking                             │  │  │
│  │  │  ┌──────────────────────────────────────────┐    │  │  │
│  │  │  │ Cilium (CNI)                             │    │  │  │
│  │  │  │  • Assigns IPs to every pod              │    │  │  │
│  │  │  │  • Routes traffic between pods           │    │  │  │
│  │  │  │  • Acts as Ingress (external → services) │    │  │  │
│  │  │  │  • Network policies (firewall rules)     │    │  │  │
│  │  │  │  • Hubble UI (network observability)     │    │  │  │
│  │  │  └──────────────────────────────────────────┘    │  │  │
│  │  │                                                  │  │  │
│  │  │  LAYER 2: Storage                                │  │  │
│  │  │  ┌──────────────────────────────────────────┐    │  │  │
│  │  │  │ Longhorn (CSI)                           │    │  │  │
│  │  │  │  • Provides PersistentVolumes            │    │  │  │
│  │  │  │  • Data survives pod restarts            │    │  │  │
│  │  │  │  • Snapshots & backups                   │    │  │  │
│  │  │  └──────────────────────────────────────────┘    │  │  │
│  │  │                                                  │  │  │
│  │  │  LAYER 3: Platform Services                      │  │  │
│  │  │  ┌────────────┐ ┌────────────┐ ┌────────────┐   │  │  │
│  │  │  │  ArgoCD    │ │ Prometheus │ │   Loki     │   │  │  │
│  │  │  │  (GitOps)  │ │ + Grafana  │ │ + Promtail │   │  │  │
│  │  │  │            │ │(monitoring)│ │ (logging)  │   │  │  │
│  │  │  └────────────┘ └────────────┘ └────────────┘   │  │  │
│  │  │                                                  │  │  │
│  │  │  LAYER 4: Applications                           │  │  │
│  │  │  ┌──────────────────────────────────────────┐    │  │  │
│  │  │  │ Portfolio Tracker (Go API)               │    │  │  │
│  │  │  │  • REST API for investment tracking      │    │  │  │
│  │  │  │  • PostgreSQL database (on Longhorn)     │    │  │  │
│  │  │  │  • CronJob for stock price updates       │    │  │  │
│  │  │  └──────────────────────────────────────────┘    │  │  │
│  │  └──────────────────────────────────────────────────┘  │  │
│  └────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

**Why these layers matter:**

Without Layer 1 (CNI), pods can't talk to each other. Without Layer 2 (CSI), databases lose data on restart. Without Layer 3, you're flying blind and deploying manually. Layer 4 is what you actually care about — but it needs everything below it.

---

## 3. Prerequisites

**Hardware:**
- Mac Mini (M1/M2/M3/M4 — any Apple Silicon or Intel)
- At least 16GB RAM (we'll use ~10GB for the VM, leaving 6GB for macOS)
- At least 100GB free disk space

**Software to install on your Mac:**
```bash
# Homebrew (macOS package manager)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Colima (lightweight Linux VM manager) — our recommended option
brew install colima docker

# kubectl (Kubernetes CLI)
brew install kubectl

# Helm (Kubernetes package manager)
brew install helm

# k9s (optional but HIGHLY recommended — terminal UI for Kubernetes)
brew install k9s
```

**Knowledge you should have:**
- Basic terminal / command line usage
- Understanding of what containers are (Docker basics)
- What a REST API is (we'll build one)

**Knowledge you DON'T need yet (we'll teach it):**
- Kubernetes specifics
- Networking concepts (CNI, CIDR, etc.)
- Storage concepts (CSI, PV, PVC)
- GitOps, Helm, ArgoCD

---

## 4. Step 1 — Linux VM on macOS

### 🧠 Why do I need a VM?

Kubernetes (and k3s) runs on **Linux**. It directly uses Linux kernel features like namespaces, cgroups, and iptables to isolate and manage containers. macOS doesn't have these. So even Docker Desktop on Mac secretly runs a Linux VM behind the scenes.

We're using **Colima** because it's lightweight and simple. Alternatives include Lima, UTM, or Multipass — they all do the same thing: run a Linux VM on your Mac.

### 🧠 How much resources should the VM get?

This is the key to "not using all the resources." We give the VM a fixed budget:

| Resource | VM Gets | macOS Keeps | Why |
|----------|---------|-------------|-----|
| CPU | 4 cores | Remaining cores | k8s system + workloads need ~3, +1 headroom |
| RAM | 10 GB | 6+ GB | Our full stack needs ~2.5GB, +7.5GB for OS/cache |
| Disk | 100 GB | Remaining | Longhorn, container images, logs, Prometheus data |

### 📋 Do It

```bash
# Start a Colima VM with our resource budget
colima start \
  --runtime containerd \
  --cpu 4 \
  --memory 10 \
  --disk 100 \
  --network-address \
  --kubernetes false  # We'll install k3s ourselves

# Verify it's running
colima status

# SSH into the VM (you'll run the remaining steps INSIDE the VM)
colima ssh
```

### 🧠 What is `--runtime containerd`?

Container runtimes are the software that actually runs containers. Docker is one option, but it has extra overhead (the Docker daemon). **containerd** is the lower-level runtime that both Docker and Kubernetes use internally. Since k3s uses containerd directly, we skip Docker entirely.

### 🧠 What is `--network-address`?

This gives the VM its own IP address on your local network, so you can access services running inside the VM directly from your Mac's browser (like Grafana dashboards).

### ✅ Checkpoint

You should now be inside a Linux terminal (via `colima ssh`). Run:
```bash
uname -a         # Should show Linux
free -h          # Should show ~10GB total memory
nproc            # Should show 4
df -h /          # Should show ~100GB disk
```

---

## 5. Step 2 — k3s: Lightweight Kubernetes

### 🧠 What is k3s and why not full Kubernetes?

**Kubernetes (k8s)** is the full orchestration platform. It's designed for hundreds of nodes and enterprise workloads. Running it requires etcd, kube-apiserver, kube-controller-manager, kube-scheduler, kube-proxy, and more — each as separate processes.

**k3s** is a certified Kubernetes distribution by Rancher Labs that:
- Bundles everything into a **single binary** (~70MB)
- Uses **SQLite** instead of etcd by default (simpler for single node)
- Removes legacy/alpha features you don't need
- Uses **~512MB RAM** vs 2-4GB for full k8s

It's 100% API-compatible with Kubernetes. Anything that works on k8s works on k3s. It's perfect for homelabs, edge computing, and IoT.

### 🧠 Understanding the k3s config

Let's break down every setting in our config:

```yaml
# /etc/rancher/k3s/config.yaml

# ---------- Networking ----------
# We DISABLE the default CNI (Flannel) because we want Cilium
flannel-backend: "none"
disable-network-policy: true

# We disable the default ingress (Traefik) and load balancer
# because Cilium will handle both
disable:
  - servicelb
  - traefik
  - local-storage  # Longhorn replaces this
```

**Why disable defaults?** k3s ships with Flannel (CNI), Traefik (ingress), and local-path-provisioner (storage). These are fine for getting started, but we're learning the real stack. Cilium is more powerful than Flannel, and Longhorn is more capable than local-path.

```yaml
# ---------- Resource Reservations ----------
kubelet-arg:
  # Reserve resources for the host OS — CRITICAL for stability
  - "system-reserved=cpu=2000m,memory=4Gi"
  - "kube-reserved=cpu=500m,memory=512Mi"
```

**Why reserve resources?** Without reservations, Kubernetes will schedule pods until the node runs out of memory, then the Linux OOM killer starts randomly killing processes — including system processes. Reservations tell Kubernetes "this much is off-limits."

- `system-reserved`: For the Linux OS, SSH, VM overhead
- `kube-reserved`: For kubelet, containerd, kube-proxy

```yaml
  # Eviction thresholds — when to start killing pods
  - "eviction-hard=memory.available<500Mi,nodefs.available<10%"
  - "eviction-soft=memory.available<1Gi,nodefs.available<15%"
```

**Eviction** is Kubernetes' graceful way of dealing with resource pressure. When available memory drops below 1GB (soft threshold), Kubernetes starts evicting low-priority pods. Below 500MB (hard threshold), it immediately kills pods. This protects the node from crashing.

```yaml
# ---------- Network CIDRs ----------
cluster-cidr: "10.42.0.0/16"   # Pod IP range (65,536 addresses)
service-cidr: "10.43.0.0/16"   # Service IP range
```

**What are CIDRs?** CIDR (Classless Inter-Domain Routing) notation defines IP address ranges. `/16` means the first 16 bits are fixed, leaving 16 bits for addresses = 65,536 IPs. Every pod in your cluster gets its own unique IP from the pod CIDR. Every Kubernetes Service gets an IP from the service CIDR.

### 📋 Do It

```bash
# Create k3s config directory
sudo mkdir -p /etc/rancher/k3s

# Copy our config (or create it — see cluster-setup/k3s-config.yaml)
sudo cp /path/to/k3s-config.yaml /etc/rancher/k3s/config.yaml

# Install k3s
curl -sfL https://get.k3s.io | INSTALL_K3S_EXEC="server" sh -

# Wait for it to start (30-60 seconds)
sleep 30

# Setup kubeconfig so kubectl works
mkdir -p ~/.kube
sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
sudo chown $(id -u):$(id -g) ~/.kube/config

# Verify
kubectl get nodes
```

### 🧠 What is kubeconfig?

kubeconfig (`~/.kube/config`) is the file that tells `kubectl` how to connect to your cluster. It contains:
- **Cluster**: The API server URL and CA certificate
- **User**: Authentication credentials
- **Context**: Which cluster + user combination to use

When you run `kubectl get pods`, kubectl reads this file to know where to send the request.

### 🧠 What happens when k3s starts?

1. k3s starts the **API server** (listens on port 6443)
2. Starts the **scheduler** (decides which node runs each pod)
3. Starts the **controller manager** (ensures desired state = actual state)
4. Starts the **kubelet** (manages containers on this node)
5. Starts **containerd** (the container runtime)
6. Creates system namespaces: `kube-system`, `kube-public`, `kube-node-lease`
7. Deploys CoreDNS (cluster DNS)

### ⚠️ Expected State After This Step

Your node will show `NotReady`:
```
NAME        STATUS     ROLES                  AGE   VERSION
mac-mini    NotReady   control-plane,master   1m    v1.31.3+k3s1
```

**This is normal!** The node is NotReady because there's no CNI installed yet. Without a CNI, pods can't get IP addresses, so nothing can run. That's Step 3.

### ✅ Checkpoint
```bash
kubectl get nodes            # Should show 1 node (NotReady is OK)
kubectl get namespaces       # Should show kube-system, kube-public, etc.
kubectl cluster-info         # Should show cluster endpoint
```

---

## 6. Step 3 — Cilium (CNI)

### 🧠 What is a CNI and why do pods need one?

**CNI = Container Network Interface.** It's a plugin that answers the question: _"How do containers talk to each other?"_

When Kubernetes creates a pod, it asks the CNI to:
1. **Create a network interface** for the pod
2. **Assign an IP address** from the cluster CIDR
3. **Set up routing** so the pod can reach other pods

Without a CNI, pods are isolated network islands. With a CNI, every pod in the cluster can reach every other pod by IP — even across multiple nodes.

### 🧠 Why Cilium over alternatives?

| CNI | How it works | Pros | Cons |
|-----|-------------|------|------|
| **Flannel** | Simple overlay network (VXLAN) | Easy, lightweight | No network policies, limited observability |
| **Calico** | BGP routing + iptables | Mature, good policies | Complex config, iptables performance |
| **Cilium** | eBPF (kernel-level programs) | Fast, great policies, Hubble UI, can replace kube-proxy + ingress | Slightly more memory |

**eBPF** is a technology that lets you run small programs inside the Linux kernel. Instead of routing packets through iptables (a chain of rules that every packet must traverse), Cilium uses eBPF to make routing decisions directly in the kernel. This is faster and more flexible.

### 🧠 What is Hubble?

Hubble is Cilium's **observability layer**. It watches all network traffic in the cluster and gives you:
- A real-time map of which services talk to which services
- Latency metrics per connection
- DNS query logs
- HTTP request/response visibility

Think of it as Wireshark for your entire Kubernetes cluster, but with a nice UI.

### 🧠 What is kube-proxy replacement?

Normally, Kubernetes runs `kube-proxy` on every node. kube-proxy uses iptables to implement Services (mapping a Service IP to pod IPs). With hundreds of Services, this creates thousands of iptables rules, which slows down.

Cilium can **replace kube-proxy** entirely using eBPF, which handles Service routing more efficiently. That's what `kubeProxyReplacement: true` does.

### 🧠 What is an Ingress Controller?

An Ingress Controller is the component that handles **traffic from outside the cluster** into your services. When you access `grafana.local` in your browser, the Ingress Controller:

1. Receives the HTTP request
2. Looks at the `Host` header (`grafana.local`)
3. Matches it to an Ingress resource
4. Forwards the request to the correct Service/pod

Common Ingress Controllers include Nginx, Traefik, and Cilium. We use Cilium's built-in ingress (`ingressController.enabled: true`) so we don't need another component.

### 📋 Do It

```bash
# Add Cilium's Helm repository
helm repo add cilium https://helm.cilium.io/
helm repo update

# Install Cilium with our settings
helm install cilium cilium/cilium \
    --version 1.16.5 \
    --namespace kube-system \
    --set operator.replicas=1 \
    --set operator.resources.limits.memory=256Mi \
    --set operator.resources.requests.memory=128Mi \
    --set resources.limits.memory=256Mi \
    --set resources.requests.memory=128Mi \
    --set hubble.enabled=true \
    --set hubble.relay.enabled=true \
    --set hubble.ui.enabled=true \
    --set ipam.mode=kubernetes \
    --set kubeProxyReplacement=true \
    --set k8sServiceHost=127.0.0.1 \
    --set k8sServicePort=6443 \
    --set ingressController.enabled=true \
    --set ingressController.default=true \
    --set ingressController.loadbalancerMode=shared

# Wait for Cilium to be ready (2-3 minutes)
kubectl -n kube-system rollout status daemonset/cilium --timeout=300s
kubectl -n kube-system rollout status deployment/cilium-operator --timeout=300s
```

### 🧠 What is a DaemonSet?

Notice Cilium runs as a **DaemonSet**, not a Deployment. A DaemonSet ensures exactly one copy of a pod runs on **every node** in the cluster. Since networking must work on every node, Cilium needs to be everywhere. If you add a new node, Kubernetes automatically starts a Cilium pod on it.

### 🧠 What is IPAM?

IPAM = **IP Address Management**. It's the component that decides which IP address each pod gets. With `ipam.mode=kubernetes`, Cilium delegates this to Kubernetes' built-in IPAM, which allocates IPs from the `cluster-cidr` we configured in k3s.

### ✅ Checkpoint

```bash
# Your node should now be Ready!
kubectl get nodes
# NAME        STATUS   ROLES                  AGE   VERSION
# mac-mini    Ready    control-plane,master   5m    v1.31.3+k3s1

# Cilium pods should be running
kubectl -n kube-system get pods -l app.kubernetes.io/name=cilium

# CoreDNS should now be Running (it was Pending without CNI)
kubectl -n kube-system get pods -l k8s-app=kube-dns

# Test pod-to-pod networking
kubectl run test-a --image=busybox --command -- sleep 3600
kubectl run test-b --image=busybox --command -- sleep 3600
sleep 10
TEST_B_IP=$(kubectl get pod test-b -o jsonpath='{.status.podIP}')
kubectl exec test-a -- ping -c 3 $TEST_B_IP
# Should succeed! Pods can talk to each other.

# Clean up test pods
kubectl delete pod test-a test-b
```

---

## 7. Step 4 — Longhorn (CSI)

### 🧠 What is a CSI and why do I need persistent storage?

**CSI = Container Storage Interface.** It's the standard way for Kubernetes to talk to storage systems.

By default, containers are **ephemeral** — when a pod restarts, all data inside it is gone. This is fine for stateless apps (web servers, APIs) but terrible for databases. If your PostgreSQL pod restarts, you don't want to lose all your data.

**PersistentVolumes (PV)** solve this. They're chunks of storage that exist independently of pods. When a pod dies and a new one starts, it re-attaches the same PersistentVolume and picks up where it left off.

The flow:
```
Your App → PersistentVolumeClaim (PVC) → PersistentVolume (PV) → Actual Storage (disk)
          "I need 10GB of storage"       Created by CSI          Managed by Longhorn
```

### 🧠 Why Longhorn?

| Storage Option | What it is | Pros | Cons |
|---------------|-----------|------|------|
| local-path | Uses the node's local disk directly | Zero overhead | No replication, no snapshots, no backups |
| OpenEBS | Container-attached storage | Feature-rich | Complex, heavy |
| Rook/Ceph | Distributed storage (enterprise) | Battle-tested, replication | Needs 3+ nodes, **very** heavy (~4GB RAM) |
| **Longhorn** | Lightweight distributed block storage | UI, snapshots, backups, simple | Designed for small clusters |

For a single-node homelab, Longhorn gives you snapshots, a management UI, and backup capability without the weight of Ceph. We set `defaultReplicaCount: 1` because we only have one node — no point replicating data to the same disk.

### 🧠 StorageClass — The Template for Volumes

A **StorageClass** defines _how_ volumes are created. When Longhorn installs, it creates a StorageClass called `longhorn`. When any pod requests storage via a PVC, Kubernetes asks the `longhorn` StorageClass to provision it.

```yaml
# This is what happens behind the scenes when PostgreSQL requests storage:
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: data-postgresql-0
spec:
  storageClassName: longhorn     # "Use Longhorn to create this volume"
  accessModes: [ReadWriteOnce]   # "Only one pod can write at a time"
  resources:
    requests:
      storage: 2Gi               # "I need 2 gigabytes"
```

### 📋 Do It

Longhorn will be installed via ArgoCD in Step 6, but here's what the ArgoCD Application does:

```yaml
# Key Longhorn settings (in argocd/applications/longhorn.yaml)
defaultSettings:
  defaultReplicaCount: 1        # Single node = single replica
  storageOverProvisioningPercentage: 100
  storageMinimalAvailablePercentage: 15  # Keep 15% disk free
persistence:
  defaultClass: true            # Make Longhorn the DEFAULT StorageClass
```

**But first**, Longhorn needs the `open-iscsi` package on the host:

```bash
# Inside your VM (colima ssh)
sudo apt-get update && sudo apt-get install -y open-iscsi
sudo systemctl enable iscsid --now
```

### 🧠 What is iSCSI?

iSCSI (Internet Small Computer Systems Interface) is a protocol for accessing storage over a network. Longhorn uses iSCSI to connect its storage engine to pods. Without it, pods can't mount Longhorn volumes.

### ✅ Checkpoint (after ArgoCD installs Longhorn)

```bash
# Longhorn pods running
kubectl -n longhorn-system get pods

# StorageClass exists
kubectl get storageclass
# NAME                 PROVISIONER          RECLAIMPOLICY   VOLUMEBINDINGMODE
# longhorn (default)   driver.longhorn.io   Delete          Immediate

# Test volume creation
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: test-pvc
spec:
  storageClassName: longhorn
  accessModes: [ReadWriteOnce]
  resources:
    requests:
      storage: 1Gi
EOF

kubectl get pvc test-pvc  # Should show "Bound"
kubectl delete pvc test-pvc
```

---

## 8. Step 5 — ArgoCD (GitOps)

### 🧠 What is GitOps?

**GitOps** is a way of managing infrastructure where **Git is the single source of truth**. Instead of running `kubectl apply -f deployment.yaml` by hand, you:

1. Push YAML files to a Git repository
2. ArgoCD watches the repo
3. ArgoCD automatically applies changes to the cluster
4. If someone manually changes the cluster, ArgoCD reverts it back to match Git

This gives you:
- **Audit trail**: Every change is a Git commit with author, timestamp, and diff
- **Rollback**: Revert a `git commit` and ArgoCD rolls back the cluster
- **Consistency**: What's in Git is what's in the cluster, always
- **Self-healing**: Manual `kubectl` changes get automatically reverted

### 🧠 Traditional vs GitOps Deployment

```
Traditional ("push" model):
  Developer → kubectl apply → Cluster
  Developer → helm install → Cluster
  Problem: Who deployed what? When? Can we undo it?

GitOps ("pull" model):
  Developer → git push → Git Repo ← ArgoCD polls → Cluster
  Everything is tracked. Rollback = git revert.
```

### 🧠 ArgoCD Architecture

ArgoCD has several components:

| Component | Role | Analogy |
|-----------|------|---------|
| **API Server** | Web UI + REST API | The dashboard you interact with |
| **Repo Server** | Clones Git repos, renders Helm/Kustomize | The librarian that reads your manifests |
| **Application Controller** | Compares desired state (Git) vs actual state (cluster) | The enforcer that keeps things in sync |
| **Redis** | Cache for the controller | Speeds up comparisons |

### 📋 Do It

```bash
# Create namespace
kubectl create namespace argocd

# Install ArgoCD via Helm
helm repo add argo https://argoproj.github.io/argo-helm
helm repo update

helm install argocd argo/argo-cd \
    --namespace argocd \
    --version 7.7.5 \
    --set server.resources.limits.memory=256Mi \
    --set controller.resources.limits.memory=512Mi \
    --set repoServer.resources.limits.memory=256Mi \
    --set redis.resources.limits.memory=128Mi \
    --set dex.enabled=false \
    --set notifications.enabled=false \
    --set server.insecure=true

# Wait for it
kubectl -n argocd rollout status deployment/argocd-server --timeout=300s

# Get the admin password
kubectl -n argocd get secret argocd-initial-admin-secret \
    -o jsonpath="{.data.password}" | base64 -d
echo  # Print newline

# Access the ArgoCD UI
kubectl port-forward svc/argocd-server -n argocd 8080:443 &
# Open https://localhost:8080 in your browser
# Login: admin / <password from above>
```

### 🧠 What is `port-forward`?

`kubectl port-forward` creates a tunnel from your local machine to a Service inside the cluster. It's like SSH port forwarding. When you visit `localhost:8080`, kubectl forwards that traffic to the ArgoCD server pod.

In production, you'd use an Ingress instead. For development, port-forward is simpler.

### 🧠 What is Helm?

We keep using `helm install`. **Helm** is a package manager for Kubernetes — like `brew` for macOS or `apt` for Ubuntu, but for Kubernetes applications.

A **Helm Chart** is a package containing:
- Templates (YAML with variables)
- Default values
- Dependencies

When you run `helm install`, Helm renders the templates with your values and applies them to the cluster. The ArgoCD Helm chart contains 20+ YAML files (Deployments, Services, RBAC, etc.) — Helm lets us install all of that with one command.

### ✅ Checkpoint

```bash
# ArgoCD pods running
kubectl -n argocd get pods
# argocd-server-...         Running
# argocd-repo-server-...    Running
# argocd-application-controller-...  Running
# argocd-redis-...           Running

# ArgoCD UI accessible
curl -k https://localhost:8080  # Should return HTML
```

## 9. Step 6 — App-of-Apps Pattern

### 🧠 The Bootstrap Problem

We have ArgoCD running. Now we need it to deploy Longhorn, monitoring, logging, and our app. We could create each ArgoCD Application manually:

```bash
kubectl apply -f argocd/applications/longhorn.yaml
kubectl apply -f argocd/applications/monitoring.yaml
kubectl apply -f argocd/applications/logging.yaml
kubectl apply -f argocd/applications/portfolio-tracker.yaml
```

But that defeats the purpose of GitOps! We'd be manually applying things. The solution is the **App-of-Apps pattern**.

### 🧠 How App-of-Apps Works

You create ONE ArgoCD Application that points to a directory containing OTHER ArgoCD Application manifests. It's recursive:

```
You manually apply:
  app-of-apps.yaml → points to argocd/app-of-apps/ directory

ArgoCD sees that directory contains:
  ├── longhorn.yaml          → ArgoCD Application (deploys Longhorn Helm chart)
  ├── monitoring.yaml        → ArgoCD Application (deploys Prometheus + Grafana)
  ├── logging.yaml           → ArgoCD Application (deploys Loki + Promtail)
  ├── cert-manager.yaml      → ArgoCD Application (deploys cert-manager)
  └── portfolio-tracker.yaml → ArgoCD Application (deploys our app)

ArgoCD creates all of these Applications, and each Application deploys its own stack.
```

So you manually apply **one file**, and your entire platform deploys itself. Add a new app? Just add a YAML file to the directory and push to Git. ArgoCD picks it up automatically.

### 🧠 Sync Waves — Ordering Deployments

Some things must be installed before others. You can't deploy PostgreSQL before Longhorn (no storage), and you can't deploy the app before PostgreSQL (no database). **Sync waves** control the order:

```yaml
metadata:
  annotations:
    argocd.argoproj.io/sync-wave: "1"  # Lower numbers deploy first
```

Our ordering:
| Wave | Components | Why |
|------|-----------|-----|
| 1 | Longhorn, cert-manager | Storage and certs must exist first |
| 2 | Monitoring, Logging | These need storage (Prometheus, Loki use PVCs) |
| 3 | Portfolio Tracker | Needs storage (PostgreSQL) and everything above |

### 🧠 Kustomize — Assembling the App List

We use **Kustomize** to tell ArgoCD "here are all the Application manifests in this directory." Kustomize is a tool built into kubectl that lets you compose Kubernetes resources without templates.

Our `kustomization.yaml` simply lists all the Application files:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../applications/longhorn.yaml
  - ../applications/cert-manager.yaml
  - ../applications/monitoring.yaml
  - ../applications/logging.yaml
  - ../applications/portfolio-tracker.yaml
```

### 📋 Do It

First, push this entire project to a Git repository:

```bash
# On your Mac (not in the VM)
cd kube-homelab

# Initialize Git
git init
git add .
git commit -m "Initial homelab setup"

# Create a GitHub repo and push
# (Create repo on github.com first, then:)
git remote add origin https://github.com/YOUR_USERNAME/kube-homelab.git
git push -u origin main
```

Now update the repo URLs in the project:

```bash
# Replace YOUR_USERNAME in all ArgoCD Application files
find argocd/ -name "*.yaml" -exec sed -i \
    's|YOUR_USERNAME|your-actual-github-username|g' {} +
git add . && git commit -m "Set repo URLs" && git push
```

Now bootstrap everything:

```bash
# Inside the VM (colima ssh)
# Apply the one manifest that rules them all
kubectl apply -f argocd/bootstrap/app-of-apps.yaml
```

### 🧠 What happens next?

1. ArgoCD sees the `app-of-apps` Application
2. It clones your Git repo
3. It reads `argocd/app-of-apps/kustomization.yaml`
4. It creates 5 ArgoCD Applications (longhorn, cert-manager, monitoring, logging, portfolio-tracker)
5. Each Application syncs: ArgoCD installs the Helm charts / manifests
6. Sync waves ensure ordering: wave 1 → wave 2 → wave 3
7. In 5-10 minutes, everything is running

Watch it happen:
```bash
# Watch ArgoCD Applications sync
kubectl -n argocd get applications -w

# Or watch all pods come up
watch kubectl get pods --all-namespaces
```

### ✅ Checkpoint

```bash
# All ArgoCD Applications should be "Synced" and "Healthy"
kubectl -n argocd get applications
# NAME                 SYNC STATUS   HEALTH STATUS
# longhorn             Synced        Healthy
# cert-manager         Synced        Healthy
# monitoring           Synced        Healthy
# logging              Synced        Healthy
# portfolio-tracker    Synced        Healthy
```

---

## 10. Step 7 — Monitoring (Prometheus + Grafana)

### 🧠 Why monitor a homelab?

Three reasons:

1. **Learn what production teams see**: Understanding Prometheus + Grafana is a valuable skill. Most companies use this exact stack.
2. **Debug problems**: When something breaks, metrics tell you _what_ and _when_. "Memory usage spiked at 3am" is more useful than "it stopped working."
3. **It's satisfying**: Watching dashboards of your own cluster is genuinely cool.

### 🧠 How Monitoring Works — The Pipeline

```
Your Apps & k8s ──metrics──▶ Prometheus ──queries──▶ Grafana ──dashboards──▶ You
                  (scrape)     (store)     (PromQL)   (visualize)
```

**Prometheus** works on a **pull model**. Every 60 seconds (our config), Prometheus reaches out to every component that exposes metrics and "scrapes" them. Components expose metrics at a `/metrics` endpoint in a text format:

```
# TYPE http_requests_total counter
http_requests_total{method="GET", path="/api/v1/portfolios"} 142
http_requests_total{method="POST", path="/api/v1/portfolios"} 7

# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{le="0.1"} 120
http_request_duration_seconds_bucket{le="0.5"} 138
```

**Grafana** queries Prometheus using **PromQL** and renders charts. Example query: `rate(http_requests_total[5m])` = "how many requests per second over the last 5 minutes?"

### 🧠 What is kube-prometheus-stack?

It's a Helm chart that bundles:

| Component | What it does |
|-----------|-------------|
| **Prometheus** | Scrapes and stores metrics |
| **Grafana** | Dashboards and visualization |
| **Alertmanager** | Routes alerts (email, Slack, PagerDuty) |
| **node-exporter** | Exposes host-level metrics (CPU, RAM, disk, network) |
| **kube-state-metrics** | Exposes Kubernetes object metrics (pod count, deployment status) |
| **Pre-built dashboards** | 20+ dashboards for k8s, node, and pod metrics |

### 🧠 Key Configuration Decisions

```yaml
# Why 60s scrape interval instead of default 30s?
scrapeInterval: 60s
# Halves the number of data points = less memory & disk.
# For a homelab, 60s granularity is fine. In production, 15-30s is common.

# Why 7-day retention?
retention: 7d
retentionSize: "5GB"
# On a homelab, you don't need months of metrics. 7 days gives you
# enough history to debug recent issues without eating disk space.

# Why disable kubeEtcd, kubeScheduler, kubeControllerManager, kubeProxy?
kubeProxy:
  enabled: false  # We use Cilium as kube-proxy replacement
kubeEtcd:
  enabled: false  # k3s uses embedded SQLite, not etcd
kubeScheduler:
  enabled: false  # k3s bundles this — Prometheus can't scrape it separately
kubeControllerManager:
  enabled: false  # Same reason
```

### 📋 Access Grafana

```bash
# After ArgoCD has deployed monitoring:
kubectl port-forward svc/monitoring-grafana -n monitoring 3000:80 &

# Open http://localhost:3000
# Login: admin / homelab-grafana
```

### 🧠 Explore These Dashboards

Once in Grafana, go to **Dashboards** → **Browse**:

1. **Node Exporter Full**: Shows your VM's CPU, RAM, disk, and network usage. Find the "Memory" panel — you'll see how much RAM your cluster is actually using.

2. **Kubernetes / Compute Resources / Namespace (Pods)**: Shows CPU and memory usage per pod in each namespace. Great for finding resource hogs.

3. **Kubernetes / Networking / Namespace (Pods)**: Shows network traffic per pod. You'll see Prometheus itself generating traffic (it scrapes every 60s).

### 🧠 Try a PromQL Query

In Grafana, go to **Explore** → select **Prometheus** → type:

```promql
# How much memory is each namespace using?
sum(container_memory_working_set_bytes{container!=""}) by (namespace)

# Which pods use the most CPU?
topk(10, sum(rate(container_cpu_usage_seconds_total{container!=""}[5m])) by (pod))

# Is our Portfolio Tracker app responding?
up{job="portfolio-tracker"}
```

### ✅ Checkpoint

```bash
# Prometheus pods running
kubectl -n monitoring get pods

# Grafana accessible
curl -s http://localhost:3000/api/health | grep ok

# Prometheus has targets
kubectl port-forward svc/monitoring-kube-prometheus-prometheus -n monitoring 9090:9090 &
# Open http://localhost:9090/targets — all should be UP
```

---

## 11. Step 8 — Logging (Loki + Promtail)

### 🧠 Monitoring vs Logging — What's the Difference?

| | Monitoring (Prometheus) | Logging (Loki) |
|---|---|---|
| **Data type** | Numbers (metrics) | Text (log lines) |
| **Question answered** | "How many?" "How fast?" "How much?" | "What happened?" "Why did it fail?" |
| **Example** | "HTTP 500 errors increased 10x at 2am" | "ERROR: connection refused to database at 2:03am" |
| **Storage cost** | Low (just numbers) | High (full text) |

You need both. Metrics tell you _something is wrong_. Logs tell you _what specifically went wrong_.

### 🧠 Why Loki Instead of Elasticsearch?

The classic logging stack is ELK (Elasticsearch + Logstash + Kibana). It's powerful but **extremely heavy**:

| | Elasticsearch | Loki |
|---|---|---|
| RAM usage | 4-8 GB minimum | 256 MB |
| Indexing | Indexes the full text of every log line | Only indexes labels (metadata) |
| Query speed | Fast full-text search | Fast label queries, slower grep |
| Complexity | High (JVM tuning, index management) | Low (single binary) |

Loki was created by Grafana Labs as "Prometheus, but for logs." It uses the same label-based approach: instead of indexing every word, it indexes labels like `namespace=monitoring, pod=grafana-xxx` and stores log lines as compressed chunks. To search within logs, it does a brute-force scan — which is fine because you usually filter by labels first.

### 🧠 How Loki + Promtail Work

```
Every Pod ──stdout/stderr──▶ Container Runtime ──log files──▶ Promtail ──push──▶ Loki
                              /var/log/pods/                   (ships logs)    (stores)
                                                                                  │
                                                                                  ▼
                                                                               Grafana
                                                                            (query + view)
```

1. **Your app** writes to stdout/stderr (as all good containerized apps should)
2. **containerd** writes these logs to files in `/var/log/pods/`
3. **Promtail** (a DaemonSet on every node) tails these log files
4. Promtail adds labels (namespace, pod name, container name) and ships to Loki
5. **Loki** stores them for 7 days (our config)
6. **Grafana** queries Loki using **LogQL**

### 🧠 SingleBinary Mode

Loki can run in different modes:
- **Microservices**: Separate components (distributor, ingester, querier, compactor) — for high scale
- **Simple Scalable**: Read path and write path separated — for medium scale
- **SingleBinary**: Everything in one process — perfect for homelab

We use SingleBinary because it uses ~256MB RAM vs 1-2GB for the distributed modes.

### 📋 Query Logs in Grafana

```bash
# Access Grafana (if not already port-forwarded)
kubectl port-forward svc/monitoring-grafana -n monitoring 3000:80 &
```

In Grafana → **Explore** → select **Loki** data source → type:

```logql
# All logs from the portfolio-tracker namespace
{namespace="portfolio-tracker"}

# Only error logs from any namespace
{namespace=~".+"} |= "error"

# ArgoCD sync logs
{namespace="argocd", container="argocd-application-controller"} |= "sync"

# Cilium agent logs (useful for debugging networking)
{namespace="kube-system", app_kubernetes_io_name="cilium"} |= "endpoint"

# Count errors per namespace over time (creates a graph!)
sum(count_over_time({namespace=~".+"} |= "error" [5m])) by (namespace)
```

### 🧠 LogQL vs PromQL

LogQL is inspired by PromQL:
- `{namespace="monitoring"}` — label selectors (same as PromQL)
- `|= "error"` — line filter (contains "error")
- `!= "debug"` — negative filter (doesn't contain "debug")
- `| json` — parse JSON logs
- `| logfmt` — parse logfmt logs
- `count_over_time()` — aggregate to metrics

### ✅ Checkpoint

```bash
# Loki running
kubectl -n logging get pods
# loki-0                Running
# promtail-xxxxx        Running

# Promtail is tailing logs
kubectl -n logging logs -l app.kubernetes.io/name=promtail --tail=5

# Grafana can reach Loki
# In Grafana: Configuration → Data Sources → Loki → Test
```

---

## 12. Step 9 — The Application: Portfolio Tracker

### 🧠 Why Build an Actual App?

Most k8s tutorials stop at "deploy nginx." But the real learning happens when you have:
- A custom application with business logic
- A database that needs persistent storage
- Background jobs (CronJobs)
- Database migrations
- Health checks
- Configuration via environment variables

Our **Portfolio Tracker** is an investment tracking API that:
- Manages portfolios of stock holdings
- Fetches real stock prices from Alpha Vantage API
- Calculates profit & loss (P&L) per holding
- Shows allocation percentages
- Runs a CronJob to periodically refresh prices

### 🧠 Application Architecture

```
                                    ┌──────────────────┐
    Browser / curl ──HTTP──▶        │ Portfolio Tracker │
                                    │    (Go binary)    │
                                    │                   │
                                    │  /api/v1/...      │──SQL──▶ PostgreSQL
                                    │  /healthz         │         (Longhorn PVC)
                                    │  /readyz          │
                                    └──────────────────┘
                                            ▲
    CronJob (every 4h) ─────────────────────┘
    "refresh-prices" command
    Fetches prices from Alpha Vantage
    Updates DB
```

### 🧠 Kubernetes Resources for This App

Let's map each Kubernetes concept to what our app needs:

| K8s Resource | Purpose | Our App |
|---|---|---|
| **Deployment** | Runs and manages pod replicas | Runs the Go API server |
| **Service** | Stable network endpoint for pods | Routes traffic to the API |
| **Ingress** | External HTTP access | `portfolio.local` → Service |
| **Secret** | Sensitive configuration | Alpha Vantage API key |
| **PVC** | Persistent storage request | PostgreSQL data directory |
| **CronJob** | Scheduled background task | Price refresh every 4 hours |
| **Job** (PreSync) | One-time task before deploy | Database migrations |

### 🧠 The Deployment — Deep Dive

```yaml
spec:
  replicas: 1
  template:
    spec:
      # --- Init Container ---
      initContainers:
        - name: wait-for-db
          image: busybox:1.36
          command: ['sh', '-c', 'until nc -z postgresql 5432; do sleep 2; done']
```

**Init containers** run before the main container starts. Ours waits for PostgreSQL to be reachable. Without this, the app would crash on startup if PostgreSQL isn't ready yet, then enter a CrashLoopBackOff.

```yaml
      containers:
        - name: portfolio-tracker
          env:
            - name: DATABASE_URL
              value: "postgres://user:pass@postgresql:5432/portfolio"
```

**Environment variables** are the standard way to configure containerized apps (following the [12-Factor App](https://12factor.net) methodology). The app reads `DATABASE_URL` at startup to know how to connect to the database.

```yaml
          livenessProbe:
            httpGet:
              path: /healthz
              port: http
            initialDelaySeconds: 10
            periodSeconds: 30
```

**Liveness probe**: Kubernetes periodically checks this endpoint. If it fails 3 times in a row, Kubernetes kills and restarts the pod. This handles the case where the app is stuck/frozen but the process hasn't crashed.

```yaml
          readinessProbe:
            httpGet:
              path: /readyz
              port: http
```

**Readiness probe**: Similar, but controls whether the pod receives traffic. If readiness fails, the Service stops sending requests to this pod. Our `/readyz` checks the database connection — if the DB is down, the pod stops receiving traffic (but doesn't restart).

### 🧠 Database Migrations — The PreSync Hook

```yaml
metadata:
  annotations:
    argocd.argoproj.io/hook: PreSync
    argocd.argoproj.io/hook-delete-policy: BeforeHookCreation
```

**PreSync hooks** are ArgoCD's way of running tasks _before_ the main deployment. Every time you push a code change, ArgoCD:

1. Runs the migration Job (creates/alters database tables)
2. Waits for it to complete
3. Then deploys the new version of the app

This ensures the database schema is always compatible with the running code.

### 🧠 CronJob — Periodic Price Refresh

```yaml
spec:
  schedule: "0 */4 * * *"  # Every 4 hours
  concurrencyPolicy: Forbid # Don't run if previous job still running
```

**CronJobs** create Jobs on a schedule (same syntax as Linux cron). Ours runs the app with a special `refresh-prices` command that:

1. Queries the DB for all unique stock symbols
2. Calls Alpha Vantage API for each symbol's current price
3. Updates the `current_price` column in the `holdings` table

`concurrencyPolicy: Forbid` prevents overlapping runs — if the API is slow and a job takes longer than 4 hours, the next scheduled run is skipped.

### 🧠 The Go Application — Key Design Decisions

**Why Go?**
- Compiles to a single binary (~15MB Docker image)
- Low memory footprint (~10-30MB at runtime)
- Excellent standard library for HTTP servers
- Fast startup time (important for CronJobs)

**Why Alpha Vantage?**
- Free tier available (5 API calls/minute, 500/day)
- Simple REST API
- Covers US stocks, ETFs, and some international markets
- No credit card required

**API Design:**

```
GET  /api/v1/portfolios              → List all portfolios
POST /api/v1/portfolios              → Create a portfolio
GET  /api/v1/portfolios/:id          → Get portfolio + holdings with P&L
POST /api/v1/portfolios/:id/holdings → Add a stock holding
GET  /api/v1/quote/:symbol           → Get real-time stock price
```

### 📋 Try It Out

After the app is deployed:

```bash
# Port-forward to the API
kubectl port-forward svc/portfolio-tracker -n portfolio-tracker 8080:8080 &

# Create a portfolio
curl -X POST http://localhost:8080/api/v1/portfolios \
  -H "Content-Type: application/json" \
  -d '{"name": "Tech Growth", "description": "High-conviction tech bets"}'

# Add some holdings
curl -X POST http://localhost:8080/api/v1/portfolios/1/holdings \
  -H "Content-Type: application/json" \
  -d '{"symbol": "AAPL", "shares": 50, "avg_cost_basis": 175.00}'

curl -X POST http://localhost:8080/api/v1/portfolios/1/holdings \
  -H "Content-Type: application/json" \
  -d '{"symbol": "NVDA", "shares": 20, "avg_cost_basis": 450.00}'

curl -X POST http://localhost:8080/api/v1/portfolios/1/holdings \
  -H "Content-Type: application/json" \
  -d '{"symbol": "MSFT", "shares": 30, "avg_cost_basis": 380.00}'

# Get a real-time stock quote
curl http://localhost:8080/api/v1/quote/AAPL

# View your portfolio with P&L calculations
curl http://localhost:8080/api/v1/portfolios/1 | python3 -m json.tool
```

### 🧠 Building and Pushing the Container Image

Before ArgoCD can deploy the app, you need a container image. The project includes a GitHub Actions workflow that:

1. Runs `go test` on every push
2. Builds the Docker image
3. Pushes to GitHub Container Registry (ghcr.io)

To build locally for testing:

```bash
cd apps/portfolio-tracker/src

# Build the image
docker build -t portfolio-tracker:dev .

# Test it locally (you'll need a PostgreSQL instance)
docker run -p 8080:8080 \
  -e DATABASE_URL="postgres://..." \
  -e ALPHA_VANTAGE_API_KEY="demo" \
  portfolio-tracker:dev
```

### ✅ Checkpoint

```bash
# App pods running
kubectl -n portfolio-tracker get pods
# portfolio-tracker-xxx     Running
# portfolio-tracker-postgresql-0  Running

# PostgreSQL has a PVC on Longhorn
kubectl -n portfolio-tracker get pvc
# data-portfolio-tracker-postgresql-0   Bound   longhorn   2Gi

# CronJob exists
kubectl -n portfolio-tracker get cronjobs
# portfolio-tracker-price-refresh   0 */4 * * *   ...

# API responds
curl http://localhost:8080/healthz  # "ok"
curl http://localhost:8080/readyz   # "ready"
```

---

## 13. Step 10 — Connecting the Dots

Now that everything is running, let's see how all the pieces interact.

### 🧠 Follow a Request Through the Stack

When you run `curl http://portfolio.local/api/v1/portfolios/1`:

```
1. DNS resolves portfolio.local → Cilium Ingress IP (via /etc/hosts or local DNS)

2. Cilium Ingress Controller receives the HTTP request
   → Reads the Host header: "portfolio.local"
   → Matches Ingress resource in portfolio-tracker namespace
   → Forwards to Service "portfolio-tracker" on port 8080

3. Kubernetes Service (kube-proxy replacement by Cilium)
   → Service has endpoints: [10.42.0.15:8080] (the pod's IP)
   → eBPF program routes directly to the pod

4. Portfolio Tracker pod receives the request
   → chi router matches GET /api/v1/portfolios/1
   → Handler queries PostgreSQL
   → SQL: SELECT * FROM holdings WHERE portfolio_id = 1
   → Calculates P&L for each holding
   → Returns JSON response

5. PostgreSQL pod
   → Receives SQL query via TCP on port 5432
   → Reads data from /var/lib/postgresql/data
   → This path is mounted from a Longhorn PersistentVolume
   → Longhorn manages the actual disk blocks via iSCSI

6. Meanwhile, Promtail
   → Tails the log files for both pods
   → Ships logs to Loki with labels:
     {namespace="portfolio-tracker", pod="portfolio-tracker-xxx"}

7. Prometheus
   → Every 60s, scrapes /metrics on the pod (if exposed)
   → Records request count, latency, error rate
```

### 🧠 Watch the GitOps Loop

Try changing something and watch ArgoCD react:

```bash
# Edit a value in your Git repo
# For example, change the Grafana password in monitoring.yaml:
#   adminPassword: "new-password-123"

git add . && git commit -m "Change Grafana password" && git push

# Watch ArgoCD detect the change (polls every 3 minutes by default)
kubectl -n argocd get applications monitoring -w

# You'll see:
# monitoring   OutOfSync   Healthy    ...
# monitoring   Syncing     Healthy    ...
# monitoring   Synced      Healthy    ...
```

### 🧠 Simulate a Failure

Try killing a pod and watch Kubernetes self-heal:

```bash
# Kill the Portfolio Tracker pod
kubectl -n portfolio-tracker delete pod -l app.kubernetes.io/name=portfolio-tracker

# Watch it come back immediately
kubectl -n portfolio-tracker get pods -w

# The Deployment controller saw "desired: 1, actual: 0" and created a new pod.
# The new pod attaches the SAME PersistentVolume — no data loss.
```

Try killing PostgreSQL:

```bash
# Kill the PostgreSQL pod
kubectl -n portfolio-tracker delete pod -l app.kubernetes.io/name=postgresql

# Watch it restart and reattach the Longhorn volume
kubectl -n portfolio-tracker get pods -w

# After it's back, your data is still there:
curl http://localhost:8080/api/v1/portfolios/1
# All your holdings are intact!
```

### 🧠 Resource Usage — Is It Actually Lightweight?

Check actual resource consumption:

```bash
# Per-node summary
kubectl top node

# Per-pod breakdown (requires metrics-server, included in kube-prometheus-stack)
kubectl top pods --all-namespaces --sort-by=memory

# Or in Grafana: Dashboards → "Kubernetes / Compute Resources / Cluster"
```

Expected totals on a healthy cluster:

| Component | Expected Memory |
|-----------|----------------|
| k3s system (kubelet, containerd, CoreDNS) | ~400MB |
| Cilium + Hubble | ~300MB |
| Longhorn | ~300MB |
| ArgoCD | ~400MB |
| Prometheus + Grafana + exporters | ~500MB |
| Loki + Promtail | ~300MB |
| Portfolio Tracker + PostgreSQL | ~300MB |
| **Total** | **~2.5GB** |

That leaves ~7.5GB of your 10GB VM for headroom, OS cache, and future apps.

---

## 14. Troubleshooting

### Pod stuck in Pending

```bash
kubectl describe pod <pod-name> -n <namespace>
# Look at the Events section at the bottom
# Common causes:
#   - "no nodes available" → Node resources exhausted
#   - "persistentvolumeclaim not found" → Longhorn not ready yet
#   - "0/1 nodes are available: insufficient memory" → Increase VM memory
```

### Pod in CrashLoopBackOff

```bash
kubectl logs <pod-name> -n <namespace>              # Current logs
kubectl logs <pod-name> -n <namespace> --previous    # Logs from crashed instance
# Common causes:
#   - Database connection refused → PostgreSQL not ready
#   - Missing env vars → Check the Secret/ConfigMap
#   - OOMKilled → Increase memory limits
```

### ArgoCD Application stuck OutOfSync

```bash
# Check the ArgoCD Application status
kubectl -n argocd describe application <app-name>
# Or in the ArgoCD UI: click the app → see sync errors

# Force a sync
kubectl -n argocd patch application <app-name> \
  -p '{"operation": {"initiatedBy": {"username": "admin"}, "sync": {}}}' \
  --type merge
```

### Longhorn volume stuck Attaching

```bash
# Check Longhorn manager logs
kubectl -n longhorn-system logs -l app=longhorn-manager --tail=50

# Check if iSCSI is running
sudo systemctl status iscsid
# If not: sudo systemctl start iscsid
```

### Can't access services from Mac browser

```bash
# Check Colima VM IP
colima status  # Look for "network address"

# Add DNS entries on your Mac (not in the VM)
sudo echo "192.168.x.x grafana.local portfolio.local argocd.local" >> /etc/hosts

# Or just use port-forward (simpler)
kubectl port-forward svc/monitoring-grafana -n monitoring 3000:80
```

---

## 15. What to Learn Next

Now that you have a working platform, here are paths to deepen your knowledge:

### 🔐 Security
- **Sealed Secrets** or **External Secrets Operator**: Encrypt secrets in Git (right now our passwords are plain text — bad!)
- **Cilium Network Policies**: Create firewall rules (e.g., only Portfolio Tracker can reach PostgreSQL)
- **RBAC**: Create limited-access ServiceAccounts instead of using admin everywhere
- **Pod Security Standards**: Enforce non-root containers, read-only filesystems

### 📦 More Apps to Deploy
- **Nextcloud**: Self-hosted Google Drive (tests your storage setup)
- **Gitea**: Self-hosted GitHub (then ArgoCD pulls from YOUR Gitea!)
- **WikiJS**: Self-hosted wiki for documenting your homelab
- **Uptime Kuma**: Monitoring dashboard for external services

### 🚀 Platform Improvements
- **External DNS**: Auto-create DNS records when you add Ingress resources
- **Let's Encrypt via cert-manager**: Real TLS certificates (requires a domain)
- **Velero**: Backup your entire cluster state + Longhorn volumes to S3
- **Renovate Bot**: Auto-update Helm chart versions in your Git repo

### 📊 Deeper Monitoring
- Add **custom metrics** to the Portfolio Tracker (request latency, DB query time)
- Create **custom Grafana dashboards** for your app
- Set up **Alertmanager** to send alerts to Slack/Discord when things break
- Add **Hubble UI** access to visualize network traffic between services

### 🏗️ Multi-Node
- Add a second Mac Mini or Raspberry Pi as a worker node
- Increase Longhorn replica count to 2 (data replicated across nodes)
- Test pod scheduling and node affinity

---

## 16. Glossary

| Term | Definition |
|------|-----------|
| **Pod** | Smallest deployable unit in Kubernetes. Contains one or more containers that share networking and storage. |
| **Deployment** | Manages a set of identical pods. Handles rolling updates and rollbacks. |
| **Service** | Stable network endpoint that routes to pods. Pods come and go, but the Service IP stays the same. |
| **Ingress** | Rules for routing external HTTP traffic to Services. |
| **Namespace** | Virtual sub-cluster for organizing resources. Like folders for your Kubernetes objects. |
| **DaemonSet** | Ensures one pod runs on every node (used by Cilium, Promtail, node-exporter). |
| **StatefulSet** | Like Deployment but for stateful apps. Pods get stable hostnames and persistent storage. |
| **CronJob** | Creates Jobs on a cron schedule. |
| **PVC** | PersistentVolumeClaim — a request for storage. |
| **PV** | PersistentVolume — an actual piece of storage provisioned by the CSI. |
| **CNI** | Container Network Interface — plugin that provides pod networking. |
| **CSI** | Container Storage Interface — plugin that provides persistent storage. |
| **Helm** | Package manager for Kubernetes. Charts are packages of pre-configured k8s resources. |
| **Kustomize** | Tool for composing and customizing k8s resources without templates. |
| **eBPF** | Extended Berkeley Packet Filter — technology for running programs in the Linux kernel. |
| **PromQL** | Prometheus Query Language — used to query metrics in Prometheus/Grafana. |
| **LogQL** | Loki Query Language — used to query logs in Loki/Grafana. |
| **GitOps** | Infrastructure management where Git is the source of truth. |
| **Sync Wave** | ArgoCD ordering mechanism — lower numbers deploy first. |
| **Init Container** | Container that runs before the main container. Used for setup tasks. |
| **Liveness Probe** | Health check — if it fails, Kubernetes restarts the pod. |
| **Readiness Probe** | Traffic check — if it fails, the pod stops receiving traffic. |

---

*Happy learning! Break things, fix things, and check `kubectl get events` when confused.* 🚀
