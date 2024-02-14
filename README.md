# gitops-drift-detector

Continuously compares live Kubernetes cluster state against Git-defined desired state. Alerts on drift. One-click reconcile back to Git truth.

## What it does

Polls your Git repos for Kubernetes manifests and diffs them against what's actually running in the cluster. Anything that's drifted (modified, missing, or unexpected) shows up in the UI with severity, a visual diff, and a reconcile button that applies the Git state back.

Supports multiple clusters. Webhook endpoint for push-triggered detection instead of polling.

## Stack:

- Backend: Go + Gin, client-go SDK, go-git
- Frontend: React + TypeScript + Vite + Tailwind + Recharts
- Database: PostgreSQL (audit log, drift history)

## Running locally

```bash
# start postgres
docker-compose up postgres -d

# backend (from backend/)
go run ./cmd/server

# frontend (from frontend/)
npm install
npm run dev
```

Or run the whole thing:

```bash
docker-compose up --build
```

UI at http://localhost:5173, API at http://localhost:8080.

## Environment variables

| Variable | Default | Notes |
|---|---|---|
| `PORT` | `8080` | Backend listen port |
| `DATABASE_URL` | `postgres://drift:drift@localhost:5432/drift?sslmode=disable` | |
| `GIT_POLL_INTERVAL` | `30s` | How often to re-clone repos |
| `ALERT_WEBHOOK_URL` | `` | Slack-compatible webhook for critical drift alerts |

## Deploying to Kubernetes

```bash
kubectl create namespace gitops-system
kubectl apply -f k8s/
```

You'll need to create a `drift-detector-secrets` secret with a `database-url` key before deploying.

## Git webhook setup

Point GitHub/GitLab push webhooks at `POST /webhooks/git`. No secret validation yet — TODO before exposing this externally.

## Architecture notes

The drift engine runs on a 30s tick independent of the git poller. Git polling and cluster state fetches are both async. The API serves the last computed snapshot synchronously, so the UI never blocks on a live cluster call.

Reconcile uses server-side apply (`fieldManager: gitops-drift-detector`) so it plays nicely with other controllers managing the same resources.
