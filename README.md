# Mixtape

> **Work in progress.** Mixtape is evolving, but its language, editor, and
> executable examples are all in this repository.

Mixtape is a compact, stack-based language for making sound, paired with an
interactive editor and player. You describe an audio signal as a sequence of
small transformations—an oscillator, an envelope, a filter, a mixer—and
Mixtape evaluates the sequence from left to right. It is implemented in Go and
ships with a practical standard library in `assets/prelude.tape`.

The project is deliberately small enough to explore from the source, while
still being useful for quick synthesis sketches. This README starts with the
normal workflow, explains the language’s mental model, and then serves as the
complete reference for built-ins and standard-library words.

> `assets/prelude.tape` is the source of truth for standard-library word
> definitions and their stack-effect comments. The `examples/` and `tests/`
> directories provide runnable patches and executable specifications.

## A first patch

Build the program, open a scratch buffer, paste this patch, and press `C-p`:

```tape
( 220 f           ; set the oscillator frequency to A3
  ~saw            ; make an infinite sawtooth stream
  1s take         ; render its first second to a finite tape
)
```

You should hear a one-second saw wave. The parentheses create a temporary
environment, `f` sets `:freq`, and `take` is the point where a lazy stream
becomes a playable audio buffer. That sequence—set controls, build a stream,
then render or play it—is the basic Mixtape workflow.

---

## Build & run

```sh
go build
./mixtape                # start GUI with an empty buffer
./mixtape mypatch.tape   # start GUI with file loaded
```

Run tests:

```sh
make test
```

The GUI needs a working desktop/OpenGL/audio environment. For language
experiments and automation, use batch evaluation instead; it does not start
the GUI.

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
- `-f <path>` — evaluate a script file and exit.
- `-e <string>` — evaluate an inline script and exit.
- `-prof <prefix>` — in batch mode, write pprof CPU and heap profiles to
  `<prefix>.cpu` and `<prefix>.mem`.

`-f` and `-e` may each be supplied more than once. Batch targets run in the
order they occur on the command line, in the same VM, so definitions from an
earlier target are available to later ones.

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

Before loading the prelude, Mixtape sets these environment variables:

- `:bpm` from `-bpm`
- `:tpb` from `-tpb`
- `:nf` = frames-per-beat = `sr / (bpm/60)`

The prelude then sets these defaults:

- Oscillators: `:freq = 440`, `:phase = 0`, `:pw = 0.5`.
- Filters: `:cutoff = 1200`, `:q = 0.7`, `:blend = 0`, `:gain = 1`.
- FM and noise: `:mod = 0`, `:index = 1`, `:seed = 0`.
- Envelopes: `:start = 0`, `:end = 1`.
- Resampling: `:resample/converter = :resample/SRC_LINEAR`.

The `f` helper sets `:freq`: `220 f` is equivalent to `220 >:freq`.

---

## The GUI editor

When you run `./mixtape [file.tape]`, Mixtape opens a source buffer. Successful
audio evaluations also show a waveform and can be played immediately. The
editor is intentionally keyboard-first, with familiar Emacs-style movement,
selection, and multi-key commands.

### Screens

- `F1` — open the read-only in-app help screen.
- `F2` — return to the editor.
- `F3` — open the sample browser. It can browse directories and play selected
  `.wav` or `.mp3` files with `C-p`. Samples load in the background and show a
  progress line at the bottom of the screen, including `Resampling` progress
  when a sample-rate conversion is needed. `C-g` stops sample-browser
  playback and cancels a pending load. `Enter` opens a selected `.tape` file
  in the editor and switches to F2; `M-w` copies a ready-to-paste
  `"/absolute/path" load` expression to the clipboard.

### Evaluating / playing

- `C-p` — evaluate buffer and **play** the resulting tape/stream.
- `C-Enter` — evaluate buffer without starting playback.
- `C-g` or `Escape` — cancel the current evaluation (and reset transient state).

`C-Enter` is useful while developing a patch: it checks the result without
interrupting what is playing. `C-p` evaluates only when the buffer changed;
otherwise it replays the last successful result. Evaluation happens in the
background, and the editor shows progress while a finite stream is rendering.

### Buffers

- `C-x n` — switch to next buffer
- `C-x p` — switch to previous buffer
- `C-x o` — switch to last buffer
- `C-x b` — open buffer switcher

### Files

- `C-x f` — open file
- `C-x s` — save the current buffer; prompts for a path if it has none.
- `C-x C-s` — save as (always prompts for a path).
- `C-x k` — kill the current buffer (asks for confirmation if it has changes).

The file and buffer switchers are searchable: type to filter, use the arrow
keys/PageUp/PageDown/Home/End to select, and `Enter` to open. Typing enters
filter mode; `Backspace` deletes the last character of the filter text.
`Escape` drops filter mode and clears the filter; press it again (or press
`C-g`) to cancel. In the file switcher, `Backspace` navigates to the parent
directory only after filter mode has been dropped.

### Quit / undo

- `C-q` — quit (asks before discarding any unsaved buffer).
- Undo:
  - `C-z`
  - `C-x u`
  - `C-S--`

### Font size

- `C-S-=` — increase font size
- `C--` — decrease font size
- `C-0` — reset to default font size

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
- `M-Backspace` — kill previous word.
- `C-u` — kill from point back to beginning of line.

The editor also syncs its internal kill/yank buffer to the system clipboard.

When a command needs an answer—such as choosing a save path or confirming a
destructive action—Mixtape opens a small modal prompt. `Escape` and `C-g`
cancel it.

---

## Mixtape DSL overview

Mixtape is a **concatenative**, **stack-based** language. Instead of nesting
function calls, place a value on the stack and then place the word that should
consume it. The result is left on the stack for the next word:

- Programs are sequences of tokens.
- Most tokens are *words* that consume values from the stack and push results.
- There is no special syntax for function calls; it is all postfix.

Example:

```tape
3 4 + 2 *    ; => (3+4)*2
```

Read this as “push 3, push 4, add them, push 2, multiply.” A program’s final
stack value is its result.

### Comments

- `;` starts a comment to end of line.

### Values / types

The language has a small set of runtime values:

- **Num** — floating point number. Can also represent booleans: `0` = false, non-zero = true.
- **Nil** — the `nil` value.
- **Str** — string literals: `"hello"`.
- **Sym** — symbol / word name, e.g. `dup`, `:bpm`, `foo`.
- **Vec** — vector (heterogeneous list): `[1 2 "x" {dup}]`.
- **Tape** — finite audio buffer (`nframes × nchannels`).
- **Stream** — potentially infinite audio stream (generator).
- **Wavetable** — table of single-cycle waves, for band-limited oscillators.

There is also a map-like, dynamically scoped environment (`set`/`get`) for
controls such as oscillator frequency and filter cutoff.

The audio types are worth distinguishing early. A `Tape` is a finite buffer of
frames, ready to display or play. A `Stream` produces frames lazily and can be
infinite; oscillators and noise generators normally return streams. Most DSP
words preserve that laziness, and `take` explicitly renders a bounded part of
a stream into a tape.

### Stack effects

Documentation uses a Forth-like stack comment form. Inputs are listed on the
left of `--`, with the deepest stack value first; outputs are on the right:

`word: ( inputs -- outputs )`
Environment usage is noted as `ENV: :var ...`. For example,
`( ENV: :cutoff | S -- s )` says that a word consumes a streamable `S`,
returns a stream, and reads its cutoff from the environment rather than the
value stack.

### Evaluation model

- Tokens are parsed into a `Vec` of `Token`s (each has position information).
- Evaluating a `Vec` evaluates its items left-to-right.
- `eval` evaluates a value, most often a quoted `Vec` used as a function body.

**Quoting**:

- `{` starts quoting; `}` ends quoting.
- A quoted block evaluates to a `Vec` of tokens: data that can be stored,
  passed to a higher-order word, or executed later.

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

Scoped environments keep local patch controls from leaking into the rest of a
program. They are especially useful for configuring an oscillator or filter:

```tape
( 880 >:freq
  ~sin
  1s take
)
```

Example:

```tape
( 100 ":bpm" set :bpm )  ; => 100, outside parens :bpm is unchanged
```

### Syntax sugar

The parser expands these shorthands before evaluation, so they are convenient
spelling rather than special runtime values:

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

Below is a categorized list of every available word, from:

- Go built-ins (`RegisterWord`, `RegisterMethod`)
- the standard library (`assets/prelude.tape`)

The reference is organized by the job a word performs rather than by where it
is implemented. You can use it either as a lookup table or as a tour: begin
with streams and oscillators, then move on to envelopes and effects when you
want to shape a sound. Examples are small, runnable fragments; surround an
audio-producing expression with `take` when you want a finite result.

### Conventions

- `b` is a boolean `Num` (`0` false, non-zero true).
- `S` means “streamable”: `Num`, `Vec` of samples, `Tape`, or `Stream`.
- Many math and DSP ops accept either `Num` or `Streamable`.

When a heading says “method,” you still write only the word itself. For
example, `[1 2 3] len` dispatches to the `Vec` implementation of `len`; there
is no `Vec.len` syntax in a patch.

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

### `sr`
`( -- n )` — push the active sample rate, set by `-sr` (48000 by default).

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

### Collection / stream utilities (stdlib)

- `sum` `([Ss|ns] -- s|n)` — sum a vector of numbers or streamables (empty
  input returns `nil`).
- `avg` `([Ss|ns] -- s|n)` — average a non-empty vector of numbers or streamables.
- `distribute` `([ns] n -- [ns])` — scale numeric values so their sum is `n`;
  the input sum must not be zero.
- `clip` `(S -- s)` — constrain samples to `[-1, 1]`.
- `cat` `([Ss] -- s)` — concatenate a vector of streamables (empty input
  returns `nil`).
- `repeat` `(S n -- s)` — concatenate `n` copies of a streamable.

---

## 6) Iteration utilities (stdlib)

Iteration is useful for building event lists, parameter sequences, and other
control data before it becomes audio. Iterators are deliberately simple: they
yield a value at a time and eventually yield `nil`.

### `for`
`( I body -- <xs> )` — evaluate `body` for each value yielded by iterator from `I`.

### `zip`
`( [xs] -- [[ys]] )` — lockstep pull from iterators until one ends.

### `seq`
`( body [syms] -- <xs> )` — sequencer helper; on each step, pulls one value from each symbol’s iterator and `set`s it, then runs `body`.

See `examples/seq.tape`.

---

## 7) Time, pitch, amplitude (stdlib)

Mixtape measures rendered audio in frames. These helpers let a patch express
musical durations and pitches without hard-coding the current sample rate or
tempo. The time suffixes are especially convenient in a patch: `1b` means one
beat at the current `:bpm`, while `250ms` is not a special literal—use
`0.25s` instead.

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

An envelope is a finite stream of control values, usually multiplied with an
oscillator to shape its amplitude. Segment words use `:start`, `:end`, and
`:nf`; the `env`, `adsr`, and `perc` builders handle those bookkeeping values
for normal multi-segment envelopes.

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

Use a tape when you need an already-rendered piece of audio: to play it in the
GUI, load a sample, inspect individual frames, splice material together, or
use one cycle as an oscillator source. Unlike streams, tapes have a known
length and channel count.

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
- `resample` `( t ratio -- t )` — resample a tape. `ratio` is output rate /
  input rate and must be between `1/16` and `16`.
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

Streams are Mixtape’s signal-flow building block. A stream is pulled only when
something needs its next frame, so a chain such as `~saw lp2 clip` does not do
work until it is rendered or played. Numeric arguments to many DSP words may
themselves be streams, which makes modulation a normal part of the language.

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

Oscillators and noise generators create infinite mono streams. Put their
controls in a local environment, then use `take` (or multiply by a finite
envelope) to decide how much audio to render. The standard oscillators are
implemented from a phase stream and a single-cycle tape, so they also provide
useful examples of small Mixtape compositions.

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

Effects consume a streamable and return a transformed stream, preserving the
input’s duration where that makes sense. This makes their order explicit:
`~saw lp2 softclip` filters before saturating, while reversing those words
produces a different sound.

### Filters and smoothing

- `dc*` `( S alpha -- s )` — DC blocker with smoothing `alpha`.
- `dc` `( S -- s )` — DC removal with `alpha = 1 - 1/SR`.
- `onepole` `( S alpha -- s )` — 1-pole smoother (higher alpha = more smoothing).
- `lp1`, `hp1`, `ap1` `( ENV: :cutoff | S -- s )` — first-order lowpass,
  highpass, and allpass filters.
- `ap2`, `notch2` `( ENV: :cutoff :q | S -- s )` — second-order allpass and
  notch filters.
- `ls2`, `hs2`, `peak2` `( ENV: :cutoff :q :gain | S -- s )` — second-order
  low-shelf, high-shelf, and peaking filters. `:gain` is a *linear* gain
  multiplier here, unlike the dB argument accepted by `gain`.

### State-variable filter ports

- `svf` `( ENV: :cutoff :q :blend | S -- s )` — continuously blends
  lowpass (`:blend = -1`), bandpass (`0`), and highpass (`1`).
- `lp2`, `bp2`, `hp2` `( ENV: :cutoff :q | S -- s )` — 2-pole low-, band-,
  and highpass ports of `svf`.
- `lp4`, `bp4`, `hp4` `( ENV: :cutoff :q | S -- s )` — corresponding 4-pole
  filters, made by cascading the 2-pole versions.
- `ap4`, `notch4` `( ENV: :cutoff :q | S -- s )` — 4-pole allpass and notch.
- `peak4` `( ENV: :cutoff :q :gain | S -- s )` — 4-pole peaking filter.

See `examples/svf_demo.tape`.

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

### Resampling and tuning

- `resample` `( S ratio -- s )` — resample a streamable using an output/input
  sample-rate ratio from `1/16` through `16`. A `Tape` input yields a `Tape`;
  other streamables yield a stream.
- `tune` `( S ratio -- s )` — change pitch by a frequency multiplier without
  changing the sample rate; it is a convenience wrapper around `resample`.

Select the converter by setting `:resample/converter` to one of these preset
values (the default is `:resample/SRC_LINEAR`):

- `:resample/SRC_SINC_BEST_QUALITY`
- `:resample/SRC_SINC_MEDIUM_QUALITY`
- `:resample/SRC_SINC_FASTEST`
- `:resample/SRC_ZERO_ORDER_HOLD`
- `:resample/SRC_LINEAR`

### Other

- `skip` `( S nframes -- s )` — drop first `nframes`.
- `pan` `( S pan -- s )` — convert the input to mono and return a stereo,
  equal-power panned stream. `pan` may be a control stream and is clamped to
  `[-1,1]`.
- `mix` `( [Ss] ratio -- s )` — interpolate between adjacent same-channel
  streams. `ratio` may be a control stream and is clamped to `[0,1]`; with
  two inputs, `0.7` means 30% of the first plus 70% of the second.

---

## 13) Wavetables and FM

A wavetable stores one or more same-length, single-cycle waves. Multiple waves
form a waveset, and `~wt` can continuously morph between them. Mixtape builds
band-limited mip levels lazily, which keeps bright waves more usable at higher
frequencies.

### `wt`
`( x -- wt )` — coerce value to `Wavetable`.

Accepted inputs:

- a `Tape` / `TapeProvider`
- a numeric `Vec` (becomes a mono tape wave)
- a `Vec` of `TapeProvider`s (waveset)
- a **finite** stream (rendered to a tape)

### `~wt`
`( ENV: :freq :phase :morph | wt -- s )` — wavetable oscillator with mipmapped band-limiting.

`:morph` may be a number or mono control stream. It is clamped to `[0,1]` and
crossfades through a waveset; omitted or invalid values default to `0`.

### `~fm`
`( ENV: :freq :mod :index :phase | wt -- s )` — wavetable FM oscillator.

`:freq`, `:mod`, and `:index` can be control streams; `:phase` is a numeric
initial phase. `:index` defaults to `1`.

Stdlib wavetables:

- `wt/sin wt/tanh wt/triangle wt/square wt/pulse wt/saw`

---

## 14) Unison

`unison` is a compact voice allocator for a quoted oscillator body. It creates
detuned copies, gives each a stereo position, and mixes them to a single stereo
stream—handy for supersaw-style patches without manually repeating the same
voice setup.

### `unison`
`( ENV: :freq :voices :spread :detune :phaseRand | body -- s )`

Evaluates `body` once per voice in an isolated environment frame, adjusting `:freq` per voice. Voices are panned and mixed down.

Parameters:

- `:voices` (Num) — number of voices (>= 1).
- `:spread` (Num) — stereo spread (0..1).
- `:detune` (Num) — detune range in cents.
- `:phaseRand` (Num) — randomize initial phase (0..1).

All four parameters are optional: their defaults are `1`, `0`, `0`, and `0`.

See `examples/unison*.tape`.

---

## Recipes

These are deliberately small, but each is a complete patch. In the GUI, place
one in a buffer and press `C-p`; in batch mode, append `frames` if you want to
print individual samples rather than a tape summary.

### Hear a 1-second 440 Hz sine

```tape
( 440 >:freq
  ~sin
  1s take
)
```

The environment frame holds the oscillator’s frequency only for this patch.
`~sin` stays lazy until `take` asks for exactly one second of frames.

### Shape a saw wave with an ADSR envelope

```tape
( 110 >:freq
  10/100b 10/100b 0.6 20/100b 1b adsr >:env
  ~saw :env *
  clip
)
```

`adsr` produces a finite control signal. Multiplying it with the infinite
oscillator therefore makes the result finite, which the GUI can render and
play directly.

### Build step-based patterns

See `examples/seq.tape` for a complete pattern. `seq` advances several named
iterators in lockstep, assigns their current values to environment variables,
and evaluates a body on each step—useful for driving pitch, timing, and other
parameters together.

---

## Project notes

- `assets/prelude.tape` is effectively the “stdlib” and includes doc comments with stack effects.
- `tests/*.tape` files contain comprehensive executable specifications of many words.
- Parsing expands syntactic sugar (`:name`, `@foo`, `>foo`, and time suffixes) *at parse time*.
- Many operators (`+`, `*`, `sin`, …) are overloaded to work on both numbers and streams (sample-wise).
- Streams are lazy; converting to a `Tape` is done with `take` (or automatically in the GUI when the eval result is finite).

If you are generating or transforming patches programmatically, prefer the
documented stack effects in this README and the comments in `prelude.tape`.
The tests are especially useful when a word’s edge cases matter.
