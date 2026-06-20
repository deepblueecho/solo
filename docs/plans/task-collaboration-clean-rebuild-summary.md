# Task collaboration clean rebuild summary

Branch: `codex/task-collab-clean`

Base: `master` at `901e2d7`

## Size

Compared with `master`:

```text
24 files changed, 1211 insertions(+), 258 deletions(-)
```

By commit:

| Commit | Purpose | Size |
|---|---|---:|
| `a54926a` | Fix master baseline verification | 7 files, +124/-6 |
| `0d6e26b` | Add minimal agent relationships | 7 files, +447/-4 |
| `741be2f` | Add task submit/accept/reject lifecycle | 8 files, +374/-107 |
| `b233169` | Fold subtasks under parent task cards | 2 files, +59/-5 |
| `35d463d` | Route wakes to mentioned agents or coordinator | 2 files, +207/-136 |

Note: the final wake-routing commit includes `gofmt` cleanup in `internal/server/service/agent.go`, so the raw line count is larger than the semantic change.

## Added vs master

### Relationships

- Added `agent_relationships` migration.
- Added minimal relationship API:
  - create
  - list
  - list by agent
  - update
  - delete
- Kept only two relationship types:
  - `assigns_to`
  - `collaborates_with`
- `collaborates_with` is unique regardless of direction.
- `assigns_to` is directional.

### Task lifecycle

- Added business lifecycle actions:
  - `submit`: claimer moves task from `in_progress` to `in_review`
  - `accept`: creator moves task from `in_review` to `done`
  - `reject`: creator moves task from `in_review` back to `in_progress`
- Added HTTP routes:
  - `POST /api/v1/channels/{channelID}/tasks/{taskID}/submit`
  - `POST /api/v1/channels/{channelID}/tasks/{taskID}/accept`
  - `POST /api/v1/channels/{channelID}/tasks/{taskID}/reject`
- Added CLI commands:
  - `solo task submit`
  - `solo task accept`
  - `solo task reject`
- Updated agent prompt so agents use lifecycle commands instead of `solo task update -s` for review flow.

### Task UI

- Parent tasks now own visible child tasks in the board.
- Child tasks no longer appear as separate cards when their parent is present in the loaded task list.
- Parent cards show a collapsible subtask list with:
  - child task status color
  - child task number
  - child task title
- Orphan child tasks still appear normally so filtering or partial data does not hide tasks.

### Wake routing

- Explicit agent mention wakes only mentioned agents.
- A message with unresolved `@...` patterns wakes no agents.
- Unmentioned channel messages wake one coordinator.
- Unmentioned thread messages wake:
  1. coordinator among existing thread agent participants
  2. root message agent if present
  3. channel coordinator fallback
- New task with mentions wakes only mentioned agents.
- New task without mentions wakes one coordinator.
- Coordinator selection:
  - prefer the root of the `assigns_to` relationship graph
  - fallback to first active channel agent by stable join/create order

### Baseline fixes

- Added `local` backend alias for Claude Code.
- Fixed memory tests to match current workspace memory path.
- Fixed one CLI test using unsupported legacy args.
- Added missing frontend type/i18n fields.
- Wrapped `/tasks` page in `Suspense` for `useSearchParams`.

## Removed or intentionally not ported

- No block/unblock task dependency model.
- No task dependency migration/table.
- No `solo task block`, `unblock`, or `blocked`.
- No dependency popover or block DAG UI.
- No relationship graph UI.
- No relationship event publisher.
- No `RELATIONSHIPS.md` generation.
- No channel-scoped relationship model.
- No extra relationship types.
- No template backend migration/API.
- No teams gallery expansion.
- No automatic execution-driven transition to review.

## Differences from the original plan

### PR0

Unchanged. We kept baseline verification as its own commit.

### PR1

Changed: templates were not ported into backend/API.

Reason: `master` already had clear frontend role templates, and adding backend templates would expand scope without solving the current task/relationship behavior problem.

### PR2

Changed: old `PATCH status` remains available.

Reason: the current server does not cleanly distinguish human UI actions from agent CLI actions at this layer. Instead, agents are guided to use `submit/accept/reject`, while the compatibility endpoint remains for existing UI.

### PR3

Unchanged in spirit, but intentionally minimal.

We implemented nested subtasks in the board without adding a full graph view or dependency visualization.

### PR4

Changed: “leader” is derived from `assigns_to`, not from a separate role column or template key.

Reason: we kept the architecture smaller. The coordinator is the agent with no parent in the assignment graph; without relationships, the first active channel agent responds.

## Possible surprises

- `internal/server/service/agent.go` has a larger diff because `gofmt` fixed existing indentation while touching the file.
- `solo task update -s` still exists for compatibility, but the agent prompt no longer teaches it as the lifecycle path.
- Relationship routing assumes `assigns_to` means `leader assigns_to worker`.
- Wake routing now reduces fan-out. This is intended, but it means a channel with no relationships will pick one stable fallback agent instead of all agents.
- Subtasks are folded only when parent and child are both present in the loaded task list.

## Verification

Passed:

```bash
go test ./... -count=1
cd frontend && npm run build
```
