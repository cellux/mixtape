(
{ ~saw ~sin ~pulse ~saw ~triangle } >:waveform
[ 110 110 220 330 110 220 ] >:freq
[
  { :waveform :freq }
  { :waveform eval .1 * 1/3b take }
  seq
] cat
)
