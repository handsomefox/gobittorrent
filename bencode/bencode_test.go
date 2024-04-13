package bencode

import (
	"reflect"
	"testing"
)

func TestDecodeList(t *testing.T) {
	type args struct {
		encodedValue string
	}
	tests := []struct {
		name     string
		args     args
		wantList List
		wantRest string
		wantErr  bool
	}{
		{
			name:     "Decode l2:hee",
			args:     args{encodedValue: "l2:hee"},
			wantList: List{String("he")},
			wantRest: "",
			wantErr:  false,
		},
		{
			name:     "Decode l2:hee123123",
			args:     args{encodedValue: "l2:hee123123"},
			wantList: List{String("he")},
			wantRest: "123123",
			wantErr:  false,
		},
		{
			name: "Decode nested l5:helloi52el2:hhee",
			args: args{encodedValue: "l5:helloi52el2:hhee"},
			wantList: List{
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
			wantList: List{
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
			gotList, gotRest, err := DecodeList(tt.args.encodedValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotList, tt.wantList) {
				t.Errorf("DecodeList() gotList = %v, want %v", gotList, tt.wantList)
			}
			if gotRest != tt.wantRest {
				t.Errorf("DecodeList() gotRest = %v, want %v", gotRest, tt.wantRest)
			}
		})
	}
}

func TestDecodeInteger(t *testing.T) {
	type args struct {
		encodedValue string
	}
	tests := []struct {
		name        string
		args        args
		wantDecoded Integer
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDecoded, gotRest, err := DecodeInteger(tt.args.encodedValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeInteger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotDecoded != tt.wantDecoded {
				t.Errorf("DecodeInteger() gotDecoded = %v, want %v", gotDecoded, tt.wantDecoded)
			}
			if gotRest != tt.wantRest {
				t.Errorf("DecodeInteger() gotRest = %v, want %v", gotRest, tt.wantRest)
			}
		})
	}
}

func TestDecodeString(t *testing.T) {
	type args struct {
		encodedValue string
	}
	tests := []struct {
		name        string
		args        args
		wantDecoded String
		wantRest    string
		wantErr     bool
	}{
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDecoded, gotRest, err := DecodeString(tt.args.encodedValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotDecoded != tt.wantDecoded {
				t.Errorf("DecodeString() gotDecoded = %v, want %v", gotDecoded, tt.wantDecoded)
			}
			if gotRest != tt.wantRest {
				t.Errorf("DecodeString() gotRest = %v, want %v", gotRest, tt.wantRest)
			}
		})
	}
}

func TestDecodeDictionary(t *testing.T) {
	type args struct {
		encodedValue string
	}
	tests := []struct {
		name     string
		args     args
		wantDict Dictionary
		wantRest string
		wantErr  bool
	}{
		{
			name:     "Decode d3:foo3:bar5:helloi52ee",
			args:     args{encodedValue: "d3:foo3:bar5:helloi52ee"},
			wantDict: Dictionary{String("foo"): String("bar"), String("hello"): Integer(52)},
			wantRest: "",
			wantErr:  false,
		},
		{
			name:     "Decode d3:foo3:bar5:helloi52ee123",
			args:     args{encodedValue: "d3:foo3:bar5:helloi52ee123"},
			wantDict: Dictionary{String("foo"): String("bar"), String("hello"): Integer(52)},
			wantRest: "123",
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDict, gotRest, err := DecodeDictionary(tt.args.encodedValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeDictionary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotDict, tt.wantDict) {
				t.Errorf("DecodeDictionary() gotDict = %v, want %v", gotDict, tt.wantDict)
			}
			if gotRest != tt.wantRest {
				t.Errorf("DecodeDictionary() gotRest = %v, want %v", gotRest, tt.wantRest)
			}
		})
	}
}

func TestInteger_Encode(t *testing.T) {
	tests := []struct {
		name    string
		i       Integer
		want    string
		wantErr bool
	}{
		{name: "Encode 52", i: Integer(52), want: "i52e", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.i.Encode()
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

func TestString_Encode(t *testing.T) {
	tests := []struct {
		name    string
		str     String
		want    string
		wantErr bool
	}{
		{name: "Encode hello", str: "hello", want: "5:hello", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.str.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("String.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("String.Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestList_Encode(t *testing.T) {
	tests := []struct {
		name    string
		list    List
		want    string
		wantErr bool
	}{
		{name: "Encode l5:helloe", list: []Bencodable{String("hello")}, want: "l5:helloe", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.list.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("List.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("List.Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDictionary_Encode(t *testing.T) {
	tests := []struct {
		name    string
		dict    Dictionary
		want    string
		wantErr bool
	}{
		{
			name: "Encode d3:foo3:bar2:hi5:helloe",
			dict: Dictionary{
				"foo": String("bar"),
				"hi":  String("hello"),
			},
			want:    "d3:foo3:bar2:hi5:helloe",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.dict.Encode()
			if (err != nil) != tt.wantErr {
				t.Errorf("Dictionary.Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Dictionary.Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}
