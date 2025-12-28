total=0
ok=0
fail=0

for t in tests/*.tape; do
  if ./mixtape -f $t -e '{ stack len 0 = } assert'; then
    ((++ok))
  else
    ((++fail))
  fi
done

if [ "$fail" -eq 0 ]; then
  echo "$ok OK"
else
  echo "$fail FAILED"
fi
