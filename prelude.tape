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

440 >:freq
0 >:phase
0.5 >:pw

{ ":freq" set } >f
{ sr * } >seconds
{ sr :bpm 60 / / * } >beats
{ sr :freq / * } >periods
{ beats :tpb / } >ticks
{ SRC_LINEAR swap resample } >stretch

{ 2 1 partition
  dup { 0 at } map >:start
  dup { 1 at } map >:end
  drop
} >start->end
