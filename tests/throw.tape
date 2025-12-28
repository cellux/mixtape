{ { "no throw in here" } catch nil? } assert
{ { "foo" throw } catch "foo" = } assert
{ { [ 1 7 9 4 3 ] { dup 2 mod 0 = { throw } if drop } each } catch 4 = } assert

{ 1 2 { 3 4 throw } catch 4 = assert 2 = assert 1 = } assert
{ [ 1 2 { 3 4 throw } catch 4 = assert ] [1 2] = } assert
{ ( 5 >:test {( 8 >:test nil throw )} catch nil? assert :test 5 = ) } assert
