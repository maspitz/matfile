// Copyright 2015 Michael Spitznagel.
// This is program is free software.  You may distribute it under the
// terms of the GNU General Public License.

// Package matfile implements the encoding and decoding of v5 MAT-File data.
package matfile

// VarReader represents a single header and a sequence of decodable variables
type VarReader interface {
	GetHeader() (Header, error)
	GetVarInfo() (VarInfo, error)
	GetVar() (Var, error)
	Next() error
}

// VarWriter encodes variables sequentially
type VarWriter interface {
	PutVar(Var)
}

// Header contains descriptive text, a version, and a byte-order indicator
type Header struct {
}

// VarInfo describes the name and type of a Var
type VarInfo struct {
}

// Var is the basic unit of data decoded from a File
type Var struct {
	VarInfo
}
