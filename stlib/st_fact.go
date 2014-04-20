package st

import (
    "bytes"
    "errors"
    "fmt"
    "math"
    "os"
)

type Fact struct {
    Calced bool
    Floor int
    Abs   bool
    Factor float64
    IgnoreConf bool
    Input  []string
    Output []string
    AverageLevel   []float64
    AverageDisp    [][]float64
    AverageDrift   [][]float64
    MaxDisp        [][]float64
    MaxDrift       [][]float64
    MaxDriftElem   [][]int
    TotalShear     [][]float64
    TotalMoment    [][]float64
    CentreOfWeight [][]float64
    CentreOfRigid  [][]float64
    SpringRadius   [][]float64
    Rigidity       [][]float64
    Eccentricity   [][]float64
}

func NewFact (n int, abs bool, factor float64) *Fact {
    f := new(Fact)
    f.Floor = n
    f.Abs   = abs
    f.IgnoreConf = true
    f.Factor = factor
    f.Input  = make([]string, 3)
    f.Output = make([]string, 3)
    f.AverageLevel   = make([]float64, n)
    f.AverageDisp    = make([][]float64, n)
    f.AverageDrift   = make([][]float64, n-1)
    f.MaxDisp        = make([][]float64, n)
    f.MaxDrift       = make([][]float64, n-1)
    f.MaxDriftElem   = make([][]int, n-1)
    f.TotalShear     = make([][]float64, n)
    f.TotalMoment    = make([][]float64, n)
    f.CentreOfWeight = make([][]float64, n)
    f.CentreOfRigid  = make([][]float64, n-1)
    f.SpringRadius   = make([][]float64, n-1)
    f.Rigidity       = make([][]float64, n-1)
    f.Eccentricity   = make([][]float64, n-1)
    return f
}

func (f *Fact) String () string {
    if f.Calced {
        var otp bytes.Buffer
        var m [][]string
        otp.WriteString("FACTS:\n")
        otp.WriteString("剛性率 Rs：○印が最小\n")
        m = maru(f.Rigidity, false)
        for i:=0; i<f.Floor-1; i++ {
            otp.WriteString(fmt.Sprintf("%2dF： %sRx=%7.5f %sRy=%7.5f\n", i+1, m[i][0], f.Rigidity[i][0], m[i][1], f.Rigidity[i][1]))
        }
        otp.WriteString("偏心率 Re：○印が最大\n")
        m = maru(f.Eccentricity, true)
        for i:=0; i<f.Floor-1; i++ {
            otp.WriteString(fmt.Sprintf("%2dF： %sRex=%8.5f %sRey=%8.5f\n", i+1, m[i][0], f.Eccentricity[i][0], m[i][1], f.Eccentricity[i][1]))
        }
        return otp.String()
    } else {
        return ""
    }
}

func maru (lis [][]float64, max bool) [][]string {
    l := len(lis)
    switch l {
    case 0:
        return nil
    case 1:
        return [][]string{[]string{"○", "○"}}
    default:
        rtn := make([][]string, l)
        for i:=0; i<l; i++ {
            rtn[i] = make([]string, 2)
        }
        tmpval := make([]float64, 2)
        ind := make([]int, 2)
        tmpval[0] = lis[0][0]; tmpval[1] = lis[0][1]
        for j:=0; j<2; j++ {
            for i:=1; i<l; i++ {
                if max {
                    if lis[i][j] > tmpval[j] {
                        tmpval[j] = lis[i][j]
                        ind[j] = i
                    }
                } else {
                    if lis[i][j] < tmpval[j] {
                        tmpval[j] = lis[i][j]
                        ind[j] = i
                    }
                }
            }
        }
        for j:=0; j<2; j++ {
            for i:=0; i<l; i++ {
                if i==ind[j] {
                    rtn[i][j] = "○"
                } else {
                    rtn[i][j] = "　"
                }
            }
        }
        return rtn
    }
}

func (f *Fact) WriteTo (fn string) error {
    var otp bytes.Buffer
    var m [][]string
    otp.WriteString("5.2：層間変形角, 剛性率, 偏心率\n\n")
    otp.WriteString("各階、両方向とも、C0=0.2相当で\n")
    otp.WriteString("  層間変形角  1/120以下\n")
    otp.WriteString("  剛性率      0.600以上\n")
    otp.WriteString("  偏心率      0.150以下\n")
    otp.WriteString("を満たしている。\n\n")
    otp.WriteString(fmt.Sprintf("入力データファイル：%s\n", f.Input[0]))
    otp.WriteString(fmt.Sprintf("                    %s\n", f.Input[1]))
    otp.WriteString(fmt.Sprintf("                    %s\n", f.Input[2]))
    otp.WriteString(fmt.Sprintf("出力データファイル：%s\n", f.Output[0]))
    otp.WriteString(fmt.Sprintf("                    %s\n", f.Output[1]))
    otp.WriteString(fmt.Sprintf("                    %s\n\n", f.Output[2]))
    otp.WriteString("それぞれ添字x, yは座標軸を表す。\n\n")
    otp.WriteString("各階平均変位 D\n")
    for i:=0; i<f.Floor; i++ {
        otp.WriteString(fmt.Sprintf("%2dFL： LEVEL= %8.5f[m] Dx= %8.5f[m] Dy= %8.5f[m]\n", i+1, f.AverageLevel[i], f.AverageDisp[i][0], f.AverageDisp[i][1]))
    }
    otp.WriteString("\n各階最大変位 Dmax\n")
    for i:=0; i<f.Floor; i++ {
        otp.WriteString(fmt.Sprintf("%2dFL： Dxmax= %8.5f[m] Dymax= %8.5f[m]\n", i+1, f.MaxDisp[i][0], f.MaxDisp[i][1]))
    }
    otp.WriteString("\n層間変形角（平均変位） D/H：○印が最大\n")
    m = maru(f.AverageDrift, true)
    for i:=0; i<f.Floor-1; i++ {
        otp.WriteString(fmt.Sprintf("%2dF： %sDx/H=1/%4d  %sDy/H=1/%4d\n", i+1, m[i][0], int(1/f.AverageDrift[i][0]), m[i][1], int(1/f.AverageDrift[i][1])))
        if f.Factor != 1.0 { otp.WriteString(fmt.Sprintf("      (C0=0.2: 1/%4d) (C0=0.2:1/%4d)\n", int(f.Factor/f.AverageDrift[i][0]), int(f.Factor/f.AverageDrift[i][1]))) }
    }
    otp.WriteString("\n層間変形角（最大値） D/H：○印が最大\n")
    m = maru(f.MaxDrift, true)
    for i:=0; i<f.Floor-1; i++ {
        otp.WriteString(fmt.Sprintf("%2dF： %sDx/H=1/%4d(部材番号：%5d)  %sDy/H=1/%4d(部材番号：%5d)\n", i+1, m[i][0], int(1/f.MaxDrift[i][0]), f.MaxDriftElem[i][0], m[i][1], int(1/f.MaxDrift[i][1]), f.MaxDriftElem[i][1]))
        if f.Factor != 1.0 { otp.WriteString(fmt.Sprintf("      (C0=0.2: 1/%4d) (C0=0.2:1/%4d)\n", int(f.Factor/f.MaxDrift[i][0]), int(f.Factor/f.MaxDrift[i][1]))) }
    }
    otp.WriteString("\n剛性率 Rs：○印が最小\n")
    m = maru(f.Rigidity, false)
    for i:=0; i<f.Floor-1; i++ {
        otp.WriteString(fmt.Sprintf("%2dF： %sRx=%7.5f %sRy=%7.5f\n", i+1, m[i][0], f.Rigidity[i][0], m[i][1], f.Rigidity[i][1]))
    }
    otp.WriteString("\n層せん断力 Q、層せん断力の重心位置 G\n")
    for i:=1; i<f.Floor; i++ {
        otp.WriteString(fmt.Sprintf("%2dF： Qx=%10.5f[tf] Qy=%10.5f[tf] Gx=%8.5f[m] Gy=%8.5f[m]\n", i, f.TotalShear[i][0], f.TotalShear[i][1], f.CentreOfWeight[i][0], f.CentreOfWeight[i][1]))
    }
    otp.WriteString("\n剛心位置 L、偏心距離 e\n")
    for i:=0; i<f.Floor-1; i++ {
        otp.WriteString(fmt.Sprintf("%2dF： Lx=%8.5f[m] Ly=%8.5f[m] ex=%8.5f[m] ey=%8.5f[m]\n", i+1, f.CentreOfRigid[i][0], f.CentreOfRigid[i][1], f.CentreOfWeight[i+1][0]-f.CentreOfRigid[i][0], f.CentreOfWeight[i+1][1]-f.CentreOfRigid[i][1]))
    }
    otp.WriteString("\n偏心率 Re：○印が最大\n")
    m = maru(f.Eccentricity, true)
    for i:=0; i<f.Floor-1; i++ {
        otp.WriteString(fmt.Sprintf("%2dF： %sRex=%8.5f %sRey=%8.5f\n", i+1, m[i][0], f.Eccentricity[i][0], m[i][1], f.Eccentricity[i][1]))
    }
    w, err := os.Create(fn)
    defer w.Close()
    if err != nil {
        return err
    }
    otp.WriteTo(w)
    return nil
}

func (f *Fact) SetFileName (inp []string, otp []string) {
    for i:=0; i<3; i++ {
        f.Input[i] = inp[i]
        f.Output[i] = otp[i]
    }
}

func (f *Fact) CalcFact (nodes [][]*Node, elems [][]*Elem) error {
    if len(nodes) < f.Floor || len(elems) < f.Floor-1 { return errors.New("CalcFact: Not Enough Data") }
    for i:=0; i<f.Floor; i++ {
        av_level := 0.0
        tmpdisp := make([]float64, 2)
        disp := make([]float64, 2)
        var drift []float64
        if i>0 { drift = make([]float64, 2) }
        shear := make([]float64, 2)
        moment := make([]float64, 2)
        num := 0; conf := make([]int, 2)
        for _, n := range nodes[i] {
            num++
            av_level += n.Coord[2]
            for j, d := range []string{"X", "Y"} {
                if f.IgnoreConf && n.Conf[j] { conf[j]++; continue }
                if n.Disp[d][j] > tmpdisp[j] {
                    tmpdisp[j] = n.Disp[d][j]
                }
                disp[j] += n.Disp[d][j]
                shear[j] += n.Force[d][j]
                moment[j] += n.Force[d][j] * n.Coord[1-j]
            }
        }
        f.MaxDisp[i] = tmpdisp
        f.AverageLevel[i] = av_level/float64(num)
        for j:=0; j<2; j++ {
            disp[j] /= float64(num-conf[j])
            if i>0 { drift[j] = (disp[j]-f.AverageDisp[i-1][j]) / (f.AverageLevel[i]-f.AverageLevel[i-1]) }
        }
        f.AverageDisp[i] = disp
        if i>0 { f.AverageDrift[i-1] = drift }
        f.TotalShear[i]  = shear
        f.TotalMoment[i] = moment
        for h:=0; h<i; h++ {
            for j:=0; j<2; j++ {
                f.TotalShear[h][j] += shear[j]
                f.TotalMoment[h][j] += moment[j]
            }
        }
    }
    for i:=0; i<f.Floor; i++ {
        f.CentreOfWeight[i] = make([]float64, 2)
        for j:=0; j<2; j++ {
            f.CentreOfWeight[i][1-j] = f.TotalMoment[i][j]/f.TotalShear[i][j]
        }
    }
    TotalStiffness := make([][]float64, f.Floor-1)
    TotalStiffCoord := make([][]float64, f.Floor-1)
    TotalStiffCoord2 := make([][]float64, f.Floor-1)
    for i:=0; i<f.Floor-1; i++ {
        tk := make([]float64, 2)
        tkx := make([]float64, 2)
        tkxx := make([]float64, 2)
        tmpmaxdrift := make([]float64, 2)
        maxdriftelem := make([]int, 2)
        for _, el := range elems[i] {
            for j, per := range []string{"X", "Y"} {
                drift := el.StoryDrift(per)
                if drift > tmpmaxdrift[j] {
                    tmpmaxdrift[j] = drift
                    maxdriftelem[j] = el.Num
                }
                coord := el.MidPoint()[1-j]
                stiff := el.LateralStiffness(per, f.Abs)
                tk[j] += stiff
                tkx[j] += stiff*coord
                tkxx[j] += stiff*coord*coord
            }
        }
        f.MaxDrift[i] = tmpmaxdrift
        f.MaxDriftElem[i] = maxdriftelem
        TotalStiffness[i] = tk
        TotalStiffCoord[i] = tkx
        TotalStiffCoord2[i] = tkxx
    }
    for i:=0; i<f.Floor-1; i++ {
        f.SpringRadius[i] = make([]float64, 2)
        f.CentreOfRigid[i] = make([]float64, 2)
        num := 0.0
        for j:=0; j<2; j++ {
            val := TotalStiffCoord[i][j] / TotalStiffness[i][j]
            f.CentreOfRigid[i][1-j] = val
            num += TotalStiffCoord2[i][j] - 2.0*val*TotalStiffCoord[i][j] + val*val*TotalStiffness[i][j]
        }
        for j:=0; j<2; j++ {
            f.SpringRadius[i][1-j] = math.Sqrt(num / TotalStiffness[i][j])
        }
    }
    for i:=0; i<f.Floor-1; i++ {
        f.Rigidity[i] = make([]float64, 2)
        f.Eccentricity[i] = make([]float64, 2)
    }
    for j:=0; j<2; j++ {
        sum := 0.0
        for i:=0; i<f.Floor-1; i++ {
            sum += 1/f.AverageDrift[i][j]
        }
        sum /= float64(f.Floor-1)
        for i:=0; i<f.Floor-1; i++ {
            f.Rigidity[i][j] = (1/f.AverageDrift[i][j])/sum
            f.Eccentricity[i][1-j] = math.Abs((f.CentreOfWeight[i+1][j] - f.CentreOfRigid[i][j]) / f.SpringRadius[i][j])
        }
    }
    f.Calced = true
    return nil
}
