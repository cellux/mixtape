[0 1 0.5 0.5 0]
[0.2 2 1 8] 1b distribute
[{9 exp/} {-3 exp/} {line/} {-3 exp/}]
env >:env

[ :env ( 220 >:freq sin~ ) *
  :env ( 440 >:freq triangle~ ) *
  :env ( 110 >:freq sin~ ) *
] {join} reduce
