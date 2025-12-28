; peak2 basic behavior

; gain=1 should be neutral (peak2 equals notch2 in this implementation)
{( 800 >:cutoff 0.707 >:q 1 >:drive 1 >:gain
   [0 1 0 1] dup peak2 swap notch2
   4 take frames swap 4 take frames =
)} assert

; varying gain as a stream should work and preserve frame count
{( 800 >:cutoff 1 >:q 1 >:drive [1 2 1 0.5] >:gain
   [1 0 1 0] peak2 4 take frames len 4 =
)} assert

; gain modulation should act per-frame.
; With q=1, the implemented formula reduces to:
;   peak2 = notch2 + (gain-1) * bp2
{( 800 >:cutoff 1 >:q 1 >:drive [1 2 0.5 1] >:gain
   [1 0 1 0] dup peak2 swap

   ; expected
   dup notch2 swap bp2
   :gain 1 - * +

   4 take frames swap 4 take frames =
)} assert
