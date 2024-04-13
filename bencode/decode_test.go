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
		wantList []any
		wantRest string
		wantErr  bool
	}{
		{
			name:     "Decode l2:hee",
			args:     args{encodedValue: "l2:hee"},
			wantList: []any{"he"},
			wantRest: "",
			wantErr:  false,
		},
		{
			name:     "Decode l2:hee123123",
			args:     args{encodedValue: "l2:hee123123"},
			wantList: []any{"he"},
			wantRest: "123123",
			wantErr:  false,
		},
		{
			name: "Decode nested l5:helloi52el2:hhee",
			args: args{encodedValue: "l5:helloi52el2:hhee"},
			wantList: []any{
				"hello",
				int64(52),
				[]any{"hh"},
			},
			wantRest: "",
			wantErr:  false,
		},
		{
			name: "Decode nested l5:helloi52el2:hhee123123",
			args: args{encodedValue: "l5:helloi52el2:hhee123123"},
			wantList: []any{
				"hello",
				int64(52),
				[]any{"hh"},
			},
			wantRest: "123123",
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotList, gotRest, err := NewDecoder(nil).decodeList(tt.args.encodedValue)
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
		wantDecoded int64
		wantRest    string
		wantErr     bool
	}{
		{
			name:        "Decode 52",
			args:        args{encodedValue: "i52e"},
			wantDecoded: int64(52),
			wantRest:    "",
			wantErr:     false,
		},
		{
			name:        "Decode 52 with the rest",
			args:        args{encodedValue: "i52e123123"},
			wantDecoded: int64(52),
			wantRest:    "123123",
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDecoded, gotRest, err := NewDecoder(nil).decodeInteger(tt.args.encodedValue)
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
		wantDecoded string
		wantRest    string
		wantErr     bool
	}{
		{
			name:        "Decode 5:hello",
			args:        args{encodedValue: "5:hello"},
			wantDecoded: "hello",
			wantRest:    "",
			wantErr:     false,
		},
		{
			name:        "Decode 5:hello123123",
			args:        args{encodedValue: "5:hello123123"},
			wantDecoded: "hello",
			wantRest:    "123123",
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDecoded, gotRest, err := NewDecoder(nil).decodeString(tt.args.encodedValue)
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
		wantDict map[string]any
		wantRest string
		wantErr  bool
	}{
		{
			name:     "Decode d3:foo3:bar5:helloi52ee",
			args:     args{encodedValue: "d3:foo3:bar5:helloi52ee"},
			wantDict: map[string]any{"foo": "bar", "hello": int64(52)},
			wantRest: "",
			wantErr:  false,
		},
		{
			name:     "Decode d3:foo3:bar5:helloi52ee123",
			args:     args{encodedValue: "d3:foo3:bar5:helloi52ee123"},
			wantDict: map[string]any{"foo": "bar", "hello": int64(52)},
			wantRest: "123",
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotDict, gotRest, err := NewDecoder(nil).decodeDictionary(tt.args.encodedValue)
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
