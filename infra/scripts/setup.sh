#!/bin/bash
set -e

# Variables from Terraform
ENVIRONMENT="${environment}"
DOMAIN_NAME="${domain_name}"
JWT_SECRET="${jwt_secret}"
POSTGRES_HOST="${postgres_host}"
POSTGRES_PASSWORD="${postgres_password}"
BUCKET_NAME="${bucket_name}"
BUCKET_REGION="${bucket_region}"

echo "=== MyGuy Application Setup - Environment: $ENVIRONMENT ==="

# Update system
apt-get update -y
apt-get upgrade -y

# Install required packages
apt-get install -y \
    curl \
    wget \
    git \
    docker.io \
    docker-compose-plugin \
    nginx \
    certbot \
    python3-certbot-nginx \
    ufw \
    htop \
    unzip \
    software-properties-common \
    apt-transport-https \
    ca-certificates \
    gnupg \
    lsb-release

# Enable and start Docker
systemctl enable docker
systemctl start docker

# Add ubuntu user to docker group
usermod -aG docker root

# Install Node.js (for frontend building)
curl -fsSL https://deb.nodesource.com/setup_18.x | bash -
apt-get install -y nodejs

# Install Go (for backend services)
GO_VERSION="1.21.5"
wget -q "https://golang.org/dl/go$GO_VERSION.linux-amd64.tar.gz"
tar -C /usr/local -xzf "go$GO_VERSION.linux-amd64.tar.gz"
rm "go$GO_VERSION.linux-amd64.tar.gz"

# Add Go to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/environment
export PATH=$PATH:/usr/local/go/bin

# Create application directory
mkdir -p /opt/myguy
cd /opt/myguy

# Create application user
useradd -r -s /bin/false myguy || true
chown -R myguy:myguy /opt/myguy

# Create environment file
cat > .env << EOF
# Environment Configuration
ENVIRONMENT=$ENVIRONMENT
DOMAIN_NAME=$DOMAIN_NAME

# Database Configuration
DB_CONNECTION="host=$POSTGRES_HOST user=postgres password=$POSTGRES_PASSWORD dbname=my_guy port=5432 sslmode=require"
POSTGRES_HOST=$POSTGRES_HOST
POSTGRES_DB=my_guy
POSTGRES_USER=postgres
POSTGRES_PASSWORD=$POSTGRES_PASSWORD

# JWT Configuration
JWT_SECRET=$JWT_SECRET

# Object Storage Configuration
BUCKET_NAME=$BUCKET_NAME
BUCKET_REGION=$BUCKET_REGION
BUCKET_ENDPOINT=$BUCKET_REGION.linodeobjects.com

# Application URLs
if [ "$ENVIRONMENT" = "production" ]; then
    CLIENT_URL=https://$DOMAIN_NAME
    API_URL=https://$DOMAIN_NAME
else
    CLIENT_URL=https://$DOMAIN_NAME
    API_URL=https://$DOMAIN_NAME
fi

# Service Ports
MAIN_BACKEND_PORT=8080
STORE_SERVICE_PORT=8081
CHAT_SERVICE_PORT=8082
FRONTEND_PORT=5173

# Node Environment
NODE_ENV=$ENVIRONMENT
EOF

# Set proper permissions
chown myguy:myguy .env
chmod 600 .env

# Configure Nginx
cat > /etc/nginx/sites-available/myguy << 'EOF'
server {
    listen 80;
    server_name DOMAIN_PLACEHOLDER;
    
    # Redirect HTTP to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name DOMAIN_PLACEHOLDER;

    # SSL Configuration (will be updated by certbot)
    ssl_certificate /etc/ssl/certs/ssl-cert-snakeoil.pem;
    ssl_certificate_key /etc/ssl/private/ssl-cert-snakeoil.key;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;
    add_header Content-Security-Policy "default-src 'self' http: https: data: blob: 'unsafe-inline'" always;

    # Frontend (Vue.js)
    location / {
        proxy_pass http://localhost:5173;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }

    # Main Backend API
    location /api/v1/ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Store Service API
    location /store/api/ {
        rewrite ^/store/api/(.*) /api/v1/$1 break;
        proxy_pass http://localhost:8081;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Chat WebSocket Service
    location /chat/ {
        proxy_pass http://localhost:8082;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # File uploads from store service
    location /uploads/ {
        proxy_pass http://localhost:8081;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Health check
    location /health {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        access_log off;
    }
}
EOF

# Replace domain placeholder
sed -i "s/DOMAIN_PLACEHOLDER/$DOMAIN_NAME/g" /etc/nginx/sites-available/myguy

# Enable the site
ln -sf /etc/nginx/sites-available/myguy /etc/nginx/sites-enabled/
rm -f /etc/nginx/sites-enabled/default

# Test nginx configuration
nginx -t

# Configure UFW firewall
ufw --force enable
ufw allow ssh
ufw allow 'Nginx Full'
ufw allow 8080  # Temporary for debugging
ufw allow 8081  # Temporary for debugging
ufw allow 8082  # Temporary for debugging

# Start nginx
systemctl enable nginx
systemctl start nginx

# Create deployment script
cat > /opt/myguy/deploy.sh << 'EOF'
#!/bin/bash
set -e

REPO_URL="https://github.com/YOUR_USERNAME/YOUR_REPO.git"
BRANCH="${1:-main}"
APP_DIR="/opt/myguy"

echo "=== Deploying MyGuy Application ==="
echo "Branch: $BRANCH"
echo "Environment: $ENVIRONMENT"

cd $APP_DIR

# Stop running services
docker compose down || true

# Pull latest code
if [ -d ".git" ]; then
    git fetch origin
    git reset --hard origin/$BRANCH
else
    git clone $REPO_URL .
    git checkout $BRANCH
fi

# Load environment variables
source .env

# Build and start services
docker compose -f docker-compose.yml -f docker-compose.$ENVIRONMENT.yml up --build -d

# Wait for services to be healthy
echo "Waiting for services to start..."
sleep 30

# Run database migrations
docker compose exec api go run cmd/migrate/main.go || true
docker compose exec chat-websocket-service npm run migrate || true

# Restart nginx to pick up any changes
systemctl reload nginx

echo "=== Deployment Complete ==="
echo "Application URL: https://$DOMAIN_NAME"
EOF

chmod +x /opt/myguy/deploy.sh

# Create systemd service for auto-deployment
cat > /etc/systemd/system/myguy-deploy.service << EOF
[Unit]
Description=MyGuy Application Deployment
After=network.target docker.service

[Service]
Type=oneshot
User=root
WorkingDirectory=/opt/myguy
ExecStart=/opt/myguy/deploy.sh
Environment="ENVIRONMENT=$ENVIRONMENT"
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable myguy-deploy.service

# Install webhook listener for GitHub deployments
cat > /opt/myguy/webhook-server.js << 'EOF'
const http = require('http');
const crypto = require('crypto');
const { exec } = require('child_process');

const SECRET = process.env.GITHUB_WEBHOOK_SECRET || 'changeme';
const PORT = process.env.WEBHOOK_PORT || 9000;

const server = http.createServer((req, res) => {
    if (req.method === 'POST' && req.url === '/webhook') {
        let body = '';
        
        req.on('data', chunk => {
            body += chunk.toString();
        });
        
        req.on('end', () => {
            const signature = req.headers['x-hub-signature-256'];
            const expectedSignature = 'sha256=' + crypto
                .createHmac('sha256', SECRET)
                .update(body)
                .digest('hex');
            
            if (signature === expectedSignature) {
                const payload = JSON.parse(body);
                
                if (payload.ref === 'refs/heads/main' || payload.ref === 'refs/heads/staging') {
                    const branch = payload.ref.split('/').pop();
                    console.log(`Deploying branch: ${branch}`);
                    
                    exec(`/opt/myguy/deploy.sh ${branch}`, (error, stdout, stderr) => {
                        if (error) {
                            console.error(`Deployment error: ${error}`);
                        } else {
                            console.log(`Deployment successful: ${stdout}`);
                        }
                    });
                }
                
                res.writeHead(200);
                res.end('OK');
            } else {
                res.writeHead(401);
                res.end('Unauthorized');
            }
        });
    } else {
        res.writeHead(404);
        res.end('Not Found');
    }
});

server.listen(PORT, () => {
    console.log(`Webhook server listening on port ${PORT}`);
});
EOF

# Create systemd service for webhook
cat > /etc/systemd/system/myguy-webhook.service << EOF
[Unit]
Description=MyGuy GitHub Webhook Server
After=network.target

[Service]
Type=simple
User=myguy
WorkingDirectory=/opt/myguy
ExecStart=/usr/bin/node webhook-server.js
Restart=always
Environment="GITHUB_WEBHOOK_SECRET=changeme"
Environment="WEBHOOK_PORT=9000"

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable myguy-webhook.service
systemctl start myguy-webhook.service

# Setup SSL certificate (Let's Encrypt)
if [ "$ENVIRONMENT" = "production" ]; then
    certbot --nginx -d $DOMAIN_NAME --non-interactive --agree-tos --email admin@$DOMAIN_NAME
else
    certbot --nginx -d $DOMAIN_NAME --non-interactive --agree-tos --email admin@$DOMAIN_NAME --staging
fi

echo "=== Setup Complete ==="
echo "Instance ready for deployment"
echo "SSH: ssh root@$(curl -s http://169.254.169.254/v1/json | grep -o '"ipv4":"[^"]*' | cut -d'"' -f4)"
echo "Application: https://$DOMAIN_NAME"