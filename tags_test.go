package gostata 

import (
	"reflect"
	"testing"
)

// TestAllTags tests a struct where all tags are provided.
type TestAllTags struct {
	A string  `stata:"name:Alpha,label:First Field,typ:str10,format:%9s"`
	B int     `stata:"typ:int"`
	C float64 `stata:"label:Currency,typ:double,format:%.2f"`
}

func TestExtractFields_AllTags(t *testing.T) {
	s := TestAllTags{
		A: "hello",
		B: 42,
		C: 3.14159,
	}
	fields, err := ExtractFields(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(fields))
	}

	// Field A: name "Alpha", label "First Field", typ "str10" -> 10, format "%9s"
	var fieldA *Field
	for _, f := range fields {
		if f.Name == "Alpha" {
			fieldA = f
			break
		}
	}
	if fieldA.Name != "Alpha" {
		t.Errorf("expected name 'Alpha', got %q", fieldA.Name)
	}
	if fieldA.Label != "First Field" {
		t.Errorf("expected label 'First Field', got %q", fieldA.Label)
	}
	if fieldA.FieldType != 10 {
		t.Errorf("expected FieldType 10, got %d", fieldA.FieldType)
	}
	if fieldA.Format != "%9s" {
		t.Errorf("expected format '%%9s', got %q", fieldA.Format)
	}
	if fieldA.data != s.A {
		t.Errorf("expected data %v, got %v", s.A, fieldA.data)
	}
}

// TestMissingNameLabel tests a struct where name and label are not provided,
// so they default to the lowercase field name.
type TestMissingNameLabel struct {
	X bool `stata:"typ:byte"`
}

func TestExtractFields_MissingNameLabel(t *testing.T) {
	s := TestMissingNameLabel{X: true}
	fields, err := ExtractFields(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(fields))
	}
	f := fields[0]
	if f.Name != "x" {
		t.Errorf("expected name 'x', got %q", f.Name)
	}
	if f.Label != "x" {
		t.Errorf("expected label 'x', got %q", f.Label)
	}
	if f.FieldType != 251 {
		t.Errorf("expected FieldType 251 for 'byte', got %d", f.FieldType)
	}
}

// TestMissingTyp tests that a field missing the required "typ" tag causes an error.
type TestMissingTyp struct {
	Y int `stata:"name:Beta,label:Second Field"`
}

func TestExtractFields_MissingTyp(t *testing.T) {
	s := TestMissingTyp{Y: 100}
	_, err := ExtractFields(s)
	if err == nil {
		t.Fatal("expected an error for missing 'typ' tag, got nil")
	}
}

// TestNoStataTag tests that a struct with no stata tags returns an error.
type TestNoStataTag struct {
	Z string
}

func TestExtractFields_NoStataTag(t *testing.T) {
	s := TestNoStataTag{Z: "test"}
	_, err := ExtractFields(s)
	if err == nil {
		t.Fatal("expected an error for no stata tags, got nil")
	}
}

// TestPointer tests that passing a pointer to a struct works correctly.
func TestExtractFields_Pointer(t *testing.T) {
	s := &TestAllTags{
		A: "pointer",
		B: 99,
		C: 2.718,
	}
	fields, err := ExtractFields(s)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(fields))
	}
	// Compare using reflect.DeepEqual on the data fields.
	for i, f := range fields {
		expected := reflect.ValueOf(s).Elem().Field(i).Interface()
		if !reflect.DeepEqual(f.data, expected) {
			t.Errorf("field %d: expected data %v, got %v", i, expected, f.data)
		}
	}
}

