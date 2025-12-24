; builtins

; nil: ( -- nil ) push Nil onto stack
; throw: ( x -- ) raise an exception carrying x
; catch: ( body -- x|nil ) run body and capture value in exception (or nil on success)
; loop: ( body -- ) execute body repeatedly until break/throw
; log: ( x -- x ) log top of stack without consuming it
; sr: ( -- n ) push global sample rate
; =: ( a b -- bool ) equality check
; stack: ( -- vec ) push current stack snapshot
; str: ( x -- str ) stringify top value
; drop: ( x -- ) drop top
; nip: ( a b -- b ) drop second-from-top
; dup: ( x -- x x ) duplicate top
; swap: ( a b -- b a ) swap top two
; over: ( a b -- a b a ) copy second to top
; (: ( -- ) push new environment
; ): ( -- ) pop environment
; [: ( -- ) mark stack position for list literal
; ]: ( ... -- vec ) collect items pushed since mark
; {: ( -- ) quote on
; }: ( -- vec ) quote off
; set: ( key val -- ) set env var named by key
; get: ( key -- val ) fetch env var named by key
; eval: ( val -- result ) evaluate value
; iter: ( iterable -- iterator ) obtain iterator
; next: ( iterator -- item|nil ) advance iterator
; /line: ( :start :end :nf -- envseg ) linear envelope segment
; /exp: ( :start :end :nf k -- envseg ) exponential envelope with curve k
; /log: ( :start :end :nf k -- envseg ) logarithmic envelope with curve k
; /cos: ( :start :end :nf -- envseg ) cosine ease-in/out envelope
; /pow: ( :start :end :nf p -- envseg ) power-law envelope with exponent p
; /sigmoid: ( :start :end :nf k -- envseg ) logistic envelope with slope k
; ~impulse: ( :freq :phase -- stream ) band-limited impulse train
; dc*: ( stream alpha -- stream ) DC-blocking IIR with smoothing alpha
; dc: ( stream -- stream ) remove DC offset with alpha = 1-1/SR
; sh: ( input rate -- output ) sample-and-hold input at rate
; comb: ( input delay fb -- stream ) feedback comb filter
; delay: ( stream nf -- stream ) delay by nf frames
; z1*: ( stream init -- stream ) one-sample delay with explicit init frame
; skip: ( stream nf -- stream ) skip first nf frames
; pan: ( input pan -- stereo ) equal-power pan mono input
; mix: ( vec ratio -- stream ) mix adjacent streams based on ratio in [0,1]
; e: ( -- n ) Euler's constant
; pi: ( -- n ) pi constant
; abs: ( x -- y ) absolute value
; sign: ( x -- -1|1 ) sign of x
; exp: ( x -- y ) natural exponential
; exp2: ( x -- y ) power-of-two exponential
; log10: ( x -- y ) base-10 logarithm
; log2: ( x -- y ) base-2 logarithm
; floor: ( x -- y ) floor
; ceil: ( x -- y ) ceiling
; trunc: ( x -- y ) truncate toward zero
; round: ( x -- y ) round to nearest integer
; sin: ( x -- y ) sine
; cos: ( x -- y ) cosine
; tan: ( x -- y ) tangent
; asin: ( x -- y ) arcsine
; acos: ( x -- y ) arccosine
; atan: ( x -- y ) arctangent
; sinh: ( x -- y ) hyperbolic sine
; cosh: ( x -- y ) hyperbolic cosine
; tanh: ( x -- y ) hyperbolic tangent
; asinh: ( x -- y ) inverse hyperbolic sine
; acosh: ( x -- y ) inverse hyperbolic cosine
; atanh: ( x -- y ) inverse hyperbolic tangent
; +: ( a b -- c ) add
; -: ( a b -- c ) subtract
; *: ( a b -- c ) multiply
; /: ( a b -- c ) divide
; mod: ( a b -- c ) modulus
; rem: ( a b -- c ) remainder
; pow: ( a b -- c ) power
; atan2: ( y x -- a ) arctangent with quadrant
; hypot: ( a b -- c ) hypotenuse length
; min: ( a b -- c ) minimum
; max: ( a b -- c ) maximum
; clamp: ( stream min max -- stream ) clamp samples to range
; softclip: ( stream mode -- stream ) smooth saturation (0=tanh, 1=atan, 2=poly, 3=softsign)
; rand: ( -- n ) random float in [0,1)
; rand/seed: ( seed -- ) reseed RNG
; ~noise: ( :seed -- stream ) white noise
; ~pink: ( :seed -- stream ) pink noise
; ~brown: ( :seed step -- stream ) brown noise with step size
; unison: ( :freq :voices :spread :detune :phaseRand body -- stream ) detuned/positioned voices
; tape1: ( nframes -- tape ) allocate mono tape
; tape2: ( nframes -- tape ) allocate stereo tape
; ~: ( val -- stream ) coerce to stream
; take: ( stream nf -- tape ) take first nf frames as a tape
; frames: ( stream -- vec ) collect all frames of stream into vec
; mono: ( stream -- stream ) sum/convert to mono
; stereo: ( stream -- stream ) ensure stereo
; wt: ( val -- wt ) coerce to wavetable
; wt/sin: ( -- wt ) sine wavetable
; wt/tanh: ( -- wt ) tanh wavetable
; wt/triangle: ( -- wt ) triangle wavetable
; wt/square: ( -- wt ) square wavetable
; wt/pulse: ( :pw -- wt ) pulse wavetable using env pulse width
; wt/saw: ( -- wt ) saw wavetable
; ~wt: ( :freq :phase :morph wt -- stream ) wavetable oscillator with env freq/phase/morph
; ~fm: ( :freq :mod :index :phase wt -- stream ) wavetable FM oscillator with env controls
; wave/sin: ( size -- wave ) sine wave (single-cycle)
; wave/tanh: ( size -- wave ) tanh-shaped wave (single-cycle)
; wave/triangle: ( size -- wave ) triangle wave (single-cycle)
; wave/square: ( size -- wave ) square wave (single-cycle)
; wave/pulse: ( :pw size -- wave ) pulse wave using env pulse width (single-cycle)
; wave/saw: ( size -- wave ) saw wave (single-cycle)
; phasor: ( :freq :phase wave -- stream ) phase-accumulating oscillator using env freq/phase
; vital/decimate: ( input factor sharp -- output ) multistage decimator
; vital/svf: ( :cutoff :q :drive :blend :saturate input -- output ) state-variable filter with env params
; vdup: ( val count -- vec ) duplicate value count times
; Sym.get: ( sym -- val ) look up environment value (uses stringified sym as key)
; Sym.set: ( sym val -- ) set environment value (uses stringified sym as key)
; Str.sym: ( str -- sym ) intern string as symbol
; Str.+: ( lhs rhs -- str ) concatenate strings
; Str.path/join: ( lhs rhs -- str ) join file system paths
; Str.parse: ( str -- vec ) parse string into AST words
; Str.parse1: ( str -- val ) parse and take first word
; Num.if: ( cond block -- ) conditional execute when nonzero
; Num.if/else: ( cond ifBlock elseBlock -- ) conditional with else branch
; Num.<: ( lhs rhs -- bool ) less-than
; Num.<=: ( lhs rhs -- bool ) less-or-equal
; Num.>=: ( lhs rhs -- bool ) greater-or-equal
; Num.>: ( lhs rhs -- bool ) greater-than
; Vec.len: ( vec -- n ) length of vector
; Vec.at: ( vec index -- item ) indexed lookup
; Vec.clone: ( vec -- vec ) shallow copy
; Vec.push: ( vec item -- vec ) append item (modifies vec)
; Vec.pop: ( vec -- vec item ) pop last item (modifies vec)
; Vec.each: ( vec block -- ) iterate and eval block per item
; Vec.map: ( vec block -- vec ) map block over items
; Vec.reduce: ( vec block -- result ) fold left with block
; Vec.partition: ( vec size step -- vecs ) window vector
; Vec.tape: ( vec -- tape ) convert numeric vector to mono tape
; Tape.nf: ( tape -- nframes ) number of frames
; Tape.shift: ( tape amount -- tape ) rotate samples by amount
; Tape.resample: ( tape converter ratio -- tape ) resample tape
; Tape.at: ( tape frame channel -- sample ) fetch sample
; Tape.slice: ( tape start end -- tape ) slice frames
; Tape.+@: ( tape rhs offset -- tape ) mix rhs into lhs at offset
; Str.load: ( path -- tape ) load audio file
; Streamable.join: ( lhs rhs -- stream ) concatenate streams channel-wise

; true: ( -- true )
-1 >true

; false: ( -- false )
0 >false

; false?: ( x -- bool )
{ 0 = } >false?

; not: ( x -- bool )
{ 0 = } >not

; !=: ( a b -- bool )
{ = false? } >!=

; true?: ( x -- bool )
{ 0 != } >true?

; nil?: ( x -- bool )
{ nil = } >nil?

; assert: ( pred -- :assert/value ) checks pred and throws error if false
{ dup >:assert/value
  eval
  false? { "assertion failed: " :assert/value str + throw } if
} >assert

; break: ( -- ) exits current loop
{ nil throw } >break

; zip: ( [xs...] -- [[zs...]] ) pull in lockstep until any iterator yields nil
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

; seq: ( body syms -- ? )
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

; :freq: ( -- num ) frequency
440 >:freq
; :phase: ( -- num ) phase
0.0 >:phase
; :pw: ( -- num ) pulse width
0.5 >:pw

; filter defaults

; :cutoff: ( -- num ) cutoff
1200  >:cutoff
; :q: ( -- num ) resonance
0.7   >:q
; :drive: ( -- num ) drive
1.0   >:drive
; :blend: ( -- num ) blend
0.0   >:blend
; :saturate: ( -- bool ) saturation toggle
false >:saturate

; FM defaults

; :mod: ( -- num ) FM phase offset (in cycles)
0.0 >:mod
; :index: ( -- num ) FM index
1.0 >:index

; noise RNG defaults

; :seed: ( -- num ) seed used by noise generators
0 >:seed

; start: ( -- num )
0 >:start
; end: ( -- num )
1 >:end

; f: ( freq -- :freq ) set default :freq
{ ":freq" swap set } >f

; seconds: ( dur -- frames )
{ sr * } >seconds

; beats: ( :bpm beats -- frames )
{ sr :bpm 60 / / * } >beats

; periods: ( :freq cycles -- frames )
{ sr :freq / * } >periods

; ticks: ( :bpm :tpb ticks -- frames )
{ beats :tpb / } >ticks

; st: ( semitones -- ratio )
{ 12 / exp2 } >st

; mtof: ( midi-note -- freq )
{ 69 - st 440 * } >mtof

; cents: ( cents -- ratio )
{ 1200 / exp2 } >cents

; db: ( db -- amp ) convert decibels to linear amplitude multiplier
{ 20 / 10 swap pow } >db

; gain: ( stream db -- stream ) apply gain in dB to a stream
{ db * } >gain

; clip: ( stream -- stream ) constrain stream samples to [-1,1]
{ -1 1 clamp } >clip

; distribute: ( vals n -- vals' ) scale vals so their sum equals n
; (e.g., [2 3 1 4] 200 distribute -> [40 60 20 80])
{( over {+} reduce / >:unit { :unit * } map )} >distribute

; cat: ( [...chunks...] -- chunk ) concatenate streams
{ {join} reduce } >cat

; repeat: ( chunk n -- chunk ) repeat and concat
{ vdup cat } >repeat

; avg: ( [streams] -- stream ) average a list of streams/numbers
{ dup len swap {+} reduce swap / } >avg

; stretch: ( stream factor -- stream )
{ SRC_LINEAR swap resample } >stretch

; start:end: ( [xs] -- :start :end )
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

; env: ( ys ds segs -- env )
{( >:envseg >:nf start:end
   [
     { :start :end :nf :envseg }
     { :envseg eval }
     seq
   ] cat
)} >env

; sum: ( [nums] -- n )
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

; w/sin: ( -- wave )
{ 0 wave/sin } >w/sin

; w/tanh: ( -- wave )
{ 0 wave/tanh } >w/tanh

; w/triangle: ( -- wave )
{ 0 wave/triangle } >w/triangle

; w/square: ( -- wave )
{ 0 wave/square } >w/square

; w/pulse: ( :pw -- wave )
{ 0 wave/pulse } >w/pulse

; w/saw: ( -- wave )
{ 0 wave/saw } >w/saw

; wave phasors

; ~sin: ( :freq :phase -- stream )
{ w/sin phasor } >~sin

; ~tanh: ( :freq :phase -- stream )
{ w/tanh phasor } >~tanh

; ~triangle: ( :freq :phase -- stream )
{ w/triangle phasor } >~triangle

; ~square: ( :freq :phase -- stream )
{ w/square phasor } >~square

; ~pulse: ( :freq :phase :pw -- stream )
{ w/pulse phasor } >~pulse

; ~saw: ( :freq :phase -- stream )
{ w/saw phasor } >~saw

; lfo

; ~lfo/sin: ( :phase freq -- stream )
{( >:freq ~sin)} >~lfo/sin

; ~lfo/tanh: ( :phase freq -- stream )
{( >:freq ~tanh)} >~lfo/tanh

; ~lfo/triangle: ( :phase freq -- stream )
{( >:freq ~triangle 0.25p skip )} >~lfo/triangle

; ~lfo/square: ( :phase freq -- stream )
{( >:freq ~square 0.26p skip)} >~lfo/square

; ~lfo/pulse: ( :phase :pw pw freq -- stream )
{( >:freq >:pw ~pulse :pw 0.01 + periods skip )} >~lfo/pulse

; ~lfo/saw: ( :phase freq -- stream )
{( >:freq ~saw 0.76p skip )} >~lfo/saw

; unipolar <-> bipolar conversions

; uni: ( bipolar -- unipolar )
{ 1 + 2 / } >uni

; bi: ( unipolar -- bipolar )
{ 2 * 1 - } >bi

; z1: ( stream -- stream ) one-sample delay with zero init
{ 0 z1* } >z1
