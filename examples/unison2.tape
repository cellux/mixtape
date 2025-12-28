; unison2.tape * Detuned chord machine + bass + FM lead
;
; A small 4-bar sketch built almost entirely around `unison`.
; - Pad: 8-voice unison "supersaw-ish" chord stack through SVF.
; - Bass: 3-voice unison low stack.
; - Lead: 5-voice unison FM plucks.
;
; Renders a 16-beat (4 bar @ :bpm=120) stereo Tape.

(
  ; --- Shared envelopes ---

  ; 4-beat pad envelope: slow-ish attack, gentle decay, short release
  [0 1 0.85 0]
  [0.9 2.7 0.4] 4b distribute
  [{/cos} {-4 /exp} {-6 /exp}] env >:pad/env

  ; 1-beat bass envelope: fast attack, snappy decay
  [0 1 0]
  [0.1 0.9] 1b distribute
  [{/cos} {-7 /exp}] env >:bass/env

  ; 1-beat lead envelope: clicky attack, exponential tail
  [0 1 0]
  [0.05 0.95] 1b distribute
  [{/cos} {-9 /exp}] env >:lead/env

  ; --- Wavetable sources ---

  ; A small multi-wave table for pad voices (morph 0..1 sweeps across it)
  [ wt/saw wt/triangle wt/square ] wt >:pad/wt

  ; --- Per-voice generators (closures) ---

  ; Minor chord voice: root + m3 + 5th (each is its own osc)
  {
    ; Slow movement inside the chord timbre (wavetable morph)
    (0.03 >:freq ~sin 0.5 * 0.5 +) >:morph

    ; Root
    :pad/wt ~wt 0.55 *

    ; Minor third (6/5)
    ( :freq 6/5 * >:freq :pad/wt ~wt ) 0.30 * +

    ; Fifth (3/2)
    ( :freq 3/2 * >:freq :pad/wt ~wt ) 0.25 * +
  } >pad/voice/min

  ; Major chord voice: root + M3 + 5th
  {
    (0.03 >:freq ~sin 0.5 * 0.5 +) >:morph
    :pad/wt ~wt 0.55 *
    ( :freq 5/4 * >:freq :pad/wt ~wt ) 0.30 * +
    ( :freq 3/2 * >:freq :pad/wt ~wt ) 0.25 * +
  } >pad/voice/maj

  ; Bass voice: one osc, but still run through unison for width/weight.
  {
    wt/saw ~wt
  } >bass/voice

  ; Lead voice: FM with per-voice detuned carrier freq
  {
    ; Modulator is tied to the *current* (per-voice) :freq, but we compute it
    ; in a nested scope so the carrier :freq remains unchanged.
    ( :freq 2 * >:freq ~sin ) 0.18 * >:mod
    2.5 >:index

    wt/sin ~fm

    ; Tiny DC cleanup (helps with heavy detune + envelopes)
    dc
  } >lead/voice

  ; --- Note/chord renderers ---

  ; ( rootHz -- Tape(4b, stereo) )
  {
    >:root
    (
      :root >:freq
      8    >:voices
      19   >:detune
      1.0  >:spread
      0.9  >:phaseRand

      "pad/voice/min" get unison

      ; Slow-ish lowpass movement + soft saturation
      (0.04 >:freq ~triangle -0.5 * 0.5 + 3200 * 250 +) >:cutoff
      (0.06 >:freq ~sin 0.20 * 0.75 +) >:q
      -1  >:blend
      1.8 * ;drive
      svf
      tanh

      :pad/env ~ *
      0.22 *
    ) 4b take
  } >pad/chord/min

  ; ( rootHz -- Tape(4b, stereo) )
  {
    >:root
    (
      :root >:freq
      8    >:voices
      19   >:detune
      1.0  >:spread
      0.9  >:phaseRand

      "pad/voice/maj" get unison

      (0.04 >:freq ~triangle -0.5 * 0.5 + 3200 * 250 +) >:cutoff
      (0.06 >:freq ~sin 0.20 * 0.75 +) >:q
      -1  >:blend
      1.8 * ;drive
      svf
      tanh

      :pad/env ~ *
      0.22 *
    ) 4b take
  } >pad/chord/maj

  ; ( rootHz -- Tape(1b, stereo) )
  {
    >:root
    (
      :root >:freq
      3    >:voices
      4    >:detune
      0.35 >:spread
      0.6  >:phaseRand

      "bass/voice" get unison

      220  >:cutoff
      0.9  >:q
      -1   >:blend
      1.6 * ;drive
      svf

      :bass/env ~ *
      0.26 *
    ) 1b take
  } >bass/note

  ; ( freqHz -- Tape(1b, stereo) )
  {
    >:f
    (
      :f >:freq
      5    >:voices
      11   >:detune
      0.85 >:spread
      0.7  >:phaseRand

      "lead/voice" get unison

      ; bright band-ish tone (blend≈0 => bandpass)
      (0.12 >:freq ~triangle -0.5 * 0.5 + 2500 * 400 +) >:cutoff
      (0.07 >:freq ~sin 0.25 * 1.1 +) >:q
      0   >:blend
      2.2 * ;drive
      svf
      tanh

      :lead/env ~ *
      0.18 *
    ) 1b take
  } >lead/note

  ; --- Arrange: 4-chord progression + bass pulses + lead riff ---

  ; Pad progression (each chord is 4 beats)
  [
    220      pad/chord/min   ; Am
    174.614  pad/chord/maj   ; F
    130.813  pad/chord/maj   ; C
    195.998  pad/chord/maj   ; G
  ] cat 16b take >:pad

  ; Bass: root each beat (16 notes)
  [
    110 110 110 110
    87.307 87.307 87.307 87.307
    65.406 65.406 65.406 65.406
    97.999 97.999 97.999 97.999
  ] { bass/note } map cat 16b take >:bass

  ; Lead: 4 notes per chord (16 notes)
  [
    440 523.251 659.255 523.251
    523.251 440    349.228 440
    391.995 329.628 261.626 329.628
    391.995 493.883 587.330 493.883
  ] { lead/note } map cat 16b take >:lead

  ; --- Mixdown ---
  16b tape2 >:mix
  :mix :pad  0 +@
  :mix :bass 0 +@
  :mix :lead 0 +@

  :mix
)
