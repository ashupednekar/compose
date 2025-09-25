# compose

> **Run Helm charts on Docker/Podman without Kubernetes**

<img width="256" height="256" alt="image" src="https://github.com/user-attachments/assets/c886c6ba-b2aa-4423-8044-13cd4a82b21c" />


## About

`compose` is an open source tool that lets you take **Helm charts** and run them directly on **Docker or Podman**, without needing a Kubernetes cluster.

It bridges the gap between Kubernetes-native packaging (Helm) and traditional container runtimes (`docker-compose.yaml`), giving you the same artifacts and deployment flows across both worlds.

## Why?

- In most environments we prefer **k3s + ArgoCD** for continuous deployment and day-2 updates.  
- But not all clients allow this — especially those running **RHEL with Podman** and no Kubernetes.  
- That forces teams back to **`docker-compose.yaml`**, often rebuilt manually → error-prone, slow, and inconsistent.  

**`compose` fixes this by:**

- Reusing the **same Helm pipeline artifacts** you already generate  
- Converting them into **docker-compose.yaml** automatically  
- Delivering a **CD-like experience** on Docker/Podman runtimes  

No drift, no duplicate work — just consistent deployments everywhere.

## Features

- Run **Helm charts** on Docker or Podman
- Generates `docker-compose.yaml` automatically
- Single static Go binary, no external dependencies
- Reuses host Docker/Podman auth
- Token refresher support for long-lived credentials
- Works in CI/CD or standalone mode

## Install

Build from source (requires Go 1.20+):

```bash
git clone https://github.com/ashupednekar/compose.git
cd compose
go build -o compose ./...
