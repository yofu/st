package st

import "slices"

func (sw *SWall) Snapshot() SectionRate {
	r := *sw
	r.Wrect = slices.Clone(sw.Wrect)
	return &r
}

func (rc *RCColumn) Snapshot() SectionRate {
	r := *rc
	r.Reins = slices.Clone(rc.Reins)
	for i := range r.Reins {
		r.Reins[i].Position = slices.Clone(rc.Reins[i].Position)
	}
	r.Hoops.Ps = slices.Clone(rc.Hoops.Ps)
	r.XFace, r.YFace = slices.Clone(rc.XFace), slices.Clone(rc.YFace)
	return &r
}

func (rw *RCWall) Snapshot() SectionRate {
	r := *rw
	r.Wrect = slices.Clone(rw.Wrect)
	r.XFace, r.YFace = slices.Clone(rw.XFace), slices.Clone(rw.YFace)
	return &r
}

func (ww *WoodWall) Snapshot() SectionRate {
	r := *ww
	r.Wrect = slices.Clone(ww.Wrect)
	return &r
}

func (sg *SGirder) Snapshot() SectionRate  { return &SGirder{*sg.SColumn.Snapshot().(*SColumn)} }
func (sb *SBrace) Snapshot() SectionRate   { return &SBrace{*sb.SColumn.Snapshot().(*SColumn)} }
func (rg *RCGirder) Snapshot() SectionRate { return &RCGirder{*rg.RCColumn.Snapshot().(*RCColumn)} }
func (rs *RCSlab) Snapshot() SectionRate   { return &RCSlab{*rs.RCWall.Snapshot().(*RCWall)} }
func (wg *WoodGirder) Snapshot() SectionRate {
	return &WoodGirder{*wg.WoodColumn.Snapshot().(*WoodColumn)}
}
func (ws *WoodSlab) Snapshot() SectionRate { return &WoodSlab{*ws.WoodWall.Snapshot().(*WoodWall)} }
