# Mailvex Deployment Artifacts

This directory contains container definitions, orchestration manifests, and deployment configurations for **Mailvex**.

## Quick References

* 📖 **Complete Step-by-Step Guide**: [`docs/deployment.md`](../docs/deployment.md)
* 🐳 **Compose Profiles**:
  * [Homelab All-in-One (`< 200MB RAM`)](./compose/docker-compose.homelab.yml)
  * [Standard Production (Decoupled API & Workers)](./compose/docker-compose.standard.yml)
  * [Enterprise High-Availability Cluster](./compose/docker-compose.cluster.yml)
* ⚓ **Helm Chart (Kubernetes)**: [`deploy/helm/mailvex`](./helm/mailvex)
* 📊 **Grafana Dashboard**: [`deploy/grafana/mailvex_dashboard.json`](./grafana/mailvex_dashboard.json)
