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

{( >seq/body >:seq/syms
   :seq/syms { get iter } map >:seq/symiters
   {
     [:seq/syms :seq/symiters] zip {
       dup 1 at # symiter
       swap 0 at # sym
       swap
       eval dup nil? { break } if set
     } each
     seq/body
   } loop
)} >seq

# oscillator defaults
440 >:freq
0.0 >:phase
0.5 >:pw

# filter defaults
1200  >:cutoff
0.7   >:q
1.0   >:drive
0.0   >:blend
false >:saturate

# FM defaults
0.0 >:mod
1.0 >:index

# RNG defaults
0 >:seed

0 >:start
1 >:end

{ ":freq" swap set } >f
{ sr * } >seconds
{ sr :bpm 60 / / * } >beats
{ sr :freq / * } >periods
{ beats :tpb / } >ticks
{ SRC_LINEAR swap resample } >stretch

{ 2 1 partition
  dup { 0 at } map >:start
  dup { 1 at } map >:end
  drop
} >start:end

{( over {+} reduce / >:unit { :unit * } map )} >distribute

{ >:envseg >:nf start:end
  [
    { :start :end :nf :envseg }
    { :envseg eval }
    seq
  ] { join } reduce
} >env

# default waves
{ 0 wave/sin } >w/sin
{ 0 wave/tanh } >w/tanh
{ 0 wave/triangle } >w/triangle
{ 0 wave/square } >w/square
{ 0 wave/pulse } >w/pulse
{ 0 wave/saw } >w/saw

# wave phasors
{ sin phasor } >~sin
{ tanh phasor } >~tanh
{ triangle phasor } >~triangle
{ square phasor } >~square
{ pulse phasor } >~pulse
{ saw phasor } >~saw
