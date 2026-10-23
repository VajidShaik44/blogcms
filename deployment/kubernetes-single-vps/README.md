# Single-VPS K3s deployment

This production overlay targets the existing Ubuntu VPS. It keeps host Nginx and Certbot on ports 80/443, uses the existing MySQL database and uploads directory, and binds the Kubernetes frontend/API ports to loopback only. It does not install K3s or change the Nginx configuration automatically.

The overlay pins both images to the main-branch commit `d815419b897b354ca442bfc351a06d7edf52520f`, whose tests, builds, and Trivy scans passed before GHCR publication.

## Preconditions

- A current production database dump is copied off the VPS and successfully restored into an isolated MySQL instance.
- The GHCR images for the pinned commit are available to the cluster. If either package is private, use a token restricted to `read:packages`.
- The VPS private address is `172.31.24.111`; the Kubernetes pod CIDR is `10.42.0.0/16`.
- AWS security-group ingress does not expose ports 3306 or 6443. Do not open database or Kubernetes API ports publicly.
- The existing uploads directory is `/home/ubuntu/app-1/blogcms/backend/storage/uploads`, owned by `ubuntu` (UID/GID 1000). The API image runs as UID 1001; the pod uses GID 1000 to share upload access with the existing systemd service.
- Confirm the uploads directory uses less than 5 GiB before binding it to the 5 GiB claim: `du -sh /home/ubuntu/app-1/blogcms/backend/storage/uploads`.
- Allow the shared group to write uploads, while keeping `ubuntu` as owner: `sudo chmod -R g+rwX /home/ubuntu/app-1/blogcms/backend/storage/uploads`.
- K3s uses its default `10.42.0.0/16` pod CIDR. If that is changed, update both the MySQL account host restriction and the cluster configuration.
- This release adds a MySQL FULLTEXT index to `posts`; the API runs GORM `AutoMigrate` at startup. Test the migration against the restored backup before starting the API pod against production, since index creation may take locks.

## 1. Restrict MySQL to the local app and private node address

The current MySQL listener is `*:3306`. Keep local API access and allow the pod network to reach the private node address, but do not bind MySQL to a public interface:

```ini
# In /etc/mysql/mysql.conf.d/mysqld.cnf, under [mysqld]
bind-address = 127.0.0.1,172.31.24.111
```

Back up the config before editing. MySQL 8.4 supports multiple comma-separated bind addresses. Restart MySQL, then verify that it listens on both addresses:

```sh
sudo cp -a /etc/mysql/mysql.conf.d/mysqld.cnf \
  "/etc/mysql/mysql.conf.d/mysqld.cnf.before-k3s.$(date +%F-%H%M%S)"
sudo systemctl restart mysql
sudo systemctl --no-pager --full status mysql
sudo ss -lntp | grep ':3306'
```

Do not continue if MySQL fails to start or the output shows `*:3306`. Restore the backup and restart MySQL if necessary. The existing systemd API continues to connect over loopback. Do not enable UFW without first allowing SSH from your current address and preserving the existing web access rules.

Create a separate database account for pods, limited to either the K3s pod subnet or this node's private address, and grant access only to the application database. Generate one password and use it for both possible MySQL client source addresses. Keep it in a protected shell variable; do not enable shell tracing (`set -x`) or paste the value into chat:

```sh
set -e
K8S_DB_PASSWORD="$(openssl rand -hex 32)"
sudo mysql <<SQL
CREATE USER 'blogcms_k8s'@'10.42.%' IDENTIFIED BY '${K8S_DB_PASSWORD}';
GRANT ALL PRIVILEGES ON blogcms.* TO 'blogcms_k8s'@'10.42.%';
CREATE USER 'blogcms_k8s'@'172.31.24.111' IDENTIFIED BY '${K8S_DB_PASSWORD}';
GRANT ALL PRIVILEGES ON blogcms.* TO 'blogcms_k8s'@'172.31.24.111';
SQL
```

The first host entry allows the pod IP if it reaches MySQL without source NAT; the second allows the node's private IP if K3s masquerades pod traffic. Keep the app's existing database and user unchanged for rollback.

## 2. Install single-node K3s without an ingress controller

Keep host Nginx as the only public web server. K3s's Traefik and ServiceLB are disabled; the workloads use loopback-only host ports 18080 and 18081.

```sh
curl -fsSL https://get.k3s.io -o /tmp/install-k3s.sh
sudo sh /tmp/install-k3s.sh server \
  --disable=traefik \
  --disable=servicelb \
  --write-kubeconfig-mode=600
rm -f /tmp/install-k3s.sh
sudo systemctl is-active k3s
sudo k3s kubectl get nodes -o wide
```

Use `sudo k3s kubectl` for the commands below so the admin kubeconfig remains root-only. K3s includes containerd; a separate Docker installation is not required.

## 3. Create namespace and credentials

Create the namespace first:

```sh
sudo k3s kubectl create namespace blogcms
```

If the GHCR packages are private, create a `ghcr-pull-secret`. This example reads the token without echoing it, writes a temporary mode-600 Docker config, and removes the file after the Kubernetes Secret is applied:

```sh
umask 077
read -rp 'GitHub username: ' GHCR_USER
read -rsp 'GitHub token with read:packages: ' GHCR_TOKEN; printf '\n'
DOCKER_CONFIG_FILE="$(mktemp)"
GHCR_AUTH="$(printf '%s:%s' "$GHCR_USER" "$GHCR_TOKEN" | base64 -w0)"
printf '{"auths":{"ghcr.io":{"auth":"%s"}}}\n' "$GHCR_AUTH" > "$DOCKER_CONFIG_FILE"
sudo k3s kubectl -n blogcms create secret generic ghcr-pull-secret \
  --type=kubernetes.io/dockerconfigjson \
  --from-file=.dockerconfigjson="$DOCKER_CONFIG_FILE" \
  --dry-run=client -o yaml | sudo k3s kubectl apply -f -
rm -f "$DOCKER_CONFIG_FILE"
unset GHCR_USER GHCR_TOKEN GHCR_AUTH
```

Use a token restricted to `read:packages`; do not put it in shell history or commit it. GHCR packages are private by default. If both packages are public, remove the `imagePullSecrets` entries from the two base Deployments before applying the overlay.

Create the ConfigMap with the VPS private address and production public URL:

```sh
sudo k3s kubectl -n blogcms create configmap blogcms-config \
  --from-literal=DB_HOST=172.31.24.111 \
  --from-literal=DB_PORT=3306 \
  --from-literal=BASE_URL=https://cmsapp.duckdns.org \
  --dry-run=client -o yaml | sudo k3s kubectl apply -f -
```

Create `blogcms-secrets` with `DB_USER=blogcms_k8s`, the generated DB password, `DB_NAME=blogcms`, and the **same effective `JWT_SECRET` as the current systemd service**. The Go app loads `backend/.env` from its working directory before falling back to its built-in JWT secret; preserve the current value to avoid invalidating existing login tokens. If the current service uses the built-in fallback, use that same value for this first cutover and schedule secret rotation separately.

Use a protected temporary env file or your secret manager to create the secret; never put literal values in a checked-in file or paste them into this guide. For a temporary env file, run `umask 077`, use `mktemp`, and enter the current `JWT_SECRET` privately with `read -rs` (from `backend/.env`, or the app's built-in fallback if none is set):

```sh
umask 077
SECRET_FILE="$(mktemp)"
read -rsp 'Current JWT_SECRET (input is hidden): ' JWT_SECRET; printf '\n'
printf 'DB_USER=blogcms_k8s\nDB_PASS=%s\nDB_NAME=blogcms\nJWT_SECRET=%s\n' \
  "$K8S_DB_PASSWORD" "$JWT_SECRET" > "$SECRET_FILE"
sudo k3s kubectl -n blogcms create secret generic blogcms-secrets \
  --from-env-file="$SECRET_FILE" --dry-run=client -o yaml \
  | sudo k3s kubectl apply -f -
rm -f "$SECRET_FILE"
unset K8S_DB_PASSWORD JWT_SECRET
```

If GHCR packages are private, create `ghcr-pull-secret` using a mode-600 temporary Docker config JSON file and remove it immediately after applying the Secret. Never put a token or password directly in a command or shell history.

## 4. Bring up and test the workloads without changing Nginx

From the repository root, apply the production overlay:

```sh
sudo k3s kubectl apply -k deployment/kubernetes-single-vps
sudo k3s kubectl -n blogcms rollout status deployment/blogcms-api --timeout=180s
sudo k3s kubectl -n blogcms rollout status deployment/blogcms-frontend --timeout=180s
sudo k3s kubectl -n blogcms get pods,services,pvc
```

Both services must be ready before proceeding. Check that the API is ready against MySQL, the frontend responds, and the host ports are loopback-only:

```sh
curl -fsS -o /dev/null -w 'Kubernetes frontend: %{http_code}\n' http://127.0.0.1:18080/
curl -fsS -o /dev/null -w 'Kubernetes API: %{http_code}\n' http://127.0.0.1:18081/readyz
sudo k3s kubectl -n blogcms logs deployment/blogcms-api --tail=100
sudo ss -lntp | grep -E ':(18080|18081)([[:space:]]|$)' || true
```

Do not change Nginx if either rollout, database readiness, or local curl check fails. Inspect `kubectl describe pod` and logs; the existing systemd/Nginx site remains live.

## 5. Switch Nginx only after local validation

Back up `/etc/nginx/sites-available/blogcms` first:

```sh
NGINX_BACKUP="/etc/nginx/sites-available/blogcms.before-k3s.$(date +%F-%H%M%S)"
sudo cp -a /etc/nginx/sites-available/blogcms "$NGINX_BACKUP"
printf 'Nginx backup: %s\n' "$NGINX_BACKUP"
```

In the HTTPS server block, change the existing `/api/` and health-check upstreams from `127.0.0.1:8080` to `127.0.0.1:18081`. Replace the static frontend `location /` with:

```nginx
location / {
    proxy_pass http://127.0.0.1:18080;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

Keep the current TLS, HTTP-to-HTTPS redirect, request-size limit, security headers, and ACME challenge location. Validate and reload (do not stop Nginx):

```sh
sudo nginx -t && sudo systemctl reload nginx
curl -fsS -o /dev/null -w 'Public frontend: %{http_code}\n' https://cmsapp.duckdns.org/
curl -fsS -o /dev/null -w 'Public API: %{http_code}\n' https://cmsapp.duckdns.org/healthz
```

After a successful cutover and verification of login, API reads/writes, and existing upload URLs, stop the old API service:

```sh
sudo systemctl stop blogcms-api
```

Keep it enabled and keep the old binary/config for rollback.

## Rollback

Restore the saved Nginx configuration and reload it. Start the old API service again:

```sh
sudo cp -a "$NGINX_BACKUP" /etc/nginx/sites-available/blogcms
sudo systemctl start blogcms-api
sudo nginx -t && sudo systemctl reload nginx
```

The same database is used by both versions. If a migration made by the container is incompatible with the old binary, restore the verified database backup only during an approved recovery; do not drop or recreate the production database.
