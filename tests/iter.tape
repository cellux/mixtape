3 iter
{ next 0 = } assert
{ next 1 = } assert
{ next 2 = } assert
{ next nil? } assert
drop

[1 2 3] iter
{ next 1 = } assert
{ next 2 = } assert
{ next 3 = } assert
{ next nil? } assert
{ next nil? } assert
drop

( 0 >:sum
  { [ 1 7 9 4 3 ] iter
    { next dup 2 mod 0 = { break } if :sum + >:sum } loop
    drop
    :sum 17 = } assert )
