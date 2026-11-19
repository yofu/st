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

## DXF library and compatibility tests

DXF output uses the published `github.com/yofu/dxf` module at
`v0.0.0-20261114161146-b17aa67668c3`. No local checkout or `replace` directive
is required. `go.mod` pins the version and `go.sum` verifies the download.

Run the portable plan, elevation and 3D export tests:

```powershell
go test -vet=off ./stlib -run '^TestDXF' -v
```

The tests read each exported DXF, audit its structure, then write and read it
again. They check member coordinates, Japanese text, steel/RC sections,
reinforcement, support points, pin groups and deformed lines.

An optional real-model test reads an existing INP/LST pair without editing it:

```powershell
$env:ST_DXF_TEST_INP = 'C:/models/model.inp'
# Optional: otherwise the same basename with .lst is used.
$env:ST_DXF_TEST_LST = 'C:/models/model.lst'
# Optional: retain outputs and portable fixture drawings here.
$env:ST_DXF_TEST_OUTPUT = 'C:/models/dxf-test-output'
go test -vet=off ./stlib -run '^TestDXF' -v
```
