# stack is empty at start
stack len 0 = assert

# truth:
#
# non-zero = true
# zero     = false

{ 1 } assert
{ -1 } assert
{ 0 not } assert

{ true -1 = } assert
{ false 0 = } assert

{ false false? } assert

{ true true? } assert
{ 10 true? } assert
{ -10 true? } assert

{ nil nil? } assert
{ 0 nil? not } assert

{ 5 3 > "gt" if "gt" = } assert

{ 5 3 > "gt" "lt" if "gt" = } assert
{ 5 3 < "gt" "lt" if "lt" = } assert

{ 7 2 5 3 > {+ 10 +} {*} if 19 = } assert
{ 7 2 5 3 < {+} {* 3 -} if 11 = } assert

# nil

{ nil nil = } assert
{ nil 0 != } assert
{ 0 nil != } assert

# numbers (Num)

## equality
{ 3 3 = } assert
{ 3 4 != } assert

## comparisons
{ 3 3 > not } assert
{ 3 3 >= } assert
{ 3 3 < not } assert
{ 3 3 <= } assert
{ 4 3 > } assert
{ 4 3 >= } assert
{ 3 4 < } assert
{ 3 4 <= } assert

## arithmetic
{ 3 4 + 7 = } assert
{ 4 3 - 1 = } assert
{ 4 3 * 12 = } assert
{ 12 3 / 4 = } assert
{ 12 5 % 2 = } assert

## ratios
{ 1/4 0.25 = } assert

# strings

{ "foo" "foo" = } assert

{ 42 str "42" = } assert

{ "hello, " "world" + "hello, world" = } assert

# symbols

{ "foo" sym "foo" sym = } assert
{ "foo" sym "foo" != } assert

# vectors

{ [] len 0 = } assert

{ [1 2 3] [1 2 3] = } assert
{ [1 2 3] len 3 = } assert

{ [ 5 2 + 3 * 5 ] [21 5] = } assert

{ [5 9 3] 0 at 5 = } assert
{ [5 9 3] 1 at 9 = } assert
{ [5 9 3] 2 at 3 = } assert

[1 2 3] iter
{ next 1 = } assert
{ next 2 = } assert
{ next 3 = } assert
{ next nil? } assert
{ next nil? } assert
drop

[1 5 2] 8 push
{ [1 5 2 8] = } assert

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

( { 0 >:sum [1 2 3] {:sum + >:sum} each :sum 6 = } assert )

({ 0 >:even
   [ 1 7 9 4 3 ] { dup 2 % 0 = { dup >:even } if drop } each
   :even 4 = } assert)

# quoting

{ {[1 2 3]} len 5 = } assert
{ {[1 2 3]} eval [1 2 3] = } assert

{ {[[1 2 3] [2 3 4]]} eval len 2 = } assert

(
  { 2 + 3 * } >test-op
  { 5 test-op 21 = } assert
)

{ 5 { 2 + 3 * } eval 21 = } assert

{ [2 3 4] { 1 + } map [3 4 5] = } assert
{ [2 4 6] { 2 / } map [1 2 3] = } assert

{ [2 3 4] {+} reduce 9 = } assert
{ [2 3 4] {*} reduce 24 = } assert

(
  { saw~ sin~ pulse~ triangle~ } >:waveform
  { :waveform len 4 = } assert
  { :waveform len 4 = } assert
  :waveform
  { pop "triangle~" sym = } assert
  drop
)

{ [ 5 2 9 4 ] 2 1 partition [ [5 2] [2 9] [9 4] ] = } assert

# throw/catch

{ "foo" throw } catch "foo" = assert
{ [ 1 7 9 4 3 ] { dup 2 % 0 = { throw } if drop } each } catch 4 = assert

# parsing

{ "1234 4321.5 *" parse {1234 4321.5 *} = } assert
{ "42 24" parse1 42 = } assert

# defaults

{ sr 48000 = } assert

{ :bpm 120 = } assert
{ :tpb 96 = } assert
{ :freq 440 = } assert
{ :phase 0 = } assert
{ :pw 0.5 = } assert

# environments

( { 100 >:bpm :bpm 100 = } assert
  ( { 90 >:bpm :bpm 90 = } assert )
  { :bpm 100 = } assert
)
{ :bpm 120 = } assert

# set/get

( { 100 "foo" set "foo" get 100 = } assert )
( { 100 "foo" set foo 100 = } assert )
( { 100 >foo "foo" get 100 = } assert )
( { 100 >foo @foo 100 = } assert )

# seconds and beats

{ 1 seconds 48000 = } assert
{ 2 seconds 96000 = } assert

{ 1 beats 24000 = } assert
{ 1 ticks 24000 96 / = } assert

{ ( 2 >:freq 1 periods ) 24000 = } assert
{ ( 10 >:freq 2 periods ) 48000 10 / 2 * = } assert

{ ( 413 f :freq ) 413 = } assert

{ 1s 1 seconds = } assert
{ 1b 1 beats = } assert
{ 1p 1 periods = } assert

{ 1/4s 1 seconds 4 / = } assert
{ 1/4b 1 beats 4 / = } assert
{ 1/4p 1 periods 4 / = } assert

# utilities

{ 3 4 5 drop + 7 = } assert
{ 5 dup * 25 = } assert
{ 2 9 swap - 7 = } assert
{ 2 9 over - 7 = assert 2 = } assert
{ 2 9 7 nip * 14 = } assert

# stack is empty at end
stack len 0 = assert
