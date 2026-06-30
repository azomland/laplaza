# 🌳 Plaza

**Create your own public square.**

No feeds. No algorithms. Just conversations.

Open-source software for creating small public squares on the Internet.
Built by [PersonnnOS](https://personnn.com).

![Plaza](https://plaza.personnn.com/og.png)

---

## Core Principles

- **Conversation over content.** No feeds, no posts, no archives.
- **Presence over identity.** No profiles, no followers, no usernames required.
- **Ephemeral by default.** When everyone leaves, the banca disappears.
- **Open by design.** MIT license. Self-host. Modify. Share.
- **Simple enough to self-host.** One binary. One command. Your own plaza.

## Philosophy

Plaza doesn't try to replace social networks. It tries to recover something
the Internet lost: small places, ephemeral conversations, human communities,
and the possibility of creating your own space without asking permission.

### The Park Rule

Every time we add a feature, we ask: *"Would this exist in a real plaza?"*

✅ A bench where people sit  
✅ Someone proposes a topic  
✅ People talking and then leaving  
✅ A plaque showing who donated the bench  
❌ An algorithm deciding who you talk to  
❌ A follower counter  
❌ An infinite feed  
❌ Notifications to bring you back  

---

## Quick Start

### One-command install (Linux server)

```bash
curl -fsSL https://getplaza.personnn.com/install.sh | bash
```

The script will ask for a domain and plaza name, then install everything
(Go, Node.js, nginx, certbot) and configure your plaza.

### Manual setup (local dev)

```bash
git clone https://github.com/azomland/personnn-laplaza.git
cd personnn-laplaza

# Backend
cd backend && go build -o plaza . && cd ..

# Frontend
cd frontend && npm install && npm run build && cd ..

# Run
./backend/plaza -config plaza.toml
```

Open http://localhost:8080

### Development mode (hot-reload)

```bash
# Terminal 1 — Go backend
cd backend && go run main.go -config ../plaza.toml

# Terminal 2 — Astro dev server
cd frontend && npm run dev
```

Open http://localhost:4321 — the dev server proxies `/api` and `/ws` to Go.

---

## Architecture

```
plaza/
├── frontend/              # Astro (static site)
├── backend/               # Go (HTTP + WebSocket server)
│   ├── main.go
│   ├── config/            # plaza.toml loader
│   ├── models/            # Plaza, Banca, Message, Store
│   ├── handlers/          # REST + WebSocket handlers
│   └── middleware/         # Security, rate limit, logging
├── install.sh             # Production installer
├── plaza.toml             # Configuration
├── LICENSE                # MIT
└── README.md
```

### Stack

| Layer | Technology |
|-------|-----------|
| Frontend | Astro (static) |
| Backend | Go 1.22+ |
| Real-time | WebSocket |
| Persistence | RAM (ephemeral by default) |
| Config | TOML |

---

## API

### `GET /api/config`

```json
{ "title": "Mi Plaza", "domain": "localhost",
  "allow_anonymous": true, "max_users_per_banca": 33 }
```

### `GET /api/bancas`

```json
[{ "id": "ab3f9c12", "title": "Hablemos de Go",
   "users": 3, "max_users": 33, "active": true }]
```

### `POST /api/bancas`

Create a banca: `{ "title": "Nueva conversación" }` → `201 Created`

### `GET /api/bancas/{id}`

Get a single banca by its 8-character hex ID.

### `GET /ws/{id}`

WebSocket. Messages are JSON:

| Type | Direction | Description |
|------|-----------|-------------|
| `message` | → | `{ "content": "..." }` |
| `set_username` | → | `{ "username": "Nico" }` |
| `typing` | → | `{}` |
| `message` | ← | Broadcast to all clients |
| `typing` | ← | `{ "username": "Nico" }` |
| `history` | ← | Recent messages on connect |
| `welcome` | ← | `{ "client_id": "...", "banca_id": "..." }` |

---

## Configuration

```toml
title = "Mi Plaza"
domain = "plaza.midominio.com"
port = 8080
max_users_per_bench = 33
allow_anonymous = true
history = false
data_dir = "./data"
```

| Key | Default | Description |
|-----|---------|-------------|
| `title` | `"Mi Plaza"` | Name shown in the nav |
| `domain` | `"localhost"` | Used for nginx/SSL/origin check |
| `port` | `8080` | HTTP port |
| `max_users_per_bench` | `33` | Max people per banca |
| `allow_anonymous` | `true` | Allow unnamed users |
| `history` | `false` | Persist messages to disk |
| `data_dir` | `"./data"` | Directory for storage |

---

## Security

- **XSS**: HTML stripped from titles, entities escaped in messages
- **CSRF**: WebSocket origin validated against configured domain
- **Rate limiting**: 120 requests/min per IP (configurable)
- **Connection limits**: Max 5 WebSocket connections per IP
- **Panic recovery**: Server never crashes from a panic
- **WS ping/pong**: Zombie connections cleaned up automatically
- **IDs**: Cryptographically random, 8-char hex (no sequential IDs)

---

## Tests

```bash
cd backend && go test -v ./...
```

53 tests covering config, models, HTTP handlers, sanitization, rate limiting,
security headers, and middleware.

---

## Deploy

See `install.sh` for production setup. It installs Go, Node.js, nginx,
certbot, and configures everything with a `systemd` service.

```bash
# Manual deploy with systemd
go build -o /usr/local/bin/plaza .
cp plaza.toml /etc/plaza.toml

cat > /etc/systemd/system/plaza.service <<EOF
[Unit]
Description=Plaza
After=network.target

[Service]
ExecStart=/usr/local/bin/plaza -config /etc/plaza.toml
Restart=always
User=plaza

[Install]
WantedBy=multi-user.target
EOF

systemctl enable --now plaza
```

---

## License

MIT © 2026 PersonnnOS
