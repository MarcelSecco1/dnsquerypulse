# ADR 0001: single-process CLI and bounded DNS engine

- Status: accepted
- Date: 2026-10-21

## Context

The MVP is a standalone Windows/Linux DNS diagnostic tool. A server, daemon, remote telemetry, and system DNS changes add risk without helping the core query, monitor, and benchmark workflows.

## Decision

Keep command orchestration in one Go process. Use contexts and deadlines for cancellation, sequential non-overlapping monitor rounds, and a fixed worker pool for benchmarks. Encode the DNS messages directly using Go's standard networking primitives and decode common answer records. Use strict YAML for the small local configuration schema.

## Consequences

The executable has a small runtime surface and does not require privileged installation. DNS protocol code needs focused controlled-fixture tests. New record types and transports must be added deliberately and documented; encrypted DNS remains future work.
