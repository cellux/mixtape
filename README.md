# Mixtape

**Work in progress.**

Mixtape is a small stack-based (Forth-like) DSL plus a minimal IDE for **non‑realtime** sound synthesis. You write a “tape” program, evaluate it, render it to a finite buffer, and play it.

The language is concatenative: tokens either push values (numbers/strings/vectors) or execute *words* that consume/produce stack values.

## Overview

- **Two main audio value types**
  - **Stream**: an (often infinite) lazy generator of frames.
  - **Tape**: a finite in-memory buffer (1 or 2 channels).
- **Workflow**
  1. Produce a `Stream` (oscillator, arithmetic, sequencing, etc.).
  2. Multiply with an envelope.
  3. Render it to a `Tape`.
  4. Play the result.
- **Defaults**
  - Sample rate: `48000` Hz (configurable)
  - Tempo: `:bpm = 120`
  - Ticks per beat: `:tpb = 96`
  - Osc defaults: `:freq = 440`, `:pw = 0.5`

## Usage

### Run the GUI/IDE

```sh
mixtape                 # open empty buffer
mixtape examples/blips.tape  # open file in the editor
```

Mixtape evaluates the current editor buffer when you press `C-Enter`. If the result is a `Tape`, it’s displayed as a waveform.

### Headless evaluation

```sh
mixtape -e '( 440 >:freq ~sin 1b take )'
mixtape -f examples/seq.tape
```

Flags:

- `-sr <int>`: sample rate (default `48000`)
- `-bpm <float>`: tempo (default `120`)
- `-tpb <int>`: ticks per beat (default `96`)
- `-loglevel <string>`: logging (default `info`)
- `-e <script>`: evaluate inline script
- `-f <file>`: evaluate file

## Key bindings

Global:

- `C-Enter`: evaluate buffer
- `C-p`: render the current result to audio (if possible) and play
- `C-z` / `C-x u` / `C-S--`: undo
- `C-x C-s`: save (only if you opened a file)
- `C-q`: quit
- `C-g` / `Escape`: reset (stop playback, clear key sequence state)

Editor basics (subset):

- Movement: `Left/Right/Up/Down`, `Home/End`, `C-a`/`C-e`, `C-Left`/`C-Right`, `PageUp`/`PageDown`
- Edit: `Backspace`, `Delete`, `Enter`, `Tab`
- Kill/yank: `C-w` (kill region), `C-y` (yank), `M-w` (copy region), `C-k` (kill to end of line)

## Language

### Core syntax

- **Whitespace separated tokens**.
- **Numbers**: floats and ratios (e.g. `0.25`, `1/4`).
- **Strings**: `"..."`.
- **Line comments**: `#` to end of line.
- **Vectors**: `[ ... ]`, evaluates code inside brackets, collects all pushed values into a vector
- **Quoted code blocks**: `{ ... }` produces a vector of unevaluated tokens; run it with `eval`.
- **Scopes**: `(` pushes a new environment, `)` pops it.

Special parsing sugar:

- `@foo` expands to `"foo" get` (load variable).
- `>foo` expands to `"foo" swap set` (store variable).
- Numeric suffixes expand to unit conversion words:
  - `1s` → `1 seconds`
  - `1b` → `1 beats`
  - `1p` → `1 periods`
  - `1t` → `1 ticks`

Truthiness:

- `0` is false, any non-zero number is true.
- `true` is `-1`, `false` is `0`.

### Stack and environment

- `dup swap over drop nip` (basic stack ops)
- `stack` pushes the entire stack as a vector
- `set` / `get` store and retrieve values in the current environment
- `:name` (a symbol starting with `:`) evaluates to the value stored under that key (e.g. `:bpm`, `:freq`).

Example:

```tape
( 100 >:bpm
  :bpm log
)
```

### Control flow

- `cond {then} if`
- `cond {then} {else} if`

`if` is a method on numbers (condition must be a `Num`).

Looping and exceptions:

- `{body} loop` runs forever or until an error.
- `break` throws a special nil value to exit `loop`.
- `throw` / `catch` provide exceptions.

### Collections and higher-order words

Vectors (`Vec`) have methods:

- `len` → length
- `at` → index
- `push` / `pop`
- `each` / `map` / `reduce`
- `partition` (sliding windows)

Example:

```tape
[1 2 3] { 10 + } map  # => [11 12 13]
```

### Envelopes

Envelopes are just 1‑channel `Tape`s (finite buffers) that you multiply with a `Stream`.

Low-level segment generators (return a `Tape`) are:

- `/line`
- `/exp k`
- `/log k`
- `/cos`
- `/pow p`
- `/sigmoid k`

They read their parameters from variables in the environment:

- `:start` (start value)
- `:end` (end value)
- `:nf` (number of frames)

The prelude defines a higher-level `env` builder that sequences multiple segments.

Example (from `examples/blips.tape`):

```tape
[0.2 2 1 8] 1b distribute
[{9 /exp} {-3 /exp} {/line} {-3 /exp}] env >:env

[ :env ( 220 >:freq ~sin ) *
  :env ( 440 >:freq ~triangle ) *
  :env ( 110 >:freq ~sin ) *
] {join} reduce
```

### Streams

A `Stream` is a lazy frame generator. Most streams are infinite.

Oscillators (produce 1-channel streams):

- `~sin`
- `~saw`
- `~triangle`
- `~pulse`

They read parameters from variables:

- `:freq` (Hz) — can be a number or a stream
- `:phase` (0..1)
- `:pw` (pulse width, 0..1) for `~pulse`

Utilities:

- `~` converts any `Streamable` value to a `Stream`.
- `delay` prepends silence: `streamable nframes delay -> Stream`
- `join` concatenates in time: `a b join -> Stream`

Rendering:

- `streamable nframes take -> Tape`

Arithmetic (`+ - * / mod rem`) is overloaded:

- `Num op Num -> Num`
- otherwise it combines streams sample-wise (auto mono/stereo adaptation)

Minimal tone:

```tape
( 440 >:freq
  ~sin
  1b take
)
```

### Tapes

A `Tape` is a finite buffer of samples.

- `tape1 nframes` creates a silent mono tape
- `tape2 nframes` creates a silent stereo tape

Tape methods:

- `nf` → frame count
- `slice start end` → sub-tape
- `shift amount` → circular rotation (amount can be frames; if `0<amount<1` it’s treated as a fraction of the tape length)
- `resample converterType ratio` → resample with libsamplerate
  - converter types: `SRC_SINC_BEST_QUALITY`, `SRC_SINC_MEDIUM_QUALITY`, `SRC_SINC_FASTEST`, `SRC_ZERO_ORDER_HOLD`, `SRC_LINEAR`
- `+@ tape offset` → mix/add RHS into LHS starting at frame offset (in-place)

Converting vectors to audio:

- `[ ...numbers... ] tape` → 1-channel tape with those exact samples.

### Other builtin words

Core words:

- Stack: `drop nip dup swap over`
- Scopes: `(` `)`
- Quoting: `{` `eval`
- Vectors: `[` `]`
- Equality: `=` and `!=` (prelude)
- Logging: `log`
- Exceptions: `throw catch`
- Iteration protocol: `iter next`

Prelude utilities (defined in `prelude.tape`):

- Booleans: `true false not true? false? nil?`
- Control: `break assert`
- List helpers: `zip`
- Sequencing: `seq` (iterate multiple vectors in lockstep and run a body)
- Time/unit conversions: `sr seconds beats ticks periods`
- Resampling shortcut: `stretch` (uses `SRC_LINEAR`)

### Loading audio

`"path" load` loads one of:

- `.tape` (DSL script)
- `.wav`
- `.mp3`

Rules:

- If you load `foo.tape`, Mixtape may cache/render it to `foo.wav` and will reuse that WAV if it’s newer than the `.tape`.
- WAV/MP3 are resampled to the current `-sr` sample rate when needed.

## Examples

- `examples/blips.tape`: oscillator + multi-segment envelope + joins
- `examples/seq.tape`: uses `seq` to generate chunks and `{join} reduce`
- `test.tape`: language regression tests and idioms
