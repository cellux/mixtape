; seq iterates over N iterables
; the N iterables have N names, received in :seq/syms
;
; seq looks up the N iterables by name
; turns them into iterators -> :seq/symiters
;
; in each iteration:
;
; 1. for each iterable I:
;
;    a. gets next value V from I (eval)
;    b. if value is nil: exit
;    c. otherwise sets name of iterable I in local env to V
;
; 2. invokes seq/body
;
; seq/body typically generates a value (piece of sound, envelope
; segment, etc.) which depends on the parameters set in step 1
;
; these values can be collected by using seq inside [...]

(
{ ~saw ~sin ~pulse ~saw ~triangle } >:waveform
[ 110 110 220 330 110 220 ] >:freq
[
  { :waveform :freq }
  { :waveform eval .1 * 1/3b take }
  seq
] cat
)
