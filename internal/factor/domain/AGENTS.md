# Factor Domain Rules

- Keep factor calculations deterministic and independent of I/O.
- Reject NaN and Inf; do not round before ranking or tie handling.
- Accept immutable market inputs through domain types or ports, not provider or database models.
