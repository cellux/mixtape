{ 1 seconds sr = } assert
{ 2 seconds sr 2 * = } assert

{ :bpm 120 = } assert

{ 1 beats sr 2 / = } assert

{ 1b 1 beats = } assert
{ 1p 1 periods = } assert

{ :tpb 96 = } assert

{ 1 ticks 1 beats 96 / = } assert

{ ( 2 >:freq 1 periods ) sr 2 / = } assert
{ ( 10 >:freq 2 periods ) sr 10 / 2 * = } assert

{ 1s 1 seconds = } assert

{ 1/4s 1 seconds 4 / = } assert
{ 1/4b 1 beats 4 / = } assert
{ 1/4p 1 periods 4 / = } assert
