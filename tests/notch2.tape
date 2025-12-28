; notch2 basic behavior

; notch rejects center frequency while sum matches input when combining lp2 and hp2 paths
{( 400 >:cutoff 1 >:q
   [0 1 0 1] dup notch2 swap bp2  ; notch + bp should reconstruct input
   + 4 take frames [0 1 0 1] =
)} assert

; notch equals lp2+hp2 directly
{( 800 >:cutoff 0.707 >:q
   [1 0 1 0] dup notch2 swap dup lp2 swap hp2 +   ; notch vs lp+hp
   4 take frames swap 4 take frames =
)} assert
