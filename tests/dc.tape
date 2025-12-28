; dc* basic behavior and initialization

; constant input is fully removed (no startup leak)
{ [1 1 1 1] 0.995 dc* 4 take frames [0 0 0 0] = } assert

; first frame is seeded, then standard HP difference equation follows (streaming path)
{ 0 ~ 1 z1* 0.5 dc* 4 take frames [0 -1 -0.5 -0.25] = } assert
