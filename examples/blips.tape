[0 1 0.5 0.5 0]
[0.2 2 1 8] 1b distribute
[{9 /exp} {-3 /exp} {/line} {-3 /exp}]
env >:env

[ :env ( a-4 mtof >:freq ~sin ) *
  :env ( a-5 mtof >:freq ~triangle ) *
  :env ( a-3 mtof >:freq ~sin ) *
] cat
