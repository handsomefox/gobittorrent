package bencode

import (
	"bytes"
	"reflect"
	"testing"
)

func TestDecodeBencodable(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantDecoded Bencodable
		wantErr     bool
	}{
		{
			name:        "Decode 52",
			input:       "i52e",
			wantDecoded: Integer(52),
			wantErr:     false,
		},
		{
			name:        "Decode 5:hello",
			input:       "5:hello",
			wantDecoded: String("hello"),
			wantErr:     false,
		},
		{
			name:        "Decode d3:foo3:bar5:helloi52ee",
			input:       "d3:foo3:bar5:helloi52ee",
			wantDecoded: Dictionary{String("foo"): String("bar"), String("hello"): Integer(52)},
			wantErr:     false,
		},
		{
			name:        "Decode l2:hee",
			input:       "l2:hee",
			wantDecoded: List{String("he")},
			wantErr:     false,
		},
		{
			name:  "Decode nested l5:helloi52el2:hhee",
			input: "l5:helloi52el2:hhee",
			wantDecoded: List{
				String("hello"),
				Integer(52),
				List{String("hh")},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decoder := NewDecoder(bytes.NewBufferString(tt.input))
			gotDecoded, err := decoder.Decode()
			if (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotDecoded, tt.wantDecoded) {
				t.Errorf("Decode() gotDecoded = %v, want %v", gotDecoded, tt.wantDecoded)
			}
		})
	}
}

func TestEncodeBencodable(t *testing.T) {
	tests := []struct {
		name    string
		input   Bencodable
		want    []byte
		wantErr bool
	}{
		{name: "Encode 52", input: Integer(52), want: []byte("i52e"), wantErr: false},
		{name: "Encode hello", input: String("hello"), want: []byte("5:hello"), wantErr: false},
		{name: "Encode l5:helloe", input: List{String("hello")}, want: []byte("l5:helloe"), wantErr: false},
		{
			name: "Encode d3:foo3:bar2:hi5:helloe",
			input: Dictionary{
				"foo": String("bar"),
				"hi":  String("hello"),
			},
			want:    []byte("d3:foo3:bar2:hi5:helloe"),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.input.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !bytes.Equal(got, tt.want) {
				t.Errorf("Encode() = %v, want %v", string(got), string(tt.want))
			}
		})
	}
}

func TestEncoderDecoder(t *testing.T) {
	tests := []struct {
		name  string
		input Bencodable
	}{
		{name: "Integer", input: Integer(52)},
		{name: "String", input: String("hello")},
		{name: "List", input: List{String("hello"), Integer(42)}},
		{name: "Dictionary", input: Dictionary{"foo": String("bar"), "answer": Integer(42)}},
		{name: "Nested", input: List{
			String("hello"),
			Integer(42),
			List{String("nested"), Integer(1)},
			Dictionary{"key": String("value")},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			encoder := NewEncoder(&buf)
			err := encoder.Encode(tt.input)
			if err != nil {
				t.Fatalf("Encode() error = %v", err)
			}

			decoder := NewDecoder(&buf)
			decoded, err := decoder.Decode()
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}

			if !reflect.DeepEqual(decoded, tt.input) {
				t.Errorf("EncoderDecoder roundtrip failed. got = %v, want %v", decoded, tt.input)
			}
		})
	}
}
