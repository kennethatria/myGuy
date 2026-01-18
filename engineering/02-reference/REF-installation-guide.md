# Installation Guide: MyGuy Platform

**Version:** 1.0
**Last Updated:** January 18, 2026

This guide covers deploying the MyGuy platform to a production server.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Server Requirements](#server-requirements)
3. [Quick Start (Docker)](#quick-start-docker)
4. [Manual Installation](#manual-installation)
5. [Environment Configuration](#environment-configuration)
6. [Database Setup](#database-setup)
7. [Reverse Proxy Setup (Nginx)](#reverse-proxy-setup-nginx)
8. [SSL/HTTPS Configuration](#sslhttps-configuration)
9. [Systemd Services](#systemd-services-manual-deployment)
10. [Health Checks & Monitoring](#health-checks--monitoring)
11. [Troubleshooting](#troubleshooting)

---

## Prerequisites

### Required Software

| Software | Minimum Version | Purpose |
|----------|-----------------|---------|
| Docker & Docker Compose | 24.0+ / 2.20+ | Container orchestration (recommended) |
| Go | 1.24+ | Backend & Store Service (manual install) |
| Node.js | 18.0+ | Chat Service (manual install) |
| PostgreSQL | 15+ | Database |
| Redis | 7+ | Socket.IO adapter (optional, for scaling) |
| Nginx | 1.24+ | Reverse proxy & SSL termination |

### Domain & SSL

- A domain name pointing to your server
- SSL certificate (Let's Encrypt recommended)

---

## Server Requirements

### Minimum (Development/Testing)

- **CPU:** 2 cores
- **RAM:** 4 GB
- **Storage:** 20 GB SSD
- **OS:** Ubuntu 22.04 LTS / Debian 12

### Recommended (Production)

- **CPU:** 4+ cores
- **RAM:** 8+ GB
- **Storage:** 50+ GB SSD
- **OS:** Ubuntu 22.04 LTS / Debian 12

### Ports Required

| Port | Service | Access |
|------|---------|--------|
| 80 | HTTP (redirect to HTTPS) | Public |
| 443 | HTTPS | Public |
| 5432 | PostgreSQL | Internal only |
| 6379 | Redis | Internal only |
| 8080 | Backend API | Internal (proxied) |
| 8081 | Store Service | Internal (proxied) |
| 8082 | Chat Service | Internal (proxied) |

---

## Quick Start (Docker)

### 1. Clone Repository

```bash
git clone https://github.com/your-org/myguy.git
cd myguy
```

### 2. Configure Environment

```bash
# Copy example environment files
cp backend/.env.example backend/.env
cp store-service/.env.example store-service/.env
cp chat-websocket-service/.env.example chat-websocket-service/.env
cp frontend/.env.example frontend/.env

# Generate secure secrets
JWT_SECRET=$(openssl rand -base64 32)
INTERNAL_API_KEY=$(openssl rand -base64 32)

# Update all .env files with the same JWT_SECRET
sed -i "s/your-secret-key-here/$JWT_SECRET/g" backend/.env
sed -i "s/your-secret-key-here/$JWT_SECRET/g" store-service/.env
sed -i "s/your-secret-key-here/$JWT_SECRET/g" chat-websocket-service/.env

# Update INTERNAL_API_KEY
sed -i "s/your-internal-api-key-change-in-production/$INTERNAL_API_KEY/g" store-service/.env
sed -i "s/your-internal-api-key-change-in-production/$INTERNAL_API_KEY/g" chat-websocket-service/.env

# Create nginx directory for configuration
mkdir -p nginx/ssl
# Note: Place your SSL certificates (fullchain.pem, privkey.pem) in nginx/ssl/
```

### 3. Create Production Docker Compose

Create `docker-compose.prod.yml`:

```yaml
services:
  api:
    build:
      context: ./backend
      dockerfile: Dockerfile
    restart: always
    environment:
      - PORT=8080
      - JWT_SECRET=${JWT_SECRET}
      - DB_HOST=postgres-db
      - DB_PORT=5432
      - DB_USER=postgres
      - DB_PASSWORD=${DB_PASSWORD}
      - DB_NAME=my_guy
      - DB_SSL_MODE=disable
    depends_on:
      postgres-db:
        condition: service_healthy
    networks:
      - myguy-network

  store-service:
    build:
      context: ./store-service
      dockerfile: Dockerfile
    restart: always
    environment:
      - PORT=8081
      - JWT_SECRET=${JWT_SECRET}
      - DB_CONNECTION=host=postgres-db user=postgres password=${DB_PASSWORD} dbname=my_guy_store port=5432 sslmode=disable
      - CHAT_API_URL=http://chat-websocket-service:8082/api/v1
      - INTERNAL_API_KEY=${INTERNAL_API_KEY}
    volumes:
      - store-uploads:/app/uploads
    depends_on:
      postgres-db:
        condition: service_healthy
    networks:
      - myguy-network

  chat-websocket-service:
    build:
      context: ./chat-websocket-service
      dockerfile: Dockerfile
    restart: always
    environment:
      - PORT=8082
      - NODE_ENV=production
      - JWT_SECRET=${JWT_SECRET}
      - DB_CONNECTION=postgresql://postgres:${DB_PASSWORD}@postgres-db:5432/my_guy_chat
      - DATABASE_URL=postgresql://postgres:${DB_PASSWORD}@postgres-db:5432/my_guy_chat
      - CLIENT_URL=${CLIENT_URL}
      - MAIN_API_URL=http://api:8080/api/v1
      - STORE_API_URL=http://store-service:8081/api/v1
      - INTERNAL_API_KEY=${INTERNAL_API_KEY}
      - REDIS_URL=redis://redis:6379/0
    depends_on:
      postgres-db:
        condition: service_healthy
      redis:
        condition: service_healthy
    networks:
      - myguy-network

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
      args:
        - VITE_API_URL=${API_URL}
        - VITE_STORE_API_URL=${STORE_API_URL}
        - VITE_STORE_API_BASE_URL=${STORE_API_BASE_URL}
        - VITE_CHAT_API_URL=${CHAT_API_URL}
        - VITE_CHAT_WS_URL=${CHAT_WS_URL}
    restart: always
    networks:
      - myguy-network

  postgres-db:
    image: postgres:15-alpine
    restart: always
    environment:
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=${DB_PASSWORD}
      - POSTGRES_DB=my_guy
      - POSTGRES_MULTIPLE_DATABASES=my_guy,my_guy_chat,my_guy_store
    volumes:
      - postgres-data:/var/lib/postgresql/data
      - ./scripts/create-multiple-databases.sh:/docker-entrypoint-initdb.d/create-databases.sh
    networks:
      - myguy-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    restart: always
    volumes:
      - redis-data:/data
    networks:
      - myguy-network
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 5
    command: redis-server --appendonly yes

  nginx:
    image: nginx:alpine
    restart: always
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./nginx/ssl:/etc/nginx/ssl:ro
      - store-uploads:/var/www/uploads:ro
    depends_on:
      - api
      - store-service
      - chat-websocket-service
      - frontend
    networks:
      - myguy-network

volumes:
  postgres-data:
  store-uploads:
  redis-data:

networks:
  myguy-network:
    driver: bridge
```

### 4. Create Production Environment File

Create `.env.prod`:

```bash
# Database
DB_PASSWORD=your-secure-database-password

# Authentication (MUST be same across all services)
JWT_SECRET=your-secure-jwt-secret-min-32-chars
INTERNAL_API_KEY=your-secure-internal-api-key

# URLs (replace with your domain)
CLIENT_URL=https://yourdomain.com
API_URL=https://yourdomain.com/api/v1
STORE_API_URL=https://yourdomain.com/store/api/v1
STORE_API_BASE_URL=https://yourdomain.com/store
CHAT_API_URL=https://yourdomain.com/chat/api/v1
CHAT_WS_URL=wss://yourdomain.com
```

### 5. Deploy

```bash
# Load environment variables
export $(cat .env.prod | xargs)

# Build and start services
docker-compose -f docker-compose.prod.yml up -d --build

# Check status
docker-compose -f docker-compose.prod.yml ps

# View logs
docker-compose -f docker-compose.prod.yml logs -f
```

---

## Manual Installation

### 1. Install Dependencies

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install Go
wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install Node.js 20
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt install -y nodejs

# Install PostgreSQL
sudo apt install -y postgresql postgresql-contrib

# Install Redis
sudo apt install -y redis-server
sudo systemctl enable redis-server

# Install Nginx
sudo apt install -y nginx
```

### 2. Create Application User

```bash
sudo useradd -m -s /bin/bash myguy
sudo mkdir -p /opt/myguy
sudo chown myguy:myguy /opt/myguy
```

### 3. Clone and Build Services

```bash
sudo -u myguy bash
cd /opt/myguy
git clone https://github.com/your-org/myguy.git .

# Build Backend
cd backend
go build -o /opt/myguy/bin/backend ./cmd/api/main.go

# Build Store Service
cd ../store-service
go build -o /opt/myguy/bin/store-service ./cmd/api/main.go

# Install Chat Service dependencies
cd ../chat-websocket-service
npm ci --production

# Build Frontend
cd ../frontend
npm ci
npm run build
```

---

## Environment Configuration

### Backend (`/opt/myguy/backend/.env`)

```env
PORT=8080
JWT_SECRET=your-secure-jwt-secret
DB_HOST=localhost
DB_PORT=5432
DB_USER=myguy
DB_PASSWORD=your-db-password
DB_NAME=my_guy
DB_SSL_MODE=disable
```

### Store Service (`/opt/myguy/store-service/.env`)

```env
PORT=8081
JWT_SECRET=your-secure-jwt-secret
DB_CONNECTION=host=localhost user=myguy password=your-db-password dbname=my_guy_store port=5432 sslmode=disable
CHAT_API_URL=http://localhost:8082/api/v1
INTERNAL_API_KEY=your-internal-api-key
```

### Chat Service (`/opt/myguy/chat-websocket-service/.env`)

```env
PORT=8082
NODE_ENV=production
JWT_SECRET=your-secure-jwt-secret
DB_CONNECTION=postgresql://myguy:your-db-password@localhost:5432/my_guy_chat
DATABASE_URL=postgresql://myguy:your-db-password@localhost:5432/my_guy_chat
CLIENT_URL=https://yourdomain.com
MAIN_API_URL=http://localhost:8080/api/v1
STORE_API_URL=http://localhost:8081/api/v1
INTERNAL_API_KEY=your-internal-api-key
REDIS_URL=redis://localhost:6379/0
```

---

## Database Setup

### 1. Create PostgreSQL User and Databases

```bash
sudo -u postgres psql

-- Create user
CREATE USER myguy WITH PASSWORD 'your-db-password';

-- Create databases
CREATE DATABASE my_guy OWNER myguy;
CREATE DATABASE my_guy_chat OWNER myguy;
CREATE DATABASE my_guy_store OWNER myguy;

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE my_guy TO myguy;
GRANT ALL PRIVILEGES ON DATABASE my_guy_chat TO myguy;
GRANT ALL PRIVILEGES ON DATABASE my_guy_store TO myguy;

\q
```

### 2. Initialize Schemas

Schemas are auto-migrated on service startup via GORM (Go services) and node-pg-migrate (Chat service).

---

## Reverse Proxy Setup (Nginx)

### Create Nginx Configuration

```bash
sudo nano /etc/nginx/sites-available/myguy
```

```nginx
# Rate limiting
limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;
limit_req_zone $binary_remote_addr zone=ws_limit:10m rate=5r/s;

# Upstream definitions
upstream backend_api {
    server 127.0.0.1:8080;
    keepalive 32;
}

upstream store_api {
    server 127.0.0.1:8081;
    keepalive 32;
}

upstream chat_api {
    server 127.0.0.1:8082;
    keepalive 32;
}

# HTTP -> HTTPS redirect
server {
    listen 80;
    server_name yourdomain.com;
    return 301 https://$server_name$request_uri;
}

# Main HTTPS server
server {
    listen 443 ssl http2;
    server_name yourdomain.com;

    # SSL Configuration
    ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256;
    ssl_prefer_server_ciphers off;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    # Gzip compression
    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml;

    # Backend API
    location /api/v1/ {
        limit_req zone=api_limit burst=20 nodelay;

        proxy_pass http://backend_api;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # Store API
    location /store/api/v1/ {
        limit_req zone=api_limit burst=20 nodelay;

        rewrite ^/store/api/v1/(.*)$ /api/v1/$1 break;
        proxy_pass http://store_api;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Store uploads (static files)
    location /uploads/ {
        alias /opt/myguy/store-service/uploads/;
        expires 30d;
        add_header Cache-Control "public, immutable";
    }

    # Chat API
    location /chat/api/v1/ {
        limit_req zone=api_limit burst=20 nodelay;

        rewrite ^/chat/api/v1/(.*)$ /api/v1/$1 break;
        proxy_pass http://chat_api;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # WebSocket endpoint
    location /socket.io/ {
        limit_req zone=ws_limit burst=10 nodelay;

        proxy_pass http://chat_api;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket timeouts
        proxy_connect_timeout 7d;
        proxy_send_timeout 7d;
        proxy_read_timeout 7d;
    }

    # Frontend (static files)
    location / {
        root /opt/myguy/frontend/dist;
        try_files $uri $uri/ /index.html;
        expires 1h;
        add_header Cache-Control "public";
    }

    # Static assets caching
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2)$ {
        root /opt/myguy/frontend/dist;
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
}
```

### Enable Site

```bash
sudo ln -s /etc/nginx/sites-available/myguy /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

---

## SSL/HTTPS Configuration

### Using Let's Encrypt (Certbot)

```bash
# Install Certbot
sudo apt install -y certbot python3-certbot-nginx

# Obtain certificate
sudo certbot --nginx -d yourdomain.com

# Auto-renewal (already configured by certbot)
sudo systemctl status certbot.timer
```

---

## Systemd Services (Manual Deployment)

### Backend Service

```bash
sudo nano /etc/systemd/system/myguy-backend.service
```

```ini
[Unit]
Description=MyGuy Backend API
After=network.target postgresql.service

[Service]
Type=simple
User=myguy
Group=myguy
WorkingDirectory=/opt/myguy/backend
ExecStart=/opt/myguy/bin/backend
Restart=always
RestartSec=5
Environment=PORT=8080

[Install]
WantedBy=multi-user.target
```

### Store Service

```bash
sudo nano /etc/systemd/system/myguy-store.service
```

```ini
[Unit]
Description=MyGuy Store Service
After=network.target postgresql.service

[Service]
Type=simple
User=myguy
Group=myguy
WorkingDirectory=/opt/myguy/store-service
ExecStart=/opt/myguy/bin/store-service
Restart=always
RestartSec=5
Environment=PORT=8081

[Install]
WantedBy=multi-user.target
```

### Chat Service

```bash
sudo nano /etc/systemd/system/myguy-chat.service
```

```ini
[Unit]
Description=MyGuy Chat WebSocket Service
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=myguy
Group=myguy
WorkingDirectory=/opt/myguy/chat-websocket-service
ExecStart=/usr/bin/node src/server.js
Restart=always
RestartSec=5
Environment=NODE_ENV=production
Environment=PORT=8082

[Install]
WantedBy=multi-user.target
```

### Enable and Start Services

```bash
sudo systemctl daemon-reload
sudo systemctl enable myguy-backend myguy-store myguy-chat
sudo systemctl start myguy-backend myguy-store myguy-chat

# Check status
sudo systemctl status myguy-backend myguy-store myguy-chat
```

---

## Health Checks & Monitoring

### Health Check Endpoints

| Service | Endpoint | Expected Response |
|---------|----------|-------------------|
| Backend | `GET /health` | `{"status": "ok"}` |
| Store | `GET /health` | `{"status": "ok"}` |
| Chat | `GET /health` | `{"status": "ok", "redis": {...}}` |

### Monitoring Script

Create `/opt/myguy/scripts/healthcheck.sh`:

```bash
#!/bin/bash

SERVICES=("http://localhost:8080/health" "http://localhost:8081/health" "http://localhost:8082/health")
NAMES=("Backend" "Store" "Chat")

for i in "${!SERVICES[@]}"; do
    STATUS=$(curl -s -o /dev/null -w "%{http_code}" "${SERVICES[$i]}")
    if [ "$STATUS" -eq 200 ]; then
        echo "${NAMES[$i]}: OK"
    else
        echo "${NAMES[$i]}: FAILED (HTTP $STATUS)"
        # Optional: Send alert
    fi
done
```

### Cron Health Check

```bash
# Add to crontab
*/5 * * * * /opt/myguy/scripts/healthcheck.sh >> /var/log/myguy-health.log 2>&1
```

---

## Troubleshooting

### Common Issues

#### Services Won't Start

```bash
# Check logs
sudo journalctl -u myguy-backend -f
sudo journalctl -u myguy-store -f
sudo journalctl -u myguy-chat -f

# Check if ports are in use
sudo lsof -i :8080
sudo lsof -i :8081
sudo lsof -i :8082
```

#### Database Connection Failed

```bash
# Test PostgreSQL connection
psql -h localhost -U myguy -d my_guy -c "SELECT 1"

# Check PostgreSQL is running
sudo systemctl status postgresql
```

#### WebSocket Connection Failed

```bash
# Check Nginx WebSocket proxy
sudo tail -f /var/log/nginx/error.log

# Test WebSocket directly
wscat -c ws://localhost:8082/socket.io/?EIO=4&transport=websocket
```

#### Redis Connection Failed

```bash
# Test Redis
redis-cli ping

# Check Redis is running
sudo systemctl status redis
```

### Log Locations

| Service | Log Location |
|---------|--------------|
| Backend | `journalctl -u myguy-backend` |
| Store | `journalctl -u myguy-store` |
| Chat | `journalctl -u myguy-chat` |
| Nginx | `/var/log/nginx/access.log`, `/var/log/nginx/error.log` |
| PostgreSQL | `/var/log/postgresql/` |

---

## Related Documentation

- [Architecture Overview](./ARCH-chat-service-architecture.md)
- [Deployment Checklist](./REF-deployment-checklist.md)
- [Environment Configuration](../../CLAUDE.md#environment-configuration)

---

**Document Version:** 1.0
**Created:** January 18, 2026
