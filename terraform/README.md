# MyGuy Infrastructure - Terraform Configuration

This directory contains Terraform configuration for deploying MyGuy application infrastructure on Akamai Cloud (Linode).

## 🏗️ Architecture Overview

```
Production Environment (€39/month):
├── 1x Linode Standard 2GB Instance (€9/month)
├── 1x Managed PostgreSQL 1GB (€15/month)
├── 1x Object Storage 250GB (€5/month)
├── 1x Load Balancer (€10/month)
└── SSL Certificate (Free - Let's Encrypt)

Staging Environment (€7/month):
├── 1x Linode Nanode 1GB Instance (€4.50/month)
├── Shared PostgreSQL (Production)
├── Object Storage 50GB (€2.50/month)
└── No Load Balancer
```

## 📋 Prerequisites

1. **Akamai Cloud Account**: Set up at [cloud.linode.com](https://cloud.linode.com)
2. **Domain**: Configure `myguy.work` DNS to point to your infrastructure
3. **GitHub Secrets**: Required secrets for automated deployment
4. **SSH Key Pair**: For server access

## 🔐 Required Secrets

Configure these secrets in your GitHub repository:

### Akamai/Linode Secrets
```bash
LINODE_TOKEN=your_linode_api_token
LINODE_OBJECT_STORAGE_ACCESS_KEY=your_object_storage_access_key
LINODE_OBJECT_STORAGE_SECRET_KEY=your_object_storage_secret_key
```

### SSH Access
```bash
SSH_PUBLIC_KEY=ssh-rsa AAAAB3NzaC1yc2E...
SSH_PRIVATE_KEY=-----BEGIN OPENSSH PRIVATE KEY-----
```

### Application Secrets
```bash
JWT_SECRET=your_staging_jwt_secret_32_chars
JWT_SECRET_PRODUCTION=your_production_jwt_secret_32_chars
POSTGRES_PASSWORD=your_staging_db_password
POSTGRES_PASSWORD_PRODUCTION=your_production_db_password
```

## 🚀 Deployment Workflows

### Staging Deployment (Cost-Optimized)
- **Create Staging**: PR opened to `main` branch → Deploy to `staging.myguy.work`
- **Update Staging**: PR updated → Redeploy staging
- **Destroy Staging**: PR closed → Automatic destruction (saves €7/month)
- **Recreate Staging**: PR reopened → Automatic recreation
- **Cost**: ~€7/month (only when PRs are active)
- **Features**: Shared database, no load balancer

### Production Deployment (Automatic)
- **Trigger**: Push to `main` branch (PR merged)
- **Process**: 
  1. Destroy staging environment first (if exists)
  2. Deploy to production `myguy.work`
  3. Run health checks and tests
- **Cost**: ~€39/month
- **Features**: Dedicated database, load balancer, SSL

### Cost Optimization Strategy
- **Development**: €46/month (staging + production)
- **Production Only**: €39/month (staging destroyed)
- **Average**: ~€40/month (staging only during testing)

## 🛠️ Manual Deployment

### Initial Setup

1. **Generate SSH Key Pair**:
```bash
ssh-keygen -t rsa -b 4096 -f ~/.ssh/myguy_deploy
```

2. **Configure Terraform Backend**:
```bash
# Create Object Storage bucket for Terraform state
# This should be done manually via Linode console first
```

3. **Initialize Terraform**:
```bash
cd terraform
terraform init
```

### Deploy Staging
```bash
terraform workspace new staging
terraform apply -var-file="environments/staging/terraform.tfvars"
```

### Deploy Production
```bash
terraform workspace new production
terraform apply -var-file="environments/production/terraform.tfvars"
```

## 📁 Directory Structure

```
terraform/
├── main.tf                    # Main infrastructure resources
├── variables.tf              # Input variables
├── outputs.tf               # Output values
├── providers.tf             # Provider configuration
├── scripts/
│   └── setup.sh            # Instance initialization script
├── environments/
│   ├── staging/
│   │   └── terraform.tfvars # Staging configuration
│   └── production/
│       └── terraform.tfvars # Production configuration
└── README.md               # This file
```

## 🔧 Configuration Files

### Environment Variables
The setup script creates `/opt/myguy/.env` with:
- Database connection strings
- JWT secrets
- Object storage configuration
- Application URLs

### Nginx Configuration
- Reverse proxy for all services
- SSL termination with Let's Encrypt
- Load balancing (production only)

### Docker Compose Overrides
- `docker-compose.staging.yml`: Staging-specific settings
- `docker-compose.production.yml`: Production optimizations

## 🌐 DNS Configuration

### Production Setup
```
Type: A
Name: myguy.work
Value: <load_balancer_ip>

Type: A  
Name: staging.myguy.work
Value: <staging_instance_ip>
```

### SSL Certificates
- Automatically provisioned via Let's Encrypt
- Renewed automatically via certbot
- Staging uses Let's Encrypt staging environment

## 📊 Cost Breakdown

### Production (€39/month)
| Service | Specification | Cost |
|---------|--------------|------|
| Compute Instance | 2GB RAM, 1 CPU | €9.00 |
| PostgreSQL | 1GB Managed DB | €15.00 |
| Object Storage | 250GB | €5.00 |
| Load Balancer | HTTP/HTTPS | €10.00 |
| **Total** | | **€39.00** |

### Staging (€7/month)
| Service | Specification | Cost |
|---------|--------------|------|
| Compute Instance | 1GB RAM, 1 CPU | €4.50 |
| Object Storage | 50GB | €2.50 |
| Database | Shared with Prod | €0.00 |
| **Total** | | **€7.00** |

## 🔍 Monitoring & Maintenance

### Health Checks
- Application endpoints: `/health`
- Database connectivity
- Object storage access

### Logging
- Application logs: `docker compose logs`
- System logs: `journalctl`
- Nginx logs: `/var/log/nginx/`

### Backup Strategy
- Database: Managed service backups (disabled for cost)
- Application: Git repository + Docker images
- User uploads: Object storage (365-day lifecycle)

## 🚨 Troubleshooting

### Common Issues

1. **Terraform State Lock**:
```bash
terraform force-unlock <lock_id>
```

2. **SSL Certificate Issues**:
```bash
sudo certbot renew --dry-run
```

3. **Service Not Starting**:
```bash
ssh root@<instance_ip>
docker compose logs <service_name>
```

4. **Database Connection**:
```bash
docker compose exec api go run cmd/health/main.go
```

### Emergency Procedures

1. **Rollback Deployment**:
```bash
ssh root@<instance_ip>
cd /opt/myguy-backup
docker compose up -d
```

2. **Scale Up Instance**:
```bash
# Update terraform.tfvars with larger instance type
terraform apply
```

## 📞 Support

For infrastructure issues:
1. Check GitHub Actions logs
2. Review Terraform state
3. SSH to instances for debugging
4. Contact Akamai Cloud support if needed

## 🔄 Updates

To update infrastructure:
1. Modify Terraform files
2. Create PR (triggers plan)
3. Merge to main (triggers apply)
4. Monitor deployment in GitHub Actions