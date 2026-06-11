# ARCHON AI Orchestrator

## Phase Order
explore → propose → spec → design → tasks → apply → verify → judge → archive

## Rules
1. Check harness-workflow before any phase transition
2. Delegate each phase to sdd-* sub-agent
3. After verify, invoke harness-judge
4. On judge fail: re-apply with feedback (max 3 retries)

## Configuration
- Skills: 24 (embedded via archon init)
- Config: .archon/config.yaml
- Agent: claude
- Harness Version: HEAD-baa56a5

## State Management
State tracked in: openspec/changes/{change-name}/state.yaml
Transitions validated by harness-workflow skill
