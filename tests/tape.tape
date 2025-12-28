; tape operations
{ [1 2 3 4] tape 1 shift frames [2 3 4 1] = } assert
{ [1 2 3 4] tape 0.5 shift frames [3 4 1 2] = } assert
{ [1 2 3 4] tape -1 shift frames [4 1 2 3] = } assert

{ [10 20 30] tape 1 at [20] = } assert

{ [1 2 3 4] tape 1 3 slice frames [2 3] = } assert

{ [1 2 3] tape [10 20] tape 1 +@ frames [1 12 23] = } assert
{ [1 2] tape [3 4 5] tape 1 +@ frames [1 5 4 5] = } assert

{ 4 tape/square frames [1 -1 -1 1] = } assert
{ 4 tape/triangle frames [0 1 0 -1] = } assert
{ 4 tape/pulse frames [1 1 -1 -1] = } assert
{ ( 0.25 >:pw 4 tape/pulse ) frames [1 -1 -1 -1] = } assert
{ 4 tape/saw frames [0 0.5 -1 -0.5] = } assert
