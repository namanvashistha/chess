#!/usr/bin/env python3
"""The Calm Chess reels, as scripts.

Each reel is written here like a director's script: the position, the moves
in chess notation with the frame each starts on, lines of light, ghosts,
captions, the camera and the sounds. Every move is checked by python-chess,
so a reel can never show an illegal or wrong move. Running this writes each
reel's data (src/reels/<id>.json, drawn by src/ChessReel.tsx) and its
soundtrack (public/audio/<id>.wav, synthesized here: no samples, no licences).

Run: uv run --with chess reels.py
"""
import json
import math
import random
import struct
import wave
from pathlib import Path

import chess

HERE = Path(__file__).resolve().parent
FPS, RATE = 30, 44100


# ---------- Sound: a small synth ----------

class Sound:
    def __init__(self, frames):
        self.s = [0.0] * int(frames / FPS * RATE)
        self.rng = random.Random(11)

    def _add(self, i, v):
        if 0 <= i < len(self.s):
            self.s[i] += v

    def tone(self, frame, freq, dur, gain, attack=0.005, decay=5.0, shape="sine", bend=0.0):
        s0, phase = int(frame / FPS * RATE), 0.0
        for k in range(int(dur * RATE)):
            t = k / RATE
            phase += 2 * math.pi * freq * (2 ** (bend * t / dur)) / RATE
            w = math.sin(phase)
            if shape == "tri":
                w = 2 / math.pi * math.asin(w)
            self._add(s0 + k, w * gain * min(1.0, t / attack) * math.exp(-decay * t / dur))

    def noise(self, frame, dur, gain, lo=0.0, hi=1.0, curve=2.0, rise=False):
        s0, n, y = int(frame / FPS * RATE), int(dur * RATE), 0.0
        for k in range(n):
            p = k / n
            a = lo + (hi - lo) * (p if rise else 1 - p)
            y += a * (self.rng.uniform(-1, 1) - y)
            self._add(s0 + k, y * gain * ((p ** curve) if rise else (1 - p) ** curve))

    def whoosh(self, frame, dur=0.45, gain=0.55):
        s0, n, y = int(frame / FPS * RATE), int(dur * RATE), 0.0
        for k in range(n):
            p = k / n
            y += (0.02 + 0.5 * math.sin(math.pi * p) ** 2) * (self.rng.uniform(-1, 1) - y)
            self._add(s0 + k, y * gain * math.sin(math.pi * p) ** 1.5)

    def heartbeat(self, frame, gain=0.9):
        for d, g in ((0, 1.0), (6.6, 0.7)):
            self.tone(frame + d, 55, 0.28, gain * g, attack=0.004, decay=7, bend=-0.6)

    def boom(self, frame, gain=1.0):
        self.tone(frame, 70, 2.2, gain, attack=0.003, decay=3.0, bend=-1.4)
        self.tone(frame, 140, 0.35, gain * 0.5, attack=0.002, decay=6)
        self.noise(frame, 0.5, gain * 0.5, lo=0.6, hi=0.6, curve=3)
        self.noise(frame, 2.5, gain * 0.12, lo=0.05, hi=0.05, curve=1.5)

    def shatter(self, frame, gain=0.7):
        self.noise(frame, 0.35, gain * 0.9, lo=0.95, hi=0.95, curve=3)
        for _ in range(18):
            self.tone(frame + self.rng.uniform(0, 13), self.rng.uniform(2400, 6200), self.rng.uniform(0.15, 0.5),
                      gain * self.rng.uniform(0.04, 0.1), decay=6)

    def scratch(self, frame, gain=0.6):
        """A record scratch: the 'wait, what?'"""
        self.noise(frame, 0.12, gain, lo=0.4, hi=0.9, curve=0.5, rise=True)
        self.tone(frame, 900, 0.25, gain * 0.3, decay=3, bend=-2.5, shape="tri")

    def sad(self, frame, gain=0.25):
        """A deflating 'wah wah'."""
        for k, f in enumerate((392.0, 370.0, 349.2)):
            self.tone(frame + k * 9, f, 0.35, gain, attack=0.03, decay=2, shape="tri")
        self.tone(frame + 27, 329.6, 1.2, gain, attack=0.03, decay=2.5, shape="tri", bend=-0.08)

    def drone(self, start, end, gain):
        for f, g in ((55.0, 1.0), (82.4, 0.6), (110.0, 0.35), (164.8, 0.18)):
            a, b = int(start / FPS * RATE), min(len(self.s), int(end / FPS * RATE))
            for i in range(a, b):
                t, span = (i - a) / RATE, (b - a) / RATE
                env = min(1, t / 0.6) * min(1, (span - t) / 0.8)
                self._add(i, math.sin(2 * math.pi * f * t + f) * gain * g * env * (0.8 + 0.2 * math.sin(2 * math.pi * 0.3 * t)))

    def riser(self, start, end, gain=0.35):
        dur = (end - start) / FPS
        self.noise(start, dur, gain, lo=0.02, hi=0.7, curve=2.2, rise=True)
        self.tone(start, 220, dur, gain * 0.17, attack=dur * 0.6, decay=0.2, bend=2.0)

    def chord(self, frame, freqs, dur, gain):
        for k, f in enumerate(freqs):
            self.tone(frame + k, f, dur, gain, attack=0.05, decay=2.2, shape="tri")

    def chime(self, frame, gain=0.09):
        """The app's great-move chime."""
        for k, f in enumerate((784.0, 988.0, 1318.5)):
            self.tone(frame + k * 2, f, 0.6, gain, decay=4)

    def brand(self, frame):
        self.chime(frame + 6)
        self.chord(frame + 15, [261.6, 329.6, 392.0, 523.3], 3.5, 0.07)

    def write(self, path, frames):
        peak = max(abs(v) for v in self.s) or 1
        end = frames / FPS
        with wave.open(str(path), "wb") as w:
            w.setnchannels(2)
            w.setsampwidth(2)
            w.setframerate(RATE)
            buf = bytearray()
            for i, v in enumerate(self.s):
                t = i / RATE
                g = 1.0 if t < end - 0.5 else max(0.0, (end - t) / 0.5)
                x = int(max(-1, min(1, v * 0.9 / peak * g)) * 32767)
                buf += struct.pack("<hh", x, x)
            w.writeframes(bytes(buf))


# ---------- Reels ----------

class Reel:
    """A reel being scripted. Moves are played on a real board, so they must be legal."""

    def __init__(self, id, fen, length, brand_at):
        self.board = chess.Board(fen)
        self.d = {"id": id, "length": length, "start": self.board.board_fen(), "moves": [], "rays": [], "glows": [], "marks": [],
                  "ghosts": [], "camera": [], "shakes": [], "flashes": [], "dim": [], "beats": [], "loop": length - 16,
                  "audio": f"audio/{id}.wav", "brand": {"at": brand_at, "lines": [], "send": []}}
        self.sound = Sound(length)

    def move(self, san, at, dur=12, lift=8, shatter=True):
        """Plays a move, starting at frame `at`; returns the frame it lands. lift < 0 means already lifted at frame 0."""
        m = self.board.parse_san(san)
        cap = None
        if self.board.is_en_passant(m):
            cap = chess.square_name(m.to_square + (-8 if self.board.turn == chess.WHITE else 8))
        entry = {"from": chess.square_name(m.from_square), "to": chess.square_name(m.to_square), "at": at, "dur": dur,
                 "liftAt": at - lift if lift >= 0 else lift, "shatter": shatter}
        if cap:
            entry["capture"] = cap
        self.d["moves"].append(entry)
        self.board.push(m)
        return at + dur

    def ray(self, a, b, color, at, until, draw=8):
        self.d["rays"].append({"from": a, "to": b, "color": color, "at": at, "until": until, "draw": draw})

    def glow(self, sq, color, at, until, opacity=0.5, pulse=False):
        self.d["glows"].append({"sq": sq, "color": color, "at": at, "until": until, "opacity": opacity, "pulse": pulse})

    def mark(self, sq, kind, color, at, until):
        self.d["marks"].append({"sq": sq, "kind": kind, "color": color, "at": at, "until": until})

    def ghost(self, code, frm, at, until, to=None, glide_at=None, glide_dur=12):
        g = {"code": code, "from": frm, "at": at, "until": until}
        if to:
            g.update({"to": to, "glideAt": glide_at, "glideDur": glide_dur})
        self.d["ghosts"].append(g)

    def camera(self, frame, zoom, x, y, roll=0, tilt=30):
        self.d["camera"].append([frame, {"zoom": zoom, "x": x, "y": y, "roll": roll, "tilt": tilt}])

    def beat(self, frm, to, *lines, top=250):
        self.d["beats"].append({"from": frm, "to": to, "top": top, "lines": [line(*l) if isinstance(l, tuple) else l for l in lines]})

    def brand(self, *lines, send):
        a = self.d["brand"]["at"]
        self.d["brand"]["lines"] = [line(t, a + 8 + i * 8, size, **kw) for i, (t, size, kw) in enumerate(lines)]
        self.d["brand"]["send"] = [line(t, a + 54 + i * 6, 56, weight=800, gap=2) for i, t in enumerate(send)]
        self.sound.brand(a)

    def write(self):
        assert self.d["brand"]["at"] + 140 <= self.d["loop"], f"{self.d['id']}: the brand needs at least 140 frames on screen"
        (HERE / "src/reels").mkdir(exist_ok=True)
        (HERE / f"src/reels/{self.d['id']}.json").write_text(json.dumps(self.d, indent=1))
        self.sound.write(HERE / f"public/audio/{self.d['id']}.wav", self.d["length"])
        print(f"{self.d['id']}: {self.d['length'] / FPS:.1f} s")


def line(text, at, size, color=None, accent=None, accent_color=None, gap=3, weight=900):
    l = {"text": text, "at": at, "size": size, "gap": gap, "weight": weight}
    if color:
        l["color"] = color
    if accent:
        l["accent"] = accent
    if accent_color:
        l["accentColor"] = accent_color
    return l


def en_passant():
    """'This capture is LEGAL.' Chess's secret rule: a pawn takes a pawn that isn't in front of it."""
    r = Reel("en-passant", "4k3/3p4/8/4P3/8/8/8/4K3 b - - 0 1", length=420, brand_at=262)
    s = r.sound
    r.camera(0, 1.55, 3.6, 2.6, -2, 29)
    r.camera(60, 1.7, 3.5, 2.3, 0, 29)
    r.camera(150, 1.4, 3.5, 2.6, 1, 31)
    r.camera(262, 1.0, 3.5, 3.5, 0, 36)
    jump = r.move("d5", 3, dur=14, lift=-20)                      # the black pawn jumps two squares
    s.whoosh(1, 0.5)
    r.beat(0, 55, ("This capture", -1, 104), line("is LEGAL.", -1, 136, accent="LEGAL.", accent_color="great"))
    r.ray("e5", "d6", "ray", 26, 70)
    r.mark("d6", "ring", "ray", 26, 70)
    land = r.move("exd6", 44, dur=12)                              # en passant
    s.whoosh(42, 0.4)
    s.shatter(land)
    s.scratch(land + 2)
    r.d["shakes"].append({"at": land, "strength": 18})
    r.d["flashes"].append({"at": land, "color": "white"})
    r.beat(land + 1, 108, line("WAIT.", land + 2, 150, color="coral"), line("WHAT?", land + 10, 150, color="coral"))
    # Why: the pawn jumped past him, so he can take it as if it had stepped once.
    r.ghost("p", "d7", 108, 196, to="d5", glide_at=112, glide_dur=14)
    r.ray("d7", "d5", "ray", 112, 196, draw=14)
    r.mark("d6", "ring", "great", 126, 196)
    r.beat(108, 196, line("It jumped past him…", 110, 84, gap=3), line("so he took it anyway.", 140, 84, gap=3, accent="anyway.", accent_color="great"))
    s.chime(126, 0.06)
    r.beat(196, 262, line("It's called", 198, 84), line("EN PASSANT.", 206, 128, color="great"),
           line("French for “in passing”.", 226, 52, color="mist", weight=800, gap=2), top=560)
    r.d["dim"].append({"at": 196, "until": 262})
    s.chord(196, [261.6, 329.6, 392.0], 2.5, 0.06)
    r.brand(("Learn the rules", 96, {}), ("nobody told you.", 96, {"accent": "nobody", "accent_color": "great"}),
            send=["Send this to the friend", "who'd call it cheating."])
    s.drone(0, 60, 0.07)
    r.write()


def stalemate():
    """'He has a QUEEN and a ROOK. Watch him throw it away.' Qb6 leaves the king no move, and no check: a draw."""
    r = Reel("stalemate", "k7/2K5/8/8/3Q4/8/8/7R w - - 0 1", length=440, brand_at=282)
    s = r.sound
    r.camera(0, 1.25, 2.4, 3.0, -2, 30)
    r.camera(40, 1.55, 1.6, 1.2, 0, 31)
    r.camera(176, 1.25, 3.5, 2.2, 1, 32)
    r.camera(282, 1.0, 3.5, 3.5, 0, 36)
    s.drone(0, 120, 0.09)
    land = r.move("Qb6", 8, dur=16, lift=-20)
    s.whoosh(6, 0.55)
    s.tone(land, 880, 0.5, 0.1, decay=4)
    r.beat(0, 46, ("He has a QUEEN", -1, 100, None, "QUEEN"), ("and a ROOK.", -1, 100, None, "ROOK."))
    r.beat(46, 120, line("Watch him", 46, 92), line("THROW it away.", 52, 120, color="coral"))
    # The king looks for a square: every one is covered.
    for k, sq in enumerate(["a7", "b7", "b8"]):
        f = 62 + k * 14
        r.ray("a8", sq, "danger", f, 150, draw=6)
        r.mark(sq, "x", "danger", f + 4, 150)
        s.tone(f + 4, 330 - k * 30, 0.2, 0.18, decay=6)
    r.beat(120, 176, line("His king can't move…", 122, 80), line("but it's NOT in check.", 140, 80, accent="NOT", accent_color="coral"))
    r.glow("a8", "great", 140, 176, 0.35)
    stal = 176
    s.sad(stal + 4)
    r.beat(stal, 230, line("STALEMATE.", stal, 140, color="mist"), line("It's a DRAW.", stal + 14, 104, accent="DRAW.", accent_color="coral"), top=520)
    r.d["dim"].append({"at": stal, "until": 232})
    # What wins: Rh8 was mate.
    r.ghost("R", "h1", 236, 282, to="h8", glide_at=240, glide_dur=12)
    r.ray("h8", "a8", "danger", 252, 282)
    r.glow("a8", "danger", 252, 282, 0.55, pulse=True)
    s.whoosh(238, 0.4)
    s.tone(252, 1318.5, 0.6, 0.08, decay=4)
    r.beat(232, 282, line("Rh8 was checkmate.", 236, 84, accent="checkmate.", accent_color="great"))
    r.brand(("In Calm Chess,", 96, {}), ("you'd just undo.", 96, {"accent": "undo.", "accent_color": "great"}),
            send=["Send this to the friend", "who always does this."])
    r.write()


def knights_tour():
    """'This knight will land on EVERY square. Once.' A knight's tour, faster and faster, painting the board."""
    path = "b1 a3 c2 a1 b3 c1 a2 b4 a6 b8 d7 f8 h7 g5 h3 g1 e2 c3 d1 b2 a4 c5 b7 a5 c4 d2 f1 h2 f3 e1 g2 h4 g6 h8 f7 d8 c6 e5 d3 f2 h1 g3 e4 d6 b5 a7 c8 b6 a8 c7 e8 g7 h5 f4 e6 d4 f5 e3 g4 h6 g8 f6 d5 e7".split()
    r = Reel("knights-tour", "8/8/8/8/8/8/8/1N6 w - - 0 1", length=450, brand_at=292)
    s = r.sound
    r.camera(0, 1.05, 3.5, 3.6, -2, 32)
    r.camera(240, 1.0, 3.5, 3.5, 2, 30)
    r.camera(292, 0.96, 3.5, 3.5, 0, 36)
    times, f = [0], 4
    notes = [261.6, 293.7, 329.6, 392.0, 440.0]
    for i, sq in enumerate(path[1:]):
        dur = max(2, round(10 * 0.935 ** i))
        r.board.turn = chess.WHITE  # the knight moves alone: it's always its turn
        land = r.move(f"N{sq}", f, dur=dur, lift=-20 if i == 0 else 3, shatter=False)
        times.append(land)
        n = notes[i % 5] * (2 ** (i // 5 / 4))
        s.tone(land, n, 0.25, 0.07 + 0.05 * i / 63, decay=5)
        f = land + 1
    end = times[-1]
    r.d["trail"] = {"squares": path, "times": times, "color": "great"}
    r.d["counter"] = {"at": 60, "until": end + 26, "top": 230, "color": "great", "label": "SQUARES",
                      "steps": [{"at": t, "text": f"{i + 1}"} for i, t in enumerate(times)]}
    r.beat(0, 60, ("This knight will land on", -1, 76), line("EVERY square.", -1, 124, accent="EVERY", accent_color="great"))
    r.d["flashes"].append({"at": end, "color": "white"})
    s.chord(end, [261.6, 329.6, 392.0, 523.3, 659.3], 3.0, 0.08)
    s.boom(end, 0.5)
    r.beat(end + 26, 292, line("Every square. Once.", end + 26, 92, accent="Once.", accent_color="great"),
           line("It's called a Knight's Tour.", end + 40, 64, color="mist", weight=800, gap=2))
    r.brand(("Chess is prettier", 96, {}), ("than you think.", 96, {"accent": "prettier", "accent_color": "great"}),
            send=["Send this to someone", "who needs something satisfying."])
    r.write()


def fools_mate():
    """'You can lose at chess in 2 moves.' Fool's Mate, played out in three seconds."""
    r = Reel("fools-mate", chess.STARTING_FEN, length=400, brand_at=244)
    s = r.sound
    r.camera(0, 1.12, 4.8, 4.6, -2, 31)
    r.camera(60, 1.3, 5.6, 4.8, 0, 31)
    r.camera(120, 1.18, 4.5, 4.0, 1, 33)
    r.camera(244, 1.0, 3.5, 3.5, 0, 36)
    r.beat(0, 96, ("You can lose at chess", -1, 84), line("in 2 MOVES.", -1, 130, accent="2 MOVES.", accent_color="coral"))
    t = r.move("f3", 4, dur=9, lift=-20); s.tone(t, 880, 0.2, 0.1, decay=6)
    r.beat(14, 40, line("Move 1.", 14, 64, color="mist"), top=600)
    t = r.move("e5", 22, dur=9); s.tone(t, 660, 0.2, 0.08, decay=6)
    t = r.move("g4", 40, dur=9); s.tone(t, 880, 0.2, 0.1, decay=6)
    r.beat(42, 64, line("Move 2.", 42, 64, color="mist"), top=600)
    s.whoosh(58, 0.45)
    mate = r.move("Qh4#", 60, dur=10)
    r.ray("h4", "e1", "danger", mate, 150)
    r.glow("e1", "danger", mate, 150, 0.6, pulse=True)
    r.d["topple"] = {"sq": "e1", "at": mate + 18}
    r.d["shakes"].append({"at": mate, "strength": 22})
    r.d["flashes"].append({"at": mate, "color": "#FF5050"})
    s.boom(mate)
    r.beat(96, 150, line("CHECKMATE.", mate + 26, 140, color="coral"))
    r.beat(150, 244, line("It's called", 152, 70, color="mist", weight=800), line("Fool's Mate.", 158, 120),
           line("The fastest loss in chess.", 178, 60, color="mist", weight=800, gap=2), top=520)
    r.d["dim"].append({"at": 150, "until": 244})
    s.chord(150, [220.0, 261.6, 329.6], 2.5, 0.06)
    r.brand(("Don't be this guy.", 96, {"accent": "guy.", "accent_color": "coral"}), ("Learn calmly.", 96, {"accent": "calmly.", "accent_color": "great"}),
            send=["Send this to the friend", "who'd play f3."])
    s.drone(0, 60, 0.07)
    r.write()


def opera():
    """'He gave away his QUEEN. On purpose.' Morphy's Opera Game, Paris 1858: Qb8+! Nxb8 Rd8#."""
    r = Reel("opera", "4kb1r/p2n1ppp/4q3/4p1B1/4P3/1Q6/PPP2PPP/2KR4 w k - 0 16", length=480, brand_at=318)
    s = r.sound
    for f, z, x, y, roll, tilt in [(0, 1.2, 1.5, 3.2, -2, 30), (18, 1.28, 2.6, 1.4, -1, 30), (64, 1.24, 2.2, 1.2, 0, 31),
                                   (79, 1.36, 1.6, 1.0, 1, 31), (110, 1.2, 3.5, 4.0, 0, 32), (142, 1.32, 3.5, 6.2, 2, 32),
                                   (152, 1.42, 4.0, 1.2, -1, 30), (190, 1.04, 3.5, 3.4, 0, 34), (318, 0.92, 3.5, 3.5, 0, 38)]:
        r.camera(f, z, x, y, roll, tilt)
    s.drone(0, 152, 0.10)
    for k in range(4):
        s.heartbeat(1.5 + k * 25.5)
    lands = r.move("Qb8+", 6, dur=12, lift=-20)
    s.whoosh(4, 0.5)
    s.tone(lands, 880, 0.5, 0.12, decay=4)
    s.tone(lands, 1318.5, 0.6, 0.08, decay=4)
    r.ray("b3", "b8", "ray", -10, lands + 10)
    r.ray("b8", "e8", "danger", lands, 64)
    r.glow("e8", "danger", lands, 64, 0.5)
    r.beat(0, 64, ("He gave away", -1, 92), line("his QUEEN.", -1, 124, accent="QUEEN."),
           line("On purpose.", 30, 96, color="great", gap=4))
    s.tone(30, 659.3, 0.9, 0.08, attack=0.02, decay=3, shape="tri")
    shatter = r.move("Nxb8", 64, dur=9)
    s.whoosh(63, 0.4, 0.5)
    s.shatter(shatter)
    s.tone(shatter, 98, 0.6, 0.5, decay=6, bend=-0.5)
    r.d["shakes"].append({"at": shatter, "strength": 14})
    r.beat(76, 110, line("Free queen?", 76, 118))
    for k in range(3):
        s.heartbeat(110 + k * 16.5, 0.8)
    r.beat(110, 152, line("Big mistake.", 112, 124, color="coral", gap=5))
    r.ray("d1", "d8", "ray", 132, 158)
    s.riser(114, 152)
    mate = r.move("Rd8#", 142, dur=10, lift=10)
    s.whoosh(140, 0.38, 0.6)
    s.boom(mate)
    r.d["shakes"].append({"at": mate, "strength": 24})
    r.d["flashes"].append({"at": mate, "color": "white"})
    r.ray("d8", "e8", "danger", mate, 216)
    r.ray("g5", "d8", "great", mate + 10, 216)
    r.glow("e8", "danger", mate, 216, 0.6, pulse=True)
    r.d["topple"] = {"sq": "e8", "at": 170}
    s.tone(170 + 7.5, 196, 0.25, 0.35, decay=8)
    s.tone(170 + 10.8, 165, 0.2, 0.2, decay=8)
    r.beat(152, 216, line("CHECKMATE.", 152, 150, color="great"), line("Two moves later.", 176, 64, color="mist", weight=800))
    r.d["dim"].append({"at": 216, "until": 318})
    r.beat(216, 270, line("Paul Morphy.", 218, 120), line("Paris, 1858.", 226, 84, color="honey"), line("At the opera.", 236, 84, color="mist"), top=620)
    s.chord(216, [261.6, 329.6, 392.0], 3.5, 0.07)
    s.tone(216, 1046.5, 2.5, 0.06, decay=3)
    r.beat(270, 318, line("Still the most famous", 272, 84, gap=2), line("checkmate ever played.", 280, 84, gap=2), top=660)
    s.chord(270, [220.0, 277.2, 329.6, 440.0], 3.0, 0.06)
    r.brand(("Learn to see", 100, {}), ("moves like this.", 100, {"accent": "this.", "accent_color": "great"}),
            send=["Send this to the friend", "who'd take the queen."])
    r.write()


if __name__ == "__main__":
    for make in (opera, en_passant, stalemate, knights_tour, fools_mate):
        make()
    ids = sorted(p.stem for p in (HERE / "src/reels").glob("*.json"))
    (HERE / "src/reels/index.ts").write_text(
        "// Written by reels.py.\n" + "".join(f'import r{i} from "./{id}.json";\n' for i, id in enumerate(ids))
        + "\nexport const REELS = [" + ", ".join(f"r{i}" for i in range(len(ids))) + "];\n")
