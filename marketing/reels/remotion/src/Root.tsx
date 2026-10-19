import React from "react";
import {Composition, staticFile} from "remotion";
import {loadFont} from "@remotion/fonts";
import {BeatSpec, ChessReel, ReelData} from "./ChessReel";
import {REELS} from "./reels";

// Nunito, the open rounded face the Android app uses (SIL Open Font License).
loadFont({family: "Nunito", url: staticFile("fonts/Nunito.ttf"), weight: "200 1000"});

/** Other opening hooks for the Opera Game, to test against "He gave away his QUEEN." */
const OPERA_HOOKS: Record<string, BeatSpec> = {
  best: {from: 0, to: 64, top: 250, lines: [
    {text: "The best move here:", at: -1, size: 92}, {text: "give away your QUEEN.", at: -1, size: 112, accent: "QUEEN."}]},
  twoMoves: {from: 0, to: 64, top: 250, lines: [
    {text: "This queen sacrifice", at: -1, size: 92}, {text: "ends it in 2 moves.", at: -1, size: 112, accent: "2 moves."}]},
};

export const RemotionRoot: React.FC = () => (
  <>
    {(REELS as unknown as ReelData[]).map((r) => (
      <Composition key={r.id} id={r.id} component={ChessReel} durationInFrames={r.length} fps={30} width={1080} height={1920}
        defaultProps={{reel: r}} />
    ))}
    {Object.entries(OPERA_HOOKS).map(([name, hook]) => (
      <Composition key={name} id={`opera-${name}`} component={ChessReel} durationInFrames={(REELS as unknown as ReelData[]).find((r) => r.id === "opera")!.length}
        fps={30} width={1080} height={1920} defaultProps={{reel: (REELS as unknown as ReelData[]).find((r) => r.id === "opera")!, hook}} />
    ))}
  </>
);
