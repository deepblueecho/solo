# Dashboard Task Entry Design

## Goal

Make Dashboard the primary place where users start work with agents. Tasks becomes the place to track work, not a second competing place to start it.

## Product Rule

Users start work from a channel or DM conversation.

Tasks created from a conversation must remain tied to that conversation, so the user can move from the task board back to the original context without learning a separate task-detail model.

## Current Problem

Solo currently exposes two mental models:

- Dashboard: send messages to agents.
- Tasks: track and manage tasks.

The split is useful, but the boundary is fuzzy. Users can wonder whether they should "message an agent" or "create a task", and after a tracked task is created they may not know whether to stay in Dashboard or go to Tasks.

## Chosen Approach

Use a message-first model:

- Dashboard is the work entry point.
- Message is the default mode.
- Task is an explicit mode inside the Dashboard input.
- Tasks is a tracking board.

This keeps Solo closer to agent collaboration than project-management software.

## User Flow

1. User opens a channel or DM in Dashboard.
2. User writes normally in the message input.
3. User can switch the input mode from `Message` to `Task`.
4. In `Task` mode, the input communicates that it will create tracked work.
5. User submits.
6. Solo creates the task from the message.
7. Solo keeps the user in the current channel or DM.
8. Solo opens the right-side thread/task panel for the new task.
9. User can continue discussion there or later track it from Tasks.

## Dashboard Input Behavior

The input has two modes:

- `Message`: normal chat message.
- `Task`: create a tracked task from the message.

Default mode is `Message`.

In `Task` mode:

- Placeholder should describe task creation, not chat.
- Submit action should read as task creation, for example `Create Task`.
- Existing attachment and mention behavior should remain unchanged.
- No separate task creation dialog is introduced.

## After Task Creation

After creating a task from Dashboard:

- Keep the user in the current channel or DM.
- Open the existing right-side thread/task panel.
- Show task status, owner/claimer, original message, and replies in that panel.
- Avoid sending the user to `/tasks` automatically.

This confirms that the task exists while preserving conversational context.

## Tasks Page Role

Tasks is for tracking:

- Filter by source, owner, creator, and status.
- Review task progress.
- Open a task's thread/task panel.
- Change task status or ownership where supported.

Tasks should not be presented as the primary place to create work.

Empty states should guide users back to Dashboard when there is no work to track.

## Task Detail Model

Do not add a separate task-detail page for this change.

Use the existing thread/task panel as the single task detail surface. A task is understood as a tracked message plus workflow state.

## Out of Scope

- New backend task model.
- New `/tasks/new` experience.
- Full task creation form.
- Jira-like fields.
- Visual restyling beyond copy and state needed for the flow.

## Files Likely Involved

- `frontend/components/dashboard/message-input.tsx`
- `frontend/components/dashboard/channel-view.tsx`
- `frontend/components/dashboard/dm-view.tsx`
- `frontend/app/tasks/page.tsx`
- `frontend/components/tasks/task-board.tsx`

## Success Criteria

- A user can start tracked work from Dashboard without leaving the conversation.
- Creating a task immediately shows where to follow up.
- Tasks reads as a tracking board, not a competing creation workflow.
- No new task creation surface is added.
- Existing message sending behavior remains unchanged.

