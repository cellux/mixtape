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

## min/max
{ 3 5 min 3 = } assert
{ 3 5 max 5 = } assert

## arithmetic
{ 3 4 + 7 = } assert
{ 4 3 - 1 = } assert
{ 4 3 * 12 = } assert
{ 12 3 / 4 = } assert

## power
{ 2 5 pow 32 = } assert

## modulo
{  5  3 mod  2 = } assert
{ -5  3 mod -2 = } assert
{  5 -3 mod  2 = } assert

## remainder
{  5  3 rem -1 = } assert
{ -5  3 rem  1 = } assert
{  5 -3 rem -1 = } assert

{  8 abs 8 = } assert
{ -8 abs 8 = } assert

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

## rand

{ rand 0.0 >= } assert
{ rand 1.0 < } assert

{ (0 >:seed rand) 0.9451961492941164 = } assert
{ (1 >:seed rand) 0.6046602879796196 = } assert

{ rand 0.9451961492941164 = } assert

## ratios
{ 1/4 0.25 = } assert

## trigonometry

{ pi sin 0.0000000000001 < } assert
{ pi 2 / sin 1 = } assert

# strings

{ "foo" "foo" = } assert

{ 42 str "42" = } assert

{ "hello, " "world" + "hello, world" = } assert

# symbols

{ "foo" sym "foo" sym = } assert
{ "foo" sym "foo" != } assert
{ { foo } 0 at "foo" sym = } assert
( { { foo } 0 at 5 set "foo" get 5 = } assert )
( { "foo" 5 set { foo } 0 at get 5 = } assert )

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
   [ 1 7 9 4 3 ] { dup 2 mod 0 = { dup >:even } if drop } each
   :even 4 = } assert)

{ [ 5 2 9 4 ] 2 1 partition [ [5 2] [2 9] [9 4] ] = } assert

{ [[1 5 2] [4 7 3]] zip [[1 4] [5 7] [2 3]] = } assert
{ [[1 5 2] [4 7]] zip [[1 4] [5 7]] = } assert
{ [[1 5] [4 7 3]] zip [[1 4] [5 7]] = } assert

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
  { ~saw ~sin ~pulse ~triangle } >:waveform
  { :waveform len 4 = } assert
  { :waveform len 4 = } assert
  :waveform
  { pop "~triangle" sym = } assert
  drop
)

# throw/catch

{ { "no throw in here" } catch nil? } assert
{ { "foo" throw } catch "foo" = } assert
{ { [ 1 7 9 4 3 ] { dup 2 mod 0 = { throw } if drop } each } catch 4 = } assert

{ 1 2 { 3 4 throw } catch 4 = assert 2 = assert 1 = } assert
{ [ 1 2 { 3 4 throw } catch 4 = assert ] [1 2] = } assert
{ ( 5 >:test {( 8 >:test nil throw )} catch nil? assert :test 5 = ) } assert

# loop/break

( 0 >:sum
  { [ 1 7 9 4 3 ] iter
    { next dup 2 mod 0 = { break } if :sum + >:sum } loop
    drop
    :sum 17 = } assert )

# seq

( [ 3 8 2 ] >:lo
  [ 7 9 5 ] >:hi
  {
    [
      { :lo :hi }
      { :hi :lo - }
      seq
    ] [ 4 1 3 ] =
  } assert
)

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

( { "foo" 100 set "foo" get 100 = } assert )
( { "foo" 100 set foo 100 = } assert )
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

{ 1/4s 1 seconds 4 / = } assert
{ 1/4b 1 beats 4 / = } assert
{ 1/4p 1 periods 4 / = } assert

# semitone offset -> freq multiplier
{ 1 st 2 1/12 pow = } assert
{ 5 st 2 5/12 pow = } assert
{ -3 st 2 -3/12 pow = } assert

# midi to hz
{ 69 mtof 440 = } assert
{ 57 mtof 220 = } assert
{ 60 mtof 261.6255653005986 = } assert
{ 1b 1 beats = } assert
{ 1p 1 periods = } assert

# cents -> freq multiplier
{ 0 cents 1 = } assert
{ 1200 cents 2 = } assert
{ -1200 cents 0.5 = } assert
{ 100 cents 1/12 exp2 = } assert

# utilities

{ 3 4 5 drop + 7 = } assert
{ 5 dup * 25 = } assert
{ 2 9 swap - 7 = } assert
{ 2 9 over - 7 = assert 2 = } assert
{ 2 9 7 nip * 14 = } assert

# stack is empty at end
stack len 0 = assert
