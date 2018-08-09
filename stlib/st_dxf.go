package st

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	addedge = false
	factor  = 0.001
)

var (
	entstart   = regexp.MustCompile("^ *0 *$")
	layeretype = regexp.MustCompile("[a-zA-Z]+")
	layersect  = regexp.MustCompile("[0-9]+")
	re_conf    = regexp.MustCompile("^CONF([01]{6})$")
)

func (frame *Frame) ReadDxf(filename string, coord []float64, eps float64) (err error) {
	var parse = 0
	f, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	var lis []string
	if len(coord) < 3 {
		coord = []float64{0.0, 0.0, 0.0}
	}
	if ok := strings.HasSuffix(string(f), "\r\n"); ok {
		lis = strings.Split(string(f), "\r\n")
	} else {
		lis = strings.Split(string(f), "\n")
	}
	tmp := make([]string, 0)
	var vertices []*Node
	namehandle := make(map[int]string)
	elemhandle := make(map[int]*Elem)
	for ind, j := range lis {
		var words []string
		for _, k := range strings.Split(j, " ") {
			if k != "" {
				words = append(words, k)
			}
		}
		if len(words) == 0 {
			continue
		}
		first := words[0]
		if first == "SECTION" {
			str := strings.ToUpper(strings.Replace(lis[ind+2], " ", "", -1))
			switch str {
			case "ENTITIES":
				parse = 1
			case "OBJECTS":
				vertices, elemhandle, err = frame.ParseDxfEntities(tmp, elemhandle, coord, vertices, eps)
				if err != nil {
					return err
				}
				tmp = []string{strings.Join(words, " ")}
				parse = 2
			default:
				parse = 0
			}
		}
		switch parse {
		case 1:
			if ind%2 == 1 {
				tmp = append(tmp, strings.Join(words, " "))
			} else {
				switch {
				default:
					tmp = append(tmp, strings.Join(words, " "))
				case entstart.MatchString(first):
					vertices, elemhandle, err = frame.ParseDxfEntities(tmp, elemhandle, coord, vertices, eps)
					if err != nil {
						return err
					}
					tmp = []string{strings.Join(words, " ")}
				}
			}
		case 2:
			if ind%2 == 1 {
				tmp = append(tmp, strings.Join(words, " "))
			} else {
				switch {
				default:
					tmp = append(tmp, strings.Join(words, " "))
				case entstart.MatchString(first):
					namehandle, err = frame.ParseDxfObjects(tmp, namehandle, elemhandle)
					if err != nil {
						return err
					}
					tmp = []string{strings.Join(words, " ")}
				}
			}
		}
	}
	frame.Name = filepath.Base(filename)
	return nil
}

func (frame *Frame) ParseDxfEntities(lis []string, elemhandle map[int]*Elem, coord []float64, vertices []*Node, eps float64) ([]*Node, map[int]*Elem, error) {
	var err error
	if len(lis) < 2 {
		return nil, elemhandle, nil
	}
	tp := lis[1]
	switch tp {
	case "LINE":
		elemhandle, err = frame.ParseDxfLine(lis, elemhandle, coord, eps)
	// case "POLYLINE":
	//     err = frame.ParseDxfPolyLine(lis)
	case "POINT":
		err = frame.ParseDxfPoint(lis, coord, eps)
	case "VERTEX":
		vertices, err = frame.ParseDxfVertex(lis, coord, vertices, eps)
	case "SEQEND":
		vertices = make([]*Node, 0)
	case "3DFACE":
		err = frame.ParseDxf3DFace(lis, coord, eps)
	case "ARC":
		err = frame.ParseDxfArc(lis, coord, eps)
	}
	return vertices, elemhandle, err
}

func (frame *Frame) ParseDxfLine(lis []string, elemhandle map[int]*Elem, coord []float64, eps float64) (map[int]*Elem, error) {
	var err error
	var index, h int64
	handle := 0
	var sect *Sect
	var etype int
	var startx, starty, startz, endx, endy, endz float64
	for i, word := range lis {
		if i%2 != 0 {
			continue
		}
		index, err = strconv.ParseInt(word, 10, 64)
		if err != nil {
			return elemhandle, err
		}
		switch int(index) {
		case 5:
			h, err = strconv.ParseInt(lis[i+1], 16, 64)
			if err != nil {
				return elemhandle, err
			}
			handle = int(h)
		case 8:
			etype = Etype(layeretype.FindString(lis[i+1]))
			if etype == 0 {
				return elemhandle, nil
			}
			tmp, err := strconv.ParseInt(layersect.FindString(lis[i+1]), 10, 64)
			if err != nil {
				return elemhandle, err
			}
			if val, ok := frame.Sects[int(tmp)]; ok {
				sect = val
			} else {
				sec := frame.AddSect(int(tmp))
				sect = sec
			}
		case 10:
			startx, err = strconv.ParseFloat(lis[i+1], 64)
		case 20:
			starty, err = strconv.ParseFloat(lis[i+1], 64)
		case 30:
			startz, err = strconv.ParseFloat(lis[i+1], 64)
		case 11:
			endx, err = strconv.ParseFloat(lis[i+1], 64)
		case 21:
			endy, err = strconv.ParseFloat(lis[i+1], 64)
		case 31:
			endz, err = strconv.ParseFloat(lis[i+1], 64)
		}
		if err != nil {
			return elemhandle, err
		}
	}
	n1, _ := frame.CoordNode(startx*factor+coord[0], starty*factor+coord[1], startz*factor+coord[2], eps)
	n2, _ := frame.CoordNode(endx*factor+coord[0], endy*factor+coord[1], endz*factor+coord[2], eps)
	el := frame.AddLineElem(-1, []*Node{n1, n2}, sect, etype)
	if handle != 0 {
		elemhandle[handle] = el
	}
	return elemhandle, nil
}

func (frame *Frame) ParseDxfArc(lis []string, coord []float64, eps float64) error {
	var err error
	var index int64
	ex := []float64{0.0, 0.0, 1.0}
	var cx, cy, cz, r float64
	var start, end float64
	for i, word := range lis {
		if i%2 != 0 {
			continue
		}
		index, err = strconv.ParseInt(word, 10, 64)
		if err != nil {
			return err
		}
		switch int(index) {
		case 10:
			cx, err = strconv.ParseFloat(lis[i+1], 64)
		case 20:
			cy, err = strconv.ParseFloat(lis[i+1], 64)
		case 30:
			cz, err = strconv.ParseFloat(lis[i+1], 64)
		case 40:
			r, err = strconv.ParseFloat(lis[i+1], 64)
		case 50:
			start, err = strconv.ParseFloat(lis[i+1], 64)
		case 51:
			end, err = strconv.ParseFloat(lis[i+1], 64)
		case 210:
			val, err := strconv.ParseFloat(lis[i+1], 64)
			if err == nil {
				ex[0] = val
			}
		case 220:
			val, err := strconv.ParseFloat(lis[i+1], 64)
			if err == nil {
				ex[1] = val
			}
		case 230:
			val, err := strconv.ParseFloat(lis[i+1], 64)
			if err == nil {
				ex[2] = val
			}
		}
		if err != nil {
			return err
		}
	}
	var mid float64
	if start > end {
		mid = 0.5*(start+end) + 180
	} else {
		mid = 0.5 * (start + end)
	}
	p, err := ArcPoints([]float64{cx*factor + coord[0], cy*factor + coord[1], cz*factor + coord[2]}, r*factor, ex, start, mid, end)
	if err != nil {
		return err
	}
	frame.AddArc(p, eps)
	return nil
}

func (frame *Frame) ParseDxfPoint(lis []string, coord []float64, eps float64) error {
	var err error
	var index int64
	var x, y, z float64
	var conf []bool
	for i, word := range lis {
		if i%2 != 0 {
			continue
		}
		index, err = strconv.ParseInt(word, 10, 64)
		if err != nil {
			return err
		}
		switch int(index) {
		case 8:
			str := strings.ToUpper(lis[i+1])
			if re_conf.MatchString(str) {
				fs := re_conf.FindStringSubmatch(str)
				conf = []bool{false, false, false, false, false, false}
				for j, s := range fs[1] {
					if s == '1' {
						conf[j] = true
					}
				}
			} else {
				switch str {
				default:
					conf = []bool{false, false, false, false, false, false}
				case "FIX":
					conf = []bool{true, true, true, true, true, true}
				case "PIN":
					conf = []bool{true, true, true, false, false, false}
				}
			}
		case 10:
			x, err = strconv.ParseFloat(lis[i+1], 64)
		case 20:
			y, err = strconv.ParseFloat(lis[i+1], 64)
		case 30:
			z, err = strconv.ParseFloat(lis[i+1], 64)
		}
		if err != nil {
			return err
		}
	}
	n, _ := frame.CoordNode(x*factor+coord[0], y*factor+coord[1], z*factor+coord[2], eps)
	if conf != nil {
		n.Conf = conf
	}
	return nil
}

func (frame *Frame) ParseDxf3DFace(lis []string, coord []float64, eps float64) error {
	var err error
	var index int64
	var sect *Sect
	var etype int
	coords := make([][]float64, 4)
	size := 0
	for i := 0; i < 4; i++ {
		coords[i] = make([]float64, 3)
	}
	check := func(c [][]float64, pos int) bool {
		for i := 0; i < 3; i++ {
			if c[pos][i] != coords[pos-1][i] {
				return true
			}
		}
		return false
	}
	for i, word := range lis {
		if i%2 != 0 {
			continue
		}
		index, err = strconv.ParseInt(word, 10, 64)
		if err != nil {
			return err
		}
		switch int(index) {
		case 8:
			etype = Etype(layeretype.FindString(lis[i+1]))
			if etype == 0 {
				return nil
			}
			tmp, err := strconv.ParseInt(layersect.FindString(lis[i+1]), 10, 64)
			if err != nil {
				return err
			}
			if val, ok := frame.Sects[int(tmp)]; ok {
				sect = val
			} else {
				sec := frame.AddSect(int(tmp))
				sect = sec
			}
		case 10, 11, 12, 13:
			coords[size][0], err = strconv.ParseFloat(lis[i+1], 64)
		case 20, 21, 22, 23:
			coords[size][1], err = strconv.ParseFloat(lis[i+1], 64)
		case 30:
			coords[0][2], err = strconv.ParseFloat(lis[i+1], 64)
			size++
		case 31, 32, 33:
			coords[size][2], err = strconv.ParseFloat(lis[i+1], 64)
			if check(coords, size) {
				size++
			}
		}
		if err != nil {
			return err
		}
	}
	enod := make([]*Node, size)
	for i := 0; i < size; i++ {
		enod[i], _ = frame.CoordNode(coords[i][0]*factor+coord[0], coords[i][1]*factor+coord[1], coords[i][2]*factor+coord[2], eps)
	}
	if addedge {
		lsect, letype := LineSect(frame, sect)
		for i := 0; i < size; i++ {
			addline := true
			var j int
			if i == size-1 {
				j = 0
			} else {
				j = i + 1
			}
			els := frame.SearchElem(enod[i], enod[j])
			for _, el := range els {
				if el.IsLineElem() {
					addline = false
					break
				}
			}
			if addline {
				frame.AddLineElem(-1, []*Node{enod[i], enod[j]}, lsect, letype)
			}
		}
	}
	frame.AddPlateElem(-1, enod, sect, etype)
	return nil
}

func (frame *Frame) ParseDxfVertex(lis []string, coord []float64, vertices []*Node, eps float64) ([]*Node, error) {
	var err error
	var index int64
	var x, y, z float64
	var etype int
	nnum := make([]int, 4)
	var size int
	var addelem bool
	var sect *Sect
	for i, word := range lis {
		if i%2 != 0 {
			continue
		}
		index, err = strconv.ParseInt(word, 10, 64)
		if err != nil {
			return vertices, err
		}
		switch int(index) {
		case 8:
			etype = Etype(layeretype.FindString(lis[i+1]))
			if etype == 0 {
				return vertices, nil
			}
			tmp, err := strconv.ParseInt(layersect.FindString(lis[i+1]), 10, 64)
			if err != nil {
				return nil, err
			}
			if val, ok := frame.Sects[int(tmp)]; ok {
				sect = val
			} else {
				sec := frame.AddSect(int(tmp))
				sect = sec
			}
		case 10:
			x, err = strconv.ParseFloat(lis[i+1], 64)
		case 20:
			y, err = strconv.ParseFloat(lis[i+1], 64)
		case 30:
			z, err = strconv.ParseFloat(lis[i+1], 64)
		case 71, 72, 73, 74:
			num, _ := strconv.ParseInt(lis[i+1], 10, 64)
			nnum[size] = int(num)
			size++
		case 100:
			if lis[i+1] == "AcDbFaceRecord" {
				addelem = true
			}
		}
		if err != nil {
			return vertices, err
		}
	}
	if addelem {
		if size < 3 {
			return vertices, nil
		}
		enod := make([]*Node, size)
		lsect, letype := LineSect(frame, sect)
		for i := 0; i < size; i++ {
			enod[i] = vertices[nnum[i]-1]
		}
		checked := make([]*Node, size)
		pos := 1
		checked[0] = enod[0]
		for i, n := range enod[1:] {
			if n.Num != enod[i].Num {
				checked[pos] = n
				pos++
			}
		}
		enod = checked[:pos]
		size = pos
		if addedge {
			for i := 0; i < size; i++ {
				addline := true
				var j int
				if i == size-1 {
					j = 0
				} else {
					j = i + 1
				}
				els := frame.SearchElem(enod[i], enod[j])
				for _, el := range els {
					if el.IsLineElem() {
						addline = false
						break
					}
				}
				if addline {
					frame.AddLineElem(-1, []*Node{enod[i], enod[j]}, lsect, letype)
				}
			}
		}
		frame.AddPlateElem(-1, enod, sect, etype)
	} else {
		n, _ := frame.CoordNode(x*factor+coord[0], y*factor+coord[1], z*factor+coord[2], eps)
		vertices = append(vertices, n)
	}
	return vertices, nil
}

func LineSect(frame *Frame, sect *Sect) (*Sect, int) {
	var lsect *Sect
	var letype int
	if sect.Num >= 700 && sect.Num < 800 {
		lsecnum := sect.Num - 200
		if sec, ok := frame.Sects[lsecnum]; ok {
			lsect = sec
		} else {
			lsect = frame.AddSect(lsecnum)
		}
		letype = GIRDER
	} else if sect.Num >= 800 && sect.Num < 900 {
		lsecnum := sect.Num - 600
		if sec, ok := frame.Sects[lsecnum]; ok {
			lsect = sec
		} else {
			lsect = frame.AddSect(lsecnum)
		}
		letype = COLUMN
	} else {
		lsecnum := sect.Num + 100
		if sec, ok := frame.Sects[lsecnum]; ok {
			lsect = sec
		} else {
			lsect = frame.AddSect(lsecnum)
		}
		letype = COLUMN
	}
	return lsect, letype
}

func (frame *Frame) ParseDxfObjects(lis []string, namehandle map[int]string, elemhandle map[int]*Elem) (map[int]string, error) {
	var err error
	if len(lis) < 2 {
		return namehandle, nil
	}
	tp := lis[1]
	switch tp {
	case "DICTIONARY":
		namehandle, err = frame.ParseDxfDictionary(lis, namehandle, elemhandle)
	case "GROUP":
		err = frame.ParseDxfGroup(lis, namehandle, elemhandle)
	}
	return namehandle, err
}

func (frame *Frame) ParseDxfDictionary(lis []string, namehandle map[int]string, elemhandle map[int]*Elem) (map[int]string, error) {
	var err error
	var index, h int64
	var name string
	for i, word := range lis {
		if i%2 != 0 {
			continue
		}
		index, err = strconv.ParseInt(word, 10, 64)
		if err != nil {
			return namehandle, err
		}
		switch int(index) {
		case 3:
			name = lis[i+1]
		case 350:
			h, err = strconv.ParseInt(lis[i+1], 16, 64)
			if err != nil {
				return namehandle, err
			}
			namehandle[int(h)] = name
		}
	}
	return namehandle, nil
}

func (frame *Frame) ParseDxfGroup(lis []string, namehandle map[int]string, elemhandle map[int]*Elem) error {
	var err error
	var index, h int64
	var bonds []*Bond
	for i, word := range lis {
		if i%2 != 0 {
			continue
		}
		index, err = strconv.ParseInt(word, 10, 64)
		if err != nil {
			return err
		}
		switch int(index) {
		case 5:
			h, err = strconv.ParseInt(lis[i+1], 16, 64)
			if err != nil {
				return err
			}
			if name, ok := namehandle[int(h)]; ok {
				switch name {
				case "PINPIN":
					bonds = []*Bond{nil, nil, nil, nil, Pin, Pin, nil, nil, nil, nil, Pin, Pin}
				case "RIGIDPIN":
					bonds = []*Bond{nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, Pin, Pin}
				case "PINRIGID":
					bonds = []*Bond{nil, nil, nil, nil, Pin, Pin, nil, nil, nil, nil, nil, nil}
				}
			} else {
				return errors.New(fmt.Sprintf("handle %X not fount", int(h)))
			}
		case 340:
			h, err = strconv.ParseInt(lis[i+1], 16, 64)
			if err != nil {
				return err
			}
			if el, ok := elemhandle[int(h)]; ok {
				for j := 0; j < 12; j++ {
					el.Bonds[j] = bonds[j]
				}
			}
		}
	}
	return nil
}
