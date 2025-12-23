; true: ( -- true)
-1 >true

; false: ( -- false)
0 >false

; false?: (x -- bool)
{ 0 = } >false?

; not: (x -- bool)
{ 0 = } >not

; !=: (a b -- bool)
{ = false? } >!=

; true?: (x -- bool)
{ 0 != } >true?

; nil?: (x -- bool)
{ nil = } >nil?

; assert: (pred --) checks pred and throws error if false
{ dup >:assert/value
  eval
  false? { "assertion failed: " :assert/value str + throw } if
} >assert

; break: ( -- bottom) exits current loop
{ nil throw } >break

; zip: ([xs...] -- [[zs...]]): pull in lockstep until any iterator yields nil
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

; seq: (body syms -- ?)
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

; :freq: ( -- n) frequency
440 >:freq
; :phase: ( -- n) phase
0.0 >:phase
; :pw: ( -- n) pulse width
0.5 >:pw

; filter defaults

; :cutoff: ( -- n) cutoff
1200  >:cutoff
; :q: ( -- n) resonance
0.7   >:q
; :drive: ( -- n) drive
1.0   >:drive
; :blend: ( -- n) blend
0.0   >:blend
; :saturate: ( -- bool) saturation toggle
false >:saturate

; FM defaults

; :mod: ( -- n) FM phase offset (in cycles)
0.0 >:mod
; :index: ( -- n) FM index
1.0 >:index

; noise RNG defaults

; :seed: ( -- n) seed used by noise generators
0 >:seed

; start: ( -- n)
0 >:start
; end: ( -- n)
1 >:end

; f: (freq --) set default :freq
{ ":freq" swap set } >f

; seconds: (dur -- frames)
{ sr * } >seconds

; beats: (beats -- frames)
{ sr :bpm 60 / / * } >beats

; periods: (cycles -- frames)
{ sr :freq / * } >periods

; ticks: (ticks -- frames)
{ beats :tpb / } >ticks

; st: (semitones -- ratio)
{ 12 / exp2 } >st

; mtof: (midi-note -- freq)
{ 69 - st 440 * } >mtof

; cents: (cents -- ratio)
{ 1200 / exp2 } >cents

; db: (db -- amp) convert decibels to linear amplitude multiplier
{ 20 / 10 swap pow } >db

; gain: (stream db -- stream) apply gain in dB to a stream
{ db * } >gain

; distribute: (vals n -- vals') scale vals so their sum equals n
; (e.g., [2 3 1 4] 200 distribute -> [40 60 20 80])
{( over {+} reduce / >:unit { :unit * } map )} >distribute

; cat: ([...chunks...] -- chunk) concatenate streams
{ {join} reduce } >cat

; repeat: (chunk n -- chunk) repeat and concat
{ vdup cat } >repeat

; mix: ([streams] -- stream) average a list of streams
{ dup len swap {~} map {+} reduce swap / } >mix

; stretch: (stream factor -- stream)
{ SRC_LINEAR swap resample } >stretch

; start:end: ([xs] -- ) sets :start to starts and :end to ends
; (e.g., after [0 2 5 6] start:end, :start = [0 2 5], :end = [2 5 6])
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

; env: (ys ds segs -- env)
{( >:envseg >:nf start:end
   [
     { :start :end :nf :envseg }
     { :envseg eval }
     seq
   ] cat
)} >env

; sum: ([nums] -- n)
{ {+} reduce } >sum

; adsr: (a d s r dur -- env)
{( >:dur >:release >:sustain/level >:decay >:attack
   :dur [:attack :decay :release] sum - >:sustain/dur
   [0 1 :sustain/level :sustain/level 0]
   [:attack :decay :sustain/dur :release]
   ;[{8 /log} {-6 /exp} {/line} {-6 /exp}]
   {/cos} 4 vdup
   env
)} >adsr

; perc: (a r -- env)
{( >:release >:attack
   [0 1 0]
   [:attack :release]
   [{0.5 /pow} {-5 /exp}]
   env
)} >perc

; default waves

; w/sin: ( -- wave)
{ 0 wave/sin } >w/sin

; w/tanh: ( -- wave)
{ 0 wave/tanh } >w/tanh

; w/triangle: ( -- wave)
{ 0 wave/triangle } >w/triangle

; w/square: ( -- wave)
{ 0 wave/square } >w/square

; w/pulse: ( -- wave)
{ 0 wave/pulse } >w/pulse

; w/saw: ( -- wave)
{ 0 wave/saw } >w/saw

; wave phasors

; ~sin: (freq -- stream)
{ w/sin phasor } >~sin

; ~tanh: (freq -- stream)
{ w/tanh phasor } >~tanh

; ~triangle: (freq -- stream)
{ w/triangle phasor } >~triangle

; ~square: (freq -- stream)
{ w/square phasor } >~square

; ~pulse: (pw freq -- stream)
{ w/pulse phasor } >~pulse

; ~saw: (freq -- stream)
{ w/saw phasor } >~saw

; lfo

; ~lfo/sin: (freq -- stream)
{( >:freq ~sin)} >~lfo/sin

; ~lfo/tanh: (freq -- stream)
{( >:freq ~tanh)} >~lfo/tanh

; ~lfo/triangle: (freq -- stream)
{( >:freq ~triangle 0.25p skip )} >~lfo/triangle

; ~lfo/square: (freq -- stream)
{( >:freq ~square 0.26p skip)} >~lfo/square

; ~lfo/pulse: (pw freq -- stream)
{( >:freq >:pw ~pulse :pw 0.01 + periods skip )} >~lfo/pulse

; ~lfo/saw: (freq -- stream)
{( >:freq ~saw 0.76p skip )} >~lfo/saw

; unipolar <-> bipolar conversions

; uni: (bipolar -- unipolar)
{ 1 + 2 / } >uni

; bi: (unipolar -- bipolar)
{ 2 * 1 - } >bi
