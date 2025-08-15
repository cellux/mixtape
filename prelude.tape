-1 >true
0 >false

{ 0 = } >false?
{ 0 = } >not
{ = false? } >!=
{ 0 != } >true?
{ nil = } >nil?

{ false? { "assertion failed" throw } if } >assert
440 >:freq
0 >:phase
0.5 >:pw

{ ":freq" set } >f
{ sr * } >seconds
{ sr :bpm 60 / / * } >beats
{ sr :freq / * } >periods
{ beats :tpb / } >ticks
{ SRC_LINEAR swap resample } >stretch
