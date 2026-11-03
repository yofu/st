```
         software
     forstructural
   analysisthename
  ofwhichstandsfor

 sigmatau  stress
structure  steel
andsometh  ing
 likethat
```

### Material yielding at member ends (Arclm101)

The optional `YIELDEND` entry in a `SECT` block selects material yielding in
the element's `ENOD` order. `1` enables yielding and `0` keeps the end elastic:

```text
         YIELDEND 1 0
```

Add this line to `SECT 221` to use its existing `NZMAX` through `MYMIN`, `EXP`
and `EXQ` at the first end only. The second end develops no plastic deformation
and is not limited by those capacities. Its forces still change through member
equilibrium and redistribution when the first end yields. The elastic stiffness
and input connection springs/releases are retained.

Omitting the entry, or specifying `YIELDEND 1 1`, retains the previous behavior.
`YIELDEND 0 1` enables only the second end; `YIELDEND 0 0` keeps both ends elastic.
This setting applies to every member using that section. Check the order of the
two node numbers in each member's `ENOD` entry; it is not inferred from height
or coordinates. Different capacities or exponents at the two ends are not
introduced by this option.

Rebuild the application, reload the `.inp`, extract the analysis frames again,
and start Arclm101 with `-init` after changing this material setting. Do not
continue an analysis with material history calculated under the old setting.
The setting survives `.inp` and arclm section exports and section snapshots.
It also applies to time-history analysis, which shares the material return solver.
Older application versions do not implement `YIELDEND`.
