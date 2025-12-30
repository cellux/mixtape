{( ~noise 1.2 resample 100 take frames >:stream-resample
   ~noise 100 take 1.2 resample 100 take frames >:tape-resample

  :stream-resample :tape-resample =
)} assert
