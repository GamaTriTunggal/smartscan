# Contributing to smartscan

Thanks for your interest in improving smartscan. This project is intentionally
focused: a self-hosted, single-company, **dynamic-QR-only** anti-counterfeit
tool. Contributions that keep that scope sharp are very welcome.

## Development setup

You need Docker + Docker Compose. For working outside containers you also need
Go 1.25+ and Node 20+.

```bash
cp .env.example .env          # set JWT_SECRET, DB_PASSWORD at minimum
docker compose up -d          # postgres, redis, backend, frontend
```

The backend auto-runs database migrations on boot. Open http://localhost:3000
and complete the setup wizard.

### Running tests

```bash
# Backend (unit tests need no DB; integration/handler tests skip when TEST_DATABASE_URL is unset)
cd backend && go test ./...

# Frontend
cd frontend && npm install && npm run test -- --run && npm run build
```

## Ground rules

- **Keep it single-company and dynamic-only.** Please don't reintroduce
  multi-tenant SaaS features, subscription/billing logic, static QR, or an
  email dependency — those were deliberately removed. Multi-brand support (one
  company, several brands) is a reasonable future direction; SaaS resale is not.
- **No external service becomes mandatory.** Redis, object storage, reverse
  geocoding, and the outbound webhook are all optional and must degrade
  gracefully when unconfigured. `docker compose up` must work with zero extra
  setup.
- **Match the surrounding code.** Go handlers follow the existing
  handler/model layout; Vue components use the existing `ui/` kit and Tailwind
  conventions. Run `go vet ./...` and `npm run build` before opening a PR.
- **Don't commit secrets, PII, or generated artifacts** (`dist/`, binaries,
  `.env`). The `.gitignore` covers the common cases.

## Pull requests

1. Branch from `main`.
2. Keep changes scoped; describe the *why*, not just the *what*.
3. Ensure backend build+vet+tests and frontend build+tests pass.
4. Update docs/`.env.example` if you add configuration.

## Reporting bugs

Open an issue with steps to reproduce, expected vs actual behavior, and your
environment (OS, Docker version). For security-sensitive reports, see
[SECURITY.md](SECURITY.md) instead of filing a public issue.
