---
name: Chess
description: Real-time two-player chess with the restraint of a well-made tool.
colors:
  accent: "#5b8def"
  accent-hover: "#4577e0"
  accent-contrast-dark: "#0c1118"
  bg-dark: "#0e1117"
  surface-dark: "#1a212d"
  surface-2-dark: "#212a38"
  surface-3-dark: "#2a3545"
  text-dark: "#e7ecf3"
  text-muted-dark: "#98a3b3"
  text-faint-dark: "#6b7686"
  border-dark: "#2a3342"
  bg-light: "#f4f5f7"
  surface-light: "#ffffff"
  text-light: "#1c2430"
  text-muted-light: "#6b7686"
  border-light: "#e2e6ec"
  board-light-sq: "#ebecd0"
  board-dark-sq: "#7d945d"
  good: "#3aa564"
  danger: "#e0564f"
  lastmove: "#ffc440"
typography:
  display:
    fontFamily: "Inter, system-ui, -apple-system, Segoe UI, Roboto, sans-serif"
    fontSize: "clamp(1.9rem, 4vw, 2.7rem)"
    fontWeight: 800
    lineHeight: 1.05
    letterSpacing: "-0.03em"
  title:
    fontFamily: "Inter, system-ui, sans-serif"
    fontSize: "1.15rem"
    fontWeight: 800
    lineHeight: 1.2
    letterSpacing: "-0.02em"
  body:
    fontFamily: "Inter, system-ui, sans-serif"
    fontSize: "1rem"
    fontWeight: 400
    lineHeight: 1.55
    letterSpacing: "normal"
  label:
    fontFamily: "Inter, system-ui, sans-serif"
    fontSize: "0.7rem"
    fontWeight: 700
    lineHeight: 1.2
    letterSpacing: "0.14em"
rounded:
  sm: "8px"
  md: "12px"
  lg: "18px"
spacing:
  xs: "8px"
  sm: "12px"
  md: "18px"
  lg: "28px"
components:
  button-primary:
    backgroundColor: "{colors.accent}"
    textColor: "{colors.accent-contrast-dark}"
    rounded: "{rounded.md}"
    padding: "16px 20px"
  button-primary-hover:
    backgroundColor: "{colors.accent-hover}"
    textColor: "{colors.accent-contrast-dark}"
    rounded: "{rounded.md}"
    padding: "16px 20px"
  button-ghost:
    backgroundColor: "{colors.surface-3-dark}"
    textColor: "{colors.text-dark}"
    rounded: "{rounded.md}"
    padding: "13px 24px"
  icon-button:
    backgroundColor: "{colors.surface-dark}"
    textColor: "{colors.text-muted-dark}"
    rounded: "{rounded.md}"
    size: "40px"
  input-text:
    backgroundColor: "{colors.surface-2-dark}"
    textColor: "{colors.text-dark}"
    rounded: "{rounded.md}"
    padding: "13px 15px"
  surface-card:
    backgroundColor: "{colors.surface-dark}"
    textColor: "{colors.text-dark}"
    rounded: "{rounded.lg}"
    padding: "16px 18px"
---

# Design System: Chess

## 1. Overview

**Creative North Star: "The Quiet Board"**

This is the interface of a tournament hall, not a game arcade. The surface recedes
so the game can lead: tinted near-black panels in the dark theme, soft off-whites in
the light theme, and a single blue accent that appears only where something is
actionable. The board itself uses flat tournament colors (sage green over bone
white), never photoreal wood. Everything is built from one token set in `theme.css`,
so the lobby, the table, and the board read as one continuous app rather than three
screens stitched together.

Density is calm and deliberate. Type hierarchy comes from scale and weight, not from
color or decoration. Numerals are tabular everywhere they represent state (timers,
move counts, material, invite codes) so the eye can track change without re-reading.
Motion is a confirmation layer: surfaces ease up on entrance, the live status dot
breathes, buttons lift a pixel or two. Nothing performs.

What this system rejects is as important as what it embraces. It is not a generic SaaS
dashboard of cream cards and gradient hero-metrics. It is not neon-on-black gamer
spectacle. It is not skeuomorphic wood and felt. It is not the cluttered, ad-heavy
chess portal with a toolbar on every edge.

**Key Characteristics:**
- One blue accent, reserved for the actionable.
- Tinted neutrals, never pure `#000` or `#fff`.
- Flat tournament board, never wood.
- Tabular numerals for all game state.
- Dark by default, light as a true peer.

## 2. Colors

A restrained, hue-tinted neutral system carrying a single cool-blue accent, with a flat
tournament board palette held apart from the UI chrome.

### Primary
- **Signal Blue** (`#5b8def`, hover `#4577e0`): the one accent. It marks the primary
  action (Create a game), focus rings, the active player's turn, selected squares, and
  live links. On a saturated blue button, text flips to near-black ink (`#0c1118`) for
  contrast. Used on roughly 10% of any screen; its scarcity is what makes it read.

### Neutral (dark theme, default)
- **Hall Black** (`#0e1117`): the page floor, served under a faint radial top-glow.
- **Panel Slate** (`#1a212d`) / **Raised Slate** (`#212a38`) / **Lifted Slate**
  (`#2a3545`): the three stacked surface tones for cards, inputs, and hover states.
- **Chalk** (`#e7ecf3`) / **Ash** (`#98a3b3`) / **Faint Ash** (`#6b7686`): primary,
  muted, and faint text. Borders sit at `#2a3342`, one notch above the floor.

### Neutral (light theme)
- **Paper** (`#f4f5f7`) page, **White** (`#ffffff`) surfaces, **Ink** (`#1c2430`) text,
  **Hairline** (`#e2e6ec`) borders. A true peer theme, not an afterthought.

### Tertiary (board + state)
- **Board Bone** (`#ebecd0`) and **Board Sage** (`#7d945d`): the flat tournament squares.
- **Win Green** (`#3aa564`) for the turn indicator and live status, **Alert Red**
  (`#e0564f`) for danger, **Last-Move Amber** (`#ffc440`) for the last move highlight.

### Named Rules
**The One Voice Rule.** Signal Blue speaks for the actionable and nothing else. If blue
appears on a passive element, it is a bug. The accent's rarity is the point.

**The Tinted Neutral Rule.** Never `#000`, never `#fff`. Every neutral carries a faint
cool tint toward the slate hue so the surfaces feel like one material.

## 3. Typography

**Display / Body / Label Font:** Inter (with `system-ui, -apple-system, Segoe UI, Roboto, sans-serif`)

**Character:** A single humanist-grotesque carries the whole system. Personality comes
from weight contrast (400 body against 800 display) and tight tracking on large sizes,
not from a second typeface. Tabular numerals are switched on wherever digits represent
live state.

### Hierarchy
- **Display** (800, `clamp(1.9rem, 4vw, 2.7rem)`, line-height 1.05, tracking -0.03em):
  the lobby greeting and other single hero lines.
- **Title** (800, 1.15rem, tracking -0.02em): brand wordmark, winner-card heading.
- **Body** (400-500, 1rem, line-height 1.55): descriptions and table data. Cap measure
  at 65-75ch (the lobby sub-line is capped at 46ch).
- **Label** (700, 0.68-0.7rem, tracking 0.07-0.14em, UPPERCASE): eyebrows, panel titles,
  turn indicators, table column heads. The system's connective tissue.

### Named Rules
**The Tabular State Rule.** Any number that changes during play (timer, move count,
material, invite code) uses `font-variant-numeric: tabular-nums`. State must never
reflow as it updates.

## 4. Elevation

A near-flat system with tonal layering as the primary depth cue and soft shadows as a
secondary one. Depth comes first from stacking the three slate surface tones (Panel ->
Raised -> Lifted), then from low-spread shadows that grow with importance. The dark
theme leans almost entirely on tonal layering; shadows there are felt more than seen.

### Shadow Vocabulary
- **`--shadow-sm`** (`0 1px 2px / 0 1px 3px rgba(18,28,45,0.06-0.08)`): resting cards,
  player bars, icon buttons.
- **`--shadow-md`** (`0 4px 12px / 0 2px 4px`): the lobby panels and side panel.
- **`--shadow-lg`** (`0 18px 40px rgba(18,28,45,0.16)`): reserved for the heaviest
  lift; use sparingly.
- **Accent glow** (`0 10px 24px -10px var(--board-glow)`): only under the primary CTA
  and the active player's bar, to tie elevation to the one accent.

### Named Rules
**The Tonal-First Rule.** Reach for the next surface tone before reaching for a shadow.
Shadows confirm hierarchy; they do not create it.

## 5. Components

### Buttons
- **Shape:** medium-rounded corners (`12px`, `--radius`).
- **Primary:** Signal Blue gradient (`135deg, accent -> accent-hover`) with near-black
  ink, generous `16px 20px` padding, an accent glow, and a leading icon chip. Lifts
  `-2px` on hover. This is the Create-a-game CTA; there is one per surface.
- **Ghost / Secondary:** Lifted Slate fill with a strong border; on hover the border and
  text shift to Signal Blue and it rises `-1px`. Used for Join and other second actions.
- **Icon button:** 40px square, resting surface fill, muted icon; border and icon go
  accent on hover. The theme toggle adds an 18-degree rotate on hover.

### Cards / Containers
- **Corner Style:** large radius (`18px`) for panels, medium (`12px`) for list items.
- **Background:** Panel Slate over the page floor; list items step down to Raised Slate.
- **Shadow Strategy:** `--shadow-md` at rest (see Elevation). No inner shadows.
- **Border:** always a 1px full border in `--border`. Never a colored side stripe.
- **Internal Padding:** `clamp(26px, 4vw, 40px)` for hero panels, `14-18px` for list rows.

### Inputs / Fields
- **Style:** Raised Slate fill, 1px `--border`, `12px` radius, `13-15px` padding. Placeholder
  in Faint Ash with relaxed tracking.
- **Focus:** border shifts to Signal Blue plus a 3px `color-mix` accent ring (20%). Always
  visible; never removed.

### Navigation
- **Topbar:** sticky, translucent glass (`backdrop-filter: blur(14px) saturate(150%)`)
  over a 1px bottom border. Left holds the brand (gradient knight chip + wordmark); right
  holds icon buttons. Shared verbatim across every surface.

### Signature Component: Live Tables List
The lobby's open-tables panel is the system's signature: a scrolling column of Raised
Slate rows, each leading with an uppercase label-cased Game ID, tabular invite code, and
player lines where an empty seat renders an italic, blinking "waiting" placeholder. A
breathing Win Green status dot signals the list is live. The Join action is an outline
button that fills Signal Blue on hover.

## 6. Do's and Don'ts

### Do:
- **Do** reserve Signal Blue (`#5b8def`) for actionable elements only: the One Voice Rule.
- **Do** tint every neutral toward slate; never use `#000` or `#fff`.
- **Do** reach for the next surface tone before adding a shadow (Tonal-First Rule).
- **Do** use `tabular-nums` for every number that changes during play.
- **Do** keep one primary action per surface; the lobby starts or joins a game, the board plays it.
- **Do** keep the topbar, tokens, and theme identical across lobby, table, and board.
- **Do** ease motion out and keep it short; respect `prefers-reduced-motion`.

### Don't:
- **Don't** build a generic SaaS dashboard: no cream cards, no gradient hero-metric blocks, no identical icon + heading + text feature grids.
- **Don't** go neon gamer / crypto: no glowing neon-on-black, no aggressive RGB, no esports energy.
- **Don't** render skeuomorphic wood: no wooden boards, felt textures, vintage chrome, or heavy drop shadows. The board is flat tournament Bone and Sage.
- **Don't** sprawl into a cluttered chess portal: no toolbars on every edge, no ad-dense panels. One task visible per surface.
- **Don't** use a colored `border-left`/`border-right` stripe as an accent; borders are full and 1px.
- **Don't** convey game state (turn, last move, check) by color alone.
