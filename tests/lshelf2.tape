; lshelf2/hshelf2 should match the SVF low/high outputs mixed with gain

[1 0.25 -0.5 0.75 -1 0.1] >:testdata

{( 2.0 >:gain 0.7 >:q 400 >:cutoff
   :testdata lshelf2 12 take frames >:actual

   :testdata lp2 >:lp
   :testdata hp2 >:hp
   :lp 2.0 * :hp + 12 take frames >:expected

   :actual :expected = )} assert

{( 0.5 >:gain 0.9 >:q 2500 >:cutoff
   :testdata hshelf2 12 take frames >:actual

   :testdata lp2 >:lp
   :testdata hp2 >:hp
   :lp :hp 0.5 * + 12 take frames >:expected

   :actual :expected = )} assert
