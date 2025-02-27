package gostata

import (
	"math/rand"
	"os"
	"strings"
	"testing"
	"unsafe"

	"github.com/matryer/is"
)

// BytesToString converts byte slice to string without allocation
func BytesToString(b []byte) string {
	// Ignore if your IDE shows an error here; it's a false positive.
	p := unsafe.SliceData(b)
	return unsafe.String(p, len(b))
}

// StringToBytes converts string to byte slice without allocation
func StringToBytes(s string) []byte {
	p := unsafe.StringData(s)
	b := unsafe.Slice(p, len(s))
	return b
}

func TestRecordWrite(t *testing.T) {
	is := is.New(t)
	sf := NewFile()
	// sf.SetNumObs(int32(2))
	sf.AddFieldMeta("bytefld", "byte field", StataByteId)
	sf.AddFieldMeta("intfld", "int field", StataIntId)
	sf.AddFieldMeta("str9fld", "str 9 field", 9)
	sf.AddFieldMeta("doublefld", "double field", StataDoubleId)

	is.NoErr(sf.BeginWrite(getTestingPath("empty_records.dta")))
	is.NoErr(sf.EndWrite())
	//TODO check stata output

	// var buf [9]byte
	is.NoErr(sf.BeginWrite(getTestingPath("two_records.dta")))

	sf.AppendByte(1)
	sf.AppendInt(999)
	sf.AppendStringN( "123456789", 9)
	sf.AppendDouble(6.284)
	is.NoErr(sf.RecordEnd())

	sf.AppendByte(2)
	sf.AppendInt(9999)
	// copy(buf[:], "1234567\x00") //must end string by \x00 if < field len
	sf.AppendStringN("1234567\x00", 9)
	sf.AppendDouble(3.142)
	is.NoErr(sf.RecordEnd())
	is.NoErr(sf.EndWrite())

	dict, err := RunScript(testDir, `
	qui {
    use two_records.dta
    count 
    noi di "N="r(N)     
    noi di "bytefld[1]=" bytefld[1]
    noi di "str9fld[2]=" str9fld[2]
    noi di "doublefld[2]=" doublefld[2]
	}	
	`)
	if err != nil {
		t.Fatalf("error running stata script from TestRecordWrite: %s", err)
	}

	if value := dict["N"]; value != "2" {
		t.Errorf("Expected N=2, found %s", value)
	}
	if value := dict["bytefld[1]"]; value != "1" {
		t.Errorf("Expected bytefld[1]=1, found %s", value)
	}
	if value := dict["str9fld[2]"]; value != "1234567" {
		t.Errorf("Expected str9fld[2]=1234567, found %s", value)
	}
	if value := dict["doublefld[2]"]; value != "3.142" {
		t.Errorf("Expected doublefld[1]=3.142, found %s", value)
	}
}

func TestFile_WriteTo(t *testing.T) {
	sf := NewFile()
	i8 := []Byte{1, 2, 3, 4, 5, 6}
	sf.AddField("i8", "int8", i8)
	i9 := []Byte{1, 2, 3, 4, 5, 6}
	sf.AddField("i9", "int8", i9)
	i16 := []Int{100, 200, 300, 400, 500, 600}
	sf.AddField("i16", "int16", i16)
	i32 := []Long{6000000, 7000000, 3000000, 4000000, 5000000, 6000000}
	sf.AddField("i32", "int32", i32)
	f32 := []Float{6.5, 7.5, 3.5, 4.5, 5.5, 6.5}
	sf.AddField("f32", "float32", f32)
	f64 := []Double{6.5, 7.5, 3.5, 4.5, 5.5, 6.5}
	sf.AddField("f64", "float64", f64)

	if err := sf.WriteFile("small.dta"); err != nil {
		t.Fatal(err)
	}
	output, err := RunStataDo(testDir, "do.do")
	if err != nil {
		t.Fatal(err)
	}
	dict := GetKeyValuePairs(output)
	t.Logf("%v", dict)
	if value := dict["N"]; value != "6" {
		t.Errorf("Expected N=5, found %s", value)
	}
	if value := dict["mean(i8)"]; value != "3.5" {
		t.Errorf("Expected mean(i8)=3.5, found %s", value)
	}

}

func TestFile_WriteToLarge(t *testing.T) {
	const N = 1e5
	sf := NewFile()
	f64 := make([]Double, N)
	for i := 0; i < N; i++ {
		f64[i] = Double(rand.NormFloat64())
	}
	sf.AddField("f64", "float64", f64)

	if err := sf.WriteFile("large.dta"); err != nil {
		t.Fatal(err)
	}
	output, err := RunStataDo(testDir, "large.do")
	if err != nil {
		t.Fatal(err)
	}
	dict := GetKeyValuePairs(output)
	t.Logf("%v", dict)
	if value := dict["N"]; value != "100000" {
		t.Errorf("Expected N=100000, found %s", value)
	}
	if value := dict["mean(f64)"]; strings.HasPrefix(value, ".0000") {
		t.Errorf("Expected mean(f64)=0, found %s", value)
	}

}

type testStruct struct {
	Name    string  `stata:"name:my_name,label:My Name,typ:str10"`
	Age     int     `stata:"label:Age in Years,typ:int"`
	Height  float64 `stata:"label:Height (meters),typ:double,format:%6.2f"`
	IsValid bool    `stata:"typ:byte"` // Example of a boolean field
}

func TestWriteStataFromStruct(t *testing.T) {
	is := is.New(t)
	data := testStruct{}
	sf, err := NewFileFromStruct(data)
	is.NoErr(err)
	fileName := getTestingPath("fromstruct.dta")
	is.NoErr(sf.BeginWrite(fileName))
	is.NoErr(sf.EndWrite())
	// err = sf.BeginWrite(fileName)
	// if err != nil {
	//         t.Fatal(err)
	// }
	// err = sf.EndWrite()
	// if err != nil {
	//         t.Fatal(err)
	// }
	//
	// Basic check: file should exist
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		t.Errorf("File %s not created", fileName)
	}

	// You would typically run Stata here and then verify the data
	// For example, using a system call to Stata in your test:
	// cmd := exec.Command("stata", "-b", "do", "verify_stata.do") // verify_stata.do would check the file
	// if err := cmd.Run(); err != nil {
	//  t.Errorf("Stata verification failed: %v", err)
	// }
}

// The default number generator is deterministic, so it'll
// produce the same sequence of numbers each time by default.
// To produce varying sequences, give it a seed that changes.
// Note that this is not safe to use for random numbers you
// intend to be secret, use `crypto/rand` for those.
// s1 := rand.NewSource(time.Now().UnixNano())
// r1 := rand.New(s1)

// // The tabwriter here helps us generate aligned output.
// w := tabwriter.NewWriter(os.Stdout, 1, 1, 1, ' ', 0)
// defer w.Flush()
// show := func(name string, v1, v2, v3 interface{}) {
// 	fmt.Fprintf(w, "%s\t%v\t%v\t%v\n", name, v1, v2, v3)
// }

// //for testing
// func printBuffer(sf *File, t *testing.T) error {
// 	var b bytes.Buffer
// 	w := bufio.NewWriter(&b)
// 	//	err := sf.WriteTo(w)
// 	err := sf.writeHeader(w)
// 	w.Flush()
// 	if len(b.Bytes()) != 109 {
// 		t.Errorf("wrong number of bytes in emitting header: %d\n", len(b.Bytes()))
// 	}
// 	fmt.Printf("len: %d\n", len(b.Bytes()))
// 	fmt.Printf("buf: %v\n", b.Bytes())

// 	b.Reset()
// 	err = sf.writeDescriptors(w)
// 	w.Flush()
// 	// len of descriptors=nvar x (1+ 33+ 12+33 + 81) + 2 (nvar+1) + 5 --> nvar * 160 + 2 * (nvar+1) + 5; 1 var=169
// 	if len(b.Bytes()) != 169 {
// 		t.Errorf("wrong number of bytes in emitting descriptors: %d\n", len(b.Bytes()))
// 	}
// 	fmt.Printf("len: %d\n", len(b.Bytes()))
// 	fmt.Printf("buf: %v\n", b.Bytes())

// 	return err
// }
