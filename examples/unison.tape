# Choir pad: saw+sine blend, 6-voice unison, slow attack/release, 10s render
(
  { saw 0.6 * sin 0.4 * + ~wt }     # per-voice source: blended saw+sine wavetable osc
  110        >:freq                 # base A2
  6          >:voices               # unison voices
  14         >:detune               # cents spread
  0.9        >:spread               # stereo width
  0.8        >:phaseRand            # randomization of initial phase for individual voices
  unison                            # build stacked osc -> stereo stream

  # gentle lowpass to soften highs (optional)
  #  2000 >:cutoff lp

  # amplitude envelope: attack=2s, decay=2s, sustain=0.7, release=4s
  # 2 2 0.7 4 adsr *
) 10s take 0.3 *
