{ "foo" sym "foo" sym = } assert
{ "foo" sym "foo" != } assert
{ { foo } 0 at "foo" sym = } assert
( { 5 { foo } 0 at set "foo" get 5 = } assert )
( { 5 "foo" set { foo } 0 at get 5 = } assert )
