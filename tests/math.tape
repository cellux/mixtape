{ 3 3 = } assert
{ 3 4 != } assert

{ 3 3 > not } assert
{ 3 3 >= } assert
{ 3 3 < not } assert
{ 3 3 <= } assert
{ 4 3 > } assert
{ 4 3 >= } assert
{ 3 4 < } assert
{ 3 4 <= } assert

{ 3 5 min 3 = } assert
{ 3 5 max 5 = } assert

{ 3 4 + 7 = } assert
{ 4 3 - 1 = } assert
{ 4 3 * 12 = } assert
{ 12 3 / 4 = } assert

{ 2 5 pow 32 = } assert

{  5  3 mod  2 = } assert
{ -5  3 mod -2 = } assert
{  5 -3 mod  2 = } assert

{  5  3 rem -1 = } assert
{ -5  3 rem  1 = } assert
{  5 -3 rem -1 = } assert

{  8 abs 8 = } assert
{ -8 abs 8 = } assert

{  8 sign 1 = } assert
{ -8 sign -1 = } assert
{  0 sign 0 = } assert

{ 4 square 16 = } assert
{ -4 square 16 = } assert

{ 1 exp 2.718281828459045 = } assert

{ 5 exp
  e 5 pow
  - abs
  0.0000000000001 < } assert

{ 4 exp2 16 = } assert

{ 10 log10 1 = } assert
{ 1000 log10 3 = } assert

{ 2 log2 1 = } assert
{ 256 log2 8 = } assert

{ 5.0 floor 5 = } assert
{ 5.3 floor 5 = } assert
{ 5.7 floor 5 = } assert
{ 6.0 floor 6 = } assert

{ 5.0 ceil 5 = } assert
{ 5.3 ceil 6 = } assert
{ 5.7 ceil 6 = } assert
{ 6.0 ceil 6 = } assert

{ 5.0 trunc 5 = } assert
{ 5.3 trunc 5 = } assert
{ 5.5 trunc 5 = } assert
{ 5.7 trunc 5 = } assert

{ 5.0 round 5 = } assert
{ 5.3 round 5 = } assert
{ 5.5 round 6 = } assert
{ 5.7 round 6 = } assert

{ pi sin 0.0000000000001 < } assert
{ pi 2 / sin 1 = } assert
