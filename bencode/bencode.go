package bencode

import (
	"bufio"
	"bytes"
	"io"
	"sort"
	"strconv"
)

type (
	Decoder struct {
		r *bufio.Reader
	}
	Encoder struct {
		w io.Writer
	}
	Bencodable interface {
		Encode() ([]byte, error)
	}
	Integer    int64
	String     string
	List       []Bencodable
	Dictionary map[String]Bencodable
)

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: bufio.NewReader(r)}
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

func (enc *Encoder) Encode(v Bencodable) error {
	b, err := v.Encode()
	if err != nil {
		return err
	}
	_, err = enc.w.Write(b)
	return err
}

func (dec *Decoder) Decode() (Bencodable, error) {
	return dec.decodeNext()
}

func (dec *Decoder) decodeNext() (Bencodable, error) {
	b, err := dec.r.Peek(1)
	if err != nil {
		return nil, err
	}

	switch b[0] {
	case 'i':
		return dec.decodeInteger()
	case 'l':
		return dec.decodeList()
	case 'd':
		return dec.decodeDictionary()
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return dec.decodeString()
	default:
		return nil, ErrUnknownValueType
	}
}

func (dec *Decoder) decodeInteger() (Integer, error) {
	_, err := dec.r.ReadByte() // consume 'i'
	if err != nil {
		return 0, err
	}

	var buf bytes.Buffer
	for {
		b, err := dec.r.ReadByte()
		if err != nil {
			return 0, err
		}
		if b == 'e' {
			break
		}
		buf.WriteByte(b)
	}

	i, err := strconv.ParseInt(buf.String(), 10, 64)
	if err != nil {
		return 0, err
	}
	return Integer(i), nil
}

func (dec *Decoder) decodeString() (String, error) {
	var length int64
	for {
		b, err := dec.r.ReadByte()
		if err != nil {
			return "", err
		}
		if b == ':' {
			break
		}
		length = length*10 + int64(b-'0')
	}

	buf := make([]byte, length)
	_, err := io.ReadFull(dec.r, buf)
	if err != nil {
		return "", err
	}
	return String(buf), nil
}

func (dec *Decoder) decodeList() (List, error) {
	_, err := dec.r.ReadByte() // consume 'l'
	if err != nil {
		return nil, err
	}

	var list List
	for {
		b, err := dec.r.Peek(1)
		if err != nil {
			return nil, err
		}
		if b[0] == 'e' {
			dec.r.ReadByte() // consume 'e'
			return list, nil
		}
		item, err := dec.decodeNext()
		if err != nil {
			return nil, err
		}
		list = append(list, item)
	}
}

func (dec *Decoder) decodeDictionary() (Dictionary, error) {
	_, err := dec.r.ReadByte() // consume 'd'
	if err != nil {
		return nil, err
	}

	dict := make(Dictionary)
	for {
		b, err := dec.r.Peek(1)
		if err != nil {
			return nil, err
		}
		if b[0] == 'e' {
			dec.r.ReadByte() // consume 'e'
			return dict, nil
		}
		key, err := dec.decodeString()
		if err != nil {
			return nil, err
		}
		value, err := dec.decodeNext()
		if err != nil {
			return nil, err
		}
		dict[key] = value
	}
}

func (i Integer) Encode() ([]byte, error) {
	return []byte("i" + strconv.FormatInt(int64(i), 10) + "e"), nil
}

func (s String) Encode() ([]byte, error) {
	return []byte(strconv.Itoa(len(s)) + ":" + string(s)), nil
}

func (l List) Encode() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('l')
	for _, item := range l {
		b, err := item.Encode()
		if err != nil {
			return nil, err
		}
		buf.Write(b)
	}
	buf.WriteByte('e')
	return buf.Bytes(), nil
}

func (d Dictionary) Encode() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('d')

	keys := make([]String, 0, len(d))
	for k := range d {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		return string(keys[i]) < string(keys[j])
	})

	for _, k := range keys {
		kb, err := k.Encode()
		if err != nil {
			return nil, err
		}
		buf.Write(kb)

		vb, err := d[k].Encode()
		if err != nil {
			return nil, err
		}
		buf.Write(vb)
	}
	buf.WriteByte('e')
	return buf.Bytes(), nil
}
