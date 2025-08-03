-1 >true
0 >false
[ ":freq" set ] >f
[ sr * ] >seconds
[ sr :bpm 60 / / * ] >beats
[ sr :freq / * ] >periods
[ beats :tpb / ] >ticks
[ SRC_LINEAR swap resample ] >stretch
[ mark swap do drop collect join ] >env
