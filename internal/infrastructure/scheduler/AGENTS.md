# Scheduler Infrastructure Rules

- Keep scheduling mechanics separate from use-case and domain rules.
- Make task execution idempotent and context-aware; do not add a distributed queue in V1.
- Persist observable task states through application-owned ports.
