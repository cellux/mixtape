{ {[1 2 3]} len 5 = } assert
{ {[1 2 3]} eval [1 2 3] = } assert

{ {[[1 2 3] [2 3 4]]} eval len 2 = } assert

(
  { 2 + 3 * } >test-op
  { 5 test-op 21 = } assert
)
