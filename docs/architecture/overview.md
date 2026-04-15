# Architecture Overview

## Summary

ChangeControlPlane starts as a modular monolith with strong seams for later extraction. The architecture is deliberately domain-oriented, event-aware, and API-first.

```mermaid
flowchart LR
    A["Web App / CLI / API Clients"] --> B["Versioned API"]
    B --> C["Application Core"]
    C --> D["Domain Modules"]
    C --> E["Risk Engine"]
    C --> F["Rollout Planner"]
    C --> G["Policy Evaluator"]
    C --> H["Audit Recorder"]
    C --> I["Integration Registry"]
    C --> J["Event Bus"]
    C --> K["Repository Interfaces"]
    K --> L["In-Memory Dev Store"]
    K --> M["PostgreSQL (planned runtime backend)"]
```

## Why This Shape

- it avoids premature microservice sprawl
- it keeps boundaries explicit and extraction-ready
- it supports high-cohesion domain modules
- it lets us move quickly while keeping enterprise-grade seams

## Primary Runtime Components

- API service for control-plane operations
- worker service for asynchronous and long-running workloads
- CLI for automation, scripting, and operator workflows
- web application for operational visibility and governance UX
- PostgreSQL-first schema and migrations
- pluggable event bus and integration adapters

## Domain Modules

The repository is organized around major product capabilities:

- org, project, team, and user administration
- service catalog and environment modeling
- change ingestion and assessment
- risk and rollout planning
- policies and audit
- integrations and eventing
- incident, simulation, and AI-ready extension points
