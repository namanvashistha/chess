#!/usr/bin/env python3
"""Composes the "Mate in 1. Easy." reel from a recording of the real app.

The gameplay is recorded, not faked: the iOS app's debug reel studio
(ios/Chess/Reel.swift) plays the scene on the iPhone simulator. This lays the
story over it at 1080 x 1920:

  Hook: "Mate in 1. Easy." over a real Scholar's Mate position.
  The queen is picked up; its lines even show f7.
  Qxh7??, the queen hangs. "bro."
  What every other chess app does: a red, shaking BLUNDER and a buzzer.
  Silence, then the Calm Chess coach: "Oops. Undo and try again."
  Undo. No judgment. "Now find it." Qxf7#: Great move, the king falls.
  The win, and the line that gets it sent to a friend.
  End card, then it fades into its own first frame, so it loops.

Caption, sound and tap timings all follow the beats measured in the recording
(the --pick, --blunder... options), so a re-recording only needs new beats.

Captions sit between Instagram's top bar and its bottom caption area; the
board and the coach card stay clear of both.

Usage: python mate_in_one.py <recording.mov> <out.mp4> [--start 5.0]
Needs Pillow and ffmpeg.
"""
import argparse
import math
import random
import shutil
import struct
import subprocess
import tempfile
import wave
from pathlib import Path

from PIL import Image, ImageDraw, ImageFilter, ImageFont

W, H = 1080, 1920
FPS = 30
HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[1]
ICON = ROOT / "ios/Chess/Assets.xcassets/AppIcon.appiconset/AppIcon.png"
ROUNDED = "/System/Library/Fonts/SFNSRounded.ttf"
EMOJI = "/System/Library/Fonts/Apple Color Emoji.ttc"

# The app's light palette.
INK = (34, 39, 44)
MIST = (104, 112, 118)
ACCENT = (46, 97, 80)
GREAT = (47, 158, 92)
DANGER = (194, 59, 59)
BG_TOP = (241, 239, 233)
BG_BOTTOM = (230, 227, 220)

# Where the recording sits: the board and coach card, cropped from the
# simulator frame (1206 x 2622) and scaled to 900 wide.
CROP_Y, CROP_H, APP_W, APP_X, APP_Y = 400, 1540, 900, 90, 490
K = APP_W / 1206


def square(name):
    """A square's centre on the reel, from where the board sits in the recording."""
    f = ord(name[0]) - 97
    r = 8 - int(name[1])
    x, y = 43 + (f + 0.5) * 140, 434 + (r + 0.5) * 139.6
    return APP_X + x * K, APP_Y + (y - CROP_Y) * K


def font(size, weight=900):
    f = ImageFont.truetype(ROUNDED, size)
    f.set_variation_by_axes([400, weight])
    return f


def is_emoji(ch):
    return ord(ch) > 0x2600 and ch not in "−→·…♚"


def draw_line(img, text, y, size, color, weight=900, center=True, x=None):
    """One line of text with colour emoji, centred (or from x) at y. Returns its height."""
    d = ImageDraw.Draw(img)
    f = font(size, weight)
    runs, cur = [], ""
    for ch in text:
        if is_emoji(ch):
            if cur:
                runs.append(("t", cur))
                cur = ""
            runs.append(("e", ch))
        elif ch != "️":
            cur += ch
    if cur:
        runs.append(("t", cur))
    pieces = []
    for kind, s in runs:
        if kind == "t":
            box = d.textbbox((0, 0), s, font=f)
            pieces.append((kind, s, box[2] - box[0], box))
        else:
            e = Image.new("RGBA", (180, 180))
            ImageDraw.Draw(e).text((0, 0), s, font=ImageFont.truetype(EMOJI, 160), embedded_color=True)
            e = e.crop(e.getbbox()).resize((int(size * 1.05), int(size * 1.05)), Image.LANCZOS)
            pieces.append((kind, e, e.width, None))
    total = sum(p[2] for p in pieces) + 10 * (len(pieces) - 1)
    cx = (W - total) / 2 if center else x
    for kind, s, w, box in pieces:
        if kind == "t":
            d.text((cx - box[0], y), s, font=f, fill=color)
        else:
            img.alpha_composite(s, (int(cx), int(y + size * 0.08)))
        cx += w + 10
    return size * 1.2


def caption(lines, path):
    """Captions in the band above the board, on a soft card so they read over anything."""
    img = Image.new("RGBA", (W, H))
    heights = sum(l[2] * 1.2 for l in lines)
    top = 300 + (180 - heights) / 2
    card = Image.new("RGBA", (W, H))
    ImageDraw.Draw(card).rounded_rectangle((70, top - 26, W - 70, top + heights + 18), 34, fill=(251, 250, 247, 235))
    shadow = card.filter(ImageFilter.GaussianBlur(18))
    shadow = Image.eval(shadow, lambda v: v)
    img.alpha_composite(Image.composite(Image.new("RGBA", (W, H), (0, 0, 0, 40)), Image.new("RGBA", (W, H)), shadow.split()[3]), (0, 10))
    img.alpha_composite(card)
    y = top
    for text, color, size in lines:
        y += draw_line(img, text, y, size, color)
    img.save(path)


def background(path):
    img = Image.new("RGB", (W, H))
    d = ImageDraw.Draw(img)
    for y in range(H):
        t = y / H
        d.line([(0, y), (W, y)], fill=tuple(int(a + (b - a) * t) for a, b in zip(BG_TOP, BG_BOTTOM)))
    glow = Image.new("RGBA", (W, H))
    ImageDraw.Draw(glow).ellipse((W / 2 - 520, -520, W / 2 + 520, 520), fill=(255, 250, 240, 160))
    img = img.convert("RGBA")
    img.alpha_composite(glow.filter(ImageFilter.GaussianBlur(160)))
    img.convert("RGB").save(path)


def tap_ring(path):
    """Where the finger lands: a soft ring of light."""
    s = 150
    img = Image.new("RGBA", (s, s))
    d = ImageDraw.Draw(img)
    d.ellipse((20, 20, s - 20, s - 20), fill=(255, 255, 255, 90), outline=(255, 255, 255, 235), width=7)
    img.filter(ImageFilter.GaussianBlur(0.6)).save(path)


def loud_app(path):
    """What every other chess app does: red, a rating drop, a verdict in capitals."""
    img = Image.new("RGBA", (W, H), (120, 0, 0, 110))
    vignette = Image.new("RGBA", (W, H))
    ImageDraw.Draw(vignette).rectangle((0, 0, W, H), outline=(220, 0, 0, 255), width=170)
    img.alpha_composite(vignette.filter(ImageFilter.GaussianBlur(90)))
    d = ImageDraw.Draw(img)
    label = "every other chess app:"
    f = font(46, 800)
    bw = d.textbbox((0, 0), label, font=f)[2]
    d.rounded_rectangle(((W - bw) / 2 - 28, 296, (W + bw) / 2 + 28, 372), 38, fill=(20, 20, 20, 230))
    d.text(((W - bw) / 2, 305), label, font=f, fill=(255, 255, 255))
    for text, size, y, color, ang in [("?? BLUNDER", 150, 720, (255, 40, 40), -6), ("Rating 1452 → 1428  (−24)", 54, 1010, (255, 255, 255), 3),
                                      ("0:03", 120, 1120, (255, 60, 60), -2), ("You missed mate in 1.", 52, 1290, (255, 255, 255), 0)]:
        layer = Image.new("RGBA", (W, 260))
        lf = font(size, 1000)
        box = ImageDraw.Draw(layer).textbbox((0, 0), text, font=lf)
        tx = (W - (box[2] - box[0])) / 2 - box[0]
        ImageDraw.Draw(layer).text((tx + 6, 26), text, font=lf, fill=(0, 0, 0, 200))
        ImageDraw.Draw(layer).text((tx, 20), text, font=lf, fill=color)
        img.alpha_composite(layer.rotate(ang, resample=Image.BICUBIC), (0, y - 60))
    img.save(path)


def end_card(path):
    """The promise, the name, and why it's different."""
    img = Image.open(path.parent / "bg.png").convert("RGBA")
    icon = Image.open(ICON).convert("RGBA").resize((300, 300), Image.LANCZOS)
    mask = Image.new("L", (300, 300))
    ImageDraw.Draw(mask).rounded_rectangle((0, 0, 299, 299), 68, fill=255)
    shadow = Image.new("RGBA", (W, H))
    ImageDraw.Draw(shadow).rounded_rectangle((390, 470, 690, 770), 68, fill=(0, 0, 0, 70))
    img.alpha_composite(shadow.filter(ImageFilter.GaussianBlur(26)))
    img.paste(icon, (390, 450), mask)
    y = 820
    y += draw_line(img, "Chess that doesn't", y, 84, INK)
    y += draw_line(img, "yell at you.", y, 84, INK)
    y += 40
    y += draw_line(img, "Calm Chess", y, 64, ACCENT)
    y += draw_line(img, "Free · No ads · Works offline", y + 10, 40, MIST, 600)
    draw_line(img, "iPhone & Android", y + 30, 40, MIST, 600)
    img.convert("RGB").save(path)


# Sound: the app's own synthesized tones, plus a clock and the loud app's buzzer.
RATE = 44100


class Track:
    def __init__(self, seconds):
        self.s = [0.0] * int(seconds * RATE)

    def tones(self, at, partials, square=False, triangle=False, gain=1.0):
        for freq, start, dur, g in partials:
            s0 = int((at + start) * RATE)
            for k in range(int(dur * RATE)):
                i = s0 + k
                if i >= len(self.s):
                    break
                t = k / RATE
                ph = 2 * math.pi * freq * t
                w = math.sin(ph)
                if square:
                    w = 1.0 if w >= 0 else -1.0
                if triangle:
                    w = 2 / math.pi * math.asin(math.sin(ph))
                self.s[i] += w * g * gain * min(1, t / 0.005) * math.exp(-5 * t / dur)

    def noise(self, at, dur, gain):
        prev = 0.0
        n = int(dur * RATE)
        for k in range(n):
            i = int(at * RATE) + k
            if i >= len(self.s):
                break
            white = random.uniform(-1, 1)
            self.s[i] += (white - prev) * gain * (1 - k / n) ** 3 * 0.5
            prev = white

    def buzzer(self, at, dur):
        """A harsh, detuned buzz under a low boom: the sound of being scolded."""
        for k in range(int(dur * RATE)):
            i = int(at * RATE) + k
            if i >= len(self.s):
                break
            t = k / RATE
            env = min(1, t / 0.01) * (1 - t / dur) ** 0.6
            buzz = sum((1 if math.sin(2 * math.pi * f * t) >= 0 else -1) for f in (146.8, 155.6, 293.7)) / 3
            boom = math.sin(2 * math.pi * (60 - 25 * t) * t) * math.exp(-4 * t)
            self.s[i] += 0.16 * buzz * env + 0.5 * boom

    def pad(self, start, end, gain):
        """A soft, slow chord: the calm after."""
        notes = [220.0, 277.18, 329.63, 415.30]
        for i in range(int(start * RATE), min(len(self.s), int(end * RATE))):
            t = i / RATE - start
            span = end - start
            env = min(1, t / 1.2) * min(1, (span - t) / 1.0)
            v = sum(math.sin(2 * math.pi * f * t + n) for n, f in enumerate(notes)) / len(notes)
            self.s[i] += v * gain * env * (0.85 + 0.15 * math.sin(2 * math.pi * 0.25 * t))

    def write(self, path, fade_out_at):
        peak = max(abs(v) for v in self.s) or 1
        scale = 0.89 / peak
        with wave.open(str(path), "wb") as w:
            w.setnchannels(2)
            w.setsampwidth(2)
            w.setframerate(RATE)
            frames = bytearray()
            for i, v in enumerate(self.s):
                t = i / RATE
                g = 1.0 if t < fade_out_at else max(0.0, 1 - (t - fade_out_at) / 0.4)
                x = int(max(-1, min(1, v * scale * g)) * 32767)
                frames += struct.pack("<hh", x, x)
            w.writeframes(bytes(frames))


SELECT = [(1320, 0, 0.12, 0.05)]
MOVE = [(880, 0, 0.25, 0.09), (1760, 0, 0.18, 0.03)]
GREAT_CHIME = [(784, 0, 0.35, 0.07), (988, 0.07, 0.35, 0.07), (1319, 0.14, 0.35, 0.07)]
BAD = [(330, 0, 0.3, 0.07), (311, 0.08, 0.35, 0.05)]
WIN = [(523, 0, 0.5, 0.07), (659, 0.09, 0.5, 0.07), (784, 0.18, 0.5, 0.07), (1046, 0.27, 0.5, 0.07), (1319, 0.36, 0.5, 0.07)]


def soundtrack(path, total, b):
    """The sound, from the reel-time beats `b`."""
    t = Track(total)
    # Tension under the hook: a clock, the thing chess usually rushes you with.
    k = 0
    while 0.15 + k * 0.45 < b["blunder"]:
        t.tones(0.15 + k * 0.45, [(2400, 0, 0.05, 0.05), (1200, 0, 0.04, 0.03)])
        k += 1
    t.tones(b["pick"], SELECT, gain=1.6)
    t.tones(b["blunder"], MOVE, gain=1.6)
    t.tones(b["blunder"], [(1980, 0, 0.3, 0.05)], gain=1.4)
    t.noise(b["blunder"], 0.35, 0.12)
    t.buzzer(b["loud"], 1.0)
    t.pad(b["calm"] + 0.2, total, 0.035)
    t.tones(b["calm"] + 0.25, BAD, triangle=True, gain=1.3)
    t.tones(b["undo"], [(990, 0, 0.1, 0.035)], gain=1.5)
    t.tones(b["pick2"], SELECT, gain=1.6)
    t.tones(b["mate"], MOVE, gain=1.6)
    t.tones(b["mate"] + 0.15, GREAT_CHIME, gain=1.5)
    t.tones(b["won"], WIN, gain=1.4)
    t.tones(b["end"] + 0.05, [(1046, 0, 0.9, 0.05), (1568, 0.05, 0.9, 0.03)], gain=1.4)
    t.write(path, total - 0.45)


def run(cmd):
    subprocess.run(cmd, check=True)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("recording")
    ap.add_argument("out")
    # Recording times of each beat, read off the frames (the first frame each shows).
    ap.add_argument("--start", type=float, default=5.3, help="the staged position is on screen")
    ap.add_argument("--pick", type=float, default=7.0, help="the queen is lifted")
    ap.add_argument("--blunder", type=float, default=8.0, help="the queen lands on h7")
    ap.add_argument("--undo", type=float, default=11.0, help="the move is taken back")
    ap.add_argument("--pick2", type=float, default=12.3, help="the queen is lifted again")
    ap.add_argument("--mate", type=float, default=13.3, help="the queen lands on f7")
    ap.add_argument("--won", type=float, default=14.5, help="the finish card arrives")
    a = ap.parse_args()
    s0 = a.start
    freeze_at = a.blunder + 0.4  # the loud app takes over the frame here, for a second
    duration = float(subprocess.run(["ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "csv=p=0", a.recording],
                                    capture_output=True, text=True).stdout)

    def reel(raw):
        """Recording time to reel time: the frozen second sits after the blunder."""
        return raw - s0 + (1.0 if raw > freeze_at else 0.0)

    main_len = round(reel(duration - 0.05), 2)
    end_len = 2.6
    total = main_len + end_len
    b = {"pick": reel(a.pick), "blunder": reel(a.blunder), "loud": reel(freeze_at), "calm": reel(freeze_at) + 1.0,
         "undo": reel(a.undo), "pick2": reel(a.pick2), "mate": reel(a.mate), "won": reel(a.won), "end": main_len}
    work = Path(tempfile.mkdtemp(prefix="reel-"))
    background(work / "bg.png")
    tap_ring(work / "ring.png")
    loud_app(work / "loud.png")
    end_card(work / "end.png")
    caps = [
        ((0.0, 0.6), [("Mate in 1.", INK, 96)]),
        ((0.6, b["blunder"]), [("Mate in 1.", INK, 96), ("Easy. 😎", INK, 72)]),
        ((b["blunder"], b["loud"]), [("bro. 💀", DANGER, 104)]),
        ((b["calm"], b["undo"]), [("Calm Chess just…", INK, 70), ("helps. 🥹", GREAT, 84)]),
        ((b["undo"], b["pick2"]), [("Undo.", INK, 88), ("No judgment.", MIST, 62)]),
        ((b["pick2"], b["mate"]), [("Now find it. 👀", INK, 84)]),
        ((b["mate"], b["won"]), [("CHECKMATE.", GREAT, 104)]),
        ((b["won"], main_len), [("send this to the friend", INK, 58), ("who hangs their queen 💀", INK, 58)]),
    ]
    for i, (_, lines) in enumerate(caps):
        caption(lines, work / f"cap{i}.png")

    # The game, as recorded: before the blunder, a frozen frame under the loud app, and after.
    app = f"crop=1206:{CROP_H}:0:{CROP_Y},scale={APP_W}:-2,fps={FPS}"
    soundtrack(work / "audio.wav", total, b)
    fc = [
        "[0:v]split=3[r0][r1][r2]",
        f"[r0]trim={s0}:{freeze_at},setpts=PTS-STARTPTS,{app}[a]",
        f"[r1]trim={freeze_at}:{freeze_at + 0.05},setpts=PTS-STARTPTS,{app},loop=loop=40:size=1,trim=0:1.0,setpts=PTS-STARTPTS[f]",
        f"[r2]trim={freeze_at}:{duration},setpts=PTS-STARTPTS,{app}[b]",
        "[a][f][b]concat=n=3:v=1:a=0[game]",
        f"[1:v]loop=loop=-1:size=1,trim=0:{main_len},fps={FPS},setpts=PTS-STARTPTS[bg]",
        f"[bg][game]overlay={APP_X}:{APP_Y}:shortest=1[base]",
    ]
    last = "base"
    # The loud app: red over the frozen frame, and the whole frame shakes.
    lo, hi = b["loud"], b["calm"]
    fc.append(f"[{last}][2:v]overlay=0:0:enable='between(t,{lo},{hi})'[loud]")
    fc.append(f"[loud]scale=1134:2016,crop=1080:1920:'27+if(between(t,{lo},{hi}),18*sin(t*97),0)':'48+if(between(t,{lo},{hi}),14*cos(t*83),0)'[shake]")
    last = "shake"
    n_in = 3
    inputs = []
    for i, ((a0, a1), _) in enumerate(caps):
        inputs += ["-loop", "1", "-t", str(main_len), "-i", str(work / f"cap{i}.png")]
        fc.append(f"[{last}][{n_in}:v]overlay=0:0:enable='between(t,{a0},{a1})'[c{i}]")
        last = f"c{i}"
        n_in += 1
    # The finger lands a beat before the board answers.
    for i, (when, sq) in enumerate([(b["pick"] - 0.1, "h5"), (b["blunder"] - 0.1, "h7"), (b["pick2"] - 0.1, "h5"), (b["mate"] - 0.1, "f7")]):
        x, y = square(sq)
        inputs += ["-loop", "1", "-t", str(main_len), "-i", str(work / "ring.png")]
        fc.append(f"[{n_in}:v]format=rgba,fade=t=out:st={when + 0.05}:d=0.35:alpha=1[ring{i}]")
        fc.append(f"[{last}][ring{i}]overlay={x - 75:.0f}:{y - 75:.0f}:enable='between(t,{when - 0.05},{when + 0.4})'[t{i}]")
        last = f"t{i}"
        n_in += 1
    # The end card fades in over the last frame, then the reel's own first
    # frame fades in over the card, so it loops without a seam.
    inputs += ["-loop", "1", "-t", str(total), "-i", str(work / "end.png")]
    fc.append(f"[{last}]tpad=stop_mode=clone:stop_duration={end_len},split[main][first]")
    fc.append(f"[{n_in}:v]fps={FPS},format=rgba,fade=t=in:st={main_len - 0.3}:d=0.3:alpha=1[end]")
    fc.append(f"[first]trim=0:0.04,setpts=PTS-STARTPTS,loop=loop={int(total * FPS)}:size=1,setpts=N/{FPS}/TB,format=rgba,"
              f"fade=t=in:st={total - 0.45}:d=0.4:alpha=1[still]")
    fc.append(f"[main][end]overlay=0:0:enable='gte(t,{main_len - 0.3})'[withend]")
    fc.append(f"[withend][still]overlay=0:0:enable='gte(t,{total - 0.45})',format=yuv420p[v]")
    n_in += 1
    cmd = ["ffmpeg", "-loglevel", "error", "-y", "-i", a.recording, "-loop", "1", "-i", str(work / "bg.png"),
           "-loop", "1", "-t", str(main_len), "-i", str(work / "loud.png"), *inputs, "-i", str(work / "audio.wav"),
           "-filter_complex", ";".join(fc), "-map", "[v]", "-map", f"{n_in}:a", "-c:v", "libx264", "-preset", "slow",
           "-crf", "17", "-pix_fmt", "yuv420p", "-r", str(FPS), "-af", "loudnorm=I=-14:TP=-1.5:LRA=11", "-c:a", "aac", "-b:a", "192k", "-ar", "44100", "-shortest",
           "-movflags", "+faststart", a.out]
    run(cmd)
    shutil.copy(work / "end.png", Path(a.out).with_suffix(".endcard.png"))
    print(a.out)


if __name__ == "__main__":
    main()
