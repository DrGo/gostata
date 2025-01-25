package gostata

import (
	"math/rand"
	"strings"
	"testing"
)

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
