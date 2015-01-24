package st

import (
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	factor = 0.001
)

var (
	entstart   = regexp.MustCompile("^ *0 *$")
	layeretype = regexp.MustCompile("[a-zA-Z]+")
	layersect  = regexp.MustCompile("[0-9]+")
)

func (frame *Frame) ReadDxf(filename string, coord []float64, eps float64) (err error) {
	var parse = false
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
			if ok, err := regexp.MatchString("^ *ENTITIES *$", lis[ind+2]); ok {
				parse = true
			} else if err != nil {
				return err
			} else {
				if parse {
					break
				}
			}
		}
		if parse {
			if ind%2 == 1 {
				tmp = append(tmp, strings.Join(words, " "))
			} else {
				switch {
				default:
					tmp = append(tmp, strings.Join(words, " "))
				case entstart.MatchString(first):
					vertices, err = frame.ParseDxf(tmp, coord, vertices, eps)
					if err != nil {
						return err
					}
					tmp = []string{strings.Join(words, " ")}
				}
			}
		}
	}
	vertices, err = frame.ParseDxf(tmp, coord, vertices, eps)
	if err != nil {
		return err
	}
	frame.Name = filepath.Base(filename)
	return nil
}

func (frame *Frame) ParseDxf(lis []string, coord []float64, vertices []*Node, eps float64) ([]*Node, error) {
	var err error
	if len(lis) < 2 {
		return nil, nil
	}
	tp := lis[1]
	switch tp {
	case "LINE":
		err = frame.ParseDxfLine(lis, coord, eps)
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
	}
	return vertices, err
}

func (frame *Frame) ParseDxfLine(lis []string, coord []float64, eps float64) error {
	var err error
	var index int64
	var sect *Sect
	var etype int
	var startx, starty, startz, endx, endy, endz float64
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
			return err
		}
	}
	n1, _ := frame.CoordNode(startx*factor+coord[0], starty*factor+coord[1], startz*factor+coord[2], eps)
	n2, _ := frame.CoordNode(endx*factor+coord[0], endy*factor+coord[1], endz*factor+coord[2], eps)
	frame.AddLineElem(-1, []*Node{n1, n2}, sect, etype)
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
			switch strings.ToUpper(lis[i+1]) {
			default:
				conf = []bool{false, false, false, false, false, false}
			case "FIX":
				conf = []bool{true, true, true, true, true, true}
			case "PIN":
				conf = []bool{true, true, true, false, false, false}
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
	// var etype int
	coords := make([][]float64, 4)
	size := 0
	for i := 0; i < 4; i++ {
		coords[i] = make([]float64, 3)
	}
	check := func(c [][]float64, pos int) bool {
		for i:=0; i<3; i++ {
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
	if sec, ok := frame.Sects[201]; ok {
		sect = sec
	} else {
		sect = frame.AddSect(201)
	}
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
			frame.AddLineElem(-1, []*Node{enod[i], enod[j]}, sect, COLUMN)
		}
	}
	if sec, ok := frame.Sects[801]; ok {
		sect = sec
	} else {
		sect = frame.AddSect(801)
	}
	frame.AddPlateElem(-1, enod, sect, WALL)
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
		enod := make([]*Node, size)
		for i := 0; i < size; i++ {
			enod[i] = vertices[nnum[i]-1]
		}
		// if sec, ok := frame.Sects[201]; ok {
		//     sect = sec
		// } else {
		//     sect = frame.AddSect(201)
		// }
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
				frame.AddLineElem(-1, []*Node{enod[i], enod[j]}, sect, etype)
			}
		}
		wsect := sect.Num + 300
		if sec, ok := frame.Sects[wsect]; ok {
			sect = sec
		} else {
			sect = frame.AddSect(wsect)
		}
		frame.AddPlateElem(-1, enod, sect, WALL)
	} else {
		n, _ := frame.CoordNode(x*factor+coord[0], y*factor+coord[1], z*factor+coord[2], eps)
		vertices = append(vertices, n)
	}
	return vertices, nil
}
