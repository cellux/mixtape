( { 0 >:sum [1 2 3] {:sum + >:sum} each :sum 6 = } assert )

({ 0 >:even
   [ 1 7 9 4 3 ] { dup 2 mod 0 = { dup >:even } if drop } each
   :even 4 = } assert)
