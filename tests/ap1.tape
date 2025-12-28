; ap1 basic behavior

; allpass leaves constant DC input unchanged
{( 1 >:cutoff [1 1 1 1] ap1 4 take frames [1 1 1 1] = )} assert

; allpass preserves sinusoid energy (after initial transient)
; use mean-square over a window to avoid phase/window max-sample artifacts
{( 400 >:cutoff
   1000 >:freq 0 >:phase
   ~sin dup ap1

   { {square} map avg } >ms

   ; stack: dry wet
   200 skip 5000 take frames ms >:wet
   200 skip 5000 take frames ms >:dry

   :wet :dry - abs
   1e-6 <
)} assert
