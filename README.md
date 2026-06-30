# qeet-notify

**Qeet Notify** is the multi-channel transactional notification platform for the Qeet Group and for external multi-tenant SaaS companies. A single API orchestrates Email, SMS, WhatsApp, Mobile Push, Web Push, and In-App notifications — with India-first compliance, per-tenant configuration, and deep integration with Qeet ID, qeet-logs, and Qeetrix.

> Status: **PRD — pre-development.** Phase 1 development targets Q3–Q4 2026 alongside 5 design partners.

---

## Why Qeet Notify exists

Every Qeet Group product needs notification infrastructure. Without a shared layer, each product independently wires email, SMS, WhatsApp, and push — duplicating TRAI DLT registration, bounce handling, preference centers, and suppression lists across products. Qeet Notify is that shared layer, also available to external engineering teams who face the same problem.

No existing platform provides multi-channel orchestration + India-first compliance + multi-tenant-native data model in a single product. Resend and Postmark are email-only. Knock and Novu are US/EU-centric. Braze and Customer.io are marketing-automation platforms, not transactional infrastructure. Qeet Notify is built for the gap.

---

## Channels

| Channel | Phase 1 | Phase 2 | Phase 3 |
|---|---|---|---|
| Email | ✓ | ✓ | ✓ |
| SMS — India (TRAI DLT) | ✓ | ✓ | ✓ |
| WhatsApp Business API | ✓ | ✓ | ✓ |
| In-App (SSE + React inbox components) | ✓ | ✓ | ✓ |
| Outbound Webhooks | ✓ | ✓ | ✓ |
| Mobile Push (FCM + APNs) | — | ✓ | ✓ |
| Web Push (VAPID) | — | ✓ | ✓ |
| RCS — India | — | ✓ | ✓ |
| SMS — International (Twilio / Plivo) | — | ✓ | ✓ |
| Slack / Teams / LINE | — | — | ✓ |

---

## Key differentiators

### India-native compliance
TRAI DLT managed registration (Principal Entity, Sender ID, template registration per telco), DPDP Act 2023 consent management, NDNC/DND registry scrubbing, promotional SMS timing enforcement, RCS as first-class channel — none of these are bolt-ons.

### Multi-tenant by construction
`tenant_id` is a required, indexed field on every record. Every query, workflow, template, suppression list, preference record, and analytics aggregation is tenant-scoped at the database layer (PostgreSQL row-level security), not via application code. Cross-tenant data access is architecturally impossible.

### Qeet ecosystem integration
- **Qeet ID:** Subscriber identity federation (no separate subscriber creation for Qeet products), OIDC SSO for the dashboard, auth event stream as notification triggers, RBAC via Qeet ID roles.
- **qeet-logs:** Every notification lifecycle event streamed as a structured, immutable audit record. Tamper-evident notification history for DPDP/GDPR audits.
- **Qeetrix:** Dashboard UI built on `@qeetrix/ui`. In-App notification inbox React components (`@qeet-notify/react`) built on Qeetrix primitives — automatically theme-aligned with host Qeet products.

### Per-workflow-run pricing
One event trigger = one workflow run. A run that fans out to email + SMS + WhatsApp costs one unit, not three. Makes multi-channel affordable at startup scale.

---

## How it works

```
Application fires event
     ↓
POST /v1/events  { tenantId, event, subscriberId, payload }
     ↓
Workflow Engine resolves: which channels? which template? any delays?
     ↓
Channel Workers deliver:  Email → SES/Resend/Postmark
                          SMS   → MSG91 / 2Factor / Plivo
                          WhatsApp → Meta Cloud API
                          Push  → FCM / APNs / VAPID
                          In-App → SSE stream → React inbox
     ↓
Lifecycle events streamed to dashboard + qeet-logs audit trail
```

---

## Competitive positioning

| | Resend | Novu | Knock | SuprSend | **Qeet Notify** |
|---|---|---|---|---|---|
| Multi-channel | ✗ (email only) | ✓ | Partial | ✓ | ✓ |
| India TRAI DLT managed | ✗ | ✗ | ✗ | Partial | ✓ |
| DPDP Act compliance tooling | ✗ | ✗ | ✗ | ✗ | ✓ |
| WhatsApp first-class | ✗ | ✓ | ✗ | ✓ | ✓ |
| RCS (India) | ✗ | ✗ | ✗ | ✗ | ✓ (Phase 2) |
| Multi-tenant native | ✗ | Partial | ✓ | ✓ | ✓ |
| Per-event-run pricing | ✗ | ✓ | ✗ | ✗ | ✓ |
| Qeet ID subscriber federation | — | — | — | — | ✓ |
| Self-hostable | ✗ | ✓ | ✗ | ✗ | ✓ (Phase 3) |

---

## Pricing summary

| Tier | Price | Workflow Runs | Channels |
|---|---|---|---|
| Free | $0 | 10,000/mo | Email + In-App |
| Starter | $29/mo | 100,000/mo | + SMS + WhatsApp |
| Growth | $99/mo | 500,000/mo | + Push + Webhooks |
| Scale | $349/mo | 2,500,000/mo | All channels |
| Enterprise | Custom | Custom | All + white-label + SLA |

SMS, WhatsApp, and RCS channel costs are passed through at provider cost + 10% platform fee (transparent billing). India data residency: included from Growth tier. Managed TRAI DLT registration: $149 one-time setup fee.

---

## Market

- CPaaS market: $26.9B in 2025, projected $108B by 2034 (18.8% CAGR).
- A2P Messaging (SMS + WhatsApp + RCS): $78.7B in 2025.
- Transactional Email: ~$1.1B in 2024, projected ~$2.75B by 2033 (12% CAGR).
- India: 500M+ WhatsApp users (highest global penetration), 100M+ RCS users projected by 2026, DPDP Act compliance deadline May 2027 driving demand for compliant notification infrastructure.

---

## Technical stack

| Layer | Technology |
|---|---|
| Backend API | Go 1.25 + chi v5 |
| Database | PostgreSQL 17 + pgx v5 (RLS for tenant isolation) |
| Message bus | NATS JetStream (durable at-least-once delivery) |
| Cache | Redis 7 (rate limiting, DND cache, delay scheduling) |
| Frontend (dashboard) | Next.js 16 + React 19 + Tailwind v4 |
| UI components | @qeetrix/ui |
| In-App SDK | @qeet-notify/react (React 19, SSE-based) |
| Auth | Qeet ID OIDC |
| Observability | qeet-logs (OTLP structured logs) + Prometheus |
| Migrations | golang-migrate (immutable SQL files) |
| Container | Docker + Kubernetes (Helm chart) |

---

## Roadmap

### Phase 1 — Foundation (Q3–Q4 2026)
Internal Qeet Group adoption complete; first 20 external paying customers.
- Core API: events, subscribers, templates, workflows (trigger + channel step + delay + condition branch).
- Channels: Email, SMS (India, TRAI DLT), WhatsApp, In-App, Outbound Webhooks.
- Managed TRAI DLT onboarding wizard.
- Preference center (hosted + embeddable widget).
- SDKs: Go, TypeScript. CLI: `qn`.
- Qeet ID integration (subscriber sync, OIDC auth, auth event stream).
- qeet-logs notification audit trail.
- DPDP subscriber erasure API. India data residency (ap-south-1).

### Phase 2 — Breadth and India-Native (Q1–Q2 2027)
150 paying customers; push, RCS, international SMS; full visual workflow editor.
- Mobile Push (FCM + APNs), Web Push (VAPID), React Native SDK, Flutter SDK.
- RCS — India (Airtel + Jio providers, SMS fallback routing).
- SMS — International (Twilio + Plivo, TCPA opt-out compliance).
- Visual drag-and-drop workflow editor; YAML git-deployable workflows.
- Digest step, channel fallback routing, fetch enrichment step.
- Visual WYSIWYG email editor. Full i18n (10 Indian languages).
- MCP server (AI coding agent integration).
- DPDP full consent management + GDPR erasure. Data warehouse connectors.
- Python + Ruby SDKs. Self-serve Stripe billing.

### Phase 3 — Enterprise and AI (Q3–Q4 2027)
$250K MRR; enterprise customers; SOC 2 Type II.
- AI send-time optimization, channel recommendation, subject line A/B.
- Natural-language workflow builder (Claude API integration).
- SAML 2.0, SCIM 2.0, HIPAA BAA, dedicated cluster, custom SLA.
- White-label tenant portal. Self-hosted Helm chart.
- Multi-region (Mumbai + Dublin + Virginia). SOC 2 Type II.
- iOS Live Activity, Slack/Teams/LINE channels.

---

## Repository structure (planned)

```
qeet-notify/
├── backend/                 # Go backend (API, workflow engine, channel workers)
│   ├── cmd/server/          # Entrypoint
│   ├── internal/
│   │   ├── api/             # HTTP handlers (chi v5)
│   │   ├── workflow/        # Workflow engine and step executors
│   │   ├── channels/        # Per-channel workers (email, sms, whatsapp, push, inapp)
│   │   ├── subscriber/      # Subscriber service
│   │   ├── template/        # Template service and rendering
│   │   ├── preference/      # Preference and suppression service
│   │   ├── analytics/       # Event aggregation and reporting
│   │   ├── india/           # TRAI DLT, DND, DPDP compliance layer
│   │   └── config/          # envconfig-driven configuration
│   └── migrations/          # golang-migrate SQL files (never edit applied)
├── frontend/                # Next.js 16 dashboard (pnpm workspace)
│   └── apps/
│       └── qn-dashboard/    # Notification management dashboard
├── sdk/
│   ├── go/                  # Go SDK (github.com/qeet-group/qeet-notify-go)
│   ├── typescript/          # TypeScript/Node.js SDK (@qeet-notify/node)
│   └── react/               # React in-app components (@qeet-notify/react)
├── cli/                     # `qn` CLI (Go)
├── mcp/                     # MCP server (Phase 2)
├── reqs/                    # Requirements and architecture documents
│   └── Product_Requirement_Document.md
├── Makefile                 # make help, make dev, make test, make migrate-up
└── README.md
```

---

## Design partners

Qeet Notify is in the design-partner phase. If you are an engineering lead or CTO at a multi-tenant SaaS company — particularly one with India users, WhatsApp notification requirements, or TRAI DLT compliance obligations — reach out at [partnerships@qeet.in](mailto:partnerships@qeet.in).

Design partners get:
- Direct access to the founding team for requirements and feedback.
- Managed TRAI DLT registration support (end-to-end).
- White-glove onboarding and migration from existing vendors.
- Locked pricing for 2 years.

---

## Full product documentation

See [Product_Requirement_Document.md](reqs/Product_Requirement_Document.md) for the complete PRD — competitive analysis, feature specifications, technical architecture, API design, pricing model, roadmap, and compliance details.

---

## Workspace context

`qeet-notify` is one of several independent projects in the [QG workspace](../CLAUDE.md). Each project has its own `.git` and toolchain. See the workspace-level `CLAUDE.md` for the full product family.

| Project | What it is |
|---|---|
| [qeet-id/](../qeet-id/) | Identity platform — the auth layer Qeet Notify uses for subscriber identity |
| [qeet-logs/](../qeet-logs/) | Log management — stores Qeet Notify's notification audit trail |
| [qeet-people/](../qeet-people/) | HCM platform — primary internal consumer of Qeet Notify |
| [qeet-in/](../qeet-in/) | Marketing site — qeet-notify will be listed when launched |
| [qeetrix/](../qeetrix/) | Design system — `@qeetrix/ui` used in the dashboard and SDKs |
