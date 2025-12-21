(  [ wt/sin wt/triangle wt/saw wt/pulse ] wt  >:wt    # multi-frame table
   (0 >:start 1 >:end 4s >:nf /line) >:morph           # 0→1 ramp in 4s
   110 >:freq
   :wt ~wt                                             # wavetable osc with morph
) 4 seconds take .2 *
