# Security Policy

## Reporting a vulnerability

Please do **not** open a public issue for security vulnerabilities. Instead,
report them privately to the maintainers via GitHub's private vulnerability
reporting (Security → Report a vulnerability on the repository) or by opening a
minimal issue that says only "security report — please provide a private
contact" without details.

We aim to acknowledge reports within a few days and to ship a fix or mitigation
as promptly as the severity warrants.

## Security model & operator responsibilities

smartscan is self-hosted software. Its security in production depends on how you
deploy it. The application enforces sensible defaults, but you must:

- **Set a strong `JWT_SECRET`** (≥ 32 random characters). The server refuses to
  start in production with a weak or missing secret.
- **Set a strong `DB_PASSWORD`** and enable TLS to the database (`DB_SSLMODE`)
  where possible.
- **Serve over HTTPS.** Auth tokens live in HttpOnly cookies; terminate TLS at a
  reverse proxy (nginx/Caddy) in front of the app.
- **Set `FRONTEND_URL` correctly** — it is embedded in every QR code.
- **Protect the server host.** The `smartscan-admin` recovery CLI and the
  database are accessible to anyone with shell access to the container/host;
  treat host access as full administrative access.
- **The `/metrics` endpoint is off by default.** It is only exposed when both
  `METRICS_USER` and `METRICS_PASS` are set; keep it behind your internal
  network.
- **Outbound webhooks** are signed with HMAC-SHA256 (`X-Smartscan-Signature`).
  Verify that signature on the receiving side.

## Built-in protections

- JWT access/refresh separation with issuer validation and Redis-backed
  revocation (fails open if Redis is unavailable — see deployment notes).
- Per-route rate limiting (auth, public scan, export, general).
- CSRF origin/referer validation, OWASP security headers, request size limits.
- Account lockout on repeated failed logins.
- HMAC-signed scan-redirect URLs to prevent scan-count manipulation.
- Uploaded-image content validation.
