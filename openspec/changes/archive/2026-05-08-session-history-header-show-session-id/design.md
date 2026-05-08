## Context

`SessionHistoryView.vue:70` currently builds the page-header suffix as:

```ts
const titleSuffix = computed(() => store.displayTitle ?? '');
```

and `useSessionHistoryStore` (`stores/sessionHistory.ts:32-35`) collapses two distinct values into one string:

```ts
const displayTitle = computed(() => {
  if (!currentMeta.value) return null;
  return currentMeta.value.displayName || currentMeta.value.sessionID;
});
```

So when the backend cannot resolve a rich `displayName` (the per-agent priority chain in `internal/sessionhistory/{claude,copilot}/reader.go`), the suffix silently degrades to the session id; when it succeeds, the user sees the rich title but never the session id. The user's request is to invert the priority — make the session id the always-visible primary line and demote the rich title to an optional, secondary line. Adding a copy button on the session id makes the header the canonical "give me this session's filesystem handle" surface.

`PageContainer.vue` (`frontend/src/layouts/PageContainer.vue:5,38-41`) currently accepts only a `titleSuffix: string` prop and renders it as plain text inside an `<h1>`. To host a session id + copy button + optional secondary line, the layout component needs a structured suffix area. There are two existing call sites of `PageContainer` that pass `titleSuffix` (BrowsePluginsView, InstalledView — both use it for "(Claude)" / "(Copilot)" labels) plus the Session History view; the contract change must remain backward-compatible.

The Wails runtime exposes `ClipboardSetText` (`frontend/wailsjs/runtime/runtime.d.ts:235`), giving us a webview-agnostic clipboard write that does not depend on `navigator.clipboard` permissions inside the embedded webkit/edge view.

## Goals / Non-Goals

**Goals:**

- Render the full, untruncated session id as the primary suffix line of the Session History page header whenever a session is selected.
- Provide a one-click copy control next to the session id that writes the same string to the system clipboard and gives transient confirmation.
- Render the resolved rich `displayName` as a secondary line **only** when it adds information (i.e. it is non-empty AND not equal to the session id).
- Preserve the no-selection state of the header (no suffix at all, no copy button, no secondary line).
- Keep `BrowsePluginsView` / `InstalledView` working with their existing `titleSuffix` prop without code changes.
- Honour `CLAUDE.md` daisyUI boundaries: `btn` classes stay inside primitives.

**Non-Goals:**

- Changing the backend `SessionMeta` shape, the per-agent display-name resolution rules, or the existing `Display name resolution for Claude/Copilot` scenarios in the spec — those are unchanged.
- Adding a "share / open in file manager" action beyond clipboard copy — out of scope.
- Adding a copy control anywhere else in the app (e.g. session list rows) — out of scope.
- Right-click / keyboard shortcut copy — only the visible button is required.
- Localising the "Copied" feedback string — English-only is acceptable for now (same baseline as existing toast messages).

## Decisions

### Decision 1: Use Wails `ClipboardSetText`, not `navigator.clipboard.writeText`

**Choice:** The copy handler calls `import { ClipboardSetText } from '@wailsjs/runtime/runtime'` and awaits it. Falls back silently on rejection (the user just sees no toast).

**Rationale:** `navigator.clipboard.writeText` requires a secure context and a user-activation gesture; both are satisfied in a click handler in the Wails webview today, but the Wails runtime API is the cross-platform, documented path and is what other Wails apps use. It also matches how the codebase already uses `EventsOn` / `EventsEmit` for runtime functionality.

**Alternatives considered:**

- `navigator.clipboard.writeText` — works but adds a fragile dependency on webview clipboard policies; not used elsewhere in this codebase.
- `document.execCommand('copy')` — deprecated, requires DOM-selection plumbing.

### Decision 2: Add `AppCopyButton` primitive in `frontend/src/components/ui/`

**Choice:** Introduce `AppCopyButton.vue` that wraps `AppButton` (or renders the `btn btn-ghost btn-circle` icon shell directly, since this is a primitive). Props: `text: string` (the value to copy), optional `size`, optional `tooltip`. Internally it calls `ClipboardSetText(text)` and shows a transient "Copied" feedback (icon swap for ~1s) plus dispatches a toast via `useToast()`.

**Rationale:** `CLAUDE.md` forbids inlining `class="btn btn-ghost btn-circle"` inside views (the documented exception only covers non-`<button>` hosts like `<RouterLink>` / `<label>`, which does not apply here — the copy control is genuinely a `<button>`). A copy button is also likely to be reused (installed-plugin id, marketplace url, future "copy command", etc.), so factoring the primitive once now is cheaper than re-extracting later. The primitive lives next to the other `App*` primitives so the daisyUI leak grep stays at zero matches.

**Alternatives considered:**

- Inline `<AppButton variant="ghost" size="sm">` with an icon slot — works for one-off, but every caller would still wire `ClipboardSetText` + toast/feedback, duplicating logic. A primitive that owns the copy-feedback state machine is a better fit.
- New ad-hoc component under `components/sessions/` — wrong layer; a copy control is cross-cutting, not session-specific.

### Decision 3: Extend `PageContainer.vue` with a named slot, keep the `titleSuffix` string prop

**Choice:** Add an optional named slot `title-suffix` to `PageContainer.vue`. When the slot has content, render it inside the existing `<span>` placeholder (replacing the prop-driven text). When the slot is empty, fall back to the existing `props.titleSuffix` string render. The slot is allowed to render any inline content (text + buttons + a second line).

**Rationale:** Backward-compatible — `BrowsePluginsView` and `InstalledView` keep passing `:title-suffix="..."` and render unchanged. The Session History view stops passing `:title-suffix` and instead provides a `<template #title-suffix>` block that renders the session id, the copy button, and the conditional display-name line. The existing visual treatment (`text-xl font-normal opacity-60 ...`) becomes the slot's default container; the view chooses whether to inherit it or override.

**Alternatives considered:**

- Replace the string prop with a richer object prop (`{ id: string, displayName?: string, copyable?: boolean }`) — couples `PageContainer` to a session-specific shape; bad layering.
- Move the entire header construction into the view and remove `PageContainer`'s suffix concern — too disruptive; existing views rely on it.

### Decision 4: Store exposes `currentSessionID` and `currentDisplayName` separately; `displayTitle` is removed

**Choice:** In `stores/sessionHistory.ts`, replace the `displayTitle` computed with two computed (or already-existing-state) accessors:

- `currentSessionID` — already in the store as `currentSessionID: ref<string>` (line 14); reuse as-is.
- `currentDisplayName` — new computed: `currentMeta.value?.displayName || ''`. Returns the empty string when no rich title was resolved.

The view derives "should the secondary line render?" from `currentDisplayName !== '' && currentDisplayName !== currentSessionID`.

**Rationale:** Keeps the store's surface a flat list of facts about the current selection, with the view in charge of presentation. The "is this a fallback?" decision belongs in the view because it is about what to render, not about what data exists. Removing `displayTitle` is fine — it has exactly one caller (`SessionHistoryView.vue:70`) and that caller is changing in this same change.

**Alternatives considered:**

- Add a `currentHasRichTitle: ComputedRef<boolean>` on the store — pushes presentation logic into the store, no real benefit.
- Keep `displayTitle` and add a sibling `currentSessionIDForHeader` getter — leaves a deprecated getter floating with no caller; pure debt.

### Decision 5: Suppress the secondary line when `displayName === sessionID`

**Choice:** Treat the existing fallback in `SessionMeta` (Claude: `sessionId[0:8]`; Copilot: `sessionId[0:8]`) as "no rich title" for header purposes. Specifically, the view hides the secondary line whenever `currentDisplayName === currentSessionID`. Because the backend's terminal fallback for Claude is the **first 8 chars** of the session id (not the full id), this exact-equality test only fires when a backend regression actually returns the full session id as the display name. In practice the fallback returns `sessionId[:8]`, which is **not equal** to the full session id, so the secondary line would render the 8-char short id — which is also undesirable duplication.

To handle both shapes cleanly, the view's "show secondary line" predicate is:

```
displayName !== ''
  && displayName !== sessionID
  && !sessionID.startsWith(displayName)   // catches the 8-char short-id fallback
```

The `startsWith` clause is intentionally narrow — it only matches the documented short-id fallback (a strict prefix of the session id) and does not accidentally hide legitimate display names like `chatbot-fix-1234` even though they could in theory be a prefix of a session id (UUIDs use `0-9a-f` and `-`; treat any displayName composed only of those characters and matching `sessionID.startsWith(displayName)` as a fallback).

**Rationale:** Without this rule the secondary line would always render, even in the common no-rich-title case where the 8-char short id adds nothing. The user's brief says "若沒有則不顯示 display title" — "if there is no [meaningful] display title, do not show it" — and the short-id fallback is what "no display title" means in practice for both adapters today.

**Alternatives considered:**

- Change the backend so `SessionMeta.displayName` is empty when only the short-id fallback fires — cleanest long-term, but expands the change surface to Go code, fixtures, and the resolver tests in two adapters. Worth doing later as a follow-up; not in this change.
- Just check `displayName !== sessionID` (no `startsWith` clause) — fails the common case, secondary line always shows the short id.

## Risks / Trade-offs

- **Risk: `ClipboardSetText` resolves `false` on platforms with restricted clipboards.**
  → Mitigation: the copy handler awaits the result; on `false` or thrown error the button does not show the success swap and the toast says "Failed to copy" instead of "Copied". No data loss, no console spam.
- **Risk: The session id is long (UUID for Claude, longer slug for Copilot) and may not fit the existing 80px-tall header strip on narrow windows.**
  → Mitigation: the existing `<h1>` already uses `flex flex-col gap-y-1` and `break-words`; the suffix slot inherits `break-words leading-tight`. If wrapping introduces a third line that pushes the optional display-name out of the strip, the strip's `h-20 overflow-hidden` will clip — acceptable for the first iteration; revisit with `min-h-20` if user-reported.
- **Risk: The `startsWith`-based fallback detection (Decision 5) becomes wrong if the backend ever returns a `displayName` shorter than the session id by coincidence.**
  → Mitigation: scoped to the documented short-id fallback shape; covered by a targeted Vitest spec asserting both "secondary line hidden when displayName is `sessionID[:8]`" and "secondary line shown when displayName is e.g. `Refactor auth flow`". Long-term: tighten by changing the backend (see Alternatives in Decision 5).
- **Trade-off: A new `AppCopyButton` primitive adds one more file under `components/ui/` for a single immediate caller.**
  → Accepted: the daisyUI boundary rule in `CLAUDE.md` makes this the lowest-friction way to add the affordance, and the cost of one small file is negligible.

## Migration Plan

This is a presentation-only change in the frontend; nothing on disk and nothing in the Wails binding shape moves. There is no data migration. Rollback is `git revert` of the change commit. No feature flag is needed because the previous behaviour (showing rich title or short-id fallback) is replaced wholesale by a strictly more informative header.

## Open Questions

- Should the copy button also be exposed on each row of `SessionListItem.vue`?
  → Defer. The user's brief is specifically about the header. If the affordance proves useful, a follow-up change can pull `AppCopyButton` into the list rows.
- Should the secondary display-name line be clickable (e.g. to also copy)?
  → Defer. Not requested; adding it now risks visual confusion with the primary copy control.
