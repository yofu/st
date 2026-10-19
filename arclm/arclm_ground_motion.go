package arclm

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// GroundMotionSample uses seconds and m/s^2. ReadGroundMotionCSV converts Gal
// (cm/s^2) to m/s^2 once, at the input boundary.
type GroundMotionSample struct{ Time, X, Y float64 }
type GroundMotion []GroundMotionSample

func ReadGroundMotionCSV(r io.Reader) (GroundMotion, error) {
	c := csv.NewReader(r)
	c.Comment, c.FieldsPerRecord, c.TrimLeadingSpace = '#', -1, true
	var motion GroundMotion
	for record := 1; ; record++ {
		row, err := c.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("ground motion record %d: %w", record, err)
		}
		if len(row) != 3 {
			return nil, fmt.Errorf("ground motion record %d: expected time,X,Y", record)
		}
		row[0] = strings.TrimPrefix(strings.TrimSpace(row[0]), "\ufeff")
		if record == 1 {
			header := strings.ToLower(row[0])
			if header == "t" || strings.HasPrefix(header, "time") || strings.HasPrefix(header, "時刻") {
				continue
			}
		}
		var values [3]float64
		for i := range values {
			values[i], err = strconv.ParseFloat(strings.TrimSpace(row[i]), 64)
			if err != nil || !finitePlastic(values[i]) {
				return nil, fmt.Errorf("ground motion record %d column %d: invalid number", record, i+1)
			}
		}
		motion = append(motion, GroundMotionSample{values[0], values[1] / 100, values[2] / 100})
	}
	return motion, motion.Validate()
}

func (m GroundMotion) Validate() error {
	if len(m) < 2 || m[0].Time != 0 {
		return fmt.Errorf("ground motion requires at least two samples and must start at time 0")
	}
	for i, p := range m {
		if !finitePlastic(p.Time) || !finitePlastic(p.X) || !finitePlastic(p.Y) || i > 0 && p.Time <= m[i-1].Time {
			return fmt.Errorf("invalid ground motion sample %d: times must increase strictly", i+1)
		}
	}
	return nil
}

// At interpolates linearly, without extrapolation or wrapping.
func (m GroundMotion) At(t float64) (GroundMotionSample, error) {
	if len(m) < 2 || !finitePlastic(t) || t < 0 || t > m[len(m)-1].Time {
		return GroundMotionSample{}, fmt.Errorf("ground motion time outside record: %g", t)
	}
	i := sort.Search(len(m), func(i int) bool { return m[i].Time >= t })
	if i == 0 || m[i].Time == t {
		return m[i], nil
	}
	a, b := m[i-1], m[i]
	s := (t - a.Time) / (b.Time - a.Time)
	return GroundMotionSample{t, a.X + s*(b.X-a.X), a.Y + s*(b.Y-a.Y)}, nil
}
