// Copyright 2015 Michael Spitznagel.
// This is program is free software.  You may distribute it under the
// terms of the GNU General Public License.

// Package matfile implements the encoding and decoding of v5 MAT-File data.
package matfile

import "encoding/binary"

// VarReader represents a single header and a sequence of decodable variables
type VarReader interface {
	Header() (Header, error)
	PeekInfo() (VarInfo, error)
	Read() (Var, error)
	Next() error
}

// VarWriter encodes variables sequentially
type VarWriter interface {
	Write(Var)
}

// Header contains descriptive text, a version, and a byte-order indicator
type Header struct {
	Description [116]byte
	Offset int64
	Version int16
	binary.ByteOrder
}

// Var is the basic unit of data decoded from a File
// Some fields are populated only for certain array classes.
type Var struct {
	VarInfo
	RealPart interface{}
	ImagPart interface{}
	RowIndex []int32 // for Sparse Array
	ColIndex []int32 // for Sparse Array
	Cells    []*Var  // Cells for Cell Array; also used as
	// Fields for Structure or Object.
	FieldNameLength int32  // for Structure or Object
	FieldNames      []int8 // for Structure or Object
	ClassName       []int8 // for Object
}

// VarInfo contains metadata for a Var
type VarInfo struct {
	IsComplex, IsGlobal, IsLogical bool
	ArrayClass
	Dimensions []int32
	Name       string
	Nzmax      uint32
}

// ArrayClass specifies the type of data contained in a variable.
// (cell array, struct array, numeric array ...)
type ArrayClass uint8

// The following numeric values correspond to the
// MAT-File specification.
const (
	ClassCell   ArrayClass = 1 // cell array
	ClassStruct ArrayClass = 2 // struct array
	ClassObject ArrayClass = 3 // object
	ClassChar   ArrayClass = 4 // character array
	ClassSparse ArrayClass = 5 // sparse array
	ClassDouble ArrayClass = 6 // specific types of numeric array:
	ClassSingle ArrayClass = 7
	ClassInt8   ArrayClass = 8
	ClassUint8  ArrayClass = 9
	ClassInt16  ArrayClass = 10
	ClassUint16 ArrayClass = 11
	ClassInt32  ArrayClass = 12
	ClassUint32 ArrayClass = 13
	ClassInt64  ArrayClass = 14
	ClassUint64 ArrayClass = 15
)

type dataElement struct {
	dataType
	data []byte
	binary.ByteOrder
}

// dataType specifies the type of data contained in a dataElement.
type dataType uint8

// The following numeric values correspond to the
// MAT-File specification.
const (
	miINT8       dataType = 1
	miUINT8      dataType = 2
	miINT16      dataType = 3
	miUINT16     dataType = 4
	miINT32      dataType = 5
	miUINT32     dataType = 6
	miSINGLE     dataType = 7
	miDOUBLE     dataType = 9
	miINT64      dataType = 12
	miUINT64     dataType = 13
	miMATRIX     dataType = 14
	miCOMPRESSED dataType = 15
	miUTF8       dataType = 16
	miUTF16      dataType = 17
	miUTF32      dataType = 18
)
