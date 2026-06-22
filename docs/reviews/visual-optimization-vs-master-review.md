# Visual Optimization Review vs Master

Date: 2026-06-22

Branch: `codex/visual-optimization`
Base: `f4fe2686643631919f9f70cc0f844c6bce1f4233`
Head: `a7f2b9b35adf8068989e1f452192bbe449ff1a38`

## Scope

Reviewed `master...HEAD`.

Diff size: 49 files changed, 1528 insertions, 1986 deletions.

This branch is broader than a visual-only cleanup. It replaces the old `/teams` list/detail layout with the relationship graph workspace, removes standalone workspace routes/components, embeds agent profile/workspace detail into the graph drawer, and unifies a large set of buttons, dialogs, inputs, selects, search dropdowns, and list rows.

## Findings

### P1: `/teams` captures `Cmd/Ctrl+Z` even while typing

File: `frontend/components/relationships/relationship-workspace.tsx:505`

The global `keydown` handler always calls `e.preventDefault()` for `Cmd/Ctrl+Z` and runs graph undo/redo. Because the handler is mounted on `window`, this also fires when focus is inside text inputs/textareas in agent forms, relationship instruction editing, search fields, or dialogs.

Impact: users cannot use native text undo while editing. Worse, graph undo may run while the user expects to undo typed text.

Suggested fix: skip graph shortcuts when `document.activeElement` is an `input`, `textarea`, `select`, or `contentEditable`, and consider disabling graph shortcuts while modal dialogs are open.

### P2: New assertion script currently fails

File: `frontend/scripts/assert-team-relationship-first.mjs:166`

Running:

```bash
node frontend/scripts/assert-team-relationship-first.mjs
```

fails with:

```text
Error: workspace file pane should be collapsible
```

The script expects `PanelLeftClose` and `isFilePaneCollapsed`, but `frontend/components/teams/teams-agent-workspace.tsx` currently has a resizable file pane, not a collapsible one.

Impact: the committed verification artifact is stale or ahead of implementation. It is also not wired into `frontend/package.json`, so CI/build will not catch it.

Suggested fix: either implement the collapse behavior, or update/delete the assertion. If the script should matter, add an npm script for it.

## Ponytail Review

`frontend/scripts/assert-team-relationship-first.mjs:1`: shrink/delete: 247 lines of brittle string assertions for UI implementation details. Replace with one small smoke check for routes/imports, or delete until it is wired into CI.

Net: about -200 lines possible if reduced to a real smoke check.

## Change Summary

### Teams and Relationships

- `/teams` now renders `RelationshipWorkspace` directly.
- Standalone `/relationships` route moved into `frontend/components/relationships/relationship-workspace.tsx`.
- Standalone `/workspace` route and old workspace helper components were removed.
- Relationship graph now owns agent node detail, relationship detail, agent creation, template creation, and embedded workspace/profile drawers.
- Relationship/agent detail panels share header and section primitives.

### Agent Detail and Workspace

- Agent detail drawer now has Profile and Workspace tabs.
- Agent profile can be embedded without redirect after delete.
- Agent workspace can be viewed inside the drawer, resized, and fullscreened.
- Workspace auto-selects the first file when available.

### Computers

- Computers detail changed from expandable card behavior to full-page detail.
- Detail sections now use shared brutal section styling.
- Connected Agents became collapsible.
- Computer name edit controls now align with Agent detail edit/save/cancel styling.

### Modal, Input, Select, Dropdown Styling

- Create Task, Create Channel, Create DM, Add Agent, Delete Channel, Delete Relationship, Channel Members, Agent Form, and related relationship dialogs were moved toward shared `Dialog`, `Button`, `Input`, `Textarea`, and `Select` primitives.
- Select focus now matches input focus.
- Inbox filter and search inputs no longer use the electric-blue focus ring.
- Global Search and Channel Search dropdowns now share cream panel, hard border, brutal shadow, black separators, and pale-yellow hover/active states.

### Shared UI Primitives

- Added shared helpers:
  - `detailSectionClass`
  - `detailSectionTitleClass`
  - `detailFieldLabelClass`
  - `detailEditActionClass`
  - `panelHeaderClass`
  - `panelTitleClass`
  - `iconActionClass`
  - `tabButtonClass`
  - `selectableRowClass`
  - `selectableRowIconClass`
- Added `Button` success variant and focused input helpers in `globals.brutal.css`.

## Verification

Passed:

```bash
git diff --check
npm run build
```

Build warning observed, pre-existing in this thread:

```text
Warning: `--localstorage-file` was provided without a valid path
```

Failed:

```bash
node frontend/scripts/assert-team-relationship-first.mjs
```

Failure:

```text
workspace file pane should be collapsible
```

Route/link sanity:

- No remaining app/component links to removed `/workspace` or `/relationships` routes were found outside the assertion script and workspace API paths.

## Manual QA Checklist

- `/teams`: graph loads, nodes render, relationship edges render.
- Create relationship by connecting two nodes.
- Edit relationship type and instruction from detail drawer.
- Delete relationship from detail drawer.
- Open agent detail from node, switch Profile/Workspace tabs.
- Try `Cmd/Ctrl+Z` while typing in a relationship instruction field.
- Create single agent from graph toolbar.
- Create agents from template.
- Open agent workspace, resize drawer, fullscreen workspace.
- Check Computers detail, connected agents collapse, and name edit controls.
- Check Inbox filter, Channel Search, and Global Search focus states.

