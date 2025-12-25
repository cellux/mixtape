;; stack comment format

; [TYPE.]NAME: ( ENV-IN* STACK-IN* -- STACK-OUT* ENV-OUT* ) DESCRIPTION

; TYPE: if present, WORD is a method of first stack argument's type
; NAME: name of word
; ENV-IN: env vars used by WORD
; STACK-IN: values expected on stack
; STACK-OUT: values left on stack
; ENV-OUT: env vars modified by WORD

;; stack comment variable names

; x y k : any value
; n : Num
; b : Num interpreted as a boolean: 0=false, anything else is true (-1)
; v : Vec
; w : Wave
; S : Streamable (Num, Vec, Wave, Tape, Stream)
; s : Stream
; t : Tape
; I : Iterable
; i : iterator
; str : Str
; sym : Symbol
; [xs] : Vec of x
; <xs> : arbitrary values on stack
; s|n : Stream or Num
; body : anything that can be evaluated
; any other name is a synonym for x

;; a note on methods

; doc comments show methods as =Type.word:= for clarity. The actual
; token is =word=, and dispatch is based on the runtime type of the
; receiver (first stack argument).

;;; BUILTINS

;; core VM words

; nil: ( -- nil ) push Nil onto stack
; throw: ( x -- ) raise an exception carrying x
; catch: ( body -- x|nil ) evaluate body and capture value carried in exception (or nil on success)
; loop: ( body -- ) evaluate body repeatedly until break/throw
; stack: ( -- v ) push current stack snapshot
; log: ( x -- x ) log top of stack without consuming it
; str: ( x -- str ) stringify
; drop: ( x -- ) drop top
; nip: ( x y -- y ) drop second-from-top
; dup: ( x -- x x ) duplicate top
; swap: ( x y -- y x ) swap top two
; over: ( x y -- x y x ) copy second to top
; (: ( -- ) push new environment
; ): ( -- ) pop environment
; [: ( -- ) mark stack position
; ]: ( <xs> -- v ) collect items pushed since last stack mark, remove mark
; {: ( -- ) quote on
; }: ( -- v ) quote off
; set: ( k x -- ) set env var named by key
; get: ( k -- x ) fetch env var named by key
; eval: ( x -- <xs> ) evaluate x
; iter: ( I -- i ) obtain iterator from iterable
; next: ( i -- i x|nil ) advance iterator
; vdup: ( x n -- [xs] ) n copies of x in vec
; Num.if: ( b then -- ) conditional execute when nonzero
; Num.if: ( b then else -- ) conditional with else branch
; Num.<: ( n n -- b ) less-than
; Num.<=: ( n n -- b ) less-or-equal
; Num.>=: ( n n -- b ) greater-or-equal
; Num.>: ( n n -- b ) greater-than
; Vec.len: ( v -- n ) length of vector
; Vec.at: ( v k -- x ) indexed lookup
; Vec.clone: ( v -- v ) shallow copy
; Vec.push: ( v x -- v ) append item (modifies vec)
; Vec.pop: ( v -- v x ) pop last item (modifies vec)
; Vec.each: ( v body -- ) iterate and eval body per item
; Vec.map: ( v body -- v ) map body over items
; Vec.reduce: ( v body -- x ) fold left with body, returns nil if v is empty
; Vec.partition: ( v size step -- [vs] ) window vector
; Vec.tape: ( v -- t ) convert numeric vector to mono tape
; Str.+: ( str1 str2 -- str ) concatenate strings
; Str.load: ( str -- t ) load audio file
; Str.path/join: ( str1 str2 -- str ) join file system paths
; Str.parse: ( str -- v ) parse string into AST words
; Str.parse1: ( str -- x ) parse and take first word

;; math

; e: ( -- n ) Euler's constant
; pi: ( -- n ) pi constant
; abs: ( S -- s|n ) absolute value
; sign: ( S -- s|n ) sign of x (-1, 0, 1)
; square: (S -- s|n ) square (x = x^2)
; exp: ( S -- s|n ) natural exponential
; exp2: ( S -- s|n ) power-of-two exponential
; log10: ( S -- s|n ) base-10 logarithm
; log2: ( S -- s|n ) base-2 logarithm
; floor: ( S -- s|n ) floor
; ceil: ( S -- s|n ) ceiling
; trunc: ( S -- s|n ) truncate toward zero
; round: ( S -- s|n ) round to nearest integer
; sin: ( S -- s|n ) sine
; cos: ( S -- s|n ) cosine
; tan: ( S -- s|n ) tangent
; asin: ( S -- s|n ) arcsine
; acos: ( S -- s|n ) arccosine
; atan: ( S -- s|n ) arctangent
; sinh: ( S -- s|n ) hyperbolic sine
; cosh: ( S -- s|n ) hyperbolic cosine
; tanh: ( S -- s|n ) hyperbolic tangent
; asinh: ( S -- s|n ) inverse hyperbolic sine
; acosh: ( S -- s|n ) inverse hyperbolic cosine
; atanh: ( S -- s|n ) inverse hyperbolic tangent
; =: ( x y -- b ) equality check
; +: ( S S -- s|n ) add
; -: ( S S -- s|n ) subtract
; *: ( S S -- s|n ) multiply
; /: ( S S -- s|n ) divide
; mod: ( S S -- s|n ) modulus
; rem: ( S S -- s|n ) remainder
; pow: ( S S -- s|n ) power
; atan2: ( S S -- s|n ) arctangent with quadrant
; hypot: ( S S -- s|n ) hypotenuse length
; min: ( S S -- s|n ) minimum
; max: ( S S -- s|n ) maximum
; clamp: ( S min max -- s|n ) clamp samples to range

;; random numbers

; rand: ( -- n ) random float in [0,1)
; rand/seed: ( n -- ) reseed RNG used by rand

;; envelope segments

; /line: ( :start :end :nf -- t ) linear envelope segment
; /exp: ( :start :end :nf k -- t ) exponential envelope segment with curve k
; /log: ( :start :end :nf k -- t ) logarithmic envelope segment with curve k
; /cos: ( :start :end :nf -- t ) cosine ease-in/out envelope segment
; /pow: ( :start :end :nf p -- t ) power-law envelope segment with exponent p
; /sigmoid: ( :start :end :nf k -- t ) logistic envelope segment with slope k

; tapes

; tape1: ( n -- t ) allocate mono tape
; tape2: ( n -- t ) allocate stereo tape
; Tape.shift: ( t amount -- t ) rotate samples by amount
; Tape.resample: ( t converter ratio -- t ) resample tape
; Tape.at: ( t frame channel -- n ) fetch sample
; Tape.slice: ( t start end -- t ) slice of frames in [start,end)
; Tape.+@: ( t t' offset -- t ) mix t' into t at offset

; stream generators

; ~: ( S -- s ) coerce to stream
; ~noise: ( :seed -- s ) white noise
; ~pink: ( :seed -- s ) pink noise
; ~brown: ( :seed step -- s ) brown noise with step size
; stream/empty: ( n -- s ) empty stream of n channels

; stream transformers

; sr: ( -- n ) push global sample rate
; ~impulse: ( :freq :phase -- s ) band-limited impulse train
; dc*: ( S alpha -- s ) DC-blocking IIR with smoothing alpha
; dc: ( S -- s ) remove DC offset with alpha = 1-1/SR
; onepole: ( S alpha -- s ) first-order IIR smoother; higher alpha = more smoothing
; peak: ( S -- s ) max(abs(x) for x in frame)
; sh: ( S rate -- s ) sample-and-hold input at rate
; comb: ( S delay fb -- s ) feedback comb filter
; delay: ( S n -- s ) delay by n frames
; z1*: ( S n|[ns] -- s ) one-sample delay with explicit init frame (Num or Vec of per-channel smps)
; pan: ( S pan -- s ) equal-power pan mono input
; mix: ( [Ss] ratio -- s ) mix streams based on ratio clamped to [0,1]
; softclip: ( S mode -- s ) smooth saturation (0=tanh, 1=atan, 2=poly, 3=softsign)
; skip: ( S n -- s ) skip first n frames
; unison: ( :freq :voices :spread :detune :phaseRand body -- s ) detuned/positioned voices
; mono: ( S -- s ) sum/convert to mono
; stereo: ( S -- s ) ensure stereo

; stream renderers

; frames: ( s -- [ns|[ns]] ) collect all frames into vec, s must be finite
; take: ( s n -- t ) take first n frames

; stream utilities

; Streamable.join: ( S S -- s ) concatenate streams
; Streamable.len ( S -- n ) length of stream, or 0 if stream is infinite

; waves and wavetables

; wt: ( x -- wt ) coerce to wavetable
; wt/sin: ( -- wt ) sine wavetable
; wt/tanh: ( -- wt ) tanh wavetable
; wt/triangle: ( -- wt ) triangle wavetable
; wt/square: ( -- wt ) square wavetable
; wt/pulse: ( :pw -- wt ) pulse wavetable using env pulse width
; wt/saw: ( -- wt ) saw wavetable
; ~wt: ( :freq :phase :morph wt -- s ) wavetable oscillator with env freq/phase/morph
; ~fm: ( :freq :mod :index :phase wt -- s ) wavetable FM oscillator with env controls
; wave/sin: ( n -- w ) sine wave (single-cycle)
; wave/tanh: ( n -- w ) tanh-shaped wave (single-cycle)
; wave/triangle: ( n -- w ) triangle wave (single-cycle)
; wave/square: ( n -- w ) square wave (single-cycle)
; wave/pulse: ( :pw n -- w ) pulse wave using env pulse width (single-cycle)
; wave/saw: ( n -- w ) saw wave (single-cycle)
; phasor: ( :freq :phase w -- s ) phase-accumulating oscillator using env freq/phase

; ports

; vital/decimate: ( S factor sharp -- s ) multistage decimator
; vital/svf: ( :cutoff :q :drive :blend :saturate S -- s ) state-variable filter with env params

;;; STANDARD LIBRARY

; true: ( -- -1 )
-1 >true

; false: ( -- 0 )
0 >false

; false?: ( x -- b )
{ 0 = } >false?

; not: ( x -- b )
{ 0 = } >not

; !=: ( x y -- b )
{ = false? } >!=

; true?: ( x -- b )
{ 0 != } >true?

; nil?: ( x -- b )
{ nil = } >nil?

; assert: ( body -- ) evaluates body and throws error if result is false
{ dup >:assert/value
  eval
  false? { "assertion failed: " :assert/value str + throw } if
} >assert

; break: ( -- ) exits current loop
{ nil throw } >break

; zip: ( [xs] -- [[ys]] ) pull in lockstep until any iterator yields nil
{( { iter } map >:zip/iters
   [] >:zip/result
   {
     [ :zip/iters { eval dup nil? { break } if } each ]
     :zip/result swap push >:zip/result
   } loop
   :zip/result
)} >zip

; seq iterates over N iterables
; the N iterables have N names, received in :seq/syms
;
; seq looks up the N iterables by name
; turns them into iterators -> :seq/symiters
;
; in each iteration:
;
; 1. for each iterable I:
;
;    a. gets next value V from I (eval)
;    b. if value is nil: exit
;    c. otherwise sets name of iterable I in local env to V
;
; 2. invokes seq/body
;
; seq/body typically generates a value (piece of sound, envelope
; segment, etc.) which depends on the parameters set in step 1
;
; these values can be collected by using seq inside [...]

; seq: ( body [syms] -- <xs> )
{( >seq/body >:seq/syms
   :seq/syms { get iter } map >:seq/symiters
   {
     [:seq/syms :seq/symiters] zip {
       dup 1 at ; symiter
       swap 0 at ; sym
       swap
       eval dup nil? { break } if set
     } each
     seq/body
   } loop
)} >seq

; oscillator defaults

; :freq: ( -- n ) frequency
440 >:freq
; :phase: ( -- n ) phase
0.0 >:phase
; :pw: ( -- n ) pulse width
0.5 >:pw

; filter defaults

; :cutoff: ( -- n ) cutoff
1200  >:cutoff
; :q: ( -- n ) resonance
0.7   >:q
; :drive: ( -- n ) drive
1.0   >:drive
; :blend: ( -- n ) blend
0.0   >:blend
; :saturate: ( -- b ) saturation toggle
false >:saturate

; FM defaults

; :mod: ( -- n ) FM phase offset (in cycles)
0.0 >:mod
; :index: ( -- n ) FM index
1.0 >:index

; noise RNG defaults

; :seed: ( -- n ) seed used by noise generators
0 >:seed

; start: ( -- n )
0 >:start
; end: ( -- n )
1 >:end

; f: ( n -- :freq ) set default :freq
{ ":freq" swap set } >f

; seconds: ( dur -- n ) length of n seconds in frames
{ sr * } >seconds

; beats: ( :bpm beats -- n ) length of n beats in frames
{ sr :bpm 60 / / * } >beats

; periods: ( :freq periods -- n ) length of n periods in frames
{ sr :freq / * } >periods

; ticks: ( :bpm :tpb ticks -- n ) length of n ticks in frames
{ beats :tpb / } >ticks

; st: ( semitones -- ratio ) frequency multiplier for n semitone steps
{ 12 / exp2 } >st

; mtof: ( midi-note -- freq ) frequency for MIDI note n
{ 69 - st 440 * } >mtof

; cents: ( cents -- ratio ) frequency multiplier for n cents
{ 1200 / exp2 } >cents

; db: ( db -- amp ) convert decibels to linear amplitude multiplier
{ 20 / 10 swap pow } >db

; gain: ( S db -- s ) apply gain in dB to a stream
{ db * } >gain

; clip: ( S -- s ) constrain stream samples to [-1,1]
{ -1 1 clamp } >clip

; distribute: ( [ns] n -- [ns] ) scale vals so their sum equals n; sum(ns) must be nonzero
; (e.g., [2 3 1 4] 200 distribute -> [40 60 20 80])
{( over {+} reduce / >:unit { :unit * } map )} >distribute

; cat: ( [Ss] -- s ) concatenate streams
{ {join} reduce } >cat

; repeat: ( S n -- s ) repeat and concat
{ vdup cat } >repeat

; avg: ( [Ss|ns] -- s|n ) average a list of streams/numbers
{ dup len swap {+} reduce swap / } >avg

; stretch: ( S factor -- s )
{ SRC_LINEAR swap resample } >stretch

; start:end: ( [ns] -- :start :end )
; after [0 2 5 6] start:end, :start = [0 2 5], :end = [2 5 6]
{ 2 1 partition
  dup { 0 at } map >:start
  dup { 1 at } map >:end
  drop
} >start:end

; env generates an envelope
;
; ys: values at envelope points
; ds: durations between envelope points (in frames)
; envsegs: closures which generate the envelope segments
;
; len(envsegs) = len(ds) = len(ys) - 1

; env: ( [ys] [ds] [segs] -- env )
{( >:envseg >:nf start:end
   [
     { :start :end :nf :envseg }
     { :envseg eval }
     seq
   ] cat
)} >env

; sum: ( [Ss|ns] -- s|n )
{ {+} reduce } >sum

; adsr: ( a d s r dur -- env )
{( >:dur >:release >:sustain/level >:decay >:attack
   :dur [:attack :decay :release] sum - >:sustain/dur
   [0 1 :sustain/level :sustain/level 0]
   [:attack :decay :sustain/dur :release]
   ;[{8 /log} {-6 /exp} {/line} {-6 /exp}]
   {/cos} 4 vdup
   env
)} >adsr

; perc: ( a r -- env )
{( >:release >:attack
   [0 1 0]
   [:attack :release]
   [{0.5 /pow} {-5 /exp}]
   env
)} >perc

; default waves (single-cycle waveforms)
; size = 0 means use hard-coded default size

; w/sin: ( -- w )
{ 0 wave/sin } >w/sin

; w/tanh: ( -- w )
{ 0 wave/tanh } >w/tanh

; w/triangle: ( -- w )
{ 0 wave/triangle } >w/triangle

; w/square: ( -- w )
{ 0 wave/square } >w/square

; w/pulse: ( :pw -- w )
{ 0 wave/pulse } >w/pulse

; w/saw: ( -- w )
{ 0 wave/saw } >w/saw

; wave phasors

; ~sin: ( :freq :phase -- s )
{ w/sin phasor } >~sin

; ~tanh: ( :freq :phase -- s )
{ w/tanh phasor } >~tanh

; ~triangle: ( :freq :phase -- s )
{ w/triangle phasor } >~triangle

; ~square: ( :freq :phase -- s )
{ w/square phasor } >~square

; ~pulse: ( :freq :phase :pw -- s )
{ w/pulse phasor } >~pulse

; ~saw: ( :freq :phase -- s )
{ w/saw phasor } >~saw

; lfo

; ~lfo/sin: ( :phase freq -- s )
{( >:freq ~sin)} >~lfo/sin

; ~lfo/tanh: ( :phase freq -- s )
{( >:freq ~tanh)} >~lfo/tanh

; ~lfo/triangle: ( :phase freq -- s )
{( >:freq ~triangle 0.25p skip )} >~lfo/triangle

; ~lfo/square: ( :phase freq -- s )
{( >:freq ~square 0.26p skip)} >~lfo/square

; ~lfo/pulse: ( :phase :pw pw freq -- s )
{( >:freq >:pw ~pulse :pw 0.01 + periods skip )} >~lfo/pulse

; ~lfo/saw: ( :phase freq -- s )
{( >:freq ~saw 0.76p skip )} >~lfo/saw

; unipolar <-> bipolar conversions

; uni: ( bipolar -- unipolar )
{ 1 + 2 / } >uni

; bi: ( unipolar -- bipolar )
{ 2 * 1 - } >bi

; z1: ( s -- s ) one-sample delay with zero init
{ 0 z1* } >z1
