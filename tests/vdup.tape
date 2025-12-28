{ {foo bar} 3 vdup [{foo bar} {foo bar} {foo bar}] = } assert

{( [1 2 3] tape ~ >:blip
   [ 1 2 3 ] 3 vdup cat frames [1 2 3 1 2 3 1 2 3] =
)} assert
