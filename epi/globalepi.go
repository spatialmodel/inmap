package epi

import (
    "sort"
    "fmt"
    "math"
    "strings"
    "strconv"
    "runtime"
//    "sync"
    "github.com/ctessum/geom"
    "github.com/ctessum/geom/encoding/shp"
    "github.com/ctessum/geom/index/rtree"
    "github.com/ctessum/geom/proj"
)

// inmapOutput holds either population or concentration data read from
// InMAP results
type inmapOutput struct {
    geom.Polygonal

    // Data holds the number of people in each population category, or
    // the TotalPM2.5 concentration in ug/m3.
    Data []float64
}

type mortality struct {
    geom.Polygonal

    // MortData holds the mortality rate for each population category
    MortData []float64 // Deaths per 100,000 people per year

    // Io holds the underlying incidence rate for each population
    // category
    Io []float64
}

func loadPopulation(f string) (*rtree.Rtree, map[string]int, error) {
    projection, _ := proj.Parse("+proj=longlat +units=degrees")
    CensusPopColumns := []string{"TotalPop"}

    var err error
    popshp, err := shp.NewDecoder(f)
    if err != nil {
        return nil, nil, err
    }
    popsr, err := popshp.SR()
    if err != nil {
        return nil, nil, err
    }
    trans, err := popsr.NewTransform(projection)
    if err != nil {
        return nil, nil, err
    }
    // Create a list of array indices for each population type.
    popIndices := make(map[string]int)
    for i, p := range CensusPopColumns {
        popIndices[p] = i
    }

    pop := rtree.NewTree(25, 50)
    for {
        g, fields, more := popshp.DecodeRowFields(CensusPopColumns...)
        if !more {
            break
        }
        p := new(inmapOutput)
        p.Data = make([]float64, len(CensusPopColumns))
        for i, pop := range CensusPopColumns {
            s, ok := fields[pop]
            if !ok {
                return nil, nil, fmt.Errorf("inmap: loading population shapefile: missing attribute column %s", pop)
            }
            p.Data[i], err = s2f(s)
            if err != nil {
                return nil, nil, err
            }
            if math.IsNaN(p.Data[i]) {
                return nil, nil, fmt.Errorf("inmap: loadPopulation: NaN population value")
            }
        }
        gg, err := g.Transform(trans)
        if err != nil {
            return nil, nil, err
        }
        switch gg.(type) {
        case geom.Polygonal:
            p.Polygonal = gg.(geom.Polygonal)
        default:
            return nil, nil, fmt.Errorf("inmap: loadPopulation: population shapes need to be polygons")
        }
        pop.Insert(p)
    }
    if err := popshp.Error(); err != nil {
        return nil, nil, err
    }
    popshp.Close()
    return pop, popIndices, nil
}

func s2f(s string) (float64, error) {
    if strings.Contains(s, "*") { // Null value
        return 0., nil
    }
    f, err := strconv.ParseFloat(s, 64)
    return f, err
}

func loadMortality(f string) ([]*mortality, map[string]int, error) {
    projection, _ := proj.Parse("+proj=longlat +units=degrees")

    mortshp, err := shp.NewDecoder(f)
    if err != nil {
        return nil, nil, err
    }

    mortshpSR, err := mortshp.SR()
    if err != nil {
        return nil, nil, err
    }
    trans, err := mortshpSR.NewTransform(projection)
    if err != nil {
        return nil, nil, err
    }

    mortIndices := make(map[string]int)
    mortRateColumns := []string{"AllCause"}
    sort.Strings(mortRateColumns)
    for i, m := range mortRateColumns {
        mortIndices[m] = i
    }
    var mortRates []*mortality
    for {
        g, fields, more := mortshp.DecodeRowFields(mortRateColumns...)
        if !more {
            break
        }
        m := new(mortality)
        m.MortData = make([]float64, len(mortRateColumns))
        m.Io = make([]float64, len(m.MortData))
        for i, mort := range mortRateColumns {
            s, ok := fields[mort]
            if !ok {
                return nil, nil, fmt.Errorf("slca: loading mortality rate shapefile: missing attribute column %s", mort)
            }
            m.MortData[i], err = s2f(s)
            if err != nil {
                return nil, nil, err
            }
            if math.IsNaN(m.MortData[i]) {
                panic("NaN mortality rate!")
            }
        }
        gg, err := g.Transform(trans)
        if err != nil {
            return nil, nil, err
        }
        switch gg.(type) {
        case geom.Polygonal:
            m.Polygonal = gg.(geom.Polygonal)
        default:
            return nil, nil, fmt.Errorf("slca: loadMortality: mortality rate shapes need to be polygons")
        }
        mortRates = append(mortRates, m)
    }
    if err := mortshp.Error(); err != nil {
        return nil, nil, err
    }
    mortshp.Close()
    return mortRates, mortIndices, nil
}

// Load the InMAP concentration output
func loadConc(f string) (*rtree.Rtree, map[string]int, error) {
    projection, _ := proj.Parse("+proj=longlat +units=degrees")
    ConcColumns := []string{"TotalPM25"}

    var err error
    concshp, err := shp.NewDecoder(f)
    if err != nil {
        return nil, nil, err
    }
    concsr, err := concshp.SR()
    if err != nil {
        return nil, nil, err
    }
    trans, err := concsr.NewTransform(projection)
    if err != nil {
        return nil, nil, err
    }
    // Create a list of array indices for each concentration type.
    concIndices := make(map[string]int)
    for i, p := range ConcColumns {
        concIndices[p] = i
    }

    conc := rtree.NewTree(25, 50)
    for {
        g, fields, more := concshp.DecodeRowFields(ConcColumns...)
        if !more {
            break
        }
        p := new(inmapOutput)
        p.Data = make([]float64, len(ConcColumns))
        for i, conc := range ConcColumns {
            s, ok := fields[conc]
            if !ok {
                return nil, nil, fmt.Errorf("inmap: loading InMAP output shapefile: missing attribute column %s", conc)
            }
            p.Data[i], err = s2f(s)
            if err != nil {
                return nil, nil, err
            }
            if math.IsNaN(p.Data[i]) {
                return nil, nil, fmt.Errorf("inmap: loadConc: NaN concentration value")
            }
        }
        gg, err := g.Transform(trans)
        if err != nil {
            return nil, nil, err
        }
        switch gg.(type) {
        case geom.Polygonal:
            p.Polygonal = gg.(geom.Polygonal)
        default:
            return nil, nil, fmt.Errorf("inmap: loadConc: InMAP output shapes need to be polygons")
        }
        conc.Insert(p)
    }
    if err := concshp.Error(); err != nil {
        return nil, nil, err
    }
    concshp.Close()
    return conc, concIndices, nil
}

// regionalIncidence calculates region-averaged underlying incidence rates.
func regionalIncidence(popIndex *rtree.Rtree, popIndices map[string]int, mort []*mortality, mortIndices map[string]int, concIndex *rtree.Rtree, concIndices map[string]int) (*rtree.Rtree, error) {
    ncpu := runtime.GOMAXPROCS(0)
    HR := GlobalGEMM

        mi, ok := mortIndices["AllCause"]
        if !ok {
            panic(fmt.Errorf("missing mortality type AllCause"))
        }
        pi, ok := popIndices["TotalPop"]
        if !ok {
            panic(fmt.Errorf("missing population type TotalPop"))
        }
//        var wg sync.WaitGroup
//        wg.Add(ncpu)
        for p := 0; p < ncpu; p++ {
//            go func(p, mi, pi int) {
                for i := p; i < len(mort); i += ncpu {
                    m := mort[i]
                    regionPopIsect := popIndex.SearchIntersect(m.Bounds())
                    regionConcIsect := concIndex.SearchIntersect(m.Bounds())
                    regionPop := make([]float64, len(regionPopIsect))
                    regionConc := make([]float64, len(regionConcIsect))
                    for i, pI := range regionPopIsect {
                        pp := pI.(*inmapOutput)
                        pArea := pp.Area()
                        isectFrac := pp.Intersection(m).Area() / pArea
                        if pArea == 0 || isectFrac == 0 {
                            continue
                        }
                        regionPop[i] = pp.Data[pi] * isectFrac

                        for _, gI := range regionConcIsect {
                            cc := gI.(*inmapOutput)
                            regionConc[i] += cc.Data[pi] * cc.Intersection(m).Area() / pArea
                        }
                    }
                    m.Io[mi] = IoRegional(regionPop, regionConc, HR, m.MortData[mi])
                }
//                wg.Done()
//            }(p, mi, pi)
//        wg.Wait()
    }
    o := rtree.NewTree(25, 50)
    for _, m := range mort {
        o.Insert(m)
    }
    return o, nil
}
