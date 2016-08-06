// A sparse array package based on
// https://github.com/skelterjohn/go.matrix/

package sparse

import (
	"fmt"
)

// BoundsCheck specifies whether to check array bounds every time
var BoundsCheck = true

// SparseArray is a sparse array with an arbitrary number of dimensions
type SparseArray struct {
	Elements map[int]float64
	ndims    int
	Shape    []int
	arrsize  int // Maximum number of Elements in array
}

// DenseArray is a dense array with an arbitrary number of dimensions
type DenseArray struct {
	Elements []float64
	ndims    int
	Shape    []int
	arrsize  int // Maximum number of Elements in array
}

// DenseArrayInt is a dense array of integers with an arbitrary number of dimensions
type DenseArrayInt struct {
	Elements []int
	ndims    int
	Shape    []int
	arrsize  int // Maximum number of Elements in array
}

// Array is a general float64 array type.
type Array interface {
	Set(val float64, index ...int)
	Get(index ...int) float64
	Scale(val float64)
	Sum() float64
	GetShape() []int
}

// ZerosSparse initializes a new sparse array.
func ZerosSparse(dims ...int) *SparseArray {
	A := new(SparseArray)
	A.Elements = make(map[int]float64)
	A.ndims = len(dims)
	A.Shape = dims
	A.arrsize = 1
	for _, i := range A.Shape {
		A.arrsize *= i
	}
	return A
}

// ZerosDense initializes a new dense array.
func ZerosDense(dims ...int) *DenseArray {
	A := new(DenseArray)
	A.ndims = len(dims)
	A.Shape = dims
	A.arrsize = 1
	for _, i := range A.Shape {
		A.arrsize *= i
	}
	A.Elements = make([]float64, A.arrsize)
	return A
}

// ZerosDenseInt initializes a new dense integer array.
func ZerosDenseInt(dims ...int) *DenseArrayInt {
	A := new(DenseArrayInt)
	A.ndims = len(dims)
	A.Shape = dims
	A.arrsize = 1
	for _, i := range A.Shape {
		A.arrsize *= i
	}
	A.Elements = make([]int, A.arrsize)
	return A
}

// Fix re-initializes the unexported fields, for example after
// transmitting via rpc.
func (A *SparseArray) Fix() {
	A.ndims = len(A.Shape)
	A.arrsize = 1
	for _, d := range A.Shape {
		A.arrsize *= d
	}
}

// Fix re-initializes the unexported fields, for example after
// transmitting via rpc.
func (A *DenseArray) Fix() {
	A.ndims = len(A.Shape)
	A.arrsize = 1
	for _, d := range A.Shape {
		A.arrsize *= d
	}
}

// GetShape returns the array shape.
func (A *DenseArray) GetShape() []int {
	return A.Shape
}

// GetShape returns the array shape.
func (A *SparseArray) GetShape() []int {
	return A.Shape
}

// Copy copies the array.
func (A *SparseArray) Copy() *SparseArray {
	B := new(SparseArray)
	ndims, shape, arrsize := A.ndims, A.Shape, A.arrsize
	B.ndims = ndims
	B.Shape = shape
	B.arrsize = arrsize
	B.Elements = make(map[int]float64)
	for i, e := range A.Elements {
		B.Elements[i] = e
	}
	return B
}

// Copy an array
func (A *DenseArray) Copy() *DenseArray {
	B := new(DenseArray)
	ndims, shape, arrsize := A.ndims, A.Shape, A.arrsize
	B.ndims = ndims
	B.Shape = shape
	B.arrsize = arrsize
	B.Elements = make([]float64, arrsize)
	for i, e := range A.Elements {
		B.Elements[i] = e
	}
	return B
}

// CheckIndex checks whether index is within array dimensions.
func (A *SparseArray) CheckIndex(index []int) error {
	if BoundsCheck {
		if len(index) != A.ndims {
			err := fmt.Errorf("Index number of dimensions (%v) does not match "+
				"array number of dimensions. (%v)", len(index), A.ndims)
			return err
		}
		for i, dim := range A.Shape {
			if index[i] < 0 {
				return fmt.Errorf(
					"index %v of dimension %v is less than 0", index[i], i)
			}
			if index[i] >= dim {
				return fmt.Errorf(
					"Index %v of dimension %v is greater than dimension "+
						"size %v.", index[i], i, A.Shape[i])
			}
		}
	}
	return nil
}

// CheckIndex checks whether index is within array dimensions.
func (A *DenseArray) CheckIndex(index []int) error {
	if BoundsCheck {
		if len(index) != A.ndims {
			err := fmt.Errorf("Index number of dimensions (%v) does not match "+
				"array number of dimensions (%v).", len(index), A.ndims)
			return err
		}
		for i, dim := range A.Shape {
			if index[i] < 0 {
				return fmt.Errorf(
					"index %v of dimension %v is less than 0", index[i], i)
			}
			if index[i] >= dim {
				err := fmt.Errorf(
					"Index %v of dimension %v is greater than dimension "+
						"size %v.", index[i], i, A.Shape[i])
				return err
			}
		}
	}
	return nil
}

// CheckIndex checks whether index is within array dimensions.
func (A *DenseArrayInt) CheckIndex(index []int) error {
	if BoundsCheck {
		if len(index) != A.ndims {
			err := fmt.Errorf("Index number of dimensions (%v) does not match "+
				"array number of dimensions (%v).", len(index), A.ndims)
			return err
		}
		for i, dim := range A.Shape {
			if index[i] < 0 {
				return fmt.Errorf(
					"index %v of dimension %v is less than 0", index[i], i)
			}
			if index[i] >= dim {
				err := fmt.Errorf(
					"Index %v of dimension %v is greater than dimension "+
						"size %v.", index[i], i, A.Shape[i])
				return err
			}
		}
	}
	return nil
}

// Make sure arrays are the same size
func (A *SparseArray) checkArray(B *SparseArray) error {
	if BoundsCheck {
		if B.ndims != A.ndims {
			err := fmt.Errorf("Number of dimensions in array A (%v) does "+
				"not match number of dimensions in array B (%v).", A.ndims, B.ndims)
			return err
		}
		for i, dim := range A.Shape {
			if B.Shape[i] != dim {
				err := fmt.Errorf(
					"Dimension %v is different in arrays A (%v) and B (%v).",
					i, A.Shape[i], B.Shape[i])
				return err
			}
		}
	}
	return nil
}

// Make sure arrays are the same size
func (A *DenseArray) checkArray(B *DenseArray) error {
	if BoundsCheck {
		if B.ndims != A.ndims {
			err := fmt.Errorf("Number of dimensions in array A (%v) does "+
				"not match number of dimensions in array B (%v).", A.ndims, B.ndims)
			return err
		}
		for i, dim := range A.Shape {
			if B.Shape[i] != dim {
				err := fmt.Errorf(
					"Dimension %v is different in arrays A (%v) and B (%v).",
					i, A.Shape[i], B.Shape[i])
				return err
			}
		}
	}
	return nil
}

// Make sure arrays are the same size
func (A *DenseArray) checkArraySparse(B *SparseArray) error {
	if BoundsCheck {
		if B.ndims != A.ndims {
			err := fmt.Errorf("Number of dimensions in array A (%v) does "+
				"not match number of dimensions in array B (%v).", A.ndims, B.ndims)
			return err
		}
		for i, dim := range A.Shape {
			if B.Shape[i] != dim {
				err := fmt.Errorf(
					"Dimension %v is different in arrays A (%v) and B (%v).",
					i, A.Shape[i], B.Shape[i])
				return err
			}
		}
	}
	return nil
}

// Convert n-dimensional index to one-dimensional index
func (A *SparseArray) Index1d(index ...int) (index1d int) {
	if err := A.CheckIndex(index); err != nil {
		panic(err)
	}
	for i := 0; i < len(index); i++ {
		mul := 1
		for j := i + 1; j < len(index); j++ {
			mul = mul * A.Shape[j]
		}
		index1d = index1d + index[i]*mul
	}
	return index1d
}

// Convert n-dimensional index to one-dimensional index
func (A *DenseArray) Index1d(index ...int) (index1d int) {
	if err := A.CheckIndex(index); err != nil {
		panic(err)
	}
	for i := 0; i < len(index); i++ {
		mul := 1
		for j := i + 1; j < len(index); j++ {
			mul = mul * A.Shape[j]
		}
		index1d = index1d + index[i]*mul
	}
	return index1d
}

// Convert n-dimensional index to one-dimensional index
func (A *DenseArrayInt) Index1d(index ...int) (index1d int) {
	if err := A.CheckIndex(index); err != nil {
		panic(err)
	}
	for i := 0; i < len(index); i++ {
		mul := 1
		for j := i + 1; j < len(index); j++ {
			mul = mul * A.Shape[j]
		}
		index1d = index1d + index[i]*mul
	}
	return index1d
}

// Convert a 1-dimensional index to an n-dimensional index
func (A *SparseArray) IndexNd(index1d int) (indexNd []int) {
	leftover := index1d
	indexNd = make([]int, A.ndims)
	for i := 0; i < A.ndims; i++ {
		stride := 1
		for j := i + 1; j < A.ndims; j++ {
			stride *= A.Shape[j]
		}
		indexNd[i] = leftover / stride
		if leftover >= stride {
			leftover = leftover % (indexNd[i] * stride)
		} else {
			leftover = 0
		}
	}
	return
}

// Convert a 1-dimensional index to an n-dimensional index
func (A *DenseArray) IndexNd(index1d int) (indexNd []int) {
	leftover := index1d
	indexNd = make([]int, A.ndims)
	for i := 0; i < A.ndims; i++ {
		stride := 1
		for j := i + 1; j < A.ndims; j++ {
			stride *= A.Shape[j]
		}
		indexNd[i] = leftover / stride
		if leftover >= stride {
			leftover = leftover % (indexNd[i] * stride)
		} else {
			leftover = 0
		}
	}
	return
}

// Set index to val.
func (A *SparseArray) Set(val float64, index ...int) {
	if val == 0. {
		return
	}
	if err := A.CheckIndex(index); err != nil {
		panic(err)
	}
	index1d := A.Index1d(index...)
	A.Elements[index1d] = val
}

// Set index to val.
func (A *DenseArray) Set(val float64, index ...int) {
	if val == 0. {
		return
	}
	if err := A.CheckIndex(index); err != nil {
		panic(err)
	}
	index1d := A.Index1d(index...)
	A.Elements[index1d] = val
}

// Set index to val.
func (A *DenseArrayInt) Set(val int, index ...int) {
	if val == 0. {
		return
	}
	if err := A.CheckIndex(index); err != nil {
		panic(err)
	}
	index1d := A.Index1d(index...)
	A.Elements[index1d] = val
}

// Get array value at index
func (A *SparseArray) Get(index ...int) float64 {
	if err := A.CheckIndex(index); err != nil {
		panic(err)
	}
	index1d := A.Index1d(index...)
	val, ok := A.Elements[index1d]
	if ok {
		return val
	} else {
		return 0.
	}
}

// Get array value at index
func (A *DenseArray) Get(index ...int) float64 {
	if err := A.CheckIndex(index); err != nil {
		panic(err)
	}
	index1d := A.Index1d(index...)
	return A.Elements[index1d]
}

// Get array value at index
func (A *DenseArrayInt) Get(index ...int) int {
	if err := A.CheckIndex(index); err != nil {
		panic(err)
	}
	index1d := A.Index1d(index...)
	return A.Elements[index1d]
}

// Get array value at one-dimensional index
func (A *SparseArray) Get1d(index1d int) float64 {
	val, ok := A.Elements[index1d]
	if ok {
		return val
	} else {
		return 0.
	}
}

// Get array value at one-dimensional index
func (A *DenseArray) Get1d(index1d int) float64 {
	return A.Elements[index1d]
}

// Add val at array index
func (A *SparseArray) AddVal(val float64, index ...int) {
	if err := A.CheckIndex(index); err != nil {
		panic(err)
	}
	index1d := A.Index1d(index...)
	A.Elements[index1d] += val
}

// AddVal adds value val at array index index.
func (A *DenseArray) AddVal(val float64, index ...int) {
	if err := A.CheckIndex(index); err != nil {
		panic(err)
	}
	index1d := A.Index1d(index...)
	A.Elements[index1d] += val
}

// AddSparse adds array B to array A in place.
func (A *SparseArray) AddSparse(B *SparseArray) {
	if err := A.checkArray(B); err != nil {
		panic(err)
	}
	for i, val := range B.Elements {
		A.Elements[i] += val
	}
}

// AddDense adds array B to array A in place.
func (A *DenseArray) AddDense(B *DenseArray) {
	if err := A.checkArray(B); err != nil {
		panic(err)
	}
	for i, val := range B.Elements {
		A.Elements[i] += val
	}
}

// Add array B to array A in place.
func (A *DenseArray) AddSparse(B *SparseArray) {
	if err := A.checkArraySparse(B); err != nil {
		panic(err)
	}
	for i, val := range B.Elements {
		A.Elements[i] += val
	}
}

// Subtract array B from array A in place.
func (A *SparseArray) SubtractSparse(B *SparseArray) {
	if err := A.checkArray(B); err != nil {
		panic(err)
	}
	for i, val := range B.Elements {
		A.Elements[i] -= val
	}
}

// Subtract val at array index
func (A *SparseArray) SubtractVal(val float64, index ...int) {
	if err := A.CheckIndex(index); err != nil {
		panic(err)
	}
	index1d := A.Index1d(index...)
	A.Elements[index1d] -= val
}

// Scale Multiplies entire array by val
func (A *SparseArray) Scale(val float64) {
	for i, _ := range A.Elements {
		A.Elements[i] *= val
	}
}

// ScaleCopy returns a copy of the array  multiplied by val
func (A *SparseArray) ScaleCopy(val float64) *SparseArray {
	out := A.Copy()
	for i, _ := range A.Elements {
		out.Elements[i] *= val
	}
	return out
}

// Scale Multiplies entire array by val
func (A *DenseArray) Scale(val float64) {
	for i, _ := range A.Elements {
		A.Elements[i] *= val
	}
}

// ScaleCopy returns a copy of the array  multiplied by val
func (A *DenseArray) ScaleCopy(val float64) *DenseArray {
	out := A.Copy()
	for i, _ := range A.Elements {
		out.Elements[i] *= val
	}
	return out
}

func ArrayMultiply(A, B *SparseArray) *SparseArray {
	if err := A.checkArray(B); err != nil {
		panic(err)
	}
	out := A.Copy()
	for i, _ := range out.Elements {
		if _, ok := B.Elements[i]; ok {
			out.Elements[i] *= B.Elements[i]
		} else {
			delete(out.Elements, i)
		}
	}
	return out
}

// IsNil returns whether the array has been allocated or not
func (A *SparseArray) IsNil() bool {
	return len(A.Elements) == 0
}

// Sum calculates the array sum.
func (A *SparseArray) Sum() float64 {
	sum := 0.
	if len(A.Elements) == 0 {
		return 0.
	}
	for _, e := range A.Elements {
		sum += e
	}
	return sum
}

// Sum calculates the array sum.
func (A *DenseArray) Sum() float64 {
	sum := 0.
	for _, e := range A.Elements {
		sum += e
	}
	return sum
}

// Nonzero returns (one dimensional) indicies of nonzero array Elements
func (A *SparseArray) Nonzero() []int {
	index := make([]int, len(A.Elements))
	i := 0
	for j, _ := range A.Elements {
		index[i] = j
		i++
	}
	return index
}

func (A *SparseArray) ToDense() []float64 {
	out := make([]float64, A.arrsize)
	for i, val := range A.Elements {
		out[i] = val
	}
	return out
}

func (A *SparseArray) ToDenseArray() *DenseArray {
	out := ZerosDense(A.Shape...)
	for i, val := range A.Elements {
		out.Elements[i] = val
	}
	return out
}

func (A *SparseArray) ToDense32() []float32 {
	out := make([]float32, A.arrsize)
	for i, val := range A.Elements {
		out[i] = float32(val)
	}
	return out
}

// returns either zero of the maximum value of the
// array; whichever is greater.
func (A *DenseArray) Max() float64 {
	max := 0.
	for _, v := range A.Elements {
		if v > max {
			max = v
		}
	}
	return max
}

// returns the maximum absolute value of the array
func (A *DenseArray) AbsMax() float64 {
	max := 0.
	for _, v := range A.Elements {
		if v > max {
			max = v
		} else if -1*v > max {
			max = -1 * v
		}
	}
	return max
}

// Subset extracts a subset of the array. Both the start and end
// indicies are inclusive
func (A *DenseArray) Subset(start []int, end []int) (B *DenseArray) {
	outputNdims := 0
	for i, s := range start {
		if end[i] > s {
			outputNdims++
		} else if end[i] < s {
			panic("End of array is before start")
		}
	}
	outDims := make([]int, outputNdims)
	outDim := 0
	for i, s := range start {
		if end[i] > s {
			outDims[outDim] = end[i] - s + 1
			outDim++
		}
	}
	B = ZerosDense(outDims...)
	B.Elements = A.Elements[A.Index1d(start...) : A.Index1d(end...)+1]
	return
}
