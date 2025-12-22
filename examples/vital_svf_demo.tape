# Vital-inspired SVF demo: three clips that stress cutoff/Q/drive/blend/saturate.
# Each clip sets :cutoff, :q, :drive, :blend, :saturate in the env; only the
# input stream sits on the stack before calling vital/svf.
[
(
  # 1) Lush low-pass sweep with gentle tremolo and soft saturation
  (0.08 >:freq ~triangle -0.5 * 0.5 + 4800 * 200 +) >:cutoff  # 200..5000 Hz sweep
  (0.12 >:freq ~sin 0.15 * 0.75 +) >:q                        # Q ~0.75..0.9
  1.5 >:drive
  -1  >:blend                                                 # pure low-pass
  1   >:saturate

  110 >:freq ~saw
  (0.05 >:freq ~sin 0.1 * 0.9 +) *                            # gentle tremolo pre-filter
  vital/svf
  8b take 0.28 *
)
(
  # 2) Acidic band-pass wobble with hot drive and tanh
  (0.6 >:freq ~triangle -0.5 * 0.5 + 2200 * 300 +) >:cutoff    # ~300..2500 Hz
  (0.5 >:freq ~sin 0.6 * 1.2 +) >:q                            # Q ~0.6..1.8
  3.0 >:drive
  0   >:blend                                                  # band-pass
  1   >:saturate

  110 >:freq ~saw 2 *                                          # extra harmonics
  vital/svf
  8b take 0.22 *
)
(
  # 3) Morphing LP->BP->HP blend with moderate drive, no saturation
  (0.18 >:freq ~triangle -0.5 * 0.5 + 4000 * 150 +) >:cutoff   # 150..4150 Hz
  (0.08 >:freq ~sin 0.2 * 0.8 +) >:q                           # Q ~0.8..1.0
  1.8 >:drive
  (1 >:freq ~sin) >:blend                                      # blend swings -1..+1
  0   >:saturate

  110 >:freq ~triangle
  vital/svf
  8b take 0.26 *
)
] {join} reduce
