# cstat

A more civilized iostat, with 100x more precision, and exportable to CSV format.

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

Compare: `vmstat 1`

Just show the busy column, polling every 5 seconds for up to 5 minutes:

```
$ cstat --poll 5s --for 5m --busy --header=false
10.734
7.532
9.400
8.460
```

Can also show the memory usage, optionally include swap memory with --swap option.

```
$ mstat
elapsed	total	used	free	shared	buffers	cached	available
1	32764724	4920732	4303108	175828	6925284	16615600	27238244
2	32764724	4920984	4302856	175828	6925292	16615592	27237992
3	32764724	4921476	4302352	175828	6925292	16615604	27237492
```

Compare: `free -k`
