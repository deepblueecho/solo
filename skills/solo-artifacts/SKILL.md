---
name: solo-artifacts
description: >-
  Use when a Solo task or thread should become an interactive, reviewable,
  self-contained HTML artifact for progress/status, review/decision, or
  comparison/leaderboard work inside Solo.
---

# solo-artifacts

Solo-brutal fork of `work-canvas`: produce **one self-contained HTML file** that renders a Solo task/thread for a human to understand and act on. The output is a capture of work, not an app and not a transcript dump.

## Workflow

1. **Pick the artifact type** and read its reference:

   | The task needs to... | Type | Read |
   |---|---|---|
   | Show where long/messy work stands, what broke, what's next | **Progress / status report** | `references/progress-report.md` |
   | Review analysis or implementation and decide accept/reject/next step | **Review / decision memo** | `references/review-decision.md` |
   | Weigh options head-to-head and pick a winner | **Comparison / leaderboard** | `references/comparison.md` |

   If blended, lead with the primary type and borrow blocks from the other.

2. **Assemble from the template.** Copy `assets/starter.html`, then:
   - Replace `[[PASTE base.css]]` with the full contents of `assets/base.css`.
   - Replace `[[PASTE interactions.js]]` with only the needed modules from `assets/interactions.js`; pasting the whole file is fine because modules self-guard.
   - Fill the header, **What needs your input**, and body sections using the component recipes in the chosen reference.

3. **Embed everything**: inline CSS/JS, embed media as `data:` URIs, no CDN or external requests.

4. **Publish back to Solo**:

   ```bash
   solo artifact publish --task <task-id> --mode <latest|final> --file <path-to-html>
   ```

## Non-Negotiables

- Keep work-canvas structure and interactions intact; only the visual skin is Solo-brutal.
- Surface only real human decisions. If nothing needs the user, say so.
- Add a legend wherever color, letters, or symbols encode meaning.
- Never modify Solo/source data from the page. Show paste-ready output with copy buttons.
- Include a provenance footer with task id, thread/channel scope, agent, model, and generation date when available.

## Files

- `assets/starter.html` — the skeleton to copy.
- `assets/base.css` — class-compatible Solo-brutal design system.
- `assets/interactions.js` — optional vanilla-JS modules: print, tabs, tables, lightbox, copy, scrollspy, persist.
- `references/*.md` — work-canvas section order, component recipes, and guardrails.
