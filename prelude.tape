-1 >true
0 >false
{ ":freq" set } >f
{ sr * } >seconds
{ sr :bpm 60 / / * } >beats
{ sr :freq / * } >periods
{ beats :tpb / } >ticks
{ SRC_LINEAR swap resample } >stretch
{ [ swap do drop ] join } >env
