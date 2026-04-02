# Reverse Proxy Templates

This page contains practical templates for running Aeterna behind Nginx, Traefik, or Caddy.

All examples assume:

- `frontend` is reachable at `127.0.0.1:5173`
- `backend` is reachable at `127.0.0.1:3000`
- public URL is `https://www.server.com`

## Required .env values

Set your backend origin allowlist to exactly match browser origins:

```env
ALLOWED_ORIGINS=https://www.server.com
BASE_URL=https://www.server.com
```

If you also serve `https://server.com`, include both:

```env
ALLOWED_ORIGINS=https://www.server.com,https://server.com
BASE_URL=https://www.server.com
```

## Nginx Template (Before Certbot)

File: `/etc/nginx/sites-available/aeterna.conf`

This template assumes Aeterna is exposed by Docker on `127.0.0.1:5000`.

```nginx
server {
    listen 80;
    server_name www.server.com server.com;

    client_max_body_size 12m;

    location / {
    proxy_pass http://127.0.0.1:5000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Apply and test:

```bash
sudo nginx -t && sudo systemctl reload nginx
```

Then obtain certificates:

```bash
sudo certbot --nginx -d server.com -d www.server.com
```

After certbot succeeds, it will create/manage the HTTPS server block automatically.


## Traefik Template (Docker labels)

Add labels to your `frontend` and `backend` services and run Traefik with Docker provider enabled.

```yaml
services:
  traefik:
    image: traefik:v3.1
    command:
      - --providers.docker=true
      - --providers.docker.exposedbydefault=false
      - --entrypoints.web.address=:80
      - --entrypoints.websecure.address=:443
      - --certificatesresolvers.le.acme.tlschallenge=true
      - --certificatesresolvers.le.acme.email=admin@server.com
      - --certificatesresolvers.le.acme.storage=/letsencrypt/acme.json
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./data/letsencrypt:/letsencrypt
    restart: unless-stopped

  frontend:
    image: ghcr.io/alpyxn/aeterna-frontend:main
    labels:
      - traefik.enable=true
      - traefik.http.routers.aeterna-web.rule=Host(`www.server.com`)
      - traefik.http.routers.aeterna-web.entrypoints=websecure
      - traefik.http.routers.aeterna-web.tls.certresolver=le
      - traefik.http.services.aeterna-web.loadbalancer.server.port=80
    restart: unless-stopped

  backend:
    image: ghcr.io/alpyxn/aeterna-backend:main
    labels:
      - traefik.enable=true
      - traefik.http.routers.aeterna-api.rule=Host(`www.server.com`) && PathPrefix(`/api`)
      - traefik.http.routers.aeterna-api.entrypoints=websecure
      - traefik.http.routers.aeterna-api.tls.certresolver=le
      - traefik.http.services.aeterna-api.loadbalancer.server.port=3000
    restart: unless-stopped
```

## Caddy Template

File: `/etc/caddy/Caddyfile`

```caddy
www.server.com {
    encode zstd gzip

    @api path /api*
    reverse_proxy @api 127.0.0.1:3000
    reverse_proxy 127.0.0.1:5173
}
```

Caddy will automatically provision TLS certificates when DNS is correct.

## Important Notes

- `localhost` is local-only and is not your public server IP.
- If users open `https://www.server.com`, `ALLOWED_ORIGINS` must include that exact origin.
- If you expose a non-standard port publicly (for example `:5000`), include that port in `ALLOWED_ORIGINS` and `BASE_URL`.
