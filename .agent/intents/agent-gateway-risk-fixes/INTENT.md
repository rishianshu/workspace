# Intent: Agent + Gateway Risk Fixes

## Goal
Address runtime risks found in the agent+gateway review: config hardcoding, missing /action route, dropped attachments, and workflow engine nil handling.

## Scope
- Use AGENT_SERVICE_URL env for chat route.
- Add /action HTTP handler in agent service.
- Accept attachedFiles in HTTP handler and inject into query.
- Guard workflow handlers when engine is unavailable.

## Out of Scope
- Large refactors of LLM routing.
- Changes to tool schemas or MCP wiring.

## Acceptance
- No hardcoded agent URL in chat route.
- /action route exists in agent HTTP server.
- attachedFiles are incorporated into the query before LLM execution.
- workflow handlers return 503 when engine is nil.
