# MyGuy Infrastructure (Terraform)

Terraform for MyGuy's Akamai Cloud (Linode) infrastructure. State and variables live in **HCP Terraform** (organization `myGuy`, workspace `dev-myguy`). Server configuration and app deployment are done by Ansible (`../configuration_management`); see the root `README.md` for the full picture.

## What it creates

| Resource | Purpose |
| :--- | :--- |
| `linode_vpc.main` + subnet `10.0.0.0/24` | Private network between the two servers |
| `linode_instance.my_guy_instance` (`g6-nanode-1`, VPC `10.0.0.2`) | App server: nginx + WAF, API, store, chat, PostgreSQL, Redis |
| `linode_instance.zipkin_instance` (`g6-nanode-1`, VPC `10.0.0.3`) | Monitoring server: Prometheus, Loki, Grafana, blackbox prober (no public ingress) |
| `linode_nodebalancer.main` | Public entry point; the domain's DNS `A` record points here |
| NodeBalancer config `:80` (HTTP mode) | Health-checks `/healthcheck/` expecting body `healthcheck`; adds `X-Forwarded-For` |
| NodeBalancer config `:443` (TCP, **PROXY protocol v2**) | TLS passthrough; nginx must listen with `proxy_protocol` |
| `linode_firewall.my_firewall` | App server: 80, 443, 22 public; node_exporter/Falco metrics from the VPC |
| `linode_firewall.zipkin_firewall` | Monitoring server: Prometheus, Loki, Grafana, SSH — VPC only |

Outputs include `instance_ip_address`, `nodebalancer_ipv4` and `zipkin_vpc_ip` (the monitoring server).

## Variables (set in the HCP Terraform workspace)

| Variable | Notes |
| :--- | :--- |
| `provider_token` | Akamai API token (sensitive). Needs **Linodes, NodeBalancers, Firewalls, VPCs, IPs: Read/Write** and **Events: Read** — without Events, instance create/delete fails with `401 ... not authorized to use this endpoint`. |
| `authorized_keys` | **Single-line SSH public key** (`ssh-ed25519 AAAA… comment`), installed for root. Validated: a private key or multi-line value fails at plan time. Its private half is the GitHub `dev` secret `SSH_PRIVATE_KEY`. |
| `root_password` | Root password (sensitive), used for the Akamai LISH console. |
| `region`, `infra_name`, `environment` | Defaults in `variables.tf`. |

## Running it

Normally through GitHub Actions: **Terraform** workflow (`.github/workflows/component.infra.tf.deploy.yml`) with `apply`. It is deliberately manual — review the plan first.

Locally (requires `terraform login` to HCP Terraform):

```sh
cd infra
terraform init
terraform plan
terraform apply
```

## Gotchas

- **Changing `authorized_keys` or the instance image replaces the servers.** Akamai applies keys only at creation. All data on them (including PostgreSQL) is lost; DNS keeps working because it points at the NodeBalancer.
- **NodeBalancer ↔ nginx are coupled.** The `:443` PROXY protocol setting and nginx's `listen 443 ssl proxy_protocol` must change together, and the `:80` health check must never be redirected — otherwise the NodeBalancer marks the server down and returns 503.
- **After servers are rebuilt**, run the **Run ansible** workflow with scope `full`, and clear old SSH host keys locally (`ssh-keygen -R <ip>`, `ssh-keygen -R 10.0.0.3`).
- **Destroy is irreversible.** The scheduled workflow trigger runs `destroy` (see the workflow file) — check before enabling schedules.

## Unused legacy files

`environments/`, `envs/*.tfvars` and `scripts/setup.sh` come from an earlier staging/production design and are not used by the workflows. The real variable values are in the HCP Terraform workspace.
