# iidx-scratch-bridge

Windows bridge that converts an IIDX entry-model controller's analog
scratch axis into stock-controller-style 133 ms keyboard pulses, so
PCSX2 + PS2 IIDX 14 GOLD stops emitting phantom POOR judgments on a
single wheel motion.

## Why this exists

```
[Entry Model 皿] (位置型アナログ軸)
    ↓
[BM2KEY] (差分検出して短いキーパルスを連発)  ← 問題箇所
    ↓
[PCSX2]   (キー → 仮想PS2十字キー Up/Down)
    ↓
[IIDX 14] (十字キー Up/Down を皿入力として読む)
```

The PS2 build of IIDX reads the wheel as a digital d-pad button. A
real PS2 controller's internal 74HC monostable converts each wheel
motion to a single ~133 ms long press. BM2KEY (made for LR2 / beatoraja)
emits short pulses on every angle delta, so a single wheel motion
produces several scratch judgments — empty POOR on no-note sections.

This tool replaces BM2KEY: it reads the wheel axis directly, accumulates
deltas, and reconstructs the stock-controller "single long press"
behaviour.

## Status

MVP. Tested against the design doc; awaiting first-pass hardware
validation. See `iidx-scratch-bridge-design.md` for the full design.

## Requirements

- Windows 10 / 11 (x86_64)
- An IIDX entry-model controller exposed as a USB HID device

## Install

Grab the latest `scratch-bridge.exe` from the
[Releases](../../releases) page. Copy `config.example.toml` next to it
as `config.toml`.

## First-time setup

The probing flow is `--list` → fill VID/PID → `--dump` → fill offsets →
run for real.

### 1. Find your VID/PID

```
scratch-bridge.exe --list
```

Lists every HID device on the system as

```
VID=0x1ccf PID=0x8048 iface=0  KONAMI / IIDX entry model  serial=...
```

Copy the matching hex VID/PID into `[device]` in `config.toml`. Leave
`scratch_axis_byte_index` and `buttons_byte_range` at the defaults for
now — they only matter for the real run.

### 2. Find the byte offsets

```
scratch-bridge.exe --config config.toml --dump
```

`--dump` prints each Input Report as raw hex regardless of the
`[device]` offsets, so you don't need to know them yet:

```
report len=8 00 00 12 34 00 AB 00 00
```

- Spin the wheel slowly. The byte that changes monotonically (wrapping
  0..255) is `scratch_axis_byte_index`.
- Tap each button. The byte(s) that flip bits are `buttons_byte_range`
  (a half-open range `[start, end)`).
- Tap each of the 7 keys plus START/SELECT and note the bit position.
  That bit index goes into `[buttons]` as `b<n>`.

Update `config.toml` with the offsets and the button-to-key mapping.

### 3. Run it

```
scratch-bridge.exe --config config.toml
```

## PCSX2 setup

1. Open PCSX2 -> Controllers -> Per-Game / Global -> Plugin Settings.
2. Bind d-pad Up to whatever `scratch.up_key` is (default `UP`) and
   d-pad Down to `scratch.down_key` (default `DOWN`).
3. Bind the seven keys + Start/Select to the same key names you set in
   `[buttons]`.

## Build from source

Requires Go 1.22+ and a working CGo toolchain (Windows: MinGW-w64).
The HID library (`github.com/bearsh/hid`) wraps hidapi and needs CGo.

```
git clone https://github.com/chun37/iidx-scratch-bridge
cd iidx-scratch-bridge
go build ./cmd/scratch-bridge
```

A non-CGo build is possible but the HID layer falls back to a no-op
stub, so the binary will start, fail to find any device, and loop on
the reconnect timer. The published Releases are built with CGo enabled
on the Windows runner.

## Tests

```
go test ./...
```

Unit tests cover the wrap-around delta logic, the pulse state machine
(including reversal, extension, and superseded-timer cases), the HID
report parser, and config validation.

## Project layout

```
cmd/scratch-bridge/  entry point + glue code
internal/config/     TOML loader + key-name parser
internal/hid/        HID device reader + Input Report parser
internal/rotation/   wrap-around delta + sub-threshold accumulation
internal/pulse/      monostable state machine (Idle/PressingUp/PressingDown)
internal/keyout/     Windows SendInput wrapper + VK code map
```

## License

MIT (see [LICENSE](LICENSE)).
