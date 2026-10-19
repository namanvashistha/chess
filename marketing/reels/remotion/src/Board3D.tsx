import React from "react";
import {at, NIGHT} from "./chess";
import {GlassPiece} from "./GlassPiece";

export type PieceState = {
  id: string;
  code: string;
  /** Column and row on the board, fractional while it glides. */
  x: number;
  y: number;
  /** How far it's lifted off the board, 0 to 1. */
  lift?: number;
  /** Degrees it has tipped over (a toppled king). */
  tip?: number;
  opacity?: number;
};

export type Ray = {from: string; to: string; color: string; progress: number; width?: number; knight?: boolean};
export type Glow = {sq: string; color: string; opacity: number};
export type Shards = {sq: string; t: number; dark: boolean};
export type Mark = {sq: string; kind: "x" | "ring"; color: string; t: number};
export type Ghost = {code: string; x: number; y: number; opacity: number};

export const BOARD = 860;
const Q = BOARD / 8;

/**
 * The board in three dimensions: the plane tilts away from the viewer and
 * the glass pieces stand upright on it, each with its shadow on the stone.
 * Lines of light, glows and glass shards lie on the board or rise from it.
 */
export const Board3D: React.FC<{
  pieces: PieceState[];
  tilt: number;
  rays?: Ray[];
  glows?: Glow[];
  shards?: Shards[];
  light?: {x: number; y: number};
  trail?: {points: {x: number; y: number}[]; color: string};
  marks?: Mark[];
  ghosts?: Ghost[];
}> = ({pieces, tilt, rays = [], glows = [], shards = [], light = {x: 0.4, y: 0.35}, trail, marks = [], ghosts = []}) => {
  return (
    <div style={{position: "absolute", left: "50%", top: "50%", width: BOARD, height: BOARD, marginLeft: -BOARD / 2, marginTop: -BOARD / 2,
      transformStyle: "preserve-3d", transform: `rotateX(${tilt}deg)`}}>
      {/* The stone slab: its edge, then its face. */}
      <div style={{position: "absolute", inset: -26, borderRadius: 22, background: "linear-gradient(160deg, #3A4040, #1E2222)",
        boxShadow: "0 60px 120px rgba(0,0,0,0.75), 0 0 0 2px rgba(255,255,255,0.05) inset", transform: "translateZ(-2px)"}} />
      <svg width={BOARD} height={BOARD} style={{position: "absolute", inset: 0, borderRadius: 6}}>
        <defs>
          <radialGradient id="spot" cx={light.x} cy={light.y} r="0.85">
            <stop offset="0" stopColor="#FFF4DC" stopOpacity="0.42" />
            <stop offset="0.55" stopColor="#FFF4DC" stopOpacity="0.06" />
            <stop offset="1" stopColor="#000" stopOpacity="0.45" />
          </radialGradient>
          <filter id="grain">
            <feTurbulence type="fractalNoise" baseFrequency="1.6" numOctaves="2" seed="7" />
            <feColorMatrix values="0 0 0 0 0  0 0 0 0 0  0 0 0 0 0  0 0 0 0.18 0" />
          </filter>
          <filter id="rayglow" x="-20%" y="-20%" width="140%" height="140%">
            <feGaussianBlur stdDeviation="6" result="b" />
            <feMerge><feMergeNode in="b" /><feMergeNode in="SourceGraphic" /></feMerge>
          </filter>
        </defs>
        {Array.from({length: 64}, (_, i) => (
          <rect key={i} x={(i % 8) * Q} y={Math.floor(i / 8) * Q} width={Q} height={Q}
            fill={(Math.floor(i / 8) + (i % 8)) % 2 ? NIGHT.dark : NIGHT.light} />
        ))}
        <rect width={BOARD} height={BOARD} filter="url(#grain)" />
        {glows.map((g, i) => {
          const p = at(g.sq);
          return <rect key={i} x={p.x * Q} y={p.y * Q} width={Q} height={Q} fill={g.color} opacity={g.opacity} />;
        })}
        <rect width={BOARD} height={BOARD} fill="url(#spot)" style={{mixBlendMode: "soft-light"}} />
        <rect width={BOARD} height={BOARD} fill="url(#spot)" opacity={0.5} />
        {trail && (
          <polyline filter="url(#rayglow)" fill="none" stroke={trail.color} strokeWidth={Q * 0.05} strokeLinecap="round" strokeLinejoin="round" opacity={0.9}
            points={trail.points.map((p) => `${(p.x + 0.5) * Q},${(p.y + 0.5) * Q}`).join(" ")} />
        )}
        {trail && trail.points.map((p, i) => <circle key={`d${i}`} cx={(p.x + 0.5) * Q} cy={(p.y + 0.5) * Q} r={Q * 0.06} fill={trail.color} />)}
        {marks.map((m, i) => {
          const p = at(m.sq), cx = (p.x + 0.5) * Q, cy = (p.y + 0.5) * Q, r = Q * 0.3 * m.t;
          return m.kind === "x" ? (
            <g key={`m${i}`} stroke={m.color} strokeWidth={Q * 0.09} strokeLinecap="round" filter="url(#rayglow)">
              <line x1={cx - r} y1={cy - r} x2={cx + r} y2={cy + r} /><line x1={cx + r} y1={cy - r} x2={cx - r} y2={cy + r} />
            </g>
          ) : <circle key={`m${i}`} cx={cx} cy={cy} r={Q * 0.4 * m.t} fill="none" stroke={m.color} strokeWidth={Q * 0.06} filter="url(#rayglow)" />;
        })}
        {rays.map((r, i) => {
          const a = at(r.from), b = at(r.to);
          const ax = (a.x + 0.5) * Q, ay = (a.y + 0.5) * Q, bx = (b.x + 0.5) * Q, by = (b.y + 0.5) * Q;
          const len = Math.hypot(bx - ax, by - ay);
          return (
            <g key={i} filter="url(#rayglow)" opacity={Math.min(1, r.progress * 3)}>
              <line x1={ax} y1={ay} x2={bx} y2={by} stroke={r.color} strokeWidth={r.width ?? Q * 0.085} strokeLinecap="round"
                strokeDasharray={len} strokeDashoffset={len * (1 - r.progress)} />
              <circle cx={bx} cy={by} r={Q * 0.13 * Math.min(1, Math.max(0, r.progress * 1.4 - 0.4))} fill={r.color} />
            </g>
          );
        })}
      </svg>
      {/* Shadows on the stone, under each standing piece. */}
      {pieces.map((p) => (
        <div key={`s${p.id}`} style={{position: "absolute", left: (p.x + 0.5) * Q - Q * 0.42, top: (p.y + 0.62) * Q - Q * 0.14,
          width: Q * 0.84, height: Q * 0.28, borderRadius: "50%", opacity: (p.opacity ?? 1) * (0.55 - 0.3 * (p.lift ?? 0)),
          background: "radial-gradient(closest-side, rgba(0,0,0,0.85), rgba(0,0,0,0))",
          transform: `translateZ(1px) scale(${1 + 0.35 * (p.lift ?? 0)})`}} />
      ))}
      {/* The pieces, upright: each turns back toward the viewer by the board's tilt. */}
      {pieces.map((p) => (
        <div key={p.id} style={{position: "absolute", left: (p.x + 0.5) * Q, top: (p.y + 0.62) * Q, width: 0, height: 0,
          transformStyle: "preserve-3d", transform: `translateZ(${2 + (p.lift ?? 0) * Q * 0.55}px)`, opacity: p.opacity ?? 1}}>
          <div style={{position: "absolute", left: -Q * 0.62, top: -Q * 1.18, width: Q * 1.24, height: Q * 1.24,
            transformOrigin: "50% 92%", transform: `rotateX(${-tilt}deg) rotateZ(${p.tip ?? 0}deg)`}}>
            <GlassPiece code={p.code} size={Q * 1.24} style={{filter: "drop-shadow(0 10px 14px rgba(0,0,0,0.45))"}} />
          </div>
        </div>
      ))}
      {/* Ghosts: a piece that isn't there, showing what could have been. */}
      {ghosts.map((g, i) => (
        <div key={`g${i}`} style={{position: "absolute", left: (g.x + 0.5) * Q, top: (g.y + 0.62) * Q, width: 0, height: 0,
          transformStyle: "preserve-3d", transform: "translateZ(60px)", opacity: g.opacity}}>
          <div style={{position: "absolute", left: -Q * 0.62, top: -Q * 1.18, width: Q * 1.24, height: Q * 1.24, transformOrigin: "50% 92%",
            transform: `rotateX(${-tilt}deg)`}}>
            <GlassPiece code={g.code} size={Q * 1.24} style={{filter: "drop-shadow(0 0 18px rgba(141,203,176,0.9))"}} />
          </div>
        </div>
      ))}
      {/* A taken piece breaking into glass, rising off the board. */}
      {shards.map((s, i) => <ShardBurst key={i} {...s} />)}
    </div>
  );
};

const ShardBurst: React.FC<Shards> = ({sq, t, dark}) => {
  if (t <= 0 || t >= 1) return null;
  const c = at(sq);
  const bits = Array.from({length: 22}, (_, k) => {
    const a = (k / 22) * Math.PI * 2 + Math.sin(k * 12.9898) * 0.6;
    const d = 0.7 + ((k * 37) % 10) / 10;
    return {dx: Math.cos(a) * d, dy: Math.sin(a) * d, up: 1.2 + ((k * 53) % 10) / 6, rot: ((k * 71) % 360) - 180, s: 0.16 + ((k * 13) % 10) / 60};
  });
  const ease = 1 - Math.pow(1 - t, 3);
  return (
    <>
      {bits.map((b, k) => (
        <div key={k} style={{position: "absolute", left: (c.x + 0.5 + b.dx * ease) * Q, top: (c.y + 0.5 + b.dy * ease) * Q,
          width: Q * b.s, height: Q * b.s, opacity: 1 - t,
          clipPath: "polygon(50% 0, 100% 70%, 20% 100%)",
          background: dark ? "linear-gradient(135deg, #8A8F98, #15171A)" : "linear-gradient(135deg, #FFFFFF, #C9CED6)",
          transform: `translateZ(${Q * b.up * Math.sin(Math.PI * Math.min(1, t * 1.3))}px) rotate(${b.rot * t * 3}deg)`,
          boxShadow: "0 0 12px rgba(255,255,255,0.35)"}} />
      ))}
    </>
  );
};
