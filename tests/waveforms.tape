(
  { ~saw ~sin ~pulse ~triangle } >:waveform
  { :waveform len 4 = } assert
  { :waveform len 4 = } assert
  :waveform
  { pop "~triangle" sym = } assert
  drop
)
