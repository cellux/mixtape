; lp2/bp2/hp2 wrappers use the correct SVF blend mapping

( [1 0.9 0.5 -0.4 0.22 -0.7] >:testdata

{( 800 >:cutoff 0.8 >:q 1 >:drive -1 >:blend
   :testdata vital/svf frames
   >:baseline

   800 >:cutoff 0.8 >:q 1 >:drive
   :testdata lp2 frames
   >:wrapper

   :baseline :wrapper = )} assert

{( 1200 >:cutoff 0.7 >:q 1 >:drive 0 >:blend
   :testdata vital/svf frames
   >:baseline

   1200 >:cutoff 0.7 >:q 1 >:drive
   :testdata bp2 frames
   >:wrapper

   :baseline :wrapper = )} assert

{( 600 >:cutoff 0.9 >:q 1 >:drive 1 >:blend
   :testdata vital/svf frames
   >:baseline

   600 >:cutoff 0.9 >:q 1 >:drive
   :testdata hp2 frames
   >:wrapper

   :baseline :wrapper = )} assert
