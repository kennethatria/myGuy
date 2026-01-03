# 🚀 MyGuy Deployment Checklist

## Pre-Deployment Setup

### ✅ 1. Akamai Cloud Account Setup
- [ ] Create account at [cloud.linode.com](https://cloud.linode.com)
- [ ] Generate API token with full access
- [ ] Create Object Storage access keys
- [ ] Note down your account details

### ✅ 2. Domain Configuration
- [ ] Ensure you own `myguy.work` domain
- [ ] Access to DNS management (registrar or Cloudflare)
- [ ] Prepare to update A records during deployment

### ✅ 3. SSH Key Generation
```bash
# Generate SSH key pair for server access
ssh-keygen -t rsa -b 4096 -f ~/.ssh/myguy_deploy
```
- [ ] SSH key pair generated
- [ ] Public key saved for GitHub secrets
- [ ] Private key saved securely

### ✅ 4. GitHub Repository Setup
- [ ] Fork or clone the MyGuy repository
- [ ] Configure GitHub Secrets (see section below)
- [ ] Ensure Actions are enabled

## 🔐 GitHub Secrets Configuration

Add these secrets in GitHub: Settings → Secrets and variables → Actions

### Akamai/Linode Secrets
```bash
LINODE_TOKEN=your_linode_api_token_here
LINODE_OBJECT_STORAGE_ACCESS_KEY=your_access_key
LINODE_OBJECT_STORAGE_SECRET_KEY=your_secret_key
```

### SSH Access Keys
```bash
SSH_PUBLIC_KEY=ssh-rsa AAAAB3NzaC1yc2E... (content of ~/.ssh/myguy_deploy.pub)
SSH_PRIVATE_KEY=-----BEGIN OPENSSH PRIVATE KEY----- (content of ~/.ssh/myguy_deploy)
```

### Application Secrets
```bash
JWT_SECRET=your_staging_jwt_secret_32_characters_minimum
JWT_SECRET_PRODUCTION=your_production_jwt_secret_32_chars_min
POSTGRES_PASSWORD=your_staging_database_password
POSTGRES_PASSWORD_PRODUCTION=your_production_database_password
```

### Optional Secrets
```bash
GITHUB_TOKEN=${{ secrets.GITHUB_TOKEN }}  # Usually auto-provided
```

## 🏗️ Infrastructure Deployment

### Option A: Automatic Deployment (Recommended)
1. **Create Pull Request**:
   - [ ] Create PR to `main` branch
   - [ ] Verify Terraform plan in PR comments
   - [ ] Check staging deployment succeeds at `staging.myguy.work`
   - [ ] Test application functionality on staging

2. **Deploy to Production**:
   - [ ] Merge PR to `main` branch
   - [ ] Staging environment automatically destroyed (saves €7/month)
   - [ ] Production deployment begins automatically
   - [ ] Monitor GitHub Actions workflow
   - [ ] Verify production deployment at `myguy.work`

3. **Development Workflow**:
   - [ ] New PRs automatically create fresh staging environments
   - [ ] Closed PRs automatically destroy staging (cost optimization)
   - [ ] Reopened PRs automatically recreate staging environments

### Option B: Manual Deployment
1. **Initialize Terraform**:
```bash
cd terraform
terraform init -backend-config="bucket=myguy-terraform-state"
```

2. **Deploy Staging**:
```bash
terraform workspace new staging
terraform apply -var-file="environments/staging/terraform.tfvars"
```

3. **Deploy Production**:
```bash
terraform workspace new production  
terraform apply -var-file="environments/production/terraform.tfvars"
```

## 🌐 DNS Configuration

After deployment, update DNS records:

### Production
```
Type: A
Name: myguy.work
Value: <load_balancer_ip_from_terraform_output>
TTL: 300
```

### Staging
```
Type: A
Name: staging.myguy.work  
Value: <staging_instance_ip_from_terraform_output>
TTL: 300
```

## ✅ Post-Deployment Verification

### 1. Infrastructure Check
- [ ] All instances running in Linode console
- [ ] Database accessible and configured
- [ ] Object storage bucket created
- [ ] Firewall rules applied

### 2. Application Health Checks
```bash
# Production
curl https://myguy.work/health
curl https://myguy.work/api/v1/server-time

# Staging  
curl https://staging.myguy.work/health
curl https://staging.myguy.work/api/v1/server-time
```

### 3. Service Verification
- [ ] **Frontend**: `https://myguy.work` loads correctly
- [ ] **Main Backend**: API endpoints respond
- [ ] **Store Service**: File uploads work
- [ ] **Chat Service**: WebSocket connections establish
- [ ] **Database**: Migrations applied successfully

### 4. SSL Certificate Check
```bash
# Check certificate is valid
curl -I https://myguy.work
openssl s_client -connect myguy.work:443 -servername myguy.work
```

### 5. Performance Check
- [ ] Page load times < 3 seconds
- [ ] API response times < 500ms
- [ ] WebSocket connections stable
- [ ] Image uploads functional

## 🔧 Common Post-Deployment Tasks

### Update Application Configuration
```bash
# SSH to production server
ssh root@<production_ip>

# Update environment variables if needed
cd /opt/myguy
nano .env

# Restart services
docker compose restart
```

### Monitor Logs
```bash
# Application logs
docker compose logs -f

# System logs
journalctl -u myguy-deploy.service

# Nginx logs
tail -f /var/log/nginx/access.log
```

### Scale Resources (if needed)
```bash
# Update instance type in terraform/environments/production/terraform.tfvars
app_instance_type = "g6-standard-2"  # Upgrade to 4GB

# Apply changes
terraform apply
```

## 💰 Cost Monitoring

**Optimized Cost Structure:**
- **Production Active**: €39/month (staging auto-destroyed)
- **Development Phase**: €46/month (when staging + production both active)
- **Average Cost**: ~€40/month (staging only active during PR testing)

**Cost Optimization Features:**
- ✅ Staging destroyed when PRs closed
- ✅ Staging destroyed when production deploys
- ✅ Staging recreated only when needed for testing
- ✅ Automatic resource cleanup

Monitor usage in Linode console to ensure costs stay within budget.

## 🚨 Emergency Procedures

### Rollback Deployment
```bash
ssh root@<instance_ip>
cd /opt/myguy-backup
docker compose up -d
systemctl reload nginx
```

### Scale Up Quickly
```bash
# Update terraform.tfvars with larger instance
production_instance_type = "g6-standard-4"
terraform apply -auto-approve
```

### Database Emergency Access
```bash
# Get database credentials from Terraform
terraform output database_password

# Connect directly
psql -h <db_host> -U postgres -d my_guy
```

## 📞 Support Contacts

- **Akamai Cloud**: [support.linode.com](https://support.linode.com)
- **GitHub Actions**: Check workflow logs and GitHub status
- **DNS Issues**: Contact your domain registrar
- **SSL Issues**: Let's Encrypt community support

## ✨ Success Criteria

Deployment is successful when:
- [ ] ✅ Both staging and production environments accessible
- [ ] ✅ All health checks passing
- [ ] ✅ SSL certificates valid
- [ ] ✅ Application fully functional
- [ ] ✅ Database migrations applied
- [ ] ✅ File uploads working
- [ ] ✅ Real-time messaging operational
- [ ] ✅ Costs within budget (< €50/month)

🎉 **Congratulations! MyGuy is now live on Akamai Cloud!**