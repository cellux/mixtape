{ [] len 0 = } assert

{ [1 2 3] [1 2 3] = } assert
{ [1 2 3] len 3 = } assert

{ [ 5 2 + 3 * 5 ] [21 5] = } assert

{ [5 9 3] 0 at 5 = } assert
{ [5 9 3] 1 at 9 = } assert
{ [5 9 3] 2 at 3 = } assert

[1 5 2] 8 push { [1 5 2 8] = } assert

[1 5 2]
{ pop 2 = } assert
{ pop 5 = } assert
{ pop 1 = } assert
{ [] = } assert

(
  { [1 5 2] swap eval
    { pop 2 = } assert
    dup { [1 5] = } assert
    9 push
    { [1 5 9] = } assert
  } >copy-with
  { clone } copy-with { [1 5 2] = } assert
  { dup } copy-with { [1 5 9] = } assert
)
