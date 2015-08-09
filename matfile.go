// Copyright 2015 Michael Spitznagel.
// This is program is free software.  You may distribute it under the
// terms of the GNU General Public License.

// Package matfile implements the encoding and decoding of v5 MAT-File data.
package matfile

import (
	"encoding/binary"
	"io"
	"math"
	"unicode/utf16"
)

// VarReader represents a file: a single header followed by
// a sequence of decodable data elements
type VarReader struct {
	Header
	elementStream
}

// Header contains descriptive text, a version, and a byte-order indicator
type Header struct {
	Description     [116]byte // descriptive text
	Offset          int64     // offset to subsystem data
	Version         int16     // version (0x0100)
	EndianIndicator [2]byte   // indicates byte order
}

// elementStream emits data elements in sequence.
// decodes the tag segment but not the data segment.
type elementStream struct {
	binary.ByteOrder
	r   io.ReaderAt
	pos int64
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

// nextElement provides the tag and a reader for the data of
// a data element, and advances the stream to the next data element.
func (er *elementStream) nextElement() (dataElement, error) {
	var de dataElement
	var tagbuf [8]byte
	_, err := er.r.ReadAt(tagbuf[:], er.pos)
	if err != nil {
		return de, err
	}
	de.tag = decodeTag(tagbuf[:], er.ByteOrder)
	if de.smallFormat == true {
		de.r = io.NewSectionReader(er.r, er.pos+4, int64(de.nBytes))
		er.pos = er.pos + 8
	} else {
		de.r = io.NewSectionReader(er.r, er.pos+8, int64(de.nBytes))
		// padding equivalent to (8 - (length % 8)) % 8
		er.pos = er.pos + 8 + int64(de.nBytes)
		if de.dataType != miCOMPRESSED {
			padding := (8 - (de.nBytes & 7)) & 7
			er.pos = er.pos + int64(padding)
		}
	}
	return de, nil
}

// TODO consider returning error if data length is not divisible the right way

func decodeElement(de dataElement, bo binary.ByteOrder) (interface{}, error) {
	switch de.dataType {
	case miINT8, miUINT8, miINT16, miUINT16, miINT32, miUINT32,
		miINT64, miUINT64, miSINGLE, miDOUBLE,
		miUTF8, miUTF16, miUTF32:
		return decodeNumeric(de, bo)
	case miMATRIX:
		return decodeArray(de, bo)
	case miCOMPRESSED:
		zde, err := decompressElement(de, bo)
		if err != nil {
			return nil, err
		}
		return decodeElement(zde, bo)
	}
	return nil, nil
}

// decodeArray decodes structured array data
func decodeArray(de dataElement, bo binary.ByteOrder) (interface{}, error) {
	panic("decode Array not implemented")
	return nil, nil
}

// decodeNumeric decodes a simple stream of numeric or character data
func decodeNumeric(de dataElement, bo binary.ByteOrder) (interface{}, error) {
	var b [8]byte
	var bs []byte
	if int(de.nBytes) > len(b) {
		bs = make([]byte, de.nBytes)
	} else {
		bs = b[:de.nBytes]
	}
	_, err := de.r.ReadAt(bs, 0)
	if err != nil {
		return nil, err
	}
	switch de.dataType {
	case miINT8:
		val := make([]int8, de.nBytes)
		for i, x := range bs {
			val[i] = int8(x)
		}
		return val, nil
	case miUINT8:
		val := make([]uint8, de.nBytes)
		copy(val, bs)
		return val, nil
	case miINT16:
		val := make([]int16, de.nBytes/2)
		for i := range bs {
			val[i] = int16(bo.Uint16(bs[2*i:]))
		}
		return val, nil
	case miUINT16:
		val := make([]uint16, de.nBytes/2)
		for i := range bs {
			val[i] = bo.Uint16(bs[2*i:])
		}
				return val, nil
	case miINT32:
		val := make([]int32, de.nBytes/4)
		for i := range bs {
			val[i] = int32(bo.Uint32(bs[4*i:]))
		}
				return val, nil
	case miUINT32:
		val := make([]uint32, de.nBytes/4)
		for i := range bs {
			val[i] = bo.Uint32(bs[4*i:])
		}
				return val, nil
	case miINT64:
		val := make([]int64, de.nBytes/8)
		for i := range bs {
			val[i] = int64(bo.Uint64(bs[8*i:]))
		}
				return val, nil
	case miUINT64:
		val := make([]uint64, de.nBytes/8)
		for i := range bs {
			val[i] = bo.Uint64(bs[8*i:])
		}
				return val, nil
	case miSINGLE:
		val := make([]float32, de.nBytes/4)
		for i := range bs {
			val[i] = math.Float32frombits(bo.Uint32(bs[4*i:]))
		}
				return val, nil
	case miDOUBLE:
		val := make([]float64, de.nBytes/8)
		for i := range bs {
			val[i] = math.Float64frombits(bo.Uint64(bs[8*i:]))
		}
				return val, nil
	case miUTF8:
		return string(bs), nil
	case miUTF16:
		x := make([]uint16, de.nBytes/2)
		for i := range bs {
			x[i] = bo.Uint16(bs[2*i:])
		}
		return string(utf16.Decode(x)), nil
	case miUTF32:
		runes := make([]rune, de.nBytes/4)
		for i := range runes {
			runes[i] = rune(bo.Uint32(bs[4*i:]))
		}
		return string(runes), nil
	}
	return nil, nil
}

// TODO decompressElement cannot handle a doubly-compressed element,
// because the zlibReaderAt does not implement io.Reader.
// Figure out if the MAT-file specification permits 2x-compressed elts.

// TODO also figure out whether miCOMPRESSED must contain only
// a single element, or if it can contain a stream of elements.

func decompressElement(in dataElement, bo binary.ByteOrder) (dataElement, error) {
	rd := in.r.(io.Reader)
	zrat, err := newzlibReaderAt(rd, int(in.nBytes))
	if err != nil {
		return dataElement{}, err
	}
	defer zrat.(io.Closer).Close()
	
	zstream := elementStream{bo, zrat, 0}
	return zstream.nextElement()
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
	r io.ReaderAt
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
