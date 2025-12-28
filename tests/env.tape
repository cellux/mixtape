{( [0 1 0.5 0.5 0]
   [0.2 2 1 8] 1b distribute
   [{9 /exp} {-3 /exp} {/line} {-3 /exp}]
   env >:env
   :env len >:envlen
   [
     :env 1 ~ *
     :env 1 ~ *
     :env 1 ~ *
   ] cat
   len
   :envlen 3 * =
)} assert
