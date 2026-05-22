[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/kennethatria/myGuy/badge)](https://scorecard.dev/viewer/?uri=github.com/kennethatria/myGuy)

# MyGuy - Task Marketplace Platform

MyGuy is a modern, microservices-based task marketplace. It allows users to post tasks they need done, and enables other users to apply, negotiate, and complete those tasks.

The platform is designed with a clean architecture, separating concerns into distinct services for task management, real-time chat, and a store/bidding marketplace.

## Architecture & Tech Stack

### Infrastructure Diagram

```mermaid
graph TB
    User["User / Browser"]
    GHA["GitHub Actions CI/CD"]
    HCP["HCP Terraform"]

    subgraph Linode["Linode Cloud"]
        NB["NodeBalancer (public entry point)"]

        subgraph AppInstance["App Instance — public + VPC 10.0.0.2"]
            F2B["Fail2ban"]
            NGINX["Nginx + ModSecurity WAF"]
            API["Backend API :8080"]
            STORE["Store Service :8081"]
            CHAT["Chat Service :8082"]
            PG["PostgreSQL"]
            REDIS["Redis"]
            NE["node_exporter :9100"]
            FALCO["Falco :8765"]
        end

        subgraph MonInstance["Monitoring Instance — VPC only 10.0.0.3"]
            ZIPKIN["Zipkin"]
            PROM["Prometheus"]
            GRAFANA["Grafana"]
            LOKI["Loki"]
        end
    end

    User -->|HTTPS| NB
    NB --> F2B
    F2B --> NGINX
    NGINX --> API
    NGINX --> STORE
    NGINX --> CHAT
    API --> PG
    STORE --> PG
    CHAT --> PG
    CHAT --> REDIS
    STORE -->|booking notify| CHAT
    API -->|traces| ZIPKIN
    STORE -->|traces| ZIPKIN
    CHAT -->|traces| ZIPKIN
    NE -->|host metrics| PROM
    FALCO -->|security metrics| PROM
    PROM --> GRAFANA
    GHA -->|provision| HCP
    GHA -->|configure + deploy| NGINX
    GHA -.->|via app instance| PROM
```

### Application Services

| Service | Language | Port | Description |
| :--- | :--- | :--- | :--- |
| **Frontend** | TypeScript (Vue.js) | `5173` | The main user interface that communicates with all backend services. |
| **Backend** | Go (Gin) | `8080` | The core API for managing users, tasks, applications, and reviews. |
| **Store Service** | Go (Gin) | `8081` | A marketplace for items with fixed-price and auction-style bidding. |
| **Chat Service** | JavaScript (Node.js) | `8082` | A real-time WebSocket service for all messaging features. |
| **Database** | PostgreSQL | `5432` | Primary data store, with each service connecting to its own database. |
| **Redis** | Redis | `6379` | Socket.IO adapter for multi-instance chat scaling. |

### Monitoring Stack (Dedicated Instance)

| Tool | Port | Description |
| :--- | :--- | :--- |
| **Zipkin** | `9411` | Distributed tracing — collects spans from all backend services. |
| **Prometheus** | `9090` | Metrics collection — scrapes CPU/memory and Falco security alerts. |
| **Grafana** | `3000` | Visualization — dashboards for app metrics, security alerts, and WAF detections. |
| **Loki** | `3100` | Log aggregation — receives ModSecurity audit logs shipped by Promtail. |

---

## Security

MyGuy runs multiple layered security controls in production.

| Layer | Tool | Role |
| :--- | :--- | :--- |
| **Network** | Linode Firewall | Allows only ports 80, 443, 22 inbound |
| **Brute force** | Fail2ban | Bans IPs after repeated SSH failures or WAF triggers |
| **HTTP** | Nginx + ModSecurity + OWASP CRS | Inspects and filters all inbound HTTP traffic |
| **Images** | Cosign | Verifies app image signatures before deployment |
| **Runtime** | Falco | Detects suspicious syscalls and container behaviour |
| **Containers** | Rootless Podman | Container escapes land as unprivileged user, not root |

### Server Access

Two non-root users are provisioned on every server. Root SSH is disabled.

| User | Purpose | SSH Access |
| :--- | :--- | :--- |
| `myguy` | Runs application containers and Ansible automation | CI/CD runner key |
| `ops` | Troubleshooting — container logs, status, restart, Fail2ban | Personal key |

---

## Observability

MyGuy has two layers of observability: **distributed tracing** via OpenTelemetry + Zipkin, and **metrics + security alerting** via Prometheus + Grafana + Falco.

In production, all monitoring tools run on a dedicated server that is only accessible within the private VPC — not exposed to the public internet.

### Distributed Tracing (OpenTelemetry + Zipkin)

Every HTTP request handled by the backend, store service, or chat service is automatically traced. Spans are exported to Zipkin where you can visualise request flows, latency, and errors across services.

| Service | OTel Implementation | Service Name in Zipkin |
| :--- | :--- | :--- |
| **Backend** | Go SDK + `otelgin` middleware | `myguy-backend` |
| **Store Service** | Go SDK + `otelgin` middleware | `myguy-store-service` |
| **Chat Service** | Node.js SDK + Express/HTTP instrumentation | `myguy-chat-service` |

Each service reads `ZIPKIN_URL` from its environment:

```env
# Local development
ZIPKIN_URL=http://localhost:9411/api/v2/spans

# Production (via VPC)
ZIPKIN_URL=http://10.0.0.3:9411/api/v2/spans
```

### Metrics & Security Alerting (Prometheus + Grafana + Falco)

**On the app instance:**
- `node_exporter` runs as a systemd service on `:9100`, exposing CPU and memory metrics
- `falco` monitors system calls for suspicious runtime behaviour and exposes Prometheus metrics on `:8765`
- `promtail` ships ModSecurity audit logs to Loki on the monitoring instance

**On the monitoring instance:**
- Prometheus scrapes `node_exporter` (`:9100`) and Falco metrics (`:8765`) on the app instance via VPC every 15 seconds
- Grafana is pre-provisioned with three dashboards:
  - **App Instance Metrics** — CPU usage (%) and memory usage (%)
  - **Falco Security Alerts** — alert rate by priority/rule and total alert count
  - **WAF — ModSecurity Detections** — ModSecurity rule triggers visualised from Loki

Both Prometheus (`:9090`) and Grafana (`:3000`) are only reachable from within the VPC. To access them locally, SSH tunnel through the app instance as the `ops` user:

```sh
ssh -N \
  -L 3000:10.0.0.3:3000 \
  -L 9090:10.0.0.3:9090 \
  -L 9411:10.0.0.3:9411 \
  -L 3100:10.0.0.3:3100 \
  ops@<app_public_ip>
```

Then open `http://localhost:3000` for Grafana, `http://localhost:9411` for Zipkin.

---

## Infrastructure & Deployment

The production infrastructure runs on Linode and is provisioned with Terraform and configured with Ansible.

### Servers

| Instance | VPC IP | Purpose |
| :--- | :--- | :--- |
| **App instance** | `10.0.0.2` | Runs the full application stack via rootless Podman Compose |
| **Monitoring instance** | `10.0.0.3` | Runs Zipkin, Prometheus, Grafana, and Loki |

The monitoring instance has no public IP. It is only reachable via the app instance as a ProxyJump host.

### Provisioning with Terraform

```sh
cd infra
terraform init
terraform apply
```

After apply, get the app instance's public IP:

```sh
terraform output -raw instance_ip_address
```

The CI/CD workflow injects this IP into `configuration_management/inventory.ini` automatically.

### Configuring with Ansible

```sh
cd configuration_management

# Run everything in order
ansible-playbook main.yml -i inventory.ini \
  -e "ci_cd_public_key=$(cat ~/.ssh/id_ed25519.pub)" \
  -e "ops_ssh_public_key=$(cat ~/.ssh/id_ed25519.pub)"

# Or run individual playbooks
ansible-playbook users.yml -i inventory.ini \
  -e "ci_cd_public_key=..." \
  -e "ops_ssh_public_key=..."    # must run first on a new server

ansible-playbook monitoring.yml -i inventory.ini
ansible-playbook site.yml -i inventory.ini
ansible-playbook security.yml -i inventory.ini
ansible-playbook observability.yml -i inventory.ini
ansible-playbook deploy.yml -i inventory.ini \
  -e "repo_url=..." \
  -e "jwt_secret=..." \
  -e "db_password=..." \
  -e "internal_api_key=..." \
  -e "registry=..." \
  -e "image_tag=..." \
  -e "github_repository=..." \
  -e "domain=..." \
  -e "certbot_email=..."
```

> **Note:** `users.yml` must run first on any new server. It connects as root to create the `myguy` and `ops` users, then disables root SSH. All subsequent playbooks connect as `myguy`.

`deploy.yml` automatically reads the Zipkin URL from the monitoring play and writes it into the app's `.env` — no manual copy-paste needed.

---

## Quick Start (Local Development)

The entire platform can be run locally using Podman Compose. Services use pre-built images from Docker Hub.

### Prerequisites
- Podman & Podman Compose
- Git

### Running the Application

1. **Clone the repository:**
   ```sh
   git clone <repository-url>
   cd myguy
   ```

2. **Create a root `.env` file** in the project root:
   ```env
   JWT_SECRET=your-secret-key-here
   DB_PASSWORD=mysecretpassword
   INTERNAL_API_KEY=your-internal-api-key-here
   ```

3. **Start the services:**
   ```sh
   podman compose up -d
   ```

4. **Access the application:**
   - **Frontend:** http://localhost:5173
   - **Backend API:** http://localhost:8080
   - **Store Service:** http://localhost:8081
   - **Chat Service:** http://localhost:8082
   - **Zipkin UI:** http://localhost:9411

---

## Documentation

- **[Backend README](./backend/README.md)**
- **[Store Service README](./store-service/README.md)**
- **[Chat Service README](./chat-websocket-service/README.md)**
- **[Frontend README](./frontend/README.md)**
