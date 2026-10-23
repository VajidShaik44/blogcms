# Kubernetes deployment

These manifests provide a starting point for deploying the API and frontend to a Kubernetes cluster. They do not create a database, install an ingress controller, or alter the existing VPS deployment.

## Prerequisites

- A Kubernetes cluster with `kubectl` and Kustomize support.
- A MySQL database reachable from the cluster. Use a dedicated database and credentials; do not point this first test deployment at the live production database.
- The API and frontend images published to GHCR by the CI workflow.
- A default StorageClass for the 5 GiB uploads claim, or a `storageClassName` added to `uploads-pvc.yaml`.

GHCR packages are private by default. Create a read-only GitHub personal access token with `read:packages` permission, then create the pull secret without committing the token:

```sh
kubectl create namespace blogcms
kubectl create secret docker-registry ghcr-pull-secret \
  --namespace blogcms \
  --docker-server=ghcr.io \
  --docker-username=YOUR_GITHUB_USERNAME \
  --docker-password=YOUR_READ_PACKAGES_TOKEN
```

Create the application configuration and credentials in the cluster. Supply a test/staging MySQL host and values; the database name and user/password are stored as Secret data:

```sh
kubectl create configmap blogcms-config \
  --namespace blogcms \
  --from-literal=DB_HOST=YOUR_MYSQL_HOST \
  --from-literal=DB_PORT=3306 \
  --from-literal=BASE_URL=https://YOUR_TEST_HOST \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl create secret generic blogcms-secrets \
  --namespace blogcms \
  --from-literal=DB_USER=YOUR_DB_USER \
  --from-literal=DB_PASS=YOUR_DB_PASSWORD \
  --from-literal=DB_NAME=YOUR_DB_NAME \
  --from-literal=JWT_SECRET=YOUR_RANDOM_SECRET \
  --dry-run=client -o yaml | kubectl apply -f -
```

Do not reuse the production database or JWT secret for an initial cluster test. Kubernetes Secrets are not encrypted in etcd unless the cluster is configured for encryption at rest; use the cluster's approved secret manager for production.

## Install the workloads

After the CI workflow has published the `main` image tags:

```sh
kubectl apply -k deployment/kubernetes
kubectl -n blogcms rollout status deployment/blogcms-api
kubectl -n blogcms rollout status deployment/blogcms-frontend
kubectl -n blogcms get pods,services,pvc
```

The API has separate liveness (`/healthz`) and readiness (`/readyz`) probes. The readiness check includes its MySQL dependency. The API is held at one replica because uploads use a single-writer persistent volume.

The sample ingress is deliberately excluded from the Kustomize resources. To expose the app, first install/configure an NGINX ingress controller, replace `blogcms.example.com` in `ingress.example.yaml`, provision the `blogcms-tls` TLS Secret, and then apply that file:

```sh
kubectl -n blogcms create secret tls blogcms-tls \
  --cert=PATH_TO_FULLCHAIN_PEM \
  --key=PATH_TO_PRIVATE_KEY
kubectl apply -f deployment/kubernetes/ingress.example.yaml
```

The ingress routes `/api` directly to the API service and other paths to the frontend. Set DNS to the ingress controller only when intentionally testing the cluster; do not change the live site's DNS as part of this preparation step.

## Pin a verified release

The initial manifests use the `main` image tags for bootstrapping. Once the workflow publishes immutable `sha-<commit>` tags, pin the deployments to the exact commit that passed tests and scans:

```sh
kubectl -n blogcms set image deployment/blogcms-api \
  api=ghcr.io/vajidshaik44/blogcms/backend:sha-FULL_COMMIT_SHA
kubectl -n blogcms set image deployment/blogcms-frontend \
  frontend=ghcr.io/vajidshaik44/blogcms/frontend:sha-FULL_COMMIT_SHA
kubectl -n blogcms rollout status deployment/blogcms-api
kubectl -n blogcms rollout status deployment/blogcms-frontend
```

These commands are instructions only; this repository change does not apply resources to a cluster.

For a local, cluster-only smoke test without ingress:

```sh
kubectl -n blogcms port-forward service/frontend 8080:80
```

Then open `http://localhost:8080` and verify the UI and API requests. Keep the browser session pointed at the test database.
