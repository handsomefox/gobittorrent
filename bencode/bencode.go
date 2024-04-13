package bencode

import (
	"io"
	"sort"
	"strconv"
	"strings"
)

type (
	Decoder struct {
		r io.Reader
	}
	Encoder struct {
		w io.Writer
	}
	Bencodable interface {
		Encode() (string, error)
	}
	Integer    int64
	String     string
	List       []Bencodable
	Dictionary map[String]Bencodable
)

func NewDecoder(r io.Reader) *Decoder { return &Decoder{r: r} }
func NewEncoder(w io.Writer) *Encoder { return &Encoder{w: w} }

func (enc *Encoder) Encode(v Bencodable) error {
	s, err := v.Encode()
	if err != nil {
		return err
	}
	_, err = enc.w.Write([]byte(s))
	return err
}

func (dec *Decoder) Decode() (Bencodable, error) {
	data, err := io.ReadAll(dec.r)
	if err != nil {
		return nil, err
	}

	decoded, _, err := decode(string(data))

	return decoded, err
}

func decode(encodedValue string) (decoded Bencodable, rest string, err error) {
	if len(encodedValue) < 1 {
		return nil, "", NewSyntaxError("bencode: length of the value is 0")
	}

	switch firstCh := encodedValue[0]; firstCh {
	// An integer looks like: 'i52e'
	case 'i':
		return DecodeInteger(encodedValue)
	// A string looks like: '5:hello'
	case '1', '2', '3', '4', '5', '6', '7', '8', '9', '0':
		return DecodeString(encodedValue)
	// A list looks like: 'l5:helloi52ee'
	case 'l':
		return DecodeList(encodedValue)
	case 'd':
		return DecodeDictionary(encodedValue)
	}

	return nil, "", ErrUnknownValueType
}

func DecodeInteger(input string) (i Integer, rest string, err error) {
	encodedInteger := input

	end := strings.Index(encodedInteger, "e")
	if end == -1 {
		return 0, "",
			NewSyntaxErrorf("bencode: failed to find 'e' when decoding an integer value (%q)\n", encodedInteger)
	}

	encodedInteger = encodedInteger[1:]
	encodedInteger = encodedInteger[:end-1]

	integer, err := strconv.ParseInt(encodedInteger, 10, 32)
	if err != nil {
		return 0, "", NewSyntaxErrorf("bencode: the provided value (%q) was encoded like an integer, but was not an integer, error: %s\n", input, err)
	}

	return Integer(integer), input[end+1:], nil
}

func DecodeString(input string) (str String, rest string, err error) {
	split := strings.SplitN(input, ":", 2)
	if len(split) < 2 {
		return "", "", NewSyntaxErrorf("bencode: failed to find ':' while decoding value (%q)\n", input)
	}

	lengthStr, rest := split[0], split[1]
	length, err := strconv.ParseInt(lengthStr, 10, 32)
	if err != nil {
		return "", "", NewSyntaxErrorf("bencode: failed to decode the length value (%q), error: %s\n", lengthStr, err)
	}

	return String(rest[:length]), rest[length:], nil
}

func DecodeList(input string) (list List, rest string, err error) {
	listValues := input[1:] // remove the 'l'
	list = make(List, 0)

	for {
		decoded, rest, err := decode(listValues)
		if err != nil {
			return nil, "", err
		}
		list = append(list, decoded)

		if strings.HasPrefix(rest, "e") {
			rest = rest[1:]

			return list, rest, nil
		}

		if rest == "" {
			return nil, "", NewSyntaxErrorf("bencode: failed to decode the list (%q), because it is not properly terminated\n", input)
		}

		listValues = rest
	}
}

func DecodeDictionary(input string) (dict Dictionary, rest string, err error) {
	// Just replace d with an l and decode the list instead, then transform into a map :)
	input = input[1:]
	input = "l" + input

	list, rest, err := DecodeList(input)
	if err != nil {
		return nil, rest, err
	}

	if len(list)%2 != 0 {
		return nil, rest, NewSyntaxErrorf("bencode: incorrect amount of items for a map (%d)\n", len(list))
	}

	dict = make(Dictionary)

	for i := 1; i < len(list); i += 2 {
		key, value := list[i-1], list[i]
		keyStr, ok := key.(String)
		if !ok {
			return nil, "", NewSyntaxErrorf("bencode: failed to decode a map key (%q), it supposed to be a byte slice", key)
		}

		dict[keyStr] = value
	}

	return dict, rest, nil
}

func (i Integer) Encode() (string, error) {
	return "i" + strconv.FormatInt(int64(i), 10) + "e", nil
}

func (str String) Encode() (string, error) {
	s := string(str)
	return strconv.FormatInt(int64(len(s)), 10) + ":" + s, nil
}

func (list List) Encode() (string, error) {
	s := "l"

	for _, item := range list {
		str, err := item.Encode()
		if err != nil {
			return "", err
		}
		s += str
	}

	return s + "e", nil
}

func (dict Dictionary) Encode() (string, error) {
	keys := make([]String, 0, len(dict))

	for k := range dict {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		return string(keys[i]) < string(keys[j])
	})

	s := "d"

	for _, k := range keys {
		key, err := k.Encode()
		if err != nil {
			return "", err
		}
		value, err := dict[k].Encode()
		if err != nil {
			return "", err
		}
		s += key + value
	}

	return s + "e", nil
}
