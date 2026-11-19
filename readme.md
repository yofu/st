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

### Dimensioned plans and elevations

Add `-dim` to use editable DXF dimensions, grid lines, story levels and
member symbols. Existing output without `-dim` keeps its previous behavior.

```text
:plan plan.dxf -floor=3 -dim
:jiku Y2.dxf -name=Y2 -dim
:crosssection elevation.dxf -axis=1 -min=-0.1 -max=0.1 -dim
```

The default coordinate conversion is model metres to drawing millimetres
(`-scale=1000`). `-height` sets text height and `-axissize` the grid bubble
radius in drawing units. Dimensions are on `DIM_SPAN`, `DIM_TOTAL` and
`DIM_HEIGHT`; grid labels and floor lines have separate layers.

Current in-memory grids take precedence. If none are loaded, a same-basename
`.kjn` is read into a private snapshot, without modifying the GUI model.
Horizontal/vertical grids and actual floor levels in that file are used for
dimensions. Ai `HEIGHT` boundaries are story-selection boundaries, not floor
levels. Without KJN definitions, grids are inferred from column axes and
levels from representative horizontal beams, plus the model base/top.
Check inferred levels before using the drawings as design documents.
The `-name` elevation selector supports grids parallel to X or Y; slanted
grids are rejected rather than silently projected onto the wrong axis.

### Sheet layouts and section schedules

```text
:drawings sheets.dxf -plans=3 -grids=Y2,X1 -sections -paper=A3
:dxfsections sections.dxf -sect=101,201 -paper=A3 -sectioncolumns=3 -sectionrows=2
```

These commands use the current model and LST definitions. A3/A4 are
landscape; A3P/A4P are portrait. `-columns`/`-rows` arrange model views;
`-sectioncolumns`/`-sectionrows` arrange section cards and automatically
start new pages. `-scale=auto` (default) selects a fitting standard scale
for each model view; `-scale=100` requires 1:100 and fails if it would crop
the drawing. Section figures select their own fitting scale.
Each paper-space layout contains a border, title, page number, view titles
and locked viewports. Geometry stays in model-space mm, with independent
blocks and dimension styles for each view. Member labels are moved away
from conflicting text with leaders; unresolved conflicts are reported.

RC column/beam cards show dimensions, concrete, actual main-bar positions,
bar counts/grades, hoop notation and use floors. Steel column/beam/brace
cards show shape, dimensions and material. Undefined or unsupported details
produce placeholders and warnings, not inferred reinforcement or connection
details. INP/LST shape disagreements are reported; current LST takes
precedence. Wall/slab metadata is included, but their section figures and
construction details are not yet implemented.

`-title=text` sets the sheet title and `-font=filename` selects a font
(default `msgothic.ttc`). Fonts are referenced by DXF, not embedded; the
receiving CAD needs the selected font or a substitute. Warnings appear in
the command history and in the Go/MCP report.

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
