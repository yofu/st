package st

import (
    "bytes"
    "fmt"
)

type Prop struct {
    Num  int
    Name string
    Hiju float64
    E    float64
    Poi  float64
    Color int
}

func (prop *Prop) InpString () string {
    var rtn bytes.Buffer
    rtn.WriteString(fmt.Sprintf("PROP %d PNAME %s\n", prop.Num, prop.Name))
    rtn.WriteString(fmt.Sprintf("         HIJU %13.5f\n", prop.Hiju))
    rtn.WriteString(fmt.Sprintf("         E    %13.3f\n", prop.E))
    rtn.WriteString(fmt.Sprintf("         POI  %13.5f\n", prop.Poi))
    rtn.WriteString(fmt.Sprintf("         PCOLOR %s\n", IntColor(prop.Color)))
    return rtn.String()
}
