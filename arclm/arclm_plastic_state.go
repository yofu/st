package arclm

// plasticElementState owns its copies: restoring a state never shares mutable
// bond definitions or hinge flags with the saved state or another element.
type plasticElementState struct {
	bonds         []*Sect
	originalBonds []*Sect
	phinge        []bool
	energy        float64
	energyb       float64
	material      *plasticMaterialState
}

func copyPlasticBonds(bonds []*Sect) []*Sect {
	if bonds == nil {
		return nil
	}
	result := make([]*Sect, len(bonds))
	for i, bond := range bonds {
		if bond != nil {
			result[i] = bond.Snapshot()
		}
	}
	return result
}

// setPlasticBond preserves the input connection on the first material-state
// transition. It also prevents modifying a bond shared with another element.
func (elem *Elem) setPlasticBond(index, num int) {
	if elem.Bonds[index].Num == num {
		return
	}
	if elem.plasticBonds == nil {
		elem.plasticBonds = make([]*Sect, len(elem.Bonds))
	}
	if elem.plasticBonds[index] == nil {
		elem.plasticBonds[index] = elem.Bonds[index].Snapshot()
	}
	bond := elem.Bonds[index].Snapshot()
	bond.Num = num
	elem.Bonds[index] = bond
}

func (elem *Elem) resetPlasticState() {
	for i, bond := range elem.plasticBonds {
		if bond != nil {
			elem.Bonds[i] = bond.Snapshot()
		}
	}
	elem.plasticBonds = nil
	elem.material = nil
	for i := range elem.Phinge {
		elem.Phinge[i] = false
	}
}

func (elem *Elem) savePlasticState() plasticElementState {
	return plasticElementState{
		bonds:         copyPlasticBonds(elem.Bonds),
		originalBonds: copyPlasticBonds(elem.plasticBonds),
		phinge:        append([]bool(nil), elem.Phinge...),
		energy:        elem.Energy,
		energyb:       elem.Energyb,
		material:      copyPlasticMaterial(elem.material),
	}
}

func (elem *Elem) restorePlasticState(state plasticElementState) {
	elem.Bonds = copyPlasticBonds(state.bonds)
	elem.plasticBonds = copyPlasticBonds(state.originalBonds)
	elem.Phinge = append([]bool(nil), state.phinge...)
	elem.Energy = state.energy
	elem.Energyb = state.energyb
	elem.material = copyPlasticMaterial(state.material)
}
