// Package stata writes data into a Stata 113 format (readable by any Stata version higher than 7)
// Source for format info https://www.stata.com/help.cgi?dta_113
// The package does not do much validation. It is up to the user to ensure that the supplied data
// meets the format specification!
package gostata

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"time"
	"unsafe"
)

const (
	stataVarSize   = 33
	stataFmtSize   = 12
	stataLabelSize = 81

	STATA_BYTE_NA     = 127
	STATA_SHORTINT_NA = 32767
	STATA_INT_NA      = 2147483647

	// DTA113MaxInt8     int8   = 0x64       // 100
	// DTA113MaxInt16    int16  = 0x7fe4     // 32740
	// DTA113MaxInt32    int32  = 0x7fffffe4 // 2147483620
	//
	// DTA113MissingInt8  int8  = 0x65       // 101
	// DTA113MissingInt16 int16 = 0x7FE5     // 32741
	// DTA113MissingInt32 int32 = 0x7FFFFFE5 // 2147483621

)

var (
	STATA_FLOAT_NA  = math.Pow(2.0, 127)
	STATA_DOUBLE_NA = math.Pow(2.0, 1023)

	// DTA113MaxFloat32 = math.Float32frombits(0x7effffff)        // ~1.7e38
	//    DTA113MaxFloat64 = math.Float64frombits(0x7fdfffffffffffff) // ~8.9e307
	//
	//    DTA113MissingFloat32 = math.Float32frombits(0x7F000000)
	//    DTA113MissingFloat64 = math.Float64frombits(0x7FE0000000000000)
	//
	littleEndian = binary.LittleEndian //only write LittleEndian
)

type (
	stataVarName [stataVarSize]byte //used for both variable and label names
	stataFmtName [stataFmtSize]byte //used for format names
	stataLabel   [stataLabelSize]byte
)

// Supported Stata var types aliased to Go types for maximum convertibility
type (
	Byte   = int8
	Int    = int16
	Long   = int32
	Float  = float32
	Double = float64
)

const (
	StataByteId   = 251 // 0xfb
	StataIntId    = 252 // 0xfc
	StataLongId   = 253 // 0xfd
	StataFloatId  = 254 // 0xfe
	StataDoubleId = 255 // 0xff
)

// field name must be exported for package Binary to see them
type header struct {
	//	Contents            Length    Format    Comments
	Version   byte       //     1    byte      contains 113 = 0x71
	ByteOrder byte       //     1    byte      0x01 -> HILO, 0x02 -> LOHI
	FileType  byte       //     1    byte      0x01
	UnUsed    byte       //     1    byte      0x01
	NumVars   int16      //		2    int       (number of vars) encoded per byteorder
	NumObs    int32      // 		4    int       (number of obs)  encoded per byteorder
	DataLabel stataLabel // 		81    char      dataset label, \0 terminated
	TimeStamp [18]byte   // 		18    char      date/time saved, \0 terminated
	//	Total                  109
}

func NewHeader() *header {
	fh := header{
		Version:   113, //113 is used in Stata versions 8-9
		ByteOrder: 2,   //LOHI
		FileType:  1,   //always 1
		UnUsed:    0,
	}
	//FIXME: leave empty for production; comment the line below
	copy(fh.DataLabel[:], "Written by VDEC Stata File Creator")
	copy(fh.TimeStamp[:], time.Now().Format("02 Jan 2006 15:04"))
	return &fh
}

// File Stata file info
type File struct {
	*header
	fields     []*Field
	recordSize int
	recBuf     []byte // buf for record appending
	offset     int    //offset within the record buffer
	f          *os.File
	w          *bufio.Writer
	internal_w bool //did we create w from a filename?
	//FIXME: remove from the struct and just declare when needed?
	//	Contents            	Length    	  Format       Comments
	typList  []byte         //         nvar    byte array
	varList  []stataVarName //      33*nvar    char array
	srtList  []byte         //    2*(nvar+1)   int array    encoded per byteorder
	fmtList  []stataFmtName //      12*nvar    char array
	lblList  []stataVarName //       33*nvar    char array
	vlblList []stataLabel
}

// NewFile returns a pointer to an initialized File.
func NewFile() *File {
	sf := File{
		header: NewHeader(),
	}
	return &sf
}

func NewFileFromStruct(data interface{}) (*File, error) {
	fields, err := ExtractFields(data)
	if err != nil {
		return nil, err
	}

	sf := &File{
		header: NewHeader(),
		fields: fields,
	}
	sf.recordSize = calcRecordSize(sf.fields)

	return sf, nil
}

// AddField adds a field to be written out to a Stata file
// It does not verify similarly-named field does not exist
// It does not verify field names and labels meet Stata requirements
// It does not verify that slice lengths are identical
func (sf *File) AddField(name, label string, slice interface{}) *Field {
	var (
		typ      byte
		sliceLen int
		format   = "%9.0g"
	)

	switch data := slice.(type) {
	case []Byte:
		typ = StataByteId
		sf.recordSize++ //one byte
		sliceLen = len(data)
	case []Int:
		typ = StataIntId
		sf.recordSize += 2
		sliceLen = len(data)
	case []Long:
		typ = StataLongId
		sf.recordSize += 4
		sliceLen = len(data)
	case []Float:
		typ = StataFloatId
		sf.recordSize += 4
		sliceLen = len(data)
	case []Double:
		typ = StataDoubleId
		sf.recordSize += 8
		sliceLen = len(data)
	default:
		panic("unsupported data type in field " + name) //must be a programmer error, so panic
		//return nil, fmt.Errorf("unsupported data type in field %s", name)
	}
	fld := &Field{
		Name:      name,
		FieldType: typ,
		Label:     label,
		Format:    format,
		data:      slice,
	}
	sf.fields = append(sf.fields, fld)
	sf.NumVars++
	if sliceLen > int(sf.NumObs) {
		sf.NumObs = int32(sliceLen)
	}
	return fld
}

// AddFieldMeta adds a description of a field in a record
// argument typ uses one of the following Stata variable types
//
//			type          code
//	       --------------------
//	       str1        1 = 0x01
//	       str2        2 = 0x02
//	       ...
//	       str244    244 = 0xf4
//	       byte      251 = 0xfb  (sic)
//	       int       252 = 0xfc
//	       long      253 = 0xfd
//	       float     254 = 0xfe
//	       double    255 = 0xff
//		--------------------
func (sf *File) AddFieldMeta(name, label string, typ byte) *Field {
	//TODO support custom formats
	format := "%9.0g" //stata not c printf formats; this works for all numeric types
	switch typ {
	case StataByteId:
		sf.recordSize++
	case StataIntId:
		sf.recordSize += 2
	case StataLongId:
		sf.recordSize += 4
	case StataFloatId:
		sf.recordSize += 4
	case StataDoubleId:
		sf.recordSize += 8
	default:
		if typ < 1 && typ > 244 {
			panic("unsupported data type " + string(typ) + " in field " + name) //must be a programmer error, so panic
		}
		// string
		sf.recordSize += int(typ)
		format = "%" + string(typ) + "s" //eg %15s
	}
	fld := &Field{
		Name:      name,
		FieldType: typ,
		Label:     label,
		Format:    format,
	}
	sf.fields = append(sf.fields, fld)
	sf.NumVars++
	return fld
}

// WriteTo writes the data to an io.Writer.
// warning: the number of written byte is not used, always zero
func (sf *File) WriteTo(w io.Writer) (int64, error) {
	return 0, sf.writeData(w)
}

func (sf *File) writeHeader(w io.Writer) error {
	// setting the header fields
	sf.NumVars = int16(len(sf.fields))
	return binary.Write(w, littleEndian, *sf.header)
}

func (sf *File) writeDescriptors(w io.Writer) error {
	sf.typList = make([]byte, sf.NumVars)
	sf.varList = make([]stataVarName, sf.NumVars)
	sf.srtList = make([]byte, 2*(sf.NumVars+1))
	sf.fmtList = make([]stataFmtName, sf.NumVars)
	sf.lblList = make([]stataVarName, sf.NumVars)
	sf.vlblList = make([]stataLabel, sf.NumVars)
	for i, f := range sf.fields {
		copy(sf.varList[i][:], f.Name) //only copy up to the size of stataVarName and pad with zeros
		sf.typList[i] = f.FieldType
		copy(sf.fmtList[i][:], f.Format)
		copy(sf.vlblList[i][:], f.Label)
	}

	if err := binary.Write(w, littleEndian, sf.typList); err != nil {
		return err
	}
	if err := binary.Write(w, littleEndian, sf.varList); err != nil {
		return err
	}
	//write an empty sort list
	if err := binary.Write(w, littleEndian, sf.srtList); err != nil {
		return err
	}
	//write var format, for now just generic numberic "%9.0g"
	if err := binary.Write(w, littleEndian, sf.fmtList); err != nil {
		return err
	}
	//write empty value lables
	if err := binary.Write(w, littleEndian, sf.lblList); err != nil {
		return err
	}
	if err := binary.Write(w, littleEndian, sf.vlblList); err != nil {
		return err
	}
	// write an empty expansion field (5 bytes of zeros)
	return binary.Write(w, littleEndian, [5]byte{0, 0, 0, 0, 0})
}

// writeData loops over the field vectors and write their binary representation to an io.Writer
// uses unsafe to  avoid using potentially slower binary.Write.
func (sf *File) writeData(w io.Writer) error {
	if sf.NumObs == 0 {
		return nil
	}
	if len(sf.fields) == 0 {
		return fmt.Errorf("No fields")
	}
	bs := make([]byte, sf.recordSize)
	for i := int32(0); i < sf.NumObs; i++ {
		offset := 0
		for _, f := range sf.fields {
			switch f.FieldType {
			case StataByteId:
				v := f.data.([]Byte)[i]
				bs[offset] = byte(v)
				offset++
			case StataIntId:
				v := f.data.([]Int)[i]
				bs[offset] = byte(v)
				offset++ //incrementing the offset instead of using bs[offset+1] to avoid doing the addition twice
				bs[offset] = byte(v >> 8)
				offset++
			case StataLongId:
				base := *(*[4]byte)(unsafe.Pointer(&f.data.([]Long)[i]))
				copy(bs[offset:], base[:])
				offset += 4
			case StataFloatId:
				base := *(*[4]byte)(unsafe.Pointer(&f.data.([]Float)[i]))
				copy(bs[offset:], base[:])
				offset += 4
			case StataDoubleId:
				base := *(*[8]byte)(unsafe.Pointer(&f.data.([]Double)[i]))
				copy(bs[offset:], base[:])
				offset += 8
			default:
				return fmt.Errorf("Field type [%d] not supported in field %s", f.FieldType, f.Name)
			}
		}
		if _, err := w.Write(bs); err != nil {
			return err
		}
	}
	return nil
}

// BeginWrite must be called once after defining all fields and before writing records
// fileName will be created or truncated if it already exists
// caveat: must set the number of observations before calling this method
// it uses a 64kb buffer as recommended by Microsoft:
// http://technet.microsoft.com/en-us/library/cc938632.aspx
func (sf *File) BeginWrite(fileName string) error {
	var err error
	sf.f, err = os.Create(fileName)
	if err != nil {
		return err
	}
	sf.w = bufio.NewWriterSize(sf.f, 64*1012) //use 64kb buffer

	if err := sf.writeHeader(sf.w); err != nil {
		return err
	}
	if err := sf.writeDescriptors(sf.w); err != nil {
		return err
	}
	sf.recBuf = make([]byte, sf.recordSize)
	sf.offset = 0
	return nil
}

func (sf *File) EndWrite() error {
	if err := sf.w.Flush(); err != nil {
		return err
	}
	// Rewind and write the header with the correct NumObs
	if _, err := sf.f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if err := sf.writeHeader(sf.w); err != nil {
		return err
	}
	if err := sf.w.Flush(); err != nil {
		return err
	}
	return sf.f.Close()
}

// RecordEnd must be called after writing all field data for the record
func (sf *File) RecordEnd() error {
	_, err := sf.w.Write(sf.recBuf)
	// if n != sf.recordSize {
	// fmt.Printf("error writing record, written=%d, recsize=%d\n", n, sf.recordSize)
	// }
	sf.offset = 0
	sf.NumObs++
	return err
}

func (sf *File) AppendByte(v Byte) {
	sf.recBuf[sf.offset] = byte(v)
	sf.offset++
}
func (sf *File) AppendInt(v Int) {
	sf.recBuf[sf.offset] = byte(v)
	sf.offset++
	sf.recBuf[sf.offset] = byte(v >> 8)
	sf.offset++
}
func (sf *File) AppendLong(v Long) {
	base := *(*[4]byte)(unsafe.Pointer(&v)) //convert t to an equivalent byte array
	copy(sf.recBuf[sf.offset:], base[:])
	sf.offset += 4
}
func (sf *File) AppendFloat(v Float) {
	base := *(*[4]byte)(unsafe.Pointer(&v))
	copy(sf.recBuf[sf.offset:], base[:])
	sf.offset += 4
}
func (sf *File) AppendDouble(v Double) {
	base := *(*[8]byte)(unsafe.Pointer(&v))
	copy(sf.recBuf[sf.offset:], base[:])
	sf.offset += 8
}

func (sf *File) AppendStringN(v string, n int) {
    b := []byte(v)
    copy(sf.recBuf[sf.offset:], b[:])
    sf.offset += n
}

func (sf *File) AppendBytesN(v []byte, n int) {
	copy(sf.recBuf[sf.offset:], v[:n])
	sf.offset += n
}

// FIXME: do not overwrite an existing file
// WriteFile
func (sf *File) WriteFile(fileName string) error {
	f, err := os.Create(fileName)
	if err != nil {
		return err
	}
	bw := bufio.NewWriterSize(f, 64*1012) //use 64kb buffer
	if _, err = sf.WriteTo(bw); err != nil {
		f.Close()
		return err
	}
	bw.Flush()
	err = f.Close()
	return err
}
