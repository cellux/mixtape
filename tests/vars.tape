( { 100 >:bpm :bpm 100 = } assert
  ( { 90 >:bpm :bpm 90 = } assert )
  { :bpm 100 = } assert
)
{ :bpm 120 = } assert

( { 100 "foo" set "foo" get 100 = } assert )
( { 100 "foo" set foo 100 = } assert )
( { 100 >foo "foo" get 100 = } assert )
( { 100 >foo @foo 100 = } assert )
