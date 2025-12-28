# Mixtape

**Work in progress**

Mixtape is a small stack-based language (the **Mixtape DSL**) plus an interactive editor/player for building and auditioning audio streams (“tapes”). It is implemented in Go and ships with a standard library in `assets/prelude.tape`.

This README documents:

- CLI usage and flags
- the built-in editor and key bindings
- the Mixtape DSL: syntax, types, evaluation model
- **all words** (built-ins + standard library), categorized with descriptions and examples

> Source of truth for most word docs: `assets/prelude.tape`.

> Working examples/tests: `tests/*.tape` and `examples/*.tape`.

---

## Build & run

```sh
go build
./mixtape               # start GUI with an empty buffer
./mixtape mypatch.tape   # start GUI with file loaded
```

Run tests:

```sh
make test
```

---

## Command line usage

`mixtape` has two modes:

1. **GUI mode** (default): open files given as positional args and start the editor/player.
2. **Batch eval mode**: evaluate a file (`-f`) or script string (`-e`) and print the resulting value.

### Flags

From `./mixtape -h`:

- `-loglevel info|debug|...` (default: `info`) — logging verbosity.
- `-sr <int>` (default: `48000`) — sample rate.
- `-bpm <float>` (default: `120`) — beats per minute.
- `-tpb <int>` (default: `96`) — ticks per beat.
- `-f <path>` — evaluate a `.tape` script file and exit.
- `-e <string>` — evaluate an inline script and exit.
- `-prof <prefix>` — write pprof CPU and heap profiles to `<prefix>.cpu` and `<prefix>.mem`.

### Examples

Evaluate a file:

```sh
./mixtape -f tests/seq.tape
```

Evaluate a one-liner:

```sh
./mixtape -e '69 mtof'
# prints: 440
```

Start the GUI with a file:

```sh
./mixtape examples/seq.tape
```

### Defaults injected into the VM

At startup Mixtape sets these environment variables:

- `:bpm` from `-bpm`
- `:tpb` from `-tpb`
- `:nf` = frames-per-beat = `sr / (bpm/60)`

The prelude then sets additional defaults like `:freq`, `:phase`, `:pw`, filter params, etc.

---

## The GUI editor

When you run `./mixtape [file.tape]` you get an editor pane and (when the result is audio) a waveform pane.

### Evaluating / playing

- `C-p` — evaluate buffer and **play** the resulting tape/stream.
- `C-Enter` — evaluate buffer without starting playback.
- `C-g` or `Escape` — cancel the current evaluation (and reset transient state).

Evaluation happens in the background; progress is shown while rendering finite streams to a tape.

### Files

- `C-x C-s` — save the current file (only works if the GUI was started with a file path).

### Quit / undo

- `C-q` — quit.
- Undo:
  - `C-z`
  - `C-x u`
  - `C-S--`

### Cursor movement

- Arrow keys — move by character/line.
- `Home` / `End` — beginning / end of line.
- `C-a` / `C-e` — beginning / end of line.
- `C-Home` / `C-End` — beginning / end of file.
- `C-Left` / `C-Right` — word left/right.
- `M-b` / `M-f` — word left/right.
- `PageUp` / `PageDown` — move by one screen.

### Editing

- Type characters — insert.
- `Enter` — insert newline.
- `Tab` — indent to next tab stop (tab width = 2 spaces).
- `Backspace` — delete char before point.
- `Delete` — delete char at point.
- `C-k` — kill to end of line (or join with next line if already at EOL).

### Region (selection) / clipboard

Mixtape has an Emacs-like mark/region.

- `C-Space` — set mark.
- `C-w` — cut region.
- `M-w` — copy (yank) region.
- `C-y` — paste (yank).
- `C-Backspace` — kill previous word.
- `M-Backspace` — kill previous word (without undo wrapper in current implementation).
- `C-u` — kill from point back to beginning of line.

The editor also syncs its internal kill/yank buffer to the system clipboard.

---

## Mixtape DSL overview

Mixtape is a **concatenative**, **stack-based** language:

- Programs are sequences of tokens.
- Most tokens are *words* that consume values from the stack and push results.
- There is no special syntax for function calls; it is all postfix.

Example:

```tape
3 4 + 2 *    ; => (3+4)*2
```

### Comments

- `;` starts a comment to end of line.

### Values / types

Runtime values implement a common `Val` interface. Core types:

- **Num** — floating point number. Can also represent booleans: `0` = false, non-zero = true.
- **Nil** — the `nil` value.
- **Str** — string literals: `"hello"`.
- **Sym** — symbol / word name, e.g. `dup`, `:bpm`, `foo`.
- **Vec** — vector (heterogeneous list): `[1 2 "x" {dup}]`.
- **Tape** — finite audio buffer (`nframes × nchannels`).
- **Stream** — potentially infinite audio stream (generator).
- **Wavetable** — table of single-cycle waves, for band-limited oscillators.

There is also a map-like environment (`set`/`get`) for variables.

### Stack effects

Documentation uses a Forth-like stack comment form:

`word: ( inputs -- outputs )`
Environment usage is noted as `ENV: :var ...`.

### Evaluation model

- Tokens are parsed into a `Vec` of `Token`s (each has position info).
- Evaluating a `Vec` evaluates its items left-to-right.
- `eval` evaluates a value (often a quoted `Vec`).

**Quoting**:

- `{` starts quoting; `}` ends quoting.
- A quoted block evaluates to a `Vec` of tokens (a “closure-like” block).

Example:

```tape
{ 2 + }    ; pushes a quoted block
5 swap eval  ; => 7
```

### Vectors

- `[` marks the stack.
- `]` collects everything pushed since the last `[` into a `Vec`.

Example:

```tape
[ 1 2 3 ]   ; => pushes Vec [1 2 3]
```

### Environments (variables)

- `set` / `get` store/fetch values from the current environment.
- Environments are **stacked**: `(` pushes a new environment frame, `)` pops it.

Example:

```tape
( 100 ":bpm" set :bpm )  ; => 100, outside parens :bpm is unchanged
```

### Syntax sugar

The parser expands these syntactic shorthands:

- `:name` → `":name" get` (fetch env var)
- `@foo` → `"foo" get`
- `>foo` → `"foo" set`

Time suffixes (numeric literals):

- `1s` → `1 seconds` (frames)
- `1b` → `1 beats` (frames)
- `1t` → `1 ticks` (frames)
- `1p` → `1 periods` (frames)

Other literal parsing:

- Ratios like `1/4` parse as a number (`0.25`).
- MIDI notes like `c-4`, `c#4` parse to MIDI numbers.

### Methods (type-dispatched words)

Some words are **methods**: the same token dispatches based on the runtime type of the receiver. For example:

- `len` works for `Vec` and `Streamable`.
- `+` works for numbers/streams and also strings (`Str.+`).

Mixtape searches for a method matching the word name and stack arity (up to 3 args).

---

## Words reference

Below is a categorized list of all available words from:

- Go built-ins (`RegisterWord`, `RegisterMethod`)
- the standard library (`assets/prelude.tape`)

Examples are small, runnable fragments.

### Conventions

- `b` is a boolean `Num` (`0` false, non-zero true).
- `S` means “streamable”: `Num`, `Vec` of samples, `Tape`, or `Stream`.
- Many math and DSP ops accept either `Num` or `Streamable`.

---

## 1) Core / stack / control

### `nil`
`( -- nil )` — push nil.

```tape
nil nil?   ; => -1
```

### `throw`
`( x -- )` — throw an exception carrying `x`.

### `catch`
`( body -- x|nil )` — evaluate `body`; if it throws, return thrown value, else `nil`.

```tape
{ "ok" } catch nil?      ; => -1
{ "err" throw } catch   ; => "err"
```

### `loop`
`( body -- )` — repeat evaluating `body` until `break`/`throw`.

### `break` (stdlib)
`( -- )` — exit current `loop` by throwing `nil`.

### `stack`
`( -- v )` — snapshot current value stack as a `Vec`.

### `log`
`( x -- x )` — log top of stack without consuming it.

### `str`
`( x -- str )` — stringify a value.

### `=`
`( x y -- b )` — equality (type-aware).

### Stack shuffles

- `drop` — `( x -- )`
- `nip` — `( x y -- y )`
- `dup` — `( x -- x x )`
- `swap` — `( x y -- y x )`
- `over` — `( x y -- x y x )`

Examples:

```tape
2 9 over -   ; => 7 (and leaves 2 under it)
```

### Environment frames

- `(` — `( -- )` push new environment frame
- `)` — `( -- )` pop environment frame

### Stack marks / vector building

- `[` — `( -- )` push a stack mark
- `]` — `( <xs> -- v )` collect values since last mark into a `Vec`

### Quoting

- `{` — start quote mode
- `}` — end quote mode and push quoted `Vec`

### Variables

- `set` — `( x k -- )` set env var named by string or symbol `k`
- `get` — `( k -- x )` fetch env var

Related syntax:

- `:foo` is shorthand for `":foo" get`.
- `>foo` is shorthand for `"foo" set`.

### `eval`
`( x -- <xs> )` — evaluate a value (often a quoted `Vec`).

### Iteration protocol

- `iter` — `( I -- i )` obtain iterator from iterable (Num/Vec)
- `next` — `( i -- i x|nil )` advance iterator (iterator is itself callable via `eval`)

Example:

```tape
3 iter
next  ; => 0
next  ; => 1
next  ; => 2
next nil?
```

### `vdup`
`( x n -- [xs] )` — vector of `n` copies of `x`.

---

## 2) Conditionals and comparisons

### `if` (Num method)

- `( b then -- )`
- `( b then else -- )`

`b` is a number (0=false).

```tape
5 3 > "gt" "lt" if   ; => "gt"
```

### Comparisons (Num methods)

- `<` `( n n -- b )`
- `<=` `( n n -- b )`
- `>=` `( n n -- b )`
- `>` `( n n -- b )`

### Boolean helpers (stdlib)

- `true` `( -- -1 )`
- `false` `( -- 0 )`
- `false?` `( x -- b )`  (true if x == 0)
- `true?` `( x -- b )`   (true if x != 0)
- `not` `( x -- b )`     (same as `0 =`)
- `!=` `( x y -- b )`
- `nil?` `( x -- b )`

### `assert` (stdlib)
`( body -- )` — evaluates `body`, throws if result is false.

---

## 3) Numbers, math, random

### Constants

- `e` `( -- n )`
- `pi` `( -- n )`

### Unary math (Num or Streamable)

Each is `( S -- s|n )`:

`abs sign square exp exp2 log10 log2 floor ceil trunc round sin cos tan asin acos atan sinh cosh tanh asinh acosh atanh`

Example:

```tape
1 exp   ; => 2.718281828459045
pi 2 / sin  ; => 1
```

### Binary math (Num or Streamable)

Each is `( S S -- s|n )`:

`+ - * / mod rem pow atan2 hypot min max`

### `clamp`
`( S min max -- s|n )` — clamp to range.

```tape
-5 0 10 clamp   ; => 0
```

### Random

- `rand` `( -- n )` — random float in `[0,1)`.
- `rand/seed` `( n -- )` — reseed RNG used by `rand`.

---

## 4) Strings, symbols, parsing, paths

### Strings

- String literal: `"hello"`

### `sym` (Str method)
`( str -- sym )` — convert a string to a symbol.

### `+` (Str method)
`( str1 str2 -- str )` — concatenate strings.

### `path/join` (Str method)
`( str1 str2 -- str )` — join filesystem paths.

### Parsing

- `parse` (Str method) `( str -- v )` — parse string into AST tokens (`Vec`).
- `parse1` (Str method) `( str -- x )` — parse and return first token.

Example:

```tape
"1234 4321.5 *" parse    ; => {1234 4321.5 *} (as a Vec)
"42 24" parse1            ; => 42
```

---

## 5) Vectors (lists)

### `len` (Vec method)
`( v -- n )`

### `at` (Vec method)
`( v k -- x )`

### `clone` (Vec method)
`( v -- v )` — shallow copy.

### `push` (Vec method)
`( v x -- v )` — append.

### `pop` (Vec method)
`( v -- v x )` — remove last item.

### Higher-order vector ops

- `each` `( v body -- )` — for each item, push it and `eval` body.
- `map` `( v body -- v )` — map body over items (body leaves one result).
- `reduce` `( v body -- x )` — fold-left, returns `nil` for empty vector.

Examples:

```tape
[2 3 4] { 1 + } map     ; => [3 4 5]
[2 3 4] {+} reduce      ; => 9
```

### `partition` (Vec method)
`( v size step -- [vs] )` — sliding windows.

### `tape` (TapeProvider method)
`( x -- t )` — convert a `TapeProvider` to a `Tape`.

Notes:

- A flat numeric `Vec` is a `TapeProvider` (mono tape).
- A `Wavetable` is also a `TapeProvider` (first wave).

---

## 6) Iteration utilities (stdlib)

### `for`
`( I body -- <xs> )` — evaluate `body` for each value yielded by iterator from `I`.

### `zip`
`( [xs] -- [[ys]] )` — lockstep pull from iterators until one ends.

### `seq`
`( body [syms] -- <xs> )` — sequencer helper; on each step, pulls one value from each symbol’s iterator and `set`s it, then runs `body`.

See `examples/seq.tape`.

---

## 7) Time, pitch, amplitude (stdlib)

### Time → frames

- `seconds` `( dur -- nframes )`
- `beats` `( ENV: :bpm | beats -- nframes )`
- `periods` `( ENV: :freq | periods -- nframes )`
- `ticks` `( ENV: :bpm :tpb | ticks -- nframes )`

Also available as literal suffixes: `1s 1b 1p 1t`.

### Pitch helpers

- `st` `( semitones -- ratio )` — semitone offset as frequency multiplier.
- `cents` `( cents -- ratio )`
- `mtof` `( midi-note -- freq )`

### Amplitude

- `db` `( db -- amp )`
- `gain` `( S db -- s )` — apply gain in dB.

### Unipolar/bipolar

- `uni` `( bipolar -- unipolar )`  maps `[-1,1] -> [0,1]`
- `bi` `( unipolar -- bipolar )`   maps `[0,1] -> [-1,1]`

---

## 8) Envelopes

### Envelope segments (built-ins)

These build a mono `Tape` segment using `:start :end :nf`:

- `/line` `( ENV: :start :end :nf | -- t )`
- `/exp` `( ENV: :start :end :nf | k -- t )`
- `/log` `( ENV: :start :end :nf | k -- t )`
- `/cos` `( ENV: :start :end :nf | -- t )`
- `/pow` `( ENV: :start :end :nf | p -- t )`
- `/sigmoid` `( ENV: :start :end :nf | k -- t )`

### Envelope builders (stdlib)

- `start:end` `( [ns] -- | SETS: :start :end )` — prepare segment endpoints.
- `env` `( [ys] [ds] [segs] -- env )` — build a multi-segment envelope.
- `adsr` `( a d s r dur -- env )`
- `perc` `( a r -- env )`

See `examples/env.tape`, `examples/adsr.tape`, `examples/perc.tape`.

---

## 9) Tapes (finite buffers)

### Allocation / generators

- `tape1` `( nframes -- t )` — mono tape.
- `tape2` `( nframes -- t )` — stereo tape.

Single-cycle wave generators (mono `Tape`; size 0 means default internal size):

- `tape/sin`
- `tape/tanh`
- `tape/triangle`
- `tape/square`
- `tape/pulse` (uses `:pw`)
- `tape/saw`

### Tape methods

- `shift` `( t amount -- t )` — rotate samples in-place (mutates).
  - `amount < 1` is treated as a fraction of length.
- `resample` `( t converter ratio -- t )` — resample.
  - converters: `SRC_SINC_BEST_QUALITY`, `SRC_SINC_MEDIUM_QUALITY`, `SRC_SINC_FASTEST`, `SRC_ZERO_ORDER_HOLD`, `SRC_LINEAR`.
- `at` `( t frameIndex -- frame )` — get a frame (always returned as a `Vec` of channel samples).
- `at/phase` `( t phaseStream -- s )` — sample a tape using a phase stream (wavetable-style).
- `slice` `( t start end -- t )` — sub-tape `[start,end)`.
- `+@` `( t t2 offset -- t )` — mix `t2` into `t` at offset (mutates, grows `t` if needed).

### Loading audio

- `load` (Str method) `( path -- t )` — load `.tape`, `.wav`, `.mp3`.
  - If you omit the extension, Mixtape tries `.tape`, `.wav`, then `.mp3`.

Example:

```tape
"~/samples/kick" load   ; loads ~/samples/kick.wav if it exists
```

---

## 10) Streams (signal processing)

### Stream basics

- `~` `( S -- s )` — coerce to stream.
  - A `Num` becomes an infinite constant stream.
  - A numeric `Vec` becomes a finite mono stream.
  - A `Tape` is streamable.

- `~empty` `( nchannels -- s )` — empty stream.

### Rendering / collecting

- `take` `( s nframes -- t )` — render first `nframes` frames into a `Tape`.
- `frames` `( s -- v )` — collect all frames into a `Vec` (stream must be finite).

### Channel utilities

- `mono` `( S -- s )` — sum/convert to mono.
- `stereo` `( S -- s )` — ensure stereo.

### Stream methods

- `len` (Streamable method) `( S -- n )` — number of frames, or `0` if infinite.
- `join` (Streamable method) `( S S -- s )` — concatenate.

---

## 11) Oscillators and noise

### Basic phase / impulse

- `~phasor` `( ENV: :freq :phase | -- s )` — phase accumulator in `[0,1)`.
- `~impulse` `( ENV: :freq :phase | -- s )` — band-limited impulse train.

### Stdlib oscillators (built from tapes + phasor)

- `~sin` `( ENV: :freq :phase | -- s )`
- `~tanh` `( ENV: :freq :phase | -- s )`
- `~triangle` `( ENV: :freq :phase | -- s )`
- `~square` `( ENV: :freq :phase | -- s )`
- `~pulse` `( ENV: :freq :phase :pw | -- s )`
- `~saw` `( ENV: :freq :phase | -- s )`

### Noise

- `~noise` `( ENV: :seed | -- s )` — white noise.
- `~pink` `( ENV: :seed | -- s )` — pink noise.
- `~brown` `( ENV: :seed | step -- s )` — brown noise random walk.

---

## 12) DSP / effects

### Filters and smoothing

- `dc*` `( S alpha -- s )` — DC blocker with smoothing `alpha`.
- `dc` `( S -- s )` — DC removal with `alpha = 1 - 1/SR`.
- `onepole` `( S alpha -- s )` — 1-pole smoother (higher alpha = more smoothing).

### Utility analysis

- `peak` `( S -- s )` — per-frame `max(abs(samples))`.

### Sample & hold

- `sh` `( S rate -- s )` — sample-and-hold.

### Delay / comb

- `delay` `( S nframes -- s )`
- `comb` `( S delay fb -- s )` — feedback comb filter.

### One-sample delay

- `z1*` `( S initFrame -- s )` — initFrame can be Num or Vec.

Stdlib convenience:

- `z1` `( s -- s )` — one-sample delay with zero init.

### Saturation

- `softclip` `( S mode -- s )`
  - `0` tanh
  - `1` atan (scaled)
  - `2` cubic soft clip
  - `3` softsign

### Other

- `skip` `( S nframes -- s )` — drop first `nframes`.
- `pan` `( S pan -- s )` — equal-power pan; pan in `[-1,1]`.
- `mix` `( [Ss] ratio -- s )` — mix streams by ratio (clamped `[0,1]`).
- `stretch` (stdlib) `( S factor -- s )` — resample/"time-stretch" using `SRC_LINEAR`.

---

## 13) Wavetables and FM

### `wt`
`( x -- wt )` — coerce value to `Wavetable`.

Accepted inputs:

- a `Tape` / `TapeProvider`
- a numeric `Vec` (becomes a mono tape wave)
- a `Vec` of `TapeProvider`s (waveset)
- a **finite** stream (rendered to a tape)

### `~wt`
`( ENV: :freq :phase :morph | wt -- s )` — wavetable oscillator with mipmapped band-limiting.

### `~fm`
`( ENV: :freq :mod :index :phase | wt -- s )` — wavetable FM oscillator.

Stdlib wavetables:

- `wt/sin wt/tanh wt/triangle wt/square wt/pulse wt/saw`

---

## 14) Unison

### `unison`
`( ENV: :freq :voices :spread :detune :phaseRand | body -- s )`

Evaluates `body` once per voice in an isolated environment frame, adjusting `:freq` per voice. Voices are panned and mixed down.

Parameters:

- `:voices` (Num) — number of voices (>= 1).
- `:spread` (Num) — stereo spread (0..1).
- `:detune` (Num) — detune range in cents.
- `:phaseRand` (Num) — randomize initial phase (0..1).

See `examples/unison*.tape`.

---

## 15) Vital-inspired ports

### `vital/decimate`
`( S factor sharp -- s )` — multistage halfband decimator.

- `factor` must be a power of two.
- `sharp` non-zero enables a sharper final stage.

### `vital/svf`
`( ENV: :cutoff :q :drive :blend | S -- s )` — state-variable filter.

- `:blend` in `[-1,1]` maps lowpass(-1) → bandpass(0) → highpass(+1).

See `examples/vital_svf_demo.tape`.

---

## Quick examples

### A 1-second 440Hz sine

```tape
( 440 >:freq
  ~sin
  1s take
)
```

### An ADSR-shaped saw and play

```tape
( 110 >:freq
  10/100b 10/100b 0.6 20/100b 1b adsr >:env
  ~saw :env *
  clip
)
```

### Sequencing (sketch)

See `examples/seq.tape` for the full pattern; `seq` is a helper for iterating multiple symbol streams in lockstep.

---

## Notes for LLMs / tooling

- `assets/prelude.tape` is effectively the “stdlib” and includes doc comments with stack effects.
- `tests/*.tape` files contain comprehensive executable specifications of many words.
- Parsing expands syntactic sugar (`:name`, `@foo`, `>foo`, and time suffixes) *at parse time*.
- Many operators (`+`, `*`, `sin`, …) are overloaded to work on both numbers and streams (sample-wise).
- Streams are lazy; converting to a `Tape` is done with `take` (or automatically in the GUI when the eval result is finite).
