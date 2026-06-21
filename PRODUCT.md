# Product

## Register

product

## Users

Hobbyists and the project author. Primarily two people who want to play a quick
game of chess against each other in real time: one creates a table, shares an
invite code, the other joins. Secondary audience is anyone evaluating the project
as a portfolio piece, so the interface doubles as a demonstration of craft. Low
volume, no accounts to manage, no competitive ladder. Sessions are short and
social: open the lobby, start or join one game, play, leave.

## Product Purpose

A real-time two-player chess app built on a bitboard engine, with a Go backend,
WebSocket move sync, Postgres + Redis persistence. It exists to let two people
play a clean game of chess with the least friction possible, and to show the
author's engineering and design ability. Success: a stranger can land on the
lobby, understand it in seconds, start a game, and never feel the seams between
lobby, table, and board.

## Brand Personality

Calm, modern, precise. The voice of a well-made tool, not a game arcade. Three
words: quiet, considered, confident. The interface should feel like Linear or
Notion applied to chess: restrained surfaces, deliberate typography, one accent
that does real work. It evokes focus and ease, never urgency or spectacle.

## Anti-references

- **Generic SaaS dashboard**: cream cards, gradient hero-metric blocks, identical
  icon + heading + text feature grids. The default AI-generated look. Avoid.
- **Neon gamer / crypto**: glowing neon-on-black, aggressive RGB, esports energy.
- **Skeuomorphic wood**: literal wooden boards, felt textures, vintage chrome,
  heavy drop shadows. The board uses flat tournament colors, not photoreal wood.
- **Cluttered chess portal**: dense, ad-heavy, toolbar-and-panel-everywhere
  chess.com sprawl. Keep one primary task visible per surface.

## Design Principles

1. **One task per surface.** The lobby is for starting or joining a game; the
   board is for playing. Never make the player hunt for the primary action.
2. **Restraint carries the brand.** Tinted neutrals plus a single blue accent
   used sparingly. Color earns its place; it is not decoration.
3. **Seamless across surfaces.** Lobby, table, and board share one token system,
   one topbar, one theme. Moving between them should feel like one app.
4. **Quiet motion.** Transitions ease out and stay short. Motion confirms an
   action or guides attention, never performs.
5. **Legible by default.** Tabular numerals, generous line length limits, clear
   hierarchy through scale and weight. The player reads state at a glance.

## Accessibility & Inclusion

Standard good defaults. Target WCAG AA contrast for text and interactive states
in both light and dark themes. Always-visible focus styling on inputs and
buttons. Respect `prefers-reduced-motion` by reducing or removing non-essential
transitions and looping animations. Do not rely on color alone to convey game
state (turn, last move, check).
