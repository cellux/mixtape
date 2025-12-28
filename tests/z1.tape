{ 1 ~ 0 z1* 3 take frames [0 1 1] = } assert
{ [1 2] ~ 0 z1* 3 take frames [0 1 2] = } assert
{ [[0.5 1.5]] ~ [0 1] z1* 2 take frames [[0 1] [0.5 1.5]] = } assert
{ 1 ~ z1 3 take frames [0 1 1] = } assert
{ [[1 2]] ~ z1 2 take frames [[0 0] [1 2]] = } assert

{ [1 2] tape 0 z1* len 3 = } assert
