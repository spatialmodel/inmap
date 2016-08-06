package carto

import (
	"fmt"
	"image/color"
	"io"
	"math"
	"sort"
	"strings"

	"github.com/gonum/plot/vg"
	"github.com/gonum/plot/vg/draw"
	//"github.com/gonum/plot/vg/vgimg"
	//"github.com/gonum/plot/vg/vgsvg"
	"github.com/gonum/plot/vg/vgpdf"
)

type Colorlist struct {
	Val, R, G, B        []float64
	HighLimit, LowLimit color.NRGBA
}

var (
	// optimized olors from http://www.cs.unm.edu/~kmorel/documents/ColorMaps/index.html
	// Originally the 255 values were 221's
	Optimized Colorlist = Colorlist{
		[]float64{-1., -0.9375, -0.875, -0.8125,
			-0.75, -0.6875, -0.625, -0.5625, -0.5, -0.4375, -0.375,
			-0.3125, -0.25, -0.1875, -0.125, -0.0625, 0., 0.0625, 0.125,
			0.1875, 0.25, 0.3125, 0.375, 0.4375, 0.5, 0.5625, 0.625,
			0.6875, 0.75, 0.8125, 0.875, 0.9375, 1.},
		[]float64{59., 68, 77, 87, 98, 108, 119, 130, 141, 152,
			163, 174, 184, 194, 204, 213, 255, 229, 236, 241, 245, 247,
			247, 247, 244, 241, 236, 229, 222, 213, 203, 192, 180},
		[]float64{76., 90, 104, 117, 130, 142, 154, 165, 176,
			185, 194, 201, 208, 213, 217, 219, 255, 216, 211, 204, 196, 187,
			177, 166, 154, 141, 127, 112, 96, 80, 62, 40, 4},
		[]float64{192., 204, 215, 225, 234, 241, 247, 251, 254,
			255, 255, 253, 249, 244, 238, 230, 255, 209, 197, 185, 173, 160,
			148, 135, 123, 111, 99, 88, 77, 66, 56, 47, 38},
		color.NRGBA{70., 6, 16, 255},
		color.NRGBA{27., 34, 85, 255}}
	OptimizedGrey Colorlist = Colorlist{
		[]float64{-1., -0.9375, -0.875, -0.8125,
			-0.75, -0.6875, -0.625, -0.5625, -0.5, -0.4375, -0.375,
			-0.3125, -0.25, -0.1875, -0.125, -0.0625, 0., 0.0625, 0.125,
			0.1875, 0.25, 0.3125, 0.375, 0.4375, 0.5, 0.5625, 0.625,
			0.6875, 0.75, 0.8125, 0.875, 0.9375, 1.},
		[]float64{59., 68, 77, 87, 98, 108, 119, 130, 141, 152, 163, 174,
			184, 194, 204, 213, 221, 229, 236, 241, 245, 247, 247, 247, 244, 241,
			236, 229, 222, 213, 203, 192, 180},
		[]float64{76., 90, 104, 117, 130, 142, 154, 165, 176, 185, 194,
			201, 208, 213, 217, 219, 221, 216, 211, 204, 196, 187, 177, 166,
			154, 141, 127, 112, 96, 80, 62, 40, 4},
		[]float64{192., 204, 215, 225, 234, 241, 247, 251, 254, 255,
			255, 253, 249, 244, 238, 230, 221, 209, 197, 185, 173, 160, 148,
			135, 123, 111, 99, 88, 77, 66, 56, 47, 38},
		color.NRGBA{70., 6, 16, 255},
		color.NRGBA{27., 34, 85, 255}}
	Jet Colorlist = Colorlist{
		[]float64{-1, -0.866666666666667, -0.733333333333333, -0.6,
			-0.466666666666667, -0.333333333333333, -0.2, -0.0666666666666668,
			0.0666666666666665, 0.2, 0.333333333333333, 0.466666666666666, 0.6,
			0.733333333333333, 0.866666666666666, 1},
		[]float64{0, 0, 0, 0, 0, 0, 66, 132, 189, 255, 255, 255,
			255, 255, 189, 132},
		[]float64{0, 0, 66, 132, 189, 255, 255, 255, 255, 255, 189,
			132, 66, 0, 0, 0},
		[]float64{189, 255, 255, 255, 255, 255, 189, 132, 66, 0, 0,
			0, 0, 0, 0, 0},
		color.NRGBA{249., 15, 225, 255},
		color.NRGBA{154., 0, 171, 255}}
	JetPosOnly Colorlist = Colorlist{
		[]float64{-1, 0, 0.0666666666666667, 0.133333333333333, 0.2,
			0.266666666666667, 0.333333333333333, 0.4, 0.466666666666667,
			0.533333333333333, 0.6, 0.666666666666667, 0.733333333333333, 0.8,
			0.866666666666667, 0.933333333333333, 1},
		[]float64{0, 0, 0, 0, 0, 0, 0, 66, 132, 189, 255, 255, 255,
			255, 255, 189, 132},
		[]float64{0, 0, 0, 66, 132, 189, 255, 255, 255, 255, 255, 189,
			132, 66, 0, 0, 0},
		[]float64{189, 189, 255, 255, 255, 255, 255, 189, 132, 66, 0,
			0, 0, 0, 0, 0, 0},
		color.NRGBA{249., 15, 225, 255},
		color.NRGBA{154., 0, 171, 255}}
)

type ColorMapType int

const (
	Linear    ColorMapType = iota // Linear color gradient
	LinCutoff                     // linear with a discontinuity at a percentile
	// specified by "CutPercentile"
)

type ColorMap struct {
	draw.Canvas
	maxval            float64
	minval            float64
	cutptlist         []float64
	Type              ColorMapType
	CutPercentile     float64 // Percentile at which discontinuity occurs for "LinCutoff" type.
	NumDivisions      int     // "Number of tick marks on legend.
	rulestring        string
	colorstops        []float64
	ticklocs          []float64
	stopcolors        []color.NRGBA
	LegendWidth       vg.Length // width of legend in inches
	LegendHeight      vg.Length // height of legend in inches
	LineWidth         vg.Length // width of lines in legend in points
	FontSize          vg.Length // font size in points.
	GradientLineWidth vg.Length // Width of lines that make up legend gradient
	Font              string    // Name of the font to use in legend
	FontColor         color.Color
	EdgeColor         color.Color // Color for legend outline
	BackgroundColor   color.Color
	negativeOutlier   bool
	positiveOutlier   bool
	ColorScheme       Colorlist
}

// Initialize new color map.
func NewColorMap(Type ColorMapType) (c *ColorMap) {
	c = new(ColorMap)
	c.cutptlist = make([]float64, 0)
	c.Type = Type
	c.CutPercentile = 99.
	c.ColorScheme = Optimized
	c.NumDivisions = 9
	c.colorstops = make([]float64, 0)
	c.stopcolors = make([]color.NRGBA, 0)
	c.LineWidth = 0.5         // points
	c.GradientLineWidth = 0.2 // points
	c.FontSize = 7.           // points
	c.negativeOutlier = false
	c.positiveOutlier = false
	c.Font = "Helvetica"
	c.FontColor = color.NRGBA{0, 0, 0, 255}
	c.EdgeColor = color.NRGBA{0, 0, 0, 255}
	c.BackgroundColor = color.NRGBA{255, 255, 255, 255}
	return
}

func (c *ColorMap) AddArray(data []float64) {
	var maxval, minval float64
	for i := 0; i < len(data); i++ {
		if data[i] > maxval {
			maxval = data[i]
		}
		if data[i] < minval {
			minval = data[i]
		}
	}
	if maxval*1.00001 > c.maxval {
		c.maxval = maxval * 1.00001
	}
	if minval*1.00001 < c.minval {
		c.minval = minval * 1.00001
	}
	if c.Type == LinCutoff {
		tmpAbs := make([]float64, len(data))
		for i := 0; i < len(data); i++ {
			tmpAbs[i] = math.Abs(data[i])
		}
		sort.Float64s(tmpAbs)
		cutpt := tmpAbs[roundInt(c.CutPercentile/100.*
			float64(len(data)))-1]
		c.cutptlist = append(c.cutptlist, cutpt)
	}
}

func (c *ColorMap) AddGeoJSON(g *GeoJSON, propertyName string) {
	vals := make([]float64, len(g.Features))
	for i, f := range g.Features {
		vals[i] = f.Properties[propertyName]
	}
	c.AddArray(vals)
}

func (c *ColorMap) AddMap(data map[string]float64) {
	vals := make([]float64, len(data))
	i := 0
	for _, val := range data {
		vals[i] = val
		i++
	}
	c.AddArray(vals)
}

func (c *ColorMap) AddArrayServer(datachan chan []float64,
	finished chan int) {
	var data []float64
	for {
		data = <-datachan
		if data == nil {
			break
		}
		c.AddArray(data)
	}
	finished <- 0
}

func (c *ColorMap) getColorOnLegend(gradLoc, barLeft,
	barRight vg.Length) color.Color {
	for i := 0; i < len(c.ticklocs)-1; i++ {
		tlspot := barLeft + (barRight-barLeft)*vg.Length(c.ticklocs[i])
		nexttlspot := barLeft + (barRight-barLeft)*vg.Length(c.ticklocs[i+1])
		if gradLoc >= tlspot && gradLoc < nexttlspot {
			val := c.colorstops[i] + (c.colorstops[i+1]-c.colorstops[i])*
				float64((gradLoc-tlspot)/(nexttlspot-tlspot))
			return c.GetColor(val)
		}
	}
	fmt.Println("gradLoc: ", gradLoc)
	fmt.Println("barLeft: ", barLeft)
	fmt.Println("barRight: ", barRight)
	fmt.Println("ticklocs: ", c.ticklocs)
	panic("Problem getting color")
}

// get color for input value. Must run c.Set() first.
func (cm *ColorMap) GetColor(v float64) color.NRGBA {
	var R, G, B uint8
	c := cm.stopcolors
	cv := cm.colorstops
	if math.IsNaN(v) || math.IsInf(v, 0) {
		fmt.Printf("Problem interpolating: %v value\n", v)
		return color.NRGBA{255, 0, 174, 255}
	}
	if len(cm.colorstops) == 0 {
		return color.NRGBA{255, 255, 255, 255}
	}
	if cv[0] > v {
		fmt.Println("x=", v, "xArray=", cm.colorstops)
		panic("Problem interpolating: x value is smaller than xArray")
	}
	if cv[len(cv)-1] < v {
		fmt.Println("x=", v, "xArray=", cm.colorstops)
		panic("Problem interpolating: x value is larger than xArray")
	}
	for i := 1; i < len(cv); i++ {
		if math.Abs(v-cv[i])/math.Abs(cv[i]) < 0.0001 {
			return c[i]
		} else if cv[i] > v {
			valFrac := (v - cv[i-1]) / (cv[i] - cv[i-1])
			R = round(float64(c[i-1].R) + (float64(c[i].R)-float64(c[i-1].R))*
				valFrac)
			G = round(float64(c[i-1].G) + (float64(c[i].G)-float64(c[i-1].G))*
				valFrac)
			B = round(float64(c[i-1].B) + (float64(c[i].B)-float64(c[i-1].B))*
				valFrac)
			return color.NRGBA{R, G, B, 255}
		}
	}
	fmt.Println("x=", v, "xArray=", cm.colorstops)
	panic("Problem interpolating.")
}

// Given an array of x values and an array of y values, find the y value at a
// given x using linear interpolation. xArray must be monotonically increasing.
func (cl *Colorlist) interpolate(v float64) color.NRGBA {
	var R, G, B uint8
	for i, val := range cl.Val {
		if math.Abs(v-val)/math.Abs(val) < 0.0001 {
			R = round(cl.R[i])
			G = round(cl.G[i])
			B = round(cl.B[i])
			return color.NRGBA{R, G, B, 255}
		} else if val > v {
			R = round(cl.R[i-1] + (cl.R[i]-cl.R[i-1])*
				(v-cl.Val[i-1])/(cl.Val[i]-cl.Val[i-1]))
			G = round(cl.G[i-1] + (cl.G[i]-cl.G[i-1])*
				(v-cl.Val[i-1])/(cl.Val[i]-cl.Val[i-1]))
			B = round(cl.B[i-1] + (cl.B[i]-cl.B[i-1])*
				(v-cl.Val[i-1])/(cl.Val[i]-cl.Val[i-1]))
			return color.NRGBA{R, G, B, 255}
		}
	}
	fmt.Println("x=", v, "xArray=", cl.Val)
	panic("Problem interpolating: x value is larger than xArray")
}

// round float to an integer
func round(x float64) uint8 {
	return uint8(x + 0.5)
}

// round float to an integer
func roundInt(x float64) int {
	return int(x + 0.5)
}

// Figure out rules for color map
func (c *ColorMap) Set() {
	var linmin, linmax, absmax float64
	cutpt := average(c.cutptlist)
	if c.minval*-1 > c.maxval {
		absmax = c.minval * -1
	} else {
		absmax = c.maxval
	}
	if absmax == 0. {
		return
	}
	if c.Type == LinCutoff && cutpt < absmax && cutpt != 0 {
		if cutpt*-1 > c.minval {
			linmin = cutpt * -1
		} else {
			linmin = c.minval
		}
		if cutpt < c.maxval {
			linmax = cutpt
		} else {
			linmax = c.maxval
		}
	} else {
		linmin = absmax * -1
		linmax = absmax
	}
	if linmax < linmin {
		panic("illegal range")
	}

	c.colorstops = make([]float64, 0)
	c.stopcolors = make([]color.NRGBA, 0)
	linabsmax := max(linmax, linmin*-1)

	if c.Type == LinCutoff && cutpt*-1 > c.minval && cutpt != 0 {
		c.colorstops = append(c.colorstops, absmax*-1)
		c.stopcolors = append(c.stopcolors, c.ColorScheme.LowLimit)
		c.negativeOutlier = true
	} else {
		c.colorstops = append(c.colorstops, c.minval)
		c.stopcolors = append(c.stopcolors,
			c.ColorScheme.interpolate(c.minval/linabsmax))
	}

	tens := math.Pow10(int(math.Floor(math.Log10(linmax - linmin))))
	n := (linmax - linmin) / tens
	for n < float64(c.NumDivisions) {
		tens /= 10
		n = (linmax - linmin) / tens
	}
	majorMult := int(n / float64(c.NumDivisions))
	switch majorMult {
	case 7:
		majorMult = 6
	case 9:
		majorMult = 8
	}
	majorDelta := float64(majorMult) * tens
	val := math.Floor(linmin/majorDelta) * majorDelta
	//tickThreshold := 0.02 // minimum distance for ticks
	for val <= linmax {
		if val >= linmin && val >= c.minval && val <= linmax &&
			val <= c.maxval {
			//math.Abs(val-c.minval) > tickThreshold*linabsmax &&
			//math.Abs(c.maxval-val) > tickThreshold*linabsmax {
			c.colorstops = append(c.colorstops, val)
			c.stopcolors = append(c.stopcolors,
				c.ColorScheme.interpolate(val/linabsmax))
		}
		if math.Nextafter(val, val+majorDelta) == val {
			break
		}
		val += majorDelta
	}
	if c.Type == LinCutoff && cutpt < c.maxval && cutpt != 0 {
		c.colorstops = append(c.colorstops, absmax)
		c.stopcolors = append(c.stopcolors, c.ColorScheme.HighLimit)
		c.positiveOutlier = true
	} else {
		c.colorstops = append(c.colorstops, c.maxval)
		c.stopcolors = append(c.stopcolors,
			c.ColorScheme.interpolate(c.maxval/linabsmax))
	}

	// calculate the locations for the tick marks
	numstops := len(c.stopcolors)
	c.ticklocs = make([]float64, numstops)
	span := linmax - linmin
	loc := 0.
	for i, stop := range c.colorstops {
		if i != 0 {
			if c.negativeOutlier && i == 1 ||
				c.positiveOutlier && i == numstops-1 {
				loc += 0.05
			} else {
				loc += (stop - c.colorstops[i-1]) / span
			}
			c.ticklocs[i] = loc
		}
	}
	for i, val := range c.ticklocs {
		c.ticklocs[i] = val / c.ticklocs[numstops-1]
	}
}

// DefaultLegendCanvas is a default canvas for drawing legends.
type DefaultLegendCanvas struct {
	draw.Canvas
}

func NewDefaultLegendCanvas() *DefaultLegendCanvas {
	const LegendWidth = 3.70 * vg.Inch
	const LegendHeight = LegendWidth * 0.1067
	const dpi = 300
	c := &DefaultLegendCanvas{
		Canvas: draw.New(vgpdf.New(LegendWidth, LegendHeight)),
		//Canvas: draw.New(vgimg.New(LegendWidth, LegendHeight, dpi)),
	}
	return c
}

func (c *DefaultLegendCanvas) WriteTo(w io.Writer) error {
	//cc := c.Canvas.Canvas.(*vgimg.Canvas)
	//_, err := vgimg.PngCanvas{cc}.WriteTo(w)
	//cc := c.Canvas.Canvas.(*vgsvg.Canvas)
	cc := c.Canvas.Canvas.(*vgpdf.Canvas)
	_, err := cc.WriteTo(w)
	return err
}

// Legend draws a legend to the supplied canvas.
func (c *ColorMap) Legend(canvas *draw.Canvas, label string) (err error) {
	c.Canvas = *canvas
	const topPad = vg.Length(0.)       // points
	const bottomPad = vg.Length(1.)    // points
	const tickPadAbove = vg.Length(2.) // pad between tick mark labels and bar
	const tickPadBelow = vg.Length(1.) // pad between tick mark labels and bar
	const labelPad = vg.Length(2.)     // pad between bar and label
	const wPad = vg.Length(10.)        // points
	const tickLength = vg.Length(3)    // points
	font, err := vg.MakeFont(c.Font, c.FontSize)
	if err != nil {
		return err
	}
	textStyle := draw.TextStyle{Color: c.FontColor, Font: font}
	//barLeft := wPad
	barLeft := c.Min.X + wPad
	//barRight := c.Max.X - c.Min.X - wPad
	barRight := c.Max.X - wPad
	//barTop := c.Max.Y - c.Min.Y - topPad - textStyle.Height(label) - labelPad
	barTop := c.Max.Y - topPad - textStyle.Height(label) - labelPad
	//barBottom := bottomPad + textStyle.Height("2.0e2") + tickPadBelow
	barBottom := c.Min.Y + bottomPad + textStyle.Height("2.0e2") + tickPadBelow
	//labelX := (c.Max.X - c.Min.X) * 0.5
	labelX := c.Min.X + (c.Max.X-c.Min.X)*0.5
	//labelY := c.Max.Y - c.Min.Y - topPad
	labelY := c.Max.Y - topPad
	unitsYunder := barBottom - tickPadBelow
	unitsYover := barTop + tickPadAbove

	// Fill in background
	//c.Canvas.FillPolygon(c.BackgroundColor, []draw.Point{
	//	draw.Point{0., 0.}, draw.Point{c.Max.X - c.Min.X, 0},
	//	draw.Point{c.Max.X - c.Min.X, c.Max.Y - c.Min.Y},
	//	draw.Point{0., c.Max.Y - c.Min.Y}, draw.Point{0, 0}})
	c.Canvas.FillPolygon(c.BackgroundColor, []vg.Point{
		vg.Point{X: c.Min.X, Y: c.Min.Y}, vg.Point{X: c.Max.X, Y: c.Min.Y},
		vg.Point{X: c.Max.X, Y: c.Max.Y},
		vg.Point{X: c.Min.X, Y: c.Max.Y}, vg.Point{X: c.Min.X, Y: c.Min.Y}})

	// Create gradient using a bunch of thin lines
	gradLoc := barLeft
	for {
		if gradLoc >= barRight {
			break
		}
		color := c.getColorOnLegend(gradLoc, barLeft, barRight)
		ls := draw.LineStyle{Color: color, Width: c.GradientLineWidth}
		c.Canvas.StrokeLine2(ls, gradLoc, barBottom, gradLoc,
			barTop)
		gradLoc += c.GradientLineWidth * 0.9
	}

	// Stroke edge of color bar
	ls := draw.LineStyle{Color: c.EdgeColor, Width: c.LineWidth}
	c.Canvas.StrokeLines(ls, []vg.Point{
		vg.Point{barLeft, barBottom},
		vg.Point{barRight, barBottom},
		vg.Point{barRight, barTop},
		vg.Point{barLeft, barTop},
		vg.Point{barLeft, barBottom}})

	for i, tickloc := range c.ticklocs {
		val := c.colorstops[i]
		var valStr string
		if math.Abs(val) < absMax(c.maxval, c.minval)*1.e-10 {
			valStr = "0"
		} else {
			valStr = strings.Replace(strings.Replace(fmt.Sprintf("%3.2g", val),
				"e+0", "e", -1), "e-0", "e-", -1)
		}
		tickx := barLeft + vg.Length(tickloc)*(barRight-barLeft)
		if c.negativeOutlier && i == 0 ||
			c.positiveOutlier && i == len(c.ticklocs)-1 {
			sty := textStyle
			sty.XAlign = -0.5
			sty.YAlign = 0
			c.Canvas.FillText(sty, vg.Point{tickx, unitsYover}, valStr)
		} else if !(c.Type == Linear && (i == 0 || i == len(c.ticklocs)-1)) {
			sty := textStyle
			sty.XAlign = -0.5
			sty.YAlign = -1
			c.Canvas.FillText(sty, vg.Point{tickx, unitsYunder}, valStr)
		}
		c.Canvas.StrokeLine2(ls, tickx, barBottom, tickx,
			barBottom+tickLength)
	}
	sty := textStyle
	sty.XAlign = -0.5
	sty.YAlign = -1
	c.Canvas.FillText(sty, vg.Point{labelX, labelY}, label)
	return
}

func max(a, b float64) float64 {
	if a > b {
		return a
	} else {
		return b
	}
}
func min(a, b float64) float64 {
	if a < b {
		return a
	} else {
		return b
	}
}
func absMax(a, b float64) float64 {
	absa := math.Abs(a)
	absb := math.Abs(b)
	if absa > absb {
		return absa
	} else {
		return absb
	}
}
func average(a []float64) (avg float64) {
	for _, val := range a {
		avg += val
	}
	avg /= float64(len(a))
	return
}
