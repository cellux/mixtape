; lp1/hp1 basic behavior

; lowpass approaches input for high cutoff (rounded)
{( 20000 >:cutoff [0 1 0 1] lp1 round 4 take frames [0 1 0 1] = )} assert

; highpass removes DC from constant input
{( 2000 >:cutoff [1 1 1 1] hp1 4 take frames [0 0 0 0] = )} assert

; highpass is input - lowpass (thus sums back to input)
{( [0 1 0 1] 400 >:cutoff dup lp1 swap hp1 + 4 take frames [0 1 0 1] = )} assert
