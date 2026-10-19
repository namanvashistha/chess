import React from "react";
import {AbsoluteFill, Easing, Img, interpolate, spring, staticFile, useCurrentFrame} from "remotion";
import {Audio} from "@remotion/media";
import {at, NIGHT, pieces} from "./chess";
import {Board3D, BOARD, Glow, PieceState, Ray, Shards} from "./Board3D";
import {GlassPiece} from "./GlassPiece";

/**
 * The reel engine. A reel is data (src/reels/*.json, written by reels.py
 * from chess notation checked move by move): a starting position, the moves
 * with their frames, lines of light, glows, ghosts, a trail, a counter, the
 * camera, captions and the brand card. This draws any of them.
 */

export type LineSpec = {text: string; at: number; size: number; color?: string; accent?: string; accentColor?: string; gap?: number; weight?: number};
export type BeatSpec = {from: number; to: number; top?: number; lines: LineSpec[]};
export type ReelData = {
  id: string;
  length: number;
  start: string;
  moves: {from: string; to: string; at: number; dur: number; liftAt: number; capture?: string; shatter?: boolean}[];
  topple?: {sq: string; at: number};
  rays: {from: string; to: string; color: string; at: number; until: number; draw?: number}[];
  glows: {sq: string; color: string; at: number; until: number; opacity?: number; pulse?: boolean}[];
  marks: {sq: string; kind: "x" | "ring"; color: string; at: number; until: number}[];
  ghosts: {code: string; from: string; to?: string; at: number; until: number; glideAt?: number; glideDur?: number}[];
  trail?: {squares: string[]; times: number[]; color: string};
  counter?: {at: number; until: number; top: number; steps: {at: number; text: string}[]; color: string; label?: string};
  camera: [number, {zoom: number; x: number; y: number; roll: number; tilt: number}][];
  shakes: {at: number; strength: number}[];
  flashes: {at: number; color: string}[];
  dim: {at: number; until: number}[];
  beats: BeatSpec[];
  brand: {at: number; lines: LineSpec[]; send: LineSpec[]};
  loop: number;
  audio: string;
};

const COLORS: Record<string, string> = {
  ray: NIGHT.ray, danger: NIGHT.danger, great: NIGHT.great, honey: NIGHT.honey, ink: NIGHT.ink, mist: NIGHT.mist, coral: "#F29C86",
};
const col = (c?: string) => (c ? COLORS[c] ?? c : NIGHT.ink);
const clamp = {extrapolateLeft: "clamp", extrapolateRight: "clamp"} as const;
const ease = Easing.bezier(0.65, 0, 0.35, 1);
const ramp = (f: number, a: number, b: number, e = ease) => (a === b ? (f >= a ? 1 : 0) : interpolate(f, [a, b], [0, 1], {...clamp, easing: e}));

/** The board at frame f: every move landed or gliding, captures gone, lifts, the toppled king. */
const boardAt = (r: ReelData, f: number) => {
  const where = new Map<string, string>(); // square -> piece id
  const all = pieces(r.start);
  all.forEach((p) => where.set(p.id, p.id));
  const pos = new Map<string, PieceState>();
  all.forEach((p) => pos.set(p.id, {...p, ...at(p.id)}));
  const shards: Shards[] = [];
  const gone = new Set<string>();
  let last: {from: string; to: string} | null = null;
  for (const m of r.moves) {
    const id = where.get(m.from);
    if (!id) continue;
    const land = m.at + m.dur;
    const capSq = m.capture ?? m.to;
    if (f >= land) {
      const victim = where.get(capSq);
      if (victim && victim !== id) {
        gone.add(victim);
        where.delete(capSq);
        if (m.shatter !== false) shards.push({sq: capSq, t: (f - land) / 26, dark: all.find((p) => p.id === victim)!.code === all.find((p) => p.id === victim)!.code.toLowerCase()});
      }
      where.delete(m.from);
      where.set(m.to, id);
      pos.set(id, {...pos.get(id)!, ...at(m.to), lift: 1 - ramp(f, land, land + 7)});
      last = {from: m.from, to: m.to};
    } else if (f >= m.liftAt || f >= m.at) {
      const t = ramp(f, m.at, land);
      const a = at(m.from), b = at(m.to);
      pos.set(id, {...pos.get(id)!, x: a.x + (b.x - a.x) * t, y: a.y + (b.y - a.y) * t, lift: ramp(f, m.liftAt, m.at)});
      break; // later moves haven't started
    } else break;
  }
  if (r.topple && f >= r.topple.at) {
    const id = where.get(r.topple.sq);
    if (id) {
      const tip = spring({frame: f - r.topple.at, fps: 30, config: {damping: 9, stiffness: 70}});
      pos.set(id, {...pos.get(id)!, tip: 84 * tip, x: pos.get(id)!.x + 0.12 * tip});
    }
  }
  const state = [...pos.values()].filter((p) => !gone.has(p.id));
  const glows: Glow[] = [];
  if (last) glows.push({sq: last.from, color: NIGHT.honey, opacity: 0.26}, {sq: last.to, color: NIGHT.honey, opacity: 0.42});
  return {pieces: state, shards, glows};
};

const camera = (r: ReelData, f: number) => {
  const keys = r.camera.map(([k]) => k);
  const pick = (k: "zoom" | "x" | "y" | "roll" | "tilt") =>
    keys.length === 1 ? r.camera[0][1][k] : interpolate(f, keys, r.camera.map(([, v]) => v[k]), {...clamp, easing: Easing.bezier(0.45, 0, 0.2, 1)});
  return {zoom: pick("zoom"), x: pick("x"), y: pick("y"), roll: pick("roll"), tilt: pick("tilt")};
};

const Scene: React.FC<{r: ReelData; f: number}> = ({r, f}) => {
  const cam = camera(r, f);
  const q = BOARD / 8;
  let sx = 0, sy = 0;
  for (const s of r.shakes) {
    const t = f - s.at;
    if (t >= 0 && t <= 12) { sx += Math.sin(t * 2.7) * s.strength * (1 - t / 12); sy += Math.cos(t * 3.3) * s.strength * (1 - t / 12); }
  }
  // Story beats dim the board; from the brand card on it stays dimmed, so nothing competes with the words.
  const dim = Math.max(0, ramp(f, r.brand.at - 8, r.brand.at + 6),
    ...r.dim.map((d) => ramp(f, d.at - 6, d.at + 10) * (d.until >= r.brand.at ? 1 : 1 - ramp(f, d.until - 10, d.until))));
  const b = boardAt(r, f);
  const rays: Ray[] = r.rays.filter((x) => f >= x.at && f < x.until).map((x) => ({
    from: x.from, to: x.to, color: col(x.color), progress: ramp(f, x.at, x.at + (x.draw ?? 8)) * (1 - ramp(f, x.until - 6, x.until)),
  }));
  const glows: Glow[] = [...b.glows, ...r.glows.filter((g) => f >= g.at && f < g.until).map((g) => ({
    sq: g.sq, color: col(g.color),
    opacity: (g.opacity ?? 0.5) * ramp(f, g.at, g.at + 5) * (1 - ramp(f, g.until - 6, g.until)) * (g.pulse ? 0.8 + 0.2 * Math.sin(f / 3) : 1),
  }))];
  // A trail: the path so far, and every square it has visited glowing.
  let trailPts: {x: number; y: number}[] = [];
  if (r.trail) {
    const {squares, times} = r.trail;
    squares.forEach((sq, i) => {
      if (times[i] <= f) glows.push({sq, color: col(r.trail!.color), opacity: 0.22 + 0.2 * (1 - ramp(f, times[i], times[i] + 10))});
    });
    trailPts = squares.filter((_, i) => times[i] <= f).map((sq) => at(sq));
    const nextI = times.findIndex((t) => t > f);
    if (nextI > 0) {
      const a = at(squares[nextI - 1]), c = at(squares[nextI]);
      const t = ramp(f, times[nextI - 1] + 1, times[nextI]);
      trailPts.push({x: a.x + (c.x - a.x) * t, y: a.y + (c.y - a.y) * t});
    }
  }
  return (
    <AbsoluteFill style={{perspective: 1700, perspectiveOrigin: "50% 40%"}}>
      <div style={{position: "absolute", left: 0, top: 360, width: 1080, height: 1500, filter: `blur(${dim * 7}px) brightness(${1 - dim * 0.55})`,
        transform: `translate(${sx}px, ${sy}px)`}}>
        <div style={{position: "absolute", inset: 0, transformStyle: "preserve-3d",
          transform: `scale(${cam.zoom}) rotateZ(${cam.roll}deg) translate(${(3.5 - cam.x) * q * 0.9}px, ${(3.5 - cam.y) * q * 0.62}px)`}}>
          <Board3D tilt={cam.tilt} pieces={b.pieces} rays={rays} glows={glows} shards={b.shards} light={{x: 0.35, y: 0.3}}
            trail={trailPts.length > 1 ? {points: trailPts, color: col(r.trail!.color)} : undefined}
            marks={r.marks.filter((m) => f >= m.at && f < m.until).map((m) => ({...m, color: col(m.color), t: ramp(f, m.at, m.at + 8, Easing.out(Easing.back(2)))}))}
            ghosts={r.ghosts.filter((g) => f >= g.at && f < g.until).map((g) => {
              const t = g.to && g.glideAt !== undefined ? ramp(f, g.glideAt, g.glideAt + (g.glideDur ?? 12)) : 0;
              const a = at(g.from), c = at(g.to ?? g.from);
              return {code: g.code, x: a.x + (c.x - a.x) * t, y: a.y + (c.y - a.y) * t,
                opacity: 0.75 * ramp(f, g.at, g.at + 6) * (1 - ramp(f, g.until - 6, g.until))};
            })} />
        </div>
      </div>
    </AbsoluteFill>
  );
};

const Word: React.FC<{children: string; at: number; color: string; size: number}> = ({children, at: start, color, size}) => {
  const f = useCurrentFrame();
  const s = start < 0 ? 1 : spring({frame: f - start, fps: 30, config: {damping: 13, stiffness: 190, mass: 0.7}});
  return (
    <span style={{display: "inline-block", marginRight: size * 0.24, color, opacity: Math.min(1, s * 1.4),
      scale: String(0.55 + 0.45 * s), translate: `0 ${(1 - s) * size * 0.35}px`, filter: `blur(${(1 - Math.min(1, s)) * 6}px)`}}>{children}</span>
  );
};

const Line: React.FC<{l: LineSpec; still?: boolean}> = ({l, still}) => (
  <div style={{fontSize: l.size, fontWeight: l.weight ?? 900, lineHeight: 1.06, letterSpacing: -l.size * 0.025, textAlign: "center",
    textShadow: "0 6px 30px rgba(0,0,0,0.7)"}}>
    {l.text.split(" ").map((w, i) => (
      <Word key={i} at={still || l.at < 0 ? -1 : l.at + i * (l.gap ?? 3)} size={l.size}
        color={l.accent && l.accent.split(" ").includes(w) ? col(l.accentColor ?? "honey") : col(l.color)}>{w}</Word>
    ))}
  </div>
);

const Beat: React.FC<{b: BeatSpec; still?: boolean}> = ({b, still}) => {
  const f = useCurrentFrame();
  if (!still && (f < b.from || f >= b.to)) return null;
  return (
    <div style={{position: "absolute", top: b.top ?? 250, left: 80, right: 80, display: "flex", flexDirection: "column", alignItems: "center", gap: 14,
      opacity: still ? 1 : 1 - ramp(f, b.to - 6, b.to)}}>
      {b.lines.filter((l) => still ? l.at < 0 : f >= l.at || l.at < 0).map((l, i) => <Line key={i} l={l} still={still} />)}
    </div>
  );
};

/** The brand, last, and held long enough to read every line. */
const BrandCard: React.FC<{r: ReelData; f: number; fade: number}> = ({r, f, fade}) => {
  const a = r.brand.at;
  const show = ramp(f, a, a + 14);
  return (
    <AbsoluteFill style={{opacity: show * (1 - fade), alignItems: "center", justifyContent: "flex-start", paddingTop: 330,
      background: `radial-gradient(110% 70% at 50% 35%, rgba(30,35,40,0.97), rgba(11,12,14,0.98))`}}>
      <Img src={staticFile("icon.png")} style={{width: 220, height: 220, borderRadius: 50, boxShadow: "0 30px 80px rgba(0,0,0,0.6)",
        scale: String(spring({frame: f - a, fps: 30, config: {damping: 12}}))}} />
      <div style={{height: 46}} />
      {r.brand.lines.map((l, i) => <Line key={i} l={l} />)}
      <div style={{height: 26}} />
      <Line l={{text: "Calm Chess", at: a + 30, size: 76, color: "great"}} />
      <Line l={{text: "Free · No ads · Works offline", at: a + 40, size: 44, color: "mist", weight: 700, gap: 2}} />
      <div style={{height: 44}} />
      {r.brand.send.map((l, i) => <Line key={i} l={l} />)}
    </AbsoluteFill>
  );
};

const BG = `radial-gradient(120% 80% at 50% 38%, #232830 0%, ${NIGHT.bgTop} 45%, ${NIGHT.bgBottom} 100%)`;

export const ChessReel: React.FC<{reel: ReelData; hook?: BeatSpec}> = ({reel, hook}) => {
  const f = useCurrentFrame();
  const beats = hook ? [hook, ...reel.beats.slice(1)] : reel.beats;
  const loop = ramp(f, reel.loop, reel.length - 1);
  const counter = reel.counter;
  const step = counter ? [...counter.steps].reverse().find((s) => s.at <= f) : undefined;
  return (
    <AbsoluteFill style={{background: BG, fontFamily: "Nunito", color: NIGHT.ink}}>
      <Scene r={reel} f={f} />
      {beats.map((b, i) => <Beat key={i} b={b} />)}
      {counter && step && f >= counter.at && f < counter.until && (
        <div style={{position: "absolute", top: counter.top, left: 0, right: 0, textAlign: "center",
          opacity: 1 - ramp(f, counter.until - 6, counter.until)}}>
          <div style={{fontSize: 170, fontWeight: 1000, color: col(counter.color), textShadow: "0 8px 40px rgba(0,0,0,0.7)", fontVariantNumeric: "tabular-nums",
            scale: String(1 + 0.18 * (1 - ramp(f, step.at, step.at + 6, Easing.out(Easing.quad))))}}>{step.text}</div>
          {counter.label && <div style={{fontSize: 46, fontWeight: 800, color: NIGHT.mist, letterSpacing: 3, marginTop: -10}}>{counter.label}</div>}
        </div>
      )}
      {reel.flashes.map((fl, i) => (
        <AbsoluteFill key={i} style={{background: col(fl.color), mixBlendMode: "overlay", pointerEvents: "none",
          opacity: f >= fl.at ? 0.6 * (1 - ramp(f, fl.at, fl.at + 12, Easing.out(Easing.quad))) : 0}} />
      ))}
      {f >= reel.brand.at && <BrandCard r={reel} f={f} fade={loop} />}
      <AbsoluteFill style={{background: "radial-gradient(90% 70% at 50% 45%, transparent 55%, rgba(0,0,0,0.55) 100%)", pointerEvents: "none"}} />
      {loop > 0 && (
        <AbsoluteFill style={{opacity: loop, background: BG}}>
          <Scene r={reel} f={0} />
          <Beat b={beats[0]} still />
        </AbsoluteFill>
      )}
      <Audio src={staticFile(reel.audio)} />
    </AbsoluteFill>
  );
};

// Kept for type completeness of ghosts drawn by the board.
export {GlassPiece};
