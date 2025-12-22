; FM oscillator (~fm)
;
; ~fm is a wavetable-based phase-modulation oscillator.
;
; Stack effect:
;   ( wt -- Stream )
;
; Required env vars:
;   :freq  (carrier frequency, Hz; number or stream)
;   :mod   (phase modulation signal, in *cycles*; number or stream)
; Optional:
;   :index (mod depth multiplier; number or stream, default 1)
;   :phase (initial phase 0..1, default 0)
;
; Gotcha: :mod is in cycles (1.0 = one full rotation), not Hz.

[

; --- Minimal: no modulation (behaves like a plain wavetable osc) ---
(
  wt/sin
  220 >:freq
  0   >:mod
  ~fm
  1b take 0.25 *
)

; --- Musical: FM bell-ish hit (ratio mod + decaying index + amp envelope) ---
(
  ; amplitude envelope over 2 beats: fast attack, exponential decay
  [0 1 0]
  [0.2 7.8] 2b distribute
  [{/line} {-6 /exp}] env >:amp

  ; FM index envelope over 2 beats: bright attack -> mellow tail
  [6 2 0.5]
  [1 7] 2b distribute
  [{-6 /exp} {-5 /exp}] env >:index

  ; modulator: a sine at 2x the carrier frequency, scaled in *cycles*
  ( 440 >:freq ~sin ) 0.18 * >:mod

  ; carrier frequency
  220 >:freq

  ; run the FM oscillator, apply amp env, render
  wt/sin ~fm
  :amp ~ *
  2b take 0.25 *
)

] {join} reduce
