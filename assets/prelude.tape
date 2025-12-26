;;; STACK COMMENT FORMAT

; [TYPE.]NAME: ( [ENV: ENV-IN |] [STACK-IN] -- [STACK-OUT] [| SETS: ENV-OUT] ) DESCRIPTION

; [...] means optional

; TYPE: if present, WORD is a method of first stack argument's type
; NAME: name of word - if ends with !, word mutates its receiver
; ENV-IN: env vars used by WORD
; STACK-IN: values expected on stack
; STACK-OUT: values left on stack
; ENV-OUT: env vars modified by WORD

;; stack comment variable names

; x y k : any value
; n : Num
; b : Num interpreted as a boolean: 0=false, anything else is true
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
; n|[ns]: sample frame, mono frame is Num; multi-channel frame is Vec[Num] (L,R,...)
; <xs> : arbitrary values on stack
; s|n : Stream or Num
; body : anything that can be evaluated
; any other name is a synonym for x

;; methods

; doc comments show methods as =Type.word:= for clarity. The actual
; token is =word=, and dispatch is based on the runtime type of the
; receiver (first stack argument).

;;; SYNTAX SUGAR

; :name -> ":name" get
; @foo -> "foo" get
; >foo -> "foo" swap set
; 1s -> 1 seconds
; 1b -> 1 beats
; 1t -> 1 ticks
; 1p -> 1 periods

;;; BUILTINS

;; core

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
; Vec.push: ( v x -- v ) append item, mutates v
; Vec.pop: ( v -- v x ) pop last item, mutates v
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

; /line: ( ENV: :start :end :nf | -- t ) linear envelope segment
; /exp: ( ENV: :start :end :nf | k -- t ) exponential envelope segment with curve k
; /log: ( ENV: :start :end :nf | k -- t ) logarithmic envelope segment with curve k
; /cos: ( ENV: :start :end :nf | -- t ) cosine ease-in/out envelope segment
; /pow: ( ENV: :start :end :nf | p -- t ) power-law envelope segment with exponent p
; /sigmoid: ( ENV: :start :end :nf | k -- t ) logistic envelope segment with slope k

;; tapes

; tape1: ( n -- t ) allocate mono tape
; tape2: ( n -- t ) allocate stereo tape
; Tape.shift: ( t amount -- t ) rotate samples by amount, mutates t
; Tape.resample: ( t converter ratio -- t ) resample tape
; Tape.at: ( t frame -- n|[ns] ) fetch frame
; Tape.slice: ( t start end -- t ) tape with frames between [start,end), mutates t
; Tape.+@: ( t t2 offset -- t ) mix t2 into t at offset, mutates t

;; stream generators

; ~: ( S -- s ) coerce to stream
; ~empty: ( n -- s ) empty stream of n channels
; ~impulse: ( ENV: :freq :phase | -- s ) band-limited impulse train

;; stream transformers

; dc*: ( S alpha -- s ) DC-blocking IIR with smoothing alpha
; dc: ( S -- s ) remove DC offset with alpha = 1-1/SR
; onepole: ( S alpha -- s ) first-order IIR smoother; higher alpha = more smoothing
; peak: ( S -- s ) max(abs(x) for x in frame)
; sh: ( S rate -- s ) sample-and-hold input at rate
; comb: ( S delay fb -- s ) feedback comb filter
; delay: ( S n -- s ) delay by n frames
; z1*: ( S n|[ns] -- s ) one-sample delay with explicit init frame (Num or Vec of per-channel smps)
; pan: ( S pan -- s ) equal-power pan mono input with pan between [-1,1]
; mix: ( [Ss] ratio -- s ) mix streams based on ratio clamped to [0,1]
; softclip: ( S mode -- s ) smooth saturation (0=tanh, 1=atan, 2=poly, 3=softsign)
; skip: ( S n -- s ) skip first n frames
; unison: ( ENV: :freq :voices :spread :detune :phaseRand | body -- s ) detuned/positioned voices
; mono: ( S -- s ) sum/convert to mono
; stereo: ( S -- s ) ensure stereo

; stream renderers

; frames: ( s -- [ns|[ns]] ) collect all frames into vec, s must be finite
; take: ( s n -- t ) take first n frames

; stream utilities

; Streamable.join: ( S S -- s ) concatenate streams
; Streamable.len ( S -- n ) length of stream, or 0 if stream is infinite

;; noise generators

; ~noise: ( ENV: :seed | -- s ) white noise
; ~pink: ( ENV: :seed | -- s ) pink noise
; ~brown: ( ENV: :seed | step -- s ) brown noise with step size

;; waves and wavetables

; wave/sin: ( n -- w ) sine wave (single-cycle)
; wave/tanh: ( n -- w ) tanh-shaped wave (single-cycle)
; wave/triangle: ( n -- w ) triangle wave (single-cycle)
; wave/square: ( n -- w ) square wave (single-cycle)
; wave/pulse: ( ENV: :pw | n -- w ) pulse wave using env pulse width (single-cycle)
; wave/saw: ( n -- w ) saw wave (single-cycle)
; phasor: ( ENV: :freq :phase | w -- s ) phase-accumulating oscillator using env freq/phase

; wt: ( x -- wt ) coerce to wavetable
; wt/sin: ( -- wt ) sine wavetable
; wt/tanh: ( -- wt ) tanh wavetable
; wt/triangle: ( -- wt ) triangle wavetable
; wt/square: ( -- wt ) square wavetable
; wt/pulse: ( ENV: :pw | -- wt ) pulse wavetable using env pulse width
; wt/saw: ( -- wt ) saw wavetable

; ~wt: ( ENV: :freq :phase :morph | wt -- s ) wavetable oscillator with env freq/phase/morph
; ~fm: ( ENV: :freq :mod :index :phase | wt -- s ) wavetable FM oscillator with env controls

;; ports

; vital/decimate: ( S factor sharp -- s ) multistage decimator
; vital/svf: ( ENV: :cutoff :q :drive :blend :saturate | S -- s ) state-variable filter

;; misc

; sr: ( -- n ) push global sample rate

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

;; oscillator parameters

; :freq: ( -- n ) frequency
440 >:freq
; :phase: ( -- n ) phase
0.0 >:phase
; :pw: ( -- n ) pulse width
0.5 >:pw

; f: ( n -- | SETS: :freq ) shorthand for setting :freq to n
{ ":freq" swap set } >f

;; filter parameters

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

;; FM parameters

; :mod: ( -- n ) FM phase offset (in cycles)
0.0 >:mod
; :index: ( -- n ) FM index
1.0 >:index

;; noise RNG parameters

; :seed: ( -- n ) seed used by noise generators
0 >:seed

;; envelope parameters

; start: ( -- n )
0 >:start
; end: ( -- n )
1 >:end

;; utilities

; sum: ( [Ss|ns] -- s|n )
{ {+} reduce } >sum

; distribute: ( [ns] n -- [ns] ) scale vals so their sum equals n; sum(ns) must be nonzero
; (e.g., [2 3 1 4] 200 distribute -> [40 60 20 80])
{( over sum / >:unit { :unit * } map )} >distribute

; avg: ( [Ss|ns] -- s|n ) average a list of streams/numbers
{ dup len swap {+} reduce swap / } >avg

; clip: ( S -- s ) constrain stream samples to [-1,1]
{ -1 1 clamp } >clip

; cat: ( [Ss] -- s ) concatenate streams
{ {join} reduce } >cat

; repeat: ( S n -- s ) repeat and concat
{ vdup cat } >repeat

; start:end: ( [ns] -- | SETS: :start :end )
; after [0 2 5 6] start:end, :start = [0 2 5], :end = [2 5 6]
{ 2 1 partition
  dup { 0 at } map >:start
  dup { 1 at } map >:end
  drop
} >start:end

; for: ( I body -- <xs> ) evaluates body for each value produced by I
{( >:for/body iter >:for/iter
   { :for/iter eval dup nil? { drop break } if :for/body eval } loop )} >for

; zip: ( [xs] -- [[ys]] ) pull in lockstep until any iterator yields nil
{( { iter } map >:zip/iters
   [] >:zip/result
   {
     [ :zip/iters { eval dup nil? { break } if } each ]
     :zip/result swap push >:zip/result
   } loop
   :zip/result
)} >zip

;; sequencer

; seq: ( body [syms] -- <xs> ) see examples/seq.tape for details
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

;; time

; seconds: ( dur -- n ) length of n seconds in frames
{ sr * } >seconds

; beats: ( :bpm beats -- n ) length of n beats in frames
{ sr :bpm 60 / / * } >beats

; periods: ( ENV: :freq | periods -- n ) length of n periods in frames
{ sr :freq / * } >periods

; ticks: ( ENV: :bpm :tpb | ticks -- n ) length of n ticks in frames
{ beats :tpb / } >ticks

;; pitch

; st: ( semitones -- ratio ) frequency multiplier for n semitone steps
{ 12 / exp2 } >st

; mtof: ( midi-note -- freq ) frequency for MIDI note n
{ 69 - st 440 * } >mtof

; cents: ( cents -- ratio ) frequency multiplier for n cents
{ 1200 / exp2 } >cents

;; amplitude

; db: ( db -- amp ) convert decibels to linear amplitude multiplier
{ 20 / 10 swap pow } >db

; gain: ( S db -- s ) apply gain in dB to a stream
{ db * } >gain

; unipolar <-> bipolar conversions

; uni: ( bipolar -- unipolar )
{ 1 + 2 / } >uni

; bi: ( unipolar -- bipolar )
{ 2 * 1 - } >bi

;; envelope generators

; env: ( [ys] [ds] [segs] -- env )
{( >:envseg >:nf start:end
   [
     { :start :end :nf :envseg }
     { :envseg eval }
     seq
   ] cat
)} >env

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

;; default waves (single-cycle waveforms)
;; size = 0 means use hard-coded default size

; w/sin: ( -- w )
{ 0 wave/sin } >w/sin

; w/tanh: ( -- w )
{ 0 wave/tanh } >w/tanh

; w/triangle: ( -- w )
{ 0 wave/triangle } >w/triangle

; w/square: ( -- w )
{ 0 wave/square } >w/square

; w/pulse: ( ENV: :pw | -- w )
{ 0 wave/pulse } >w/pulse

; w/saw: ( -- w )
{ 0 wave/saw } >w/saw

;; wave phasors

; ~sin: ( ENV: :freq :phase | -- s )
{ w/sin phasor } >~sin

; ~tanh: ( ENV: :freq :phase | -- s )
{ w/tanh phasor } >~tanh

; ~triangle: ( ENV: :freq :phase | -- s )
{ w/triangle phasor } >~triangle

; ~square: ( ENV: :freq :phase | -- s )
{ w/square phasor } >~square

; ~pulse: ( ENV: :freq :phase :pw | -- s )
{ w/pulse phasor } >~pulse

; ~saw: ( ENV: :freq :phase | -- s )
{ w/saw phasor } >~saw

;; lfo

; ~lfo/sin: ( ENV: :freq :phase | -- s )
{ ~sin } >~lfo/sin

; ~lfo/tanh: ( ENV: :freq :phase | -- s )
{ ~tanh } >~lfo/tanh

; ~lfo/triangle: ( ENV: :freq :phase | -- s )
{ ~triangle 0.25p skip } >~lfo/triangle

; ~lfo/square: ( ENV: :freq :phase | -- s )
{ ~square 0.26p skip } >~lfo/square

; ~lfo/pulse: ( ENV: :freq :phase :pw | -- s )
{ ~pulse :pw 0.01 + periods skip } >~lfo/pulse

; ~lfo/saw: ( ENV: :freq :phase | -- s )
{ ~saw 0.76p skip } >~lfo/saw

;; dsp

; z1: ( s -- s ) one-sample delay with zero init
{ 0 z1* } >z1

; stretch: ( S factor -- s )
{ SRC_LINEAR swap resample } >stretch
