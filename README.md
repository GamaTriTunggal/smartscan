# smartscan

**Open-source, self-hosted QR authentication for brand owners.**

smartscan lets a brand generate a unique QR code for every physical unit it produces, then verify authenticity when a consumer scans one. Because each code is *dynamic* (one code = one unit), the system can enforce a per-code scan limit, detect impossible-travel (velocity) anomalies, and flag duplicate warranty registrations — signals that only make sense when a code cannot legitimately appear on more than one product.

It is designed to be run **inside a single company** by its own IT team. There is no SaaS layer, no billing, no multi-tenant onboarding — you `docker compose up`, complete a one-time setup wizard, and it's yours.

## Why dynamic-only?

A static QR (the same code printed on every unit) fundamentally cannot do anti-counterfeit work: with a million identical codes in the wild, no scan can be told apart from any other, so scan-count thresholds, velocity checks, and per-unit geofencing are all meaningless. smartscan therefore supports **dynamic QR only** — every code is unique and individually tracked.

## Features

- **Products & batches** — define a product, generate a batch of unique QR codes (asynchronously, backed by a Redis Streams queue), and export them.
- **Print-ready output** — download codes as CSV/Excel for industrial label printers, or as a **vector QR PDF** (crisp at any size) for self-printing.
- **Public verification** — each code resolves to a customizable landing page showing an authentic / counterfeit verdict, product info, certifications, gallery, and your company contact details.
- **Counterfeit detection** — per-code scan-limit thresholds (with a QR → batch → product → company override hierarchy), impossible-travel velocity checks, consumer-submitted counterfeit reports, and duplicate-warranty-registration flagging.
- **Warranty registration** — consumers activate a warranty by scanning; you collect their details for export or stream them to your CRM in real time via webhook.
- **Geofencing** — mark a distribution zone per batch and get alerted on out-of-zone scans.
- **QC & warehouse scanning** — internal roles for quality-control and warehouse staff.
- **In-app notifications + webhook** — no email server required; alerts appear in-app and can be pushed to Slack/Discord/ntfy/your CRM via one HMAC-signed webhook.

## Tech stack

- **Backend** — Go 1.25 · Gin · GORM · PostgreSQL 18 (uses the native `uuidv7()`) · Redis
- **Frontend** — Vue 3.5 · Vite · Pinia · Tailwind CSS
- **Auth** — JWT in HttpOnly cookies (access + refresh)

## Quick start

Requires Docker and Docker Compose.

```bash
git clone https://github.com/GamaTriTunggal/smartscan.git
cd smartscan
cp .env.example .env
# edit .env — set JWT_SECRET, DB_PASSWORD, and FRONTEND_URL
docker compose up -d
```

Then open <http://localhost:3000> and complete the first-run setup wizard to create your company and administrator account.

- **Frontend**: http://localhost:3000
- **API**: http://localhost:8080/api/v1

## Configuration

All configuration is via environment variables — see [`.env.example`](.env.example) for the full list. The essentials:

| Variable | Purpose |
|---|---|
| `JWT_SECRET` | Signing secret for auth tokens — **set a strong random value** (≥ 32 chars in production). |
| `DB_PASSWORD` | PostgreSQL password. |
| `FRONTEND_URL` | Public base URL of your deployment — **it is baked into every QR code**, so set it correctly before generating codes. |
| `BIGDATACLOUD_API_KEY` | *(optional)* enables server-side reverse geocoding for scan analytics. |
| `R2_*` | *(optional)* S3-compatible object storage for uploads; falls back to local disk. |

## Account recovery

There is no email in smartscan, so password recovery works like most internal tools:

- **A staff member forgot their password** → an admin resets it from **Staff → Reset Password**; the one-time password is shown on screen to hand over directly.
- **The last admin is locked out** → run the operator CLI on the server:
  ```bash
  docker compose exec backend ./smartscan-admin reset-password admin@yourcompany.com
  ```
  It prints a one-time password. (Anyone with server access already controls the database, so this adds no new trust boundary.)

## License

[AGPL-3.0](LICENSE). You may run, modify, and self-host smartscan freely. If you offer it to others as a network service, the AGPL requires you to make your modified source available to those users.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Security issues: see [SECURITY.md](SECURITY.md).
