# Lessons (tutor self-improvement log)

Patterns to avoid repeating, captured after corrections from the user.
Format: `- <date> — <what went wrong> → <rule for next time>`
- 2026-10-04 — Asked to build the reference solution: the user wants it built incrementally, not in one shot → one TDD step per commit, tag per lesson and per Part, docs updated from the tagged code afterwards.
- 2026-10-04 — Branched `solution` from a stale local `main` (missing two merged PRs) → always `git fetch origin main` and branch from `origin/main`.
