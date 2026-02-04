---
name: workspace-sdlc
description: Use for any development task in Workspace to enforce SDLC artifacts, contract/ontology discipline, and review/testing gates.
---

# Workspace SDLC Skill

## When to use
- Any code change, feature, or bugfix in Workspace.
- Planning or reviewing work for agentic coding features.

## Workflow (lightweight, deterministic)
1) Intake: restate intent, define scope and non-goals. If non-trivial, create/update requirement artifacts:
   - `.agent/intents/<slug>/INTENT.md`
   - `.agent/intents/<slug>/SPEC.md`
   - `.agent/intents/<slug>/ACCEPTANCE.md`
   - `.agent/runs/<slug>/RUNCARD.md` (when handing off to Codex)
2) Contract-first: prefer explicit schemas, enums, and stable error codes; update README/docs when public surface changes.
   - Policy: code, docs, and ontology must stay aligned; update `docs/ontology/INDEX.md` for any contract or service change.
3) No special-casing: avoid hardcoded IDs; use capability descriptors and config.
4) Determinism: same inputs produce same outputs; avoid LLM-driven logic unless required by spec.
5) Verification: add/adjust tests for new behavior; capture evidence (tests or logs).
6) Review gate: run a local Codex review before committing changes.
7) Record decisions and blockers in `.agent/runs/<slug>/DECISIONS.md` and `.agent/runs/<slug>/QUESTIONS.md`.

## Notes
- If a doc index/ontology exists in the repo, read and update it when contracts change.
- Keep artifacts small and concrete; no TODOs in INTENT/SPEC.
