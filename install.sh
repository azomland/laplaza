#!/usr/bin/env bash
set -euo pipefail

# ──────────────────────────────────────────────
# 🌳 Plaza Installer
# ──────────────────────────────────────────────

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m'

echo ""
echo -e "  ${GREEN}🌳 Plaza Installer${NC}"
echo ""

# ── 1. Domain / IP ───────────────────────────
echo -e "${BOLD}¿Cuál es el dominio de tu Plaza?${NC}"
echo "  (ej: plaza.midominio.com)"
read -r DOMAIN
DOMAIN=${DOMAIN:-localhost}

echo ""
echo -e "${BOLD}¿Cómo quieres llamar a tu Plaza?${NC}"
echo "  (ej: Plaza de José)"
read -r TITLE
TITLE=${TITLE:-Mi Plaza}

echo ""
echo -e "${BOLD}¿Puerto interno? (default: 8080)${NC}"
read -r PORT
PORT=${PORT:-8080}

echo ""
echo -e "${BOLD}¿Usas Cloudflare como DNS/proxy? (s/n)${NC}"
read -r USE_CLOUDFLARE

# ── Cloudflare check ─────────────────────────
CLOUDFLARE_STEP=false
if [[ "$USE_CLOUDFLARE" =~ ^[sSyY] ]]; then
  CLOUDFLARE_STEP=true
  echo ""
  echo -e "  ${YELLOW}╔══════════════════════════════════════════════════════╗${NC}"
  echo -e "  ${YELLOW}║  IMPORTANTE: Cloudflare Proxy (nube naranja)        ║${NC}"
  echo -e "  ${YELLOW}║                                                    ║${NC}"
  echo -e "  ${YELLOW}║  1. Ve a tu panel de Cloudflare                    ║${NC}"
  echo -e "  ${YELLOW}║  2. Busca el registro DNS de $DOMAIN        ║${NC}"
  echo -e "  ${YELLOW}║  3. Cambia el icono de la nube naranja ☁️          ║${NC}"
  echo -e "  ${YELLOW}║     a nube gris (DNS only) 💨                      ║${NC}"
  echo -e "  ${YELLOW}║                                                    ║${NC}"
  echo -e "  ${YELLOW}║  Espera 2-3 min a que propague el cambio.         ║${NC}"
  echo -e "  ${YELLOW}║  Después del SSL, podrás volver a naranja.        ║${NC}"
  echo -e "  ${YELLOW}╚══════════════════════════════════════════════════════╝${NC}"
  echo ""
  echo -e "  ${BOLD}Presiona Enter cuando hayas cambiado la nube a gris...${NC}"
  read -r
fi

# ── 2. Dependencies ──────────────────────────
echo ""
echo -e "  ${GREEN}✔${NC} Checking server..."

if ! command -v git &>/dev/null; then
  echo -e "  ${BOLD}Instalando git...${NC}"
  apt-get update -qq && apt-get install -y -qq git
fi

# Go
if command -v go &>/dev/null; then
  echo -e "  ${GREEN}✔${NC} Go $(go version | awk '{print $3}')"
else
  echo -e "  ${BOLD}Instalando Go...${NC}"
  if [[ "$(uname)" == "Linux" ]]; then
    wget -q https://go.dev/dl/go1.22.5.linux-amd64.tar.gz -O /tmp/go.tar.gz
    tar -C /usr/local -xzf /tmp/go.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
  elif [[ "$(uname)" == "Darwin" ]]; then
    if command -v brew &>/dev/null; then
      brew install go
    else
      echo "Instala Go manualmente: https://go.dev/dl/"
      exit 1
    fi
  fi
  echo -e "  ${GREEN}✔${NC} Go instalado"
fi

# Node.js
if command -v node &>/dev/null; then
  echo -e "  ${GREEN}✔${NC} Node $(node -v)"
else
  echo -e "  ${BOLD}Instalando Node.js...${NC}"
  if command -v curl &>/dev/null; then
    curl -fsSL https://deb.nodesource.com/setup_22.x | bash -
    apt-get install -y nodejs
  fi
  echo -e "  ${GREEN}✔${NC} Node.js instalado"
fi

# ── 3. Clone / Download ─────────────────────
REPO_DIR="/opt/plaza"
if [ -d "$REPO_DIR" ]; then
  echo -e "  ${GREEN}✔${NC} Plaza ya existe en $REPO_DIR, actualizando..."
  cd "$REPO_DIR" && git pull
else
  echo -e "  ${BOLD}Descargando Plaza...${NC}"
  git clone https://github.com/azomland/personnn-laplaza.git "$REPO_DIR" 2>/dev/null || {
    echo "No se pudo clonar. ¿Tienes git instalado?"
    exit 1
  }
  cd "$REPO_DIR"
fi

# ── 4. Config ────────────────────────────────
echo -e "  ${BOLD}Configurando plaza.toml...${NC}"
cat > plaza.toml <<EOF
title = "$TITLE"
domain = "$DOMAIN"
port = $PORT
max_users_per_bench = 33
allow_anonymous = true
history = false
ads = false
data_dir = "./data"
EOF

# ── 5. Build Go backend ─────────────────────
echo -e "  ${BOLD}🪑  Construyendo las bancas...${NC}"
cd backend
go mod tidy
go build -o /usr/local/bin/plaza .
cd ..

# ── 6. Build Astro frontend ─────────────────
echo -e "  ${BOLD}🌱  Plantando los árboles...${NC}"
cd frontend
npm install
npm run build
cd ..

# ── 6.5 Create plaza user ────────────────────
id -u plaza &>/dev/null || useradd -r -s /bin/false plaza
mkdir -p "$REPO_DIR/data"
chown -R plaza:plaza "$REPO_DIR"

# ── 7. Nginx ─────────────────────────────────
if command -v nginx &>/dev/null; then
  echo -e "  ${GREEN}✔${NC} Nginx detectado"
else
  echo -e "  ${BOLD}Instalando Nginx...${NC}"
  apt-get install -y -qq nginx
fi

cat > /etc/nginx/sites-available/plaza <<EOF
server {
    listen 80;
    server_name $DOMAIN;

    location / {
        proxy_pass http://127.0.0.1:$PORT;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }
}
EOF

if [ -d "/etc/nginx/sites-enabled" ]; then
  ln -sf /etc/nginx/sites-available/plaza /etc/nginx/sites-enabled/
fi
nginx -t && systemctl reload nginx 2>/dev/null || nginx -s reload 2>/dev/null || true
echo -e "  ${GREEN}✔${NC} Nginx configurado"

# ── 8. SSL / Certbot ─────────────────────────
echo ""
echo -e "  ${BOLD}¿Generar certificado SSL con Let's Encrypt? (s/n)${NC}"
read -r USE_SSL

if [[ "$USE_SSL" =~ ^[sSyY] ]]; then
  if ! command -v certbot &>/dev/null; then
    echo -e "  ${BOLD}Instalando Certbot...${NC}"
    apt-get install -y -qq certbot python3-certbot-nginx
  fi

  if $CLOUDFLARE_STEP; then
    echo ""
    echo -e "  ${YELLOW}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "  ${YELLOW}║  Certbot va a validar el dominio.                 ║${NC}"
    echo -e "  ${YELLOW}║  Asegúrate que la nube Cloudflare siga en GRIS.   ║${NC}"
    echo -e "  ${YELLOW}║  Temporaliza el proxy:                            ║${NC}"
    echo -e "  ${YELLOW}║   - DNS only (nube gris) → Certbot → SSL          ║${NC}"
    echo -e "  ${YELLOW}║   - Luego vuelve a naranja ☁️                     ║${NC}"
    echo -e "  ${YELLOW}╚══════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "  ${BOLD}Presiona Enter cuando estés listo...${NC}"
    read -r
  fi

  certbot --nginx -d "$DOMAIN" --non-interactive --agree-tos --email "admin@$DOMAIN" || {
    echo -e "  ${YELLOW}⚠️  Certbot falló. Posibles causas:${NC}"
    echo "     - Cloudflare proxy aún en naranja ☁️"
    echo "     - El dominio no apunta a este servidor"
    echo "     - Puerto 80 no accesible"
    echo ""
    echo -e "  ${BOLD}Arregla el problema y ejecuta: certbot --nginx -d $DOMAIN${NC}"
  }

  if $CLOUDFLARE_STEP; then
    echo ""
    echo -e "  ${YELLOW}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "  ${YELLOW}║  SSL generado. Ahora puedes volver a activar      ║${NC}"
    echo -e "  ${YELLOW}║  el proxy de Cloudflare (nube naranja ☁️)         ║${NC}"
    echo -e "  ${YELLOW}║  en el panel de DNS.                              ║${NC}"
    echo -e "  ${YELLOW}╚══════════════════════════════════════════════════════╝${NC}"
  fi
fi

# ── 9. Systemd ─────────────────────────────────
echo -e "  ${BOLD}☕  Preparando un café...${NC}"
cat > /etc/systemd/system/plaza.service <<EOF
[Unit]
Description=Plaza
After=network.target

[Service]
ExecStart=/usr/local/bin/plaza -config $REPO_DIR/plaza.toml
Restart=always
User=plaza
Group=plaza
WorkingDirectory=$REPO_DIR
LimitNOFILE=4096

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable plaza
systemctl restart plaza

# ── Done ──────────────────────────────────────
echo ""
echo -e "  ${GREEN}████████████ 100%${NC}"
echo ""
echo -e "  🌳 ${BOLD}Tu Plaza está viva.${NC}"
echo ""
echo -e "  Accede:  ${GREEN}https://$DOMAIN${NC}"
echo ""
echo -e "  Config:  $REPO_DIR/plaza.toml"
echo -e "  Logs:    journalctl -u plaza -f"
echo -e "  Status:  systemctl status plaza"
echo ""
echo -e "  🧹 Las bancas vacías se limpian solas."
echo -e "  🌱  Todo es efímero por diseño."
echo ""
