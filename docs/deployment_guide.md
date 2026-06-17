# Deployment Guide
## Institution OS
**Version**: 1.0.0

---

## 1. Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Docker | 24+ | Containers |
| Docker Compose | 2.20+ | Local orchestration |
| Go | 1.22+ | Backend development |
| Node.js | 20+ | Frontend development |
| PostgreSQL client | 15+ | Database management |
| golang-migrate | latest | DB migrations |

---

## 2. Local Development

### 2.1 Clone & Configure
```bash
git clone https://github.com/your-org/institution-os.git
cd institution-os

# Copy environment files
cp backend/.env.example backend/.env
cp frontend/.env.example frontend/.env.local

# Edit backend/.env — set DB password, JWT keys, M-Pesa stub keys
```

### 2.2 Generate RSA Keys for JWT
```bash
cd backend
openssl genrsa -out jwt_private.pem 2048
openssl rsa -in jwt_private.pem -pubout -out jwt_public.pem
# Add paths to .env: JWT_PRIVATE_KEY_PATH, JWT_PUBLIC_KEY_PATH
```

### 2.3 Start All Services
```bash
docker-compose up -d postgres redis minio
# Wait for postgres to be ready (~5 seconds)

# Run migrations
cd backend
migrate -path ./migrations -database "$DATABASE_URL" up

# Seed test data
psql "$DATABASE_URL" -f ../scripts/seed_test_data.sql

# Start backend (with hot reload via Air)
air  # or: go run ./cmd/server/

# Start frontend
cd ../frontend
npm install
npm run dev
```

### 2.4 Access
| Service | URL |
|---------|-----|
| Frontend | http://localhost:3000 |
| Backend API | http://localhost:8080 |
| API Docs | http://localhost:8080/docs |
| MinIO Console | http://localhost:9001 |
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3001 |

### 2.5 Default Seed Users
| Email | Password | Role |
|-------|----------|------|
| principal@upendo.ac.ke | Admin1234! | Super Admin |
| finance@upendo.ac.ke | Admin1234! | Accountant |
| teacher1@upendo.ac.ke | Admin1234! | Teacher |
| parent1@upendo.ac.ke | Admin1234! | Parent |

---

## 3. Production Deployment (Single VPS)

### 3.1 Server Requirements
- Ubuntu 22.04 LTS
- 4 vCPU, 8GB RAM, 100GB SSD (minimum for 850 students)
- Static IP address
- Domain name with A record pointing to server IP

### 3.2 Initial Server Setup
```bash
# On VPS as root
apt-get update && apt-get upgrade -y
apt-get install -y docker.io docker-compose-plugin nginx certbot python3-certbot-nginx ufw

# Firewall
ufw allow 22/tcp
ufw allow 80/tcp
ufw allow 443/tcp
ufw enable

# Create app user
useradd -m -s /bin/bash appuser
usermod -aG docker appuser

# Create app directory
mkdir -p /opt/institution-os
chown appuser:appuser /opt/institution-os
```

### 3.3 Nginx Configuration
```nginx
# /etc/nginx/sites-available/institution-os
server {
    listen 80;
    server_name app.upendo.ac.ke;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name app.upendo.ac.ke;

    ssl_certificate     /etc/letsencrypt/live/app.upendo.ac.ke/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/app.upendo.ac.ke/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;

    # Security headers
    add_header Strict-Transport-Security "max-age=63072000" always;
    add_header X-Frame-Options DENY always;
    add_header X-Content-Type-Options nosniff always;
    add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; connect-src 'self';" always;

    # Backend API
    location /api/ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        client_max_body_size 20M;
    }

    # Frontend
    location / {
        proxy_pass http://localhost:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

```bash
ln -s /etc/nginx/sites-available/institution-os /etc/nginx/sites-enabled/
certbot --nginx -d app.upendo.ac.ke
nginx -t && systemctl reload nginx
```

### 3.4 Production Docker Compose
```bash
# /opt/institution-os/docker-compose.prod.yml
# Uses production images from GitHub Container Registry
# See infrastructure/docker-compose.yml for full file
```

### 3.5 Environment Variables (Production)
```bash
# /opt/institution-os/.env.prod
DATABASE_URL=postgres://iosuser:STRONG_PASSWORD@localhost:5432/institution_os
REDIS_URL=redis://:REDIS_PASSWORD@localhost:6379/0
JWT_PRIVATE_KEY_PATH=/run/secrets/jwt_private.pem
JWT_PUBLIC_KEY_PATH=/run/secrets/jwt_public.pem
BACKUP_ENCRYPTION_KEY=<64-char random hex>
BACKUP_BUCKET=institution-os-backups
MPESA_CONSUMER_KEY=<from Safaricom>
MPESA_CONSUMER_SECRET=<from Safaricom>
MPESA_SHORTCODE=<paybill number>
MPESA_PASSKEY=<from Safaricom>
MPESA_CALLBACK_URL=https://app.upendo.ac.ke/api/v1/finance/mpesa/callback
SMS_API_KEY=<Africa's Talking key>
SENDGRID_API_KEY=<SendGrid key>
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=<minio key>
MINIO_SECRET_KEY=<minio secret>
```

---

## 4. GitHub Actions CI/CD

### 4.1 CI Pipeline (on every push)
```yaml
# .github/workflows/ci.yml
# - Lint Go code (golangci-lint)
# - Run Go tests (go test ./...)
# - Build Go binary
# - Lint TypeScript (eslint)
# - Build Next.js (npm run build)
# - Run frontend tests
```

### 4.2 CD Pipeline (on push to main)
```yaml
# .github/workflows/cd.yml
# - Build Docker images
# - Push to ghcr.io
# - SSH to VPS
# - Pull new images
# - docker-compose up -d
# - Run migrations
# - Health check
```

---

## 5. Database Migrations

```bash
# Apply all pending migrations
migrate -path ./backend/migrations -database "$DATABASE_URL" up

# Rollback one migration
migrate -path ./backend/migrations -database "$DATABASE_URL" down 1

# Check current version
migrate -path ./backend/migrations -database "$DATABASE_URL" version

# Force version (if dirty state)
migrate -path ./backend/migrations -database "$DATABASE_URL" force VERSION
```

---

## 6. Health Monitoring

```bash
# Check service health
curl https://app.upendo.ac.ke/api/v1/health/ready

# View logs
docker-compose logs -f backend
docker-compose logs -f postgres

# Prometheus metrics
curl http://localhost:8080/metrics
```
