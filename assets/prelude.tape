-1 >true
0 >false

{ 0 = } >false?
{ 0 = } >not
{ = false? } >!=
{ 0 != } >true?
{ nil = } >nil?

{ dup >:assert/value
  eval
  false? { "assertion failed: " :assert/value str + throw } if
} >assert

{ nil throw } >break

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
440 >:freq
0.0 >:phase
0.5 >:pw

; filter defaults
1200  >:cutoff
0.7   >:q
1.0   >:drive
0.0   >:blend
false >:saturate

; FM defaults
0.0 >:mod
1.0 >:index

; RNG defaults
0 >:seed

0 >:start
1 >:end

{ ":freq" swap set } >f
{ sr * } >seconds
{ sr :bpm 60 / / * } >beats
{ sr :freq / * } >periods

{ beats :tpb / } >ticks
{ SRC_LINEAR swap resample } >stretch

; semitone offset -> freq multiplier
{ 12 / exp2 } >st

; MIDI note -> frequency
{ 69 - st 440 * } >mtof

; cents -> freq multiplier
{ 1200 / exp2 } >cents

{ 2 1 partition
  dup { 0 at } map >:start
  dup { 1 at } map >:end
  drop
} >start:end

{( over {+} reduce / >:unit { :unit * } map )} >distribute

{ {join} reduce } >cat
{ vdup cat } >repeat
{ dup len swap {~} map {+} reduce swap / } >mix

{ >:envseg >:nf start:end
  [
    { :start :end :nf :envseg }
    { :envseg eval }
    seq
  ] cat
} >env

; default waves
{ 0 wave/sin } >w/sin
{ 0 wave/tanh } >w/tanh
{ 0 wave/triangle } >w/triangle
{ 0 wave/square } >w/square
{ 0 wave/pulse } >w/pulse
{ 0 wave/saw } >w/saw

; wave phasors
{ w/sin phasor } >~sin
{ w/tanh phasor } >~tanh
{ w/triangle phasor } >~triangle
{ w/square phasor } >~square
{ w/pulse phasor } >~pulse
{ w/saw phasor } >~saw
