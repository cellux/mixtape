; env generates an envelope
;
; ys: values at envelope points
; ds: durations between envelope points (in frames)
; envsegs: closures which generate the envelope segments
;
; len(envsegs) = len(ds) = len(ys) - 1

[0 1 0.85 0]
[0.9 2.7 0.4] 4b distribute
[{/cos} {-4 /exp} {-6 /exp}] env
