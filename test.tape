# stack is empty at start
stack len 0 = assert

# numbers

## truth values
1 assert
-1 assert
0 not assert

## equality
3 3 = assert
3 4 != assert

## comparisons
3 3 > not assert
3 3 >= assert
3 3 < not assert
3 3 <= assert
4 3 > assert
4 3 >= assert
3 4 < assert
3 4 <= assert

## arithmetic
3 4 + 7 = assert
4 3 - 1 = assert
4 3 * 12 = assert
12 3 / 4 = assert
12 5 % 2 = assert

## ratios
1/4 0.25 = assert

# vectors

[1 2 3] len 3 = assert

5 [ 2 + 3 * ] execute 21 = assert

[ 2 + 3 * ] >test-op
5 test-op 21 = assert

# defaults

:bpm 120 = assert
:sr 48000 = assert
:freq 440 = assert
:phase 0 = assert
:width 0.5 = assert

# environments

( 100 >:bpm :bpm 100 = assert
  ( 90 >:bpm :bpm 90 = assert )
  :bpm 100 = assert
)
:bpm 120 = assert

# env/set env/get
( 100 "foo" env/set "foo" env/get 100 = assert )
( 100 "foo" env/set foo 100 = assert )
( 100 >foo "foo" env/get 100 = assert )

# conversions

42 str "42" = assert
"42" num 42 = assert

# seconds and beats

1 seconds 48000 = assert
2 seconds 96000 = assert

1 beats 24000 = assert

1s 1 seconds = assert
1b 1 beats = assert

1/4s 1 seconds 4 / = assert
1/4b 1 beats 4 / = assert

( 60 >:bpm 44100 >:sr 1 beats 44100 = assert )
( 60 >:bpm 48000 >:sr 1 beats 48000 = assert )
( 120 >:bpm 44100 >:sr 1 beats 22050 = assert )

# utilities

5 dup * 25 = assert

# stack is empty at end
stack len 0 = assert
