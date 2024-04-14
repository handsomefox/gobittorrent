package bencode

import (
	"reflect"
	"testing"
)

func TestDecodeBencodable(t *testing.T) {
	type args struct {
		encodedValue string
	}
	tests := []struct {
		name        string
		args        args
		wantDecoded Bencodable
		wantRest    string
		wantErr     bool
	}{
		{
			name:        "Decode 52",
			args:        args{encodedValue: "i52e"},
			wantDecoded: Integer(52),
			wantRest:    "",
			wantErr:     false,
		},
		{
			name:        "Decode 52 with the rest",
			args:        args{encodedValue: "i52e123123"},
			wantDecoded: Integer(52),
			wantRest:    "123123",
			wantErr:     false,
		},
		{
			name:        "Decode 5:hello",
			args:        args{encodedValue: "5:hello"},
			wantDecoded: String("hello"),
			wantRest:    "",
			wantErr:     false,
		},
		{
			name:        "Decode 5:hello123123",
			args:        args{encodedValue: "5:hello123123"},
			wantDecoded: String("hello"),
			wantRest:    "123123",
			wantErr:     false,
		},
		{
			name:        "Decode d3:foo3:bar5:helloi52ee",
			args:        args{encodedValue: "d3:foo3:bar5:helloi52ee"},
			wantDecoded: Dictionary{String("foo"): String("bar"), String("hello"): Integer(52)},
			wantRest:    "",
			wantErr:     false,
		},
		{
			name:        "Decode d3:foo3:bar5:helloi52ee123",
			args:        args{encodedValue: "d3:foo3:bar5:helloi52ee123"},
			wantDecoded: Dictionary{String("foo"): String("bar"), String("hello"): Integer(52)},
			wantRest:    "123",
			wantErr:     false,
		},
		{
			name:        "Decode l2:hee",
			args:        args{encodedValue: "l2:hee"},
			wantDecoded: List{String("he")},
			wantRest:    "",
			wantErr:     false,
		},
		{
			name:        "Decode l2:hee123123",
			args:        args{encodedValue: "l2:hee123123"},
			wantDecoded: List{String("he")},
			wantRest:    "123123",
			wantErr:     false,
		},
		{
			name: "Decode nested l5:helloi52el2:hhee",
			args: args{encodedValue: "l5:helloi52el2:hhee"},
			wantDecoded: List{
				String("hello"),
				Integer(52),
				List{String("hh")},
			},
			wantRest: "",
			wantErr:  false,
		},
		{
			name: "Decode nested l5:helloi52el2:hhee123123",
			args: args{encodedValue: "l5:helloi52el2:hhee123123"},
			wantDecoded: List{
				String("hello"),
				Integer(52),
				List{String("hh")},
			},
			wantRest: "123123",
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDecoded, gotRest, err := decode(tt.args.encodedValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("decode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotDecoded, tt.wantDecoded) {
				t.Errorf("decode() gotDecoded = %v, want %v", gotDecoded, tt.wantDecoded)
			}
			if gotRest != tt.wantRest {
				t.Errorf("decode() gotRest = %v, want %v", gotRest, tt.wantRest)
			}
		})
	}
}

func TestEncodeBencodable(t *testing.T) {
	tests := []struct {
		name    string
		input   Bencodable
		want    string
		wantErr bool
	}{
		{name: "Encode 52", input: Integer(52), want: "i52e", wantErr: false},
		{name: "Encode hello", input: String("hello"), want: "5:hello", wantErr: false},
		{name: "Encode l5:helloe", input: List{String("hello")}, want: "l5:helloe", wantErr: false},
		{
			name: "Encode d3:foo3:bar2:hi5:helloe",
			input: Dictionary{
				"foo": String("bar"),
				"hi":  String("hello"),
			},
			want:    "d3:foo3:bar2:hi5:helloe",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.input.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("Integer.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Integer.Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}
