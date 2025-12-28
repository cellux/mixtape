{ [[1] tape [2] tape [3] tape] cat frames [1 2 3] = } assert

{( [1 2 3] tape ~ >:blip
   [ :blip :blip :blip ] cat frames [1 2 3 1 2 3 1 2 3] =
)} assert

{( [1 2 3] tape ~ >:blip
   [ :blip :blip :blip ] cat len 9 =
)} assert

{( [1 2 3] tape ~ >:blip
   [ :blip 2 * :blip 3 * :blip 4 * ] cat frames
   [2 4 6 3 6 9 4 8 12] =
)} assert

{( [1 2] tape ~ >:blip
   [ :blip 1 + :blip 2 + ] cat frames
   [2 3 3 4] =
)} assert
