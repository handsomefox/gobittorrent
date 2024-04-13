package bencode

import (
	"io"
	"strconv"
	"strings"
)

type Decoder struct {
	r    io.Reader
	data []byte
}

func NewDecoder(r io.Reader) *Decoder { return &Decoder{r: r, data: nil} }

func (dec *Decoder) Decode() (any, error) {
	data, err := io.ReadAll(dec.r)
	if err != nil {
		return nil, err
	}

	decoded, _, err := dec.decode(string(data))

	return decoded, err
}

func (dec *Decoder) decode(encodedValue string) (decoded any, rest string, err error) {
	if len(encodedValue) < 1 {
		return "", "", NewSyntaxError("bencode: length of the value is 0")
	}

	switch firstCh := encodedValue[0]; firstCh {
	// An integer looks like: 'i52e'
	case 'i':
		return dec.decodeInteger(encodedValue)
	// A string looks like: '5:hello'
	case '1', '2', '3', '4', '5', '6', '7', '8', '9', '0':
		return dec.decodeString(encodedValue)
	// A list looks like: 'l5:helloi52ee'
	case 'l':
		return dec.decodeList(encodedValue)
	case 'd':
		return dec.decodeDictionary(encodedValue)
	}

	return "", "", ErrUnknownValueType
}

func (dec *Decoder) decodeDictionary(encodedValue string) (dict map[string]any, rest string, err error) {
	// Just replace d with an l and decode the list instead, then transform into a map :)
	encodedValue = encodedValue[1:]
	encodedValue = "l" + encodedValue

	list, rest, err := dec.decodeList(encodedValue)
	if err != nil {
		return nil, rest, err
	}
	if len(list)%2 != 0 {
		return nil, rest, NewSyntaxErrorf("bencode: incorrect amount of items for a map (%d)\n", len(list))
	}

	dict = make(map[string]any)
	for i := 1; i < len(list); i += 2 {
		key, value := list[i-1], list[i]
		keyStr, ok := key.(string)
		if !ok {
			return nil, "",
				NewSyntaxErrorf("bencode: failed to decode a map key (%q), it supposed to be a byte slice", key)
		}

		dict[keyStr] = value
	}

	return dict, rest, nil
}

func (dec *Decoder) decodeList(encodedValue string) (list []any, rest string, err error) {
	listValues := encodedValue[1:] // remove the 'l'
	list = make([]any, 0)

	for {
		decoded, rest, err := dec.decode(listValues)
		if err != nil {
			return nil, "", err
		}
		list = append(list, decoded)

		if strings.HasPrefix(rest, "e") {
			rest = rest[1:]
			return list, rest, nil
		}

		if rest == "" {
			return nil, "", NewSyntaxErrorf("bencode: failed to decode the list (%q), because it is not properly terminated\n", encodedValue)
		}

		listValues = rest
	}
}

func (dec *Decoder) decodeInteger(encodedValue string) (decoded int64, rest string, err error) {
	encodedInteger := encodedValue

	end := strings.Index(encodedInteger, "e")
	if end == -1 {
		return 0, "",
			NewSyntaxErrorf("bencode: failed to find 'e' when decoding an integer value (%q)\n", encodedInteger)
	}

	encodedInteger = encodedInteger[1:]
	encodedInteger = encodedInteger[:end-1]

	integer, err := strconv.ParseInt(encodedInteger, 10, 32)
	if err != nil {
		return 0, "", NewSyntaxErrorf("bencode: the provided value (%q) was encoded like an integer, but was not an integer, error: %s\n", encodedValue, err)
	}

	return integer, encodedValue[end+1:], nil
}

func (dec *Decoder) decodeString(encodedValue string) (decoded, rest string, err error) {
	split := strings.SplitN(encodedValue, ":", 2)
	if len(split) < 2 {
		return "", "", NewSyntaxErrorf("bencode: failed to find ':' while decoding value (%q)\n", encodedValue)
	}

	lengthStr, rest := split[0], split[1]
	length, err := strconv.ParseInt(lengthStr, 10, 32)
	if err != nil {
		return "", "", NewSyntaxErrorf("bencode: failed to decode the length value (%q), error: %s\n", lengthStr, err)
	}

	return rest[:length], rest[length:], nil
}
