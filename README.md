# cstat

A more civilized iostat, with 100x more precision.

This has only been tested so far on macOS.

```
$ cstat
elapsed	busy%	sys%	user%	nice%	idle%
1	2.707	1.140	1.567	0.000	97.293
2	1.702	0.567	1.135	0.000	98.298
3	1.994	0.997	0.997	0.000	98.006
4	1.569	0.571	0.999	0.000	98.431
5	6.695	1.994	4.701	0.000	93.305
6	6.553	2.707	3.846	0.000	93.447
```

Just show the busy column, polling every 5 seconds for up to 5 minutes:

```
$ cstat --poll 5s --for 5m --busy --header=false
10.734
7.532
9.400
8.460
```

