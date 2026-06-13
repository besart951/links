# Links

Links ist ein Produktgeruest mit getrennten Oberflaechen fuer oeffentliche Besucher, interne Betreiber und Support-Faelle. Dieser Kontext beschreibt die gemeinsame Sprache fuer diese sichtbaren Produktbereiche und ihre operativen Zustandssignale.

## Language

### Product Areas

**Landing Page**:
The public entry point for Links on the main domain.
_Avoid_: Website, homepage, marketing page

**Admin Panel**:
The internal operator-facing area for checking and later managing Links.
_Avoid_: Admin app, back office, dashboard

**Support Portal**:
The user-facing area for help, contact, and assistance flows.
_Avoid_: Help page, support app, customer service site

**Links Surface**:
A distinct user-facing or operator-facing area of Links, such as the Landing Page, Admin Panel, or Support Portal.
_Avoid_: Frontend, web app, site

### Operational State

**System Status**:
The high-level availability signal for Links from the product's perspective.
_Avoid_: Health, backend status, uptime

**Data Store Status**:
The availability signal for Links' persisted data as surfaced to operators.
_Avoid_: DB status, database health, Postgres status

## Platform Core

Links hat einen zentralen SaaS-Core im Go-Backend. Dieser Core ist die einzige Autorisierungsquelle fuer Tenants, Memberships, Produktlizenzen, Produktzuweisungen und Permissions. UI-seitige Access-Checks duerfen Funktionen verstecken, ersetzen aber nie Backend-Authorization.

### Core Concepts

**Tenant**:
Ein Kunden- oder Workspace-Kontext, in dem Mitglieder, Lizenzpools und Produktzuweisungen verwaltet werden.
_Avoid_: Account, company, organisation, workspace when the code path needs authorization semantics

**Tenant Membership**:
Die Beziehung zwischen User und Tenant, inklusive Tenant-Rolle und Status.
_Avoid_: User role without tenant context

**Product License Pool**:
Der lizenzierte Seat-Pool eines Tenants fuer ein Produkt. Im MVP ist die Quelle `manual_self_service` und Owner/Billing-Admins koennen bis zu 10 Seats pro Produkt setzen.
_Avoid_: Subscription when no payment provider is involved

**Product Assignment**:
Die konkrete Freischaltung eines Tenant-Mitglieds fuer ein Produkt, inklusive Produktrolle. Nur aktive Assignments verbrauchen Seats und erzeugen Produktzugriff.
_Avoid_: Feature flag, entitlement when a user-specific seat is meant

**Access Snapshot**:
Ein vom Backend gebauter, lesbarer Zustand fuer UI-Gating: User, aktiver Tenant, verfuegbare Tenants und Produktzugriffe. Der Snapshot ist Komfort fuer Frontends, nicht Sicherheitsquelle.
_Avoid_: Client auth truth, JWT claims source of truth

**Permission**:
Eine feingranulare Produktfaehigkeit wie `planner.task.read` oder `finance.invoice.read`. Permissions entstehen zentral aus Produktrolle und Produkt.
_Avoid_: Checking raw UI route names

## Module Boundaries

**Transport**:
Connect RPC Handler in `app/backend/internal/server`. Transport liest Cookies/Header, ruft Application Use Cases auf und mappt Application Errors auf Connect-Codes.
_Avoid_: Business decisions in handlers

**Application Service**:
Orchestrierung in `app/backend/internal/platform`. Diese Schicht koordiniert Transaktionen, Repositories/SQL-Zugriffe und Domain Policies.
_Avoid_: Permission matrices, password algorithms or seat rules inline in RPC handlers

**Entitlements Domain**:
`app/backend/internal/platform/entitlements/domain` enthaelt Produktkeys, Tenant-/Produktrollen, Permissions und die zentrale AuthorizationPolicy.
_Avoid_: Doppelte Rollen-/Permission-Maps in Frontends or handlers

**Licensing Domain**:
`app/backend/internal/platform/licensing/domain` enthaelt Seat-Limits, Assignment-Seat-Regeln und License-Event-Typen.
_Avoid_: Seat checks scattered inside request handlers

**Auth Security Adapter**:
`app/backend/internal/platform/auth/adapters/security` implementiert Argon2id hinter dem `auth/ports.PasswordHasher` Port.
_Avoid_: Password hashing as a free helper in business logic

## Frontend Shared Packages

**`@links/access-core`**:
Reine Access-Domain fuer TypeScript: Produktkeys, Rollen, AccessSnapshot und AccessPolicy mit `can`, `hasProduct`, `hasTenantRole` und `roleAtLeast`.

**`@links/auth-client`**:
Application-Service-Fassade fuer Auth-State. Proto wird nur in der Infrastruktur/Mapper-Grenze gelesen und in normale Domain-Modelle gemappt.

**`@links/ui-svelte`**:
Svelte-Komponenten und Stores, darunter `<Access>`, die auf `@links/access-core` aufbauen.
