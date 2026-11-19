package st

import (
	"math"
	"sort"
	"strings"

	"github.com/yofu/dxf"
	"github.com/yofu/dxf/drawing"
	"github.com/yofu/dxf/entity"
)

func overlapDXFBox(a, b [4]float64, pad float64) bool {
	return a[0] < b[2]+pad && a[2] > b[0]-pad && a[1] < b[3]+pad && a[3] > b[1]-pad
}

// Keep grid/dimension text fixed, and move colliding member symbols along
// their text normal. Leaders retain their connection to the original member.
// Return unresolved collisions rather than silently claiming a clean layout.
func arrangeDXFMemberLabels(d *drawing.Drawing) int {
	var labels []*entity.Text
	var boxes [][4]float64
	addBox := func(e entity.Entity) { a, b := e.BBox(); boxes = append(boxes, [4]float64{a[0], a[1], b[0], b[1]}) }
	for _, e := range d.ModelSpaceEntities() {
		if t, ok := e.(*entity.Text); ok {
			if strings.HasPrefix(t.Layer().Name(), "TXT") {
				labels = append(labels, t)
			} else {
				addBox(t)
			}
		}
		if dim, ok := e.(*entity.Dimension); ok {
			if block, err := d.Block(dim.BlockName); err == nil {
				for _, ge := range block.Entities {
					switch ge.(type) {
					case *entity.Text, *entity.MText:
						addBox(ge)
					}
				}
			}
		}
	}
	sort.SliceStable(labels, func(i, j int) bool {
		a, b := labels[i], labels[j]
		if a.Coord1[0] != b.Coord1[0] {
			return a.Coord1[0] < b.Coord1[0]
		}
		if a.Coord1[1] != b.Coord1[1] {
			return a.Coord1[1] < b.Coord1[1]
		}
		return a.Value < b.Value
	})
	d.AddLayer("LABEL_LEADER", dxf.DefaultColor, dxf.DefaultLineType, false)
	unresolved := 0
	for _, t := range labels {
		a, b := t.BBox()
		original := [4]float64{a[0], a[1], b[0], b[1]}
		theta := t.Rotation * math.Pi / 180
		chosen := [4]float64{}
		dx, dy := 0., 0.
		found := false
		for pass := 0; pass < 81; pass++ {
			step := float64((pass + 1) / 2)
			if pass%2 == 0 {
				step = -step
			}
			if pass == 0 {
				step = 0
			}
			dx, dy = -math.Sin(theta)*step*t.Height*1.6, math.Cos(theta)*step*t.Height*1.6
			chosen = [4]float64{original[0] + dx, original[1] + dy, original[2] + dx, original[3] + dy}
			collision := false
			for _, box := range boxes {
				if overlapDXFBox(chosen, box, .15*t.Height) {
					collision = true
					break
				}
			}
			if !collision {
				found = true
				break
			}
		}
		if !found {
			unresolved++
			chosen = original
			dx, dy = 0, 0
		}
		if dx != 0 || dy != 0 {
			if err := d.Move(dx, dy, 0, t); err != nil {
				unresolved++
				chosen = original
			} else {
				d.Layer("LABEL_LEADER", true)
				d.Line((original[0]+original[2])/2, (original[1]+original[3])/2, 0, (chosen[0]+chosen[2])/2, chosen[1]-.15*t.Height, 0)
			}
		}
		boxes = append(boxes, chosen)
	}
	return unresolved
}
