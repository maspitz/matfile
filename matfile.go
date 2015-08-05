// Copyright 2015 Michael Spitznagel.
// This is program is free software.  You may distribute it under the
// terms of the GNU General Public License.

// Package matfile implements the encoding and decoding of v5 MAT-File data.
package matfile

import (
	"encoding/binary"
	"io"
	"io/ioutil"
	"math"
	"unicode/utf16"
)

// VarReader represents a file: a single header followed by
// a sequence of decodable data elements
type VarReader struct {
	Header
	decoder
}

// Header contains descriptive text, a version, and a byte-order indicator
type Header struct {
	Description     [116]byte // descriptive text
	Offset          int64     // offset to subsystem data
	Version         int16     // version (0x0100)
	EndianIndicator [2]byte   // indicates byte order
}

// NOTES ON INTERACTION WITH UNDERLYING READER
// In terms of OS file reads, what we want is something like:
// read file header
// read element tag
//   record element type
//   calculate & record offset to next element
// if tag is miCOMPRESSED,
//   set up a decompressing reader
//   and go ahead and read from that (should obtain miMATRIX)
// else if tag is miMATRIX,
//   [if we want to read the whole data element from file at once]
//     read all data into a []byte
//     decode sub - data elements from slices.  calculate offsets.
//   [if we want to read array info only]
//     read and decode sufficient sub - data elements using a SectionReader.

// Now, a cell array may itself contain miMATRIX data.
// If we expect to handle this using an underlying []byte instead of
// a SectionReader, then we need to have two different methods
// to decode data elements of type miMATRIX.
// This is needless duplication of effort.  We could wrap the
// []byte in a Reader but that defeats the purpose of having a lighter-weight
// data source for dataelement decoding.  So...  should use only
// SectionReaders I would say.  Assuming SectionReader is what I think
// it is.

// decoder is able to decode data elements in sequence
type decoder struct {
	binary.ByteOrder
	r io.ReaderAt
}

type tag struct {
	dataType
	nBytes      uint32
	smallFormat bool
}

// decodeTag assumes len(buf) >= 8
func decodeTag(buf []byte, bo binary.ByteOrder) tag {
	var t tag
	smallTag := bo.Uint32(buf[:])
	t.dataType = dataType(smallTag)
	t.smallFormat = (smallTag >> 16) != 0
	if t.smallFormat == true {
		t.nBytes = uint32(smallTag >> 16)
	} else {
		t.nBytes = bo.Uint32(buf[4:])
	}
	return t
}

// readFullElement reads an entire data element and advances
// the reader to the beginning of the next data element.
func readFullElement(r io.Reader, bo binary.ByteOrder) (dataElement, error) {
	var de dataElement
	var tagbuf [8]byte
	_, err := io.ReadFull(r, tagbuf[:])
	if err != nil {
		return de, err
	}
	de.tag = decodeTag(tagbuf[:], bo)
	de.data = make([]byte, de.nBytes)
	if de.smallFormat == true {
		copy(de.data, tagbuf[4:])
	} else {
		_, err = io.ReadFull(r, de.data)
		if err != nil {
			return de, err
		}

		if de.dataType != miCOMPRESSED {
			// padding equivalent to (8 - (length % 8)) % 8
			padding := (8 - (de.nBytes & 7)) & 7
			// advance reader past padding to next element (ignore errors)
			_, _ = io.CopyN(ioutil.Discard, r, int64(padding))
		}
	}
	return de, nil
}

// TODO consider signalling error if data length is not divisible
// TODO furthermore consider accepting dataType and []byte as arguments
//      instead of dataElement.
func decodeElement(de dataElement, bo binary.ByteOrder) interface{} {
	switch de.dataType {
	case miINT8:
		val := make([]int8, de.nBytes)
		for i, x := range de.data {
			val[i] = int8(x)
		}
		return val
	case miUINT8:
		val := make([]uint8, de.nBytes)
		copy(val, de.data)
		return val
	case miINT16:
		val := make([]int16, de.nBytes/2)
		for i := range val {
			val[i] = int16(bo.Uint16(de.data[2*i:]))
		}
		return val
	case miUINT16:
		val := make([]uint16, de.nBytes/2)
		for i := range val {
			val[i] = bo.Uint16(de.data[2*i:])
		}
		return val
	case miINT32:
		val := make([]int32, de.nBytes/4)
		for i := range val {
			val[i] = int32(bo.Uint32(de.data[4*i:]))
		}
		return val
	case miUINT32:
		val := make([]uint32, de.nBytes/4)
		for i := range val {
			val[i] = bo.Uint32(de.data[4*i:])
		}
		return val
	case miSINGLE:
		val := make([]float32, de.nBytes/4)
		for i := range val {
			val[i] = math.Float32frombits(bo.Uint32(de.data[4*i:]))
		}
		return val
	case miDOUBLE:
		val := make([]float64, de.nBytes/8)
		for i := range val {
			val[i] = math.Float64frombits(bo.Uint64(de.data[8*i:]))
		}
		return val
	case miINT64:
		val := make([]int64, de.nBytes/8)
		for i := range val {
			val[i] = int64(bo.Uint64(de.data[8*i:]))
		}
		return val
	case miUINT64:
		val := make([]uint64, de.nBytes/8)
		for i := range val {
			val[i] = bo.Uint64(de.data[8*i:])
		}
		return val
	case miMATRIX:
		return nil
		//return decodeFullArray(de)
	case miCOMPRESSED:
		return nil
		//return decodeFullCompressed(de)
	case miUTF8:
		return string(de.data)
	case miUTF16:
		val := make([]uint16, de.nBytes/2)
		for i := range val {
			val[i] = bo.Uint16(de.data[2*i:])
		}
		return string(utf16.Decode(val))
	case miUTF32:
		runes := make([]rune, de.nBytes/4)
		for i := range runes {
			runes[i] = rune(bo.Uint32(de.data[4*i:]))
		}
		return string(runes)
	}
	return nil
}

// VarWriter encodes variables sequentially
type VarWriter interface {
	Write(Var)
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
	tag
	data []byte
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
