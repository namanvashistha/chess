import React, {useId} from "react";
import {ART, PLINTH} from "./chess";

/**
 * One chess piece in glass, exactly as the app draws it: milk glass for
 * White, smoked glass for Black, lit from the top left, with a fine contour
 * that keeps it clear of both squares.
 */
export const GlassPiece: React.FC<{code: string; size: number; style?: React.CSSProperties}> = ({code, size, style}) => {
  const id = useId().replace(/:/g, "");
  const white = code === code.toUpperCase();
  const art = ART[code.toLowerCase()];
  const shape = PLINTH + art.body;
  // The art box, padded and a touch larger, standing on its plinth.
  const place = "translate(50 85.7) scale(1.05) translate(-50 -85.7) translate(4 2) scale(0.92)";
  return (
    <svg viewBox="0 0 100 100" width={size} height={size} style={{overflow: "visible", ...style}}>
      <defs>
        <linearGradient id={`fill${id}`} x1="0" y1="0" x2="1" y2="1">
          {white ? (
            <>
              <stop offset="0" stopColor="#FFFFFF" />
              <stop offset="0.45" stopColor="#F9F7F5" />
              <stop offset="1" stopColor="#CCD0D6" />
            </>
          ) : (
            <>
              <stop offset="0" stopColor="#383C42" />
              <stop offset="0.5" stopColor="#212329" />
              <stop offset="1" stopColor="#131418" />
            </>
          )}
        </linearGradient>
        <radialGradient id={`hi${id}`} cx="0.34" cy="0.22" r="0.42">
          <stop offset="0" stopColor="#FFFFFF" stopOpacity={white ? 0.9 : 0.16} />
          <stop offset="1" stopColor="#FFFFFF" stopOpacity={0} />
        </radialGradient>
        <linearGradient id={`lit${id}`} x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor={white ? "#FFFFFF" : "#E8EDF5"} stopOpacity={white ? 1 : 0.4} />
          <stop offset="0.55" stopColor={white ? "#FFFFFF" : "#E8EDF5"} stopOpacity={white ? 0.25 : 0.08} />
          <stop offset="1" stopColor="#FFFFFF" stopOpacity={0} />
        </linearGradient>
        <linearGradient id={`shade${id}`} x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#293036" stopOpacity={0} />
          <stop offset="0.5" stopColor="#293036" stopOpacity={0.32} />
          <stop offset="1" stopColor="#293036" stopOpacity={white ? 0.5 : 0.6} />
        </linearGradient>
        <clipPath id={`clip${id}`}>
          <path d={shape} />
        </clipPath>
      </defs>
      <g transform={place}>
        <path d={shape} fill={white ? "rgba(255,255,255,0.8)" : "rgba(0,0,0,0.8)"} />
        <path
          d={shape}
          fill={`url(#fill${id})`}
          style={{filter: `drop-shadow(0 0 1.6px ${white ? "rgba(0,0,0,0.58)" : "rgba(255,255,255,0.46)"})`}}
        />
        <g clipPath={`url(#clip${id})`}>
          <rect x="0" y="0" width="100" height="100" fill={`url(#hi${id})`} style={{mixBlendMode: "screen"}} />
          <path d={shape} fill="none" stroke={`url(#lit${id})`} strokeWidth={6} />
          <path d={shape} fill="none" stroke={`url(#shade${id})`} strokeWidth={6} />
        </g>
        <path d={shape} fill="none" stroke={white ? "rgba(0,0,0,0.6)" : "rgba(255,255,255,0.38)"} strokeWidth={1.7} strokeLinejoin="round" />
        {art.detail && (
          <path d={art.detail} fill="none" stroke={white ? "rgba(102,102,102,0.6)" : "rgba(230,230,230,0.45)"} strokeWidth={1.6} strokeLinecap="round" />
        )}
        {art.eye && <circle cx={art.eye[0]} cy={art.eye[1]} r={art.eye[2]} fill={white ? "rgba(64,64,64,0.7)" : "rgba(235,235,235,0.65)"} />}
      </g>
    </svg>
  );
};
