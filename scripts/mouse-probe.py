#!/usr/bin/env python3
"""Mouse and touch probe for rdxt. Shows a fixed checklist that updates in place.

Usage: mouse-probe.py RESULTS.json
"""
import json, os, re, select, sys, termios, time, tty

OUT = sys.argv[1] if len(sys.argv) > 1 else "mouse-results.json"
ESC = "\x1b"
fd = sys.stdin.fileno()

STEPS = [
    ("click", "Mac: plain click"),
    ("shift", "Mac: shift-click"),
    ("alt", "Mac: option-click"),
    ("cmd", "Mac: cmd-click (shows as nothing, or as something else)"),
    ("drag", "Mac: click and drag"),
    ("wheel", "Mac: scroll wheel / trackpad"),
    ("tap", "iPad: tap"),
    ("swipe", "iPad: swipe up and down"),
    ("ipad-scroll", "iPad: trackpad or mouse scroll (if you have one)"),
]

step = 0
log = {k: [] for k, _ in STEPS}
recent = []


def w(s):
    sys.stdout.write(s)
    sys.stdout.flush()


def describe(b, x, y, kind):
    mods = [n for bit, n in ((4, "shift"), (8, "alt"), (16, "ctrl")) if b & bit]
    base = b & ~(4 | 8 | 16)
    what = {0: "left", 1: "middle", 2: "right", 3: "move", 64: "wheel-up", 65: "wheel-down",
            66: "wheel-left", 67: "wheel-right"}.get(base & ~32, str(base))
    if base & 32 and what != "move":
        what += "-drag"
    if not what.startswith("wheel") and not what.endswith("drag"):
        what += " down" if kind == "M" else " up"
    return {"event": what, "mods": mods, "x": x, "y": y}


def summary(events):
    counts = {}
    for e in events:
        k = e["event"] + ("+" + "+".join(e["mods"]) if e["mods"] else "")
        counts[k] = counts.get(k, 0) + 1
    return ", ".join("%s×%d" % (k, v) for k, v in counts.items()) or "-"


def draw():
    out = [ESC + "[H" + ESC + "[2J", ESC + "[1mrdxt mouse probe" + ESC + "[0m", ""]
    out.append("Do each step, then press " + ESC + "[1mSpace" + ESC + "[0m (or tap the [ Next ] button) for the next one.")
    out.append("Press " + ESC + "[1mq" + ESC + "[0m to finish.")
    out.append("")
    for i, (k, label) in enumerate(STEPS):
        mark = "▶" if i == step else ("✓" if i < step else " ")
        style = ESC + "[1m" if i == step else (ESC + "[2m" if i > step else "")
        out.append("%s %s%-52s%s %s" % (mark, style, label, ESC + "[0m", summary(log[k])[:60]))
    out.append("")
    out.append(ESC + "[7m [ Next ] " + ESC + "[0m")
    out.append("")
    out.append(ESC + "[2mLast events:" + ESC + "[0m")
    for e in recent[-5:]:
        out.append(ESC + "[2m  %-16s %-12s at %d,%d" % (e["event"], "+".join(e["mods"]) or "-", e["x"], e["y"]) + ESC + "[0m")
    w("\r\n".join(out))


next_row = 6 + len(STEPS) + 2  # 1-based row of the [ Next ] button

old = termios.tcgetattr(fd)
try:
    tty.setraw(fd)
    w(ESC + "[?1049h" + ESC + "[?25l" + ESC + "[?1000h" + ESC + "[?1002h" + ESC + "[?1006h")
    draw()
    buf = ""
    done = False
    while not done and step < len(STEPS):
        r, _, _ = select.select([fd], [], [], 0.5)
        if not r:
            continue
        buf += os.read(fd, 4096).decode("utf-8", "replace")
        while buf:
            m = re.match(r"\x1b\[<(\d+);(\d+);(\d+)([Mm])", buf)
            if m:
                e = describe(int(m.group(1)), int(m.group(2)), int(m.group(3)), m.group(4))
                buf = buf[m.end():]
                if e["y"] == next_row and e["x"] <= 11 and e["event"] == "left up":
                    step += 1
                else:
                    log[STEPS[step][0]].append(e)
                    recent.append(e)
                continue
            if buf.startswith("\x1b") and len(buf) < 32 and not re.match(r"\x1b(\[[\d;<]*)?[^\d;<\[]", buf):
                break  # incomplete escape sequence; wait for more
            ch, buf = buf[0], buf[1:]
            if ch == " ":
                step += 1
            elif ch in "qQ\x03":
                done = True
                break
        if step < len(STEPS):
            draw()
finally:
    w(ESC + "[?1006l" + ESC + "[?1002l" + ESC + "[?1000l" + ESC + "[?25h" + ESC + "[?1049l")
    termios.tcsetattr(fd, termios.TCSADRAIN, old)
    with open(OUT, "w") as f:
        json.dump({k: log[k] for k, _ in STEPS}, f, indent=1)
    print("rdxt mouse probe results:")
    for k, label in STEPS:
        print("  %-50s %s" % (label, summary(log[k])))
    print("\nSaved. You can close this pane.")
