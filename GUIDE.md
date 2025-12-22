# GUIDE.md — Instructions for LLMs writing Mixtape DSL examples

This repository contains **Mixtape**, a small concatenative (Forth‑like) DSL for **non‑realtime** audio synthesis. Your job (as an LLM) is to generate *small, correct, idiomatic* `.tape` snippets that illustrate the use of a **particular word**.

This guide explains the language model, core data types, common idioms, and the conventions Mixtape examples should follow.

---

## 0) What a “good example” looks like

When asked for an example of a word `WORD`, produce:

1. **A short title** and 1–3 sentence intent summary.
2. **A minimal snippet** that isolates `WORD` (demonstrate stack effect).
3. **A musical / practical snippet** that uses `WORD` in a realistic patch.
4. If the word has tricky edge cases, add a tiny **pitfall / gotcha** note.

### Output conventions for snippets

- Use fenced code blocks tagged `tape`.
- Prefer examples that end by leaving a **Tape** on the stack (so the GUI can show/play it):
  - typically: `... 1b take` or `... 1s take`.
- Keep levels safe: aim for peak amplitude ≤ ~0.8 to avoid clipping.
- Keep durations short: 1–4 beats is usually enough.
- Avoid relying on external files unless the example is specifically about `load`.

### Stack commentary

Mixtape is stack-based; examples are clearer with occasional stack comments:

```tape
# ( a b -- result )  ; stack effect style comment (optional)
```

---

## 1) The execution model (very important)

Mixtape evaluates a buffer left-to-right:

- Tokens either **push literal values** (numbers/strings/vectors) or **execute words/methods**.
- Words consume arguments from the stack and push results.
- Many DSP words read parameters from the **environment** (variables) rather than from the stack.

Typical workflow:

1. Build a (often infinite) **Stream**.
2. Optionally process/mix it.
3. Multiply by an envelope (often a 1‑channel Tape used as a stream).
4. Convert to a finite **Tape** using `take`.

---

## 2) Core syntax

### Literals

- Numbers: `123`, `0.25`, `-3`, `1/4` (ratios parse to floats).
- Strings: `"hello"`
- Comments: `#` to end of line.

### Vectors and quoting

Mixtape has two bracket-like constructs that look similar but behave differently:

1) **Vector builder**: `[ ... ]`

- Evaluates the code inside and collects *all values pushed* into a `Vec`.

```tape
[ 5 2 + 3 * 5 ]  # => [21 5]
```

2) **Quoted code block**: `{ ... }`

- Produces a `Vec` of unevaluated tokens.
- Run it with `eval`.

```tape
{ 2 + 3 * }  # pushes a quoted block (a Vec)
5 swap eval # => 21
```

### Environments / scopes

- `(` pushes a new environment (scope).
- `)` pops it.

Variables are stored in the environment and are commonly named with a leading `:`.

```tape
( 100 >:bpm
  :bpm
)
```

**Important:** Many DSP words read from variables like `:freq`, `:phase`, etc.

---

## 3) Syntactic sugar you MUST use correctly

Mixtape has a few reader expansions (these appear in tests and the README):

- `>foo` expands to `"foo" swap set` (store).
- `@foo` expands to `"foo" get` (load).

And for `:names` (common for parameters):

- `:freq` is a symbol that evaluates to the value stored under the key `":freq"`.
- `>:freq` is the preferred “store into parameter” pattern.

Time suffixes:

- `1s` → `1 seconds`
- `1b` → `1 beats`
- `1p` → `1 periods`
- `1t` → `1 ticks`

---

## 4) Truthiness, control flow, and errors

Truthiness:

- `0` is false.
- Any non-zero number is true.
- Prelude defines `true = -1`, `false = 0`.

Conditionals are **methods on numbers**:

- `cond {then} if`
- `cond {then} {else} if`

Example:

```tape
{ 5 3 > "gt" "lt" if "gt" = } assert
```

### Exceptions and looping

- `throw` throws the top-of-stack value.
- `catch` runs a quoted block and returns either `nil` (no throw) or the thrown value.
- `loop` repeats a quoted block forever until it errors; a special `break` is implemented as `nil throw`.

Prelude provides:

- `break` (implemented as `{ nil throw }`)
- `assert` (throws a message on failure)

---

## 5) Core value types you will encounter

### `Num`

- Floating number (ratios are parsed to float).
- Supports comparisons via methods: `< <= >= >`.
- `+ - * / %` are **words** that do:
  - `Num op Num -> Num`
  - otherwise, sample-wise combination of streams.

### `Str`

Methods:

- `sym` : `"foo" sym -> foo` (a symbol)
- `+` : string concatenation
- `path/join` : filesystem path joining
- `parse` : parse a string into quoted code (`Vec` of tokens)
- `parse1` : parse and return first token

### `Sym`

- Symbols name words/methods/vars.
- If a symbol starts with `:`, it evaluates to the corresponding env value (e.g. `:bpm`).

Methods:

- `get` / `set` (symbolic environment access)

### `Vec`

Vectors are central: they represent lists and also represent quoted programs.

Methods:

- `len`
- `at` (bounds checked)
- `clone`
- `push` / `pop` (note: `pop` returns **two** values: the shortened vec and the popped item)
- `each` (iterate, running a closure per item)
- `map`
- `reduce`
- `partition size step` (sliding windows)
- `tape` (convert numeric vec to a mono Tape of exact samples)

### `Stream`

A `Stream` is a lazy generator of audio frames (mono or stereo). Usually infinite.

Words:

- `~` : convert any Streamable to a Stream (Num, Tape, Stream, etc.)
- `mono` / `stereo` : channel conversion
- `take` : `streamable nframes take -> Tape`

Method on Streamables:

- `join` : concatenate streams in time

### `Tape`

A `Tape` is a finite in-memory buffer.

Words:

- `tape1 nframes` : silent mono tape
- `tape2 nframes` : silent stereo tape

Methods:

- `nf` : number of frames
- `slice start end`
- `shift amount` : circular rotate; amount can be frames or (0..1) fraction of length
- `resample converterType ratio` : sample-rate conversion (uses libsamplerate)
- `+@ offset tape` : mix/add RHS into LHS starting at offset (in-place)

### Loading audio / scripts

- `"path" load` loads `.tape`, `.wav`, or `.mp3`.
- If you load `foo.tape`, Mixtape may cache a rendered `foo.wav` next to it and reuse it if newer.

---

## 6) Built-in words (as implemented in Go)

These are registered in Go code.

### Core stack / meta

- `nil`
- `drop` `nip` `dup` `swap` `over`
- `stack` (pushes the whole value stack as a `Vec`)
- `str` (stringify top value)
- `log` (log top value)
- `sr` (sample rate)
- `=` (equality; prelude defines `!=`)

### Quoting / evaluation / iteration protocol

- `(` `)` (push/pop environment)
- `[` `]` (mark/collect into a vector)
- `{` (begin quote; ends at matching `}` in source)
- `eval`
- `iter` / `next` (iterator protocol used by prelude helpers like `zip`)

### Exceptions / looping

- `throw` `catch`
- `loop`

### Streams

- `~` `take` `mono` `stereo`
- arithmetic stream combinators: `+ - * / %`

### DSP / utilities

- `~impulse` (reads `:freq`, optional `:phase`)
- `dc` / `dc* alpha` (DC blockers)
- `sh` (sample & hold): `input rate -- output`
- `comb` (feedback comb): `input delayFrames feedback -- output`
- `delay` (prepend silence): `streamable nframes delay -- Stream`
- `pan` (equal-power pan): `input pan -- stereo`

### Noise

- `~noise` (reads optional `:seed`)
- `~pink` (reads optional `:seed`)
- `~brown step` (reads optional `:seed`)

### Envelopes (segment generators)

These read from env vars `:start`, `:end`, `:nf`.

- `/line`
- `/exp k`
- `/log k`
- `/cos`
- `/pow p`
- `/sigmoid k`

### Wavetables / oscillators

Wave constructors (used with `phasor`):

- `wave/sin`, `wave/tanh`, `wave/triangle`, `wave/square`, `wave/pulse`, `wave/saw` (take a `size` Num)
- `phasor` (turn a WaveProvider into an oscillator; prelude wraps these as `~sin`, `~saw`, etc.)

Wavetable system:

- `wt` (convert a value to a wavetable)
- `wt/sin`, `wt/tanh`, `wt/triangle`, `wt/square`, `wt/pulse` (uses `:pw`), `wt/saw`
- `~wt` (wavetable osc; reads `:freq`, optional `:phase`, optional `:morph` stream)
- `~fm` (FM osc; reads `:freq`, `:mod`, optional `:index`, optional `:phase`)

### Unison

- `unison` (expects a closure on the stack)
- Reads env vars: `:voices`, `:spread` (stereo spread 0..1), `:detune` (cents), `:phaseRand` (0..1), and requires `:freq`.
- Produces a **stereo** stream.

### “Vital” ports

- `vital/decimate` : `input_stream factor sharp_flag -> output_stream` (factor must be power of two)
- `vital/svf` : state variable filter; reads `:cutoff`, `:q`, `:drive`, `:blend`, `:saturate` and consumes `input` from stack.

---

## 7) Prelude words (defined in `prelude.tape`)

These are not Go-builtins, but are always available.

### Booleans and predicates

- `true` `false`
- `not`
- `true?` `false?`
- `nil?`
- `!=`

### Testing / control

- `assert` : evaluates a quoted block; throws on failure
- `break` : exits `loop`

### Iteration helpers

- `zip` : zips vectors (uses `iter`/`next`)
- `seq` : iterates several vectors in lockstep, binding symbols each step

### Time / units

- `seconds` : seconds → frames (`sr *`)
- `beats`   : beats → frames (`sr * bpm/60`)
- `ticks`   : ticks → frames (`beats * tpb^-1`)
- `periods` : oscillator periods → frames (`sr/freq * periods`)

Convenience:

- `f` : `413 f` stores 413 into `:freq`
- `stretch` : `tape ratio stretch` uses `SRC_LINEAR` resample

### Envelope builder utilities

- `start:end` : helper used to unpack start/end vectors
- `distribute` : normalize a vector to sum to 1 and scale by unit
- `env` : sequences multiple envelope segments and joins them

### Default oscillators

Prelude defines convenient oscillator words backed by `wave/*` + `phasor`:

- `~sin` `~tanh` `~triangle` `~square` `~pulse` `~saw`

These read env vars:

- `:freq` (Hz)
- `:phase` (0..1)
- `:pw` for pulse (0..1)

---

## 8) How to write examples for a particular word

When given a word `WORD`, follow this checklist:

1. **Identify the stack signature** (what it pops/pushes).
   - Many words are easy to infer from tests or Go code.
   - Example: `delay` is `streamable nframes -- stream`.

2. **Identify environment dependencies**.
   - Example: `~sin` requires `:freq` (default is 440 from prelude), optionally `:phase`.
   - Example: `/line` needs `:start`, `:end`, `:nf`.

3. **Demonstrate it in isolation**.
   - If it returns a Stream, end with `take` to produce a Tape.

4. **Demonstrate composition**.
   - Compose with one other orthogonal concept (e.g., envelope + oscillator).

5. **Keep snippets deterministic**.
   - Prefer deterministic sources (`~sin`, `~impulse`, seeded noise) and short renders.

6. **Use scoping** when setting temporary parameters.

```tape
( 220 >:freq
  ~sin
  1b take
)
```

This prevents contaminating later code.

---

## 9) Example templates (copy/paste)

### Template A: “word returns a number / vec”

```tape
# Demonstrate WORD
{ ... WORD ... expected = } assert
```

### Template B: “word returns a stream”

```tape
# Demonstrate WORD (render 1 beat)
(
  # set any required params
  ...
  ... WORD ...
  1b take
)
```

### Template C: “word modifies the environment”

Use `(` `)` to localize changes:

```tape
(
  ... >:someParam
  ...
)
```

---

## 10) Common pitfalls (avoid these)

- **Forgetting to render a Stream**: if your final value is a Stream, the GUI won’t show a waveform unless you `take` it.
- **Forgetting required env vars**: e.g. `unison` requires `:freq`.
- **Confusing `{}` vs `[]`**:
  - `{}` produces quoted code (a Vec of tokens) that does nothing until `eval`.
  - `[]` runs code and collects resulting values.
- **`Vec.pop` returns two values** (vec then item). Write examples accordingly.
- **`Tape.shift` amount rules**:
  - `0 < amount < 1` means fraction of tape length.
  - negative values wrap.
- **Clipping**: mixing many voices or applying high `:drive` can exceed ±1; scale down.

---

## 11) Quick “known-good” idioms (from tests)

### Map/reduce

```tape
{ [2 3 4] { 1 + } map [3 4 5] = } assert
{ [2 3 4] {+} reduce 9 = } assert
```

### Simple oscillator

```tape
(
  440 >:freq
  ~sin
  1b take
)
```

### Envelope segment (manual)

```tape
(
  0 >:start
  1 >:end
  1b >:nf
  /cos        # => Tape (mono)
)
```

### Multiply envelope with oscillator

```tape
(
  (0 >:start 1 >:end 1b >:nf /cos) >:env
  220 >:freq
  :env ~ *
  1b take
)
```

### Using `catch` / `throw`

```tape
{ { "boom" throw } catch "boom" = } assert
```

---

## 12) Validation (what maintainers will do)

Your examples should be compatible with Mixtape’s test/eval pipeline.

- Repo regression tests live in `test.tape`.
- The project is commonly validated with:

```sh
make test
```

If you include assertions, keep them simple and deterministic.

---

## 13) If you’re missing a detail

If the prompt asks for a word whose stack effect is unclear:

- Prefer an example that **only depends on behavior already demonstrated** in `README.md` and `test.tape`.
- State assumptions explicitly in a comment.
- Keep the example minimal so it’s easy to adjust.

---

## Appendix: Minimal word reference by category (fast lookup)

### Stack

`drop nip dup swap over stack`

### Quoting / eval

`{ eval  [ ]  ( )`

### Control

`if loop break throw catch`

### Environment

`set get  >foo @foo  :name  >:name`

### Streams

`~ take mono stereo join  + - * / %`

### Osc / noise

`~sin ~saw ~triangle ~pulse ~square ~tanh  ~impulse  ~noise ~pink ~brown`

### Envelopes

`/line /cos /exp /log /pow /sigmoid  env`

### Tape

`tape1 tape2  nf slice shift resample +@  load`

### Wavetable / FM / Unison

`wt wt/* ~wt ~fm unison`

### Filters / FX

`vital/svf vital/decimate dc dc* comb sh pan delay`
