package bencode

import (
	"fmt"
	"net/url"
	"strconv"
)

type AnnounceMessage struct {
	Announce   string
	InfoHash   string // the info hash of the torrent
	PeerID     string // a unique identifier for the client
	Port       int64  // 6881 - the port client is listening on
	Uploaded   int64  // 0 - the total amount uploaded so far
	Downloaded int64  // 0 - the total amount downloaded so far
	Left       int64  // the number of bytes left to download
	Compact    int64  // 1 - whether the peer list should use the compact representation
}

func (req *AnnounceMessage) URL() (string, error) {
	u, err := url.Parse(req.Announce)
	if err != nil {
		return "", fmt.Errorf("%w %q, because %w", ErrParseAnnounceURL, req.Announce, err)
	}
	var (
		// This is stupid, but it works :)
		// It's basically a map for A-z and - . ~ _ hex values of ascii characters
		unreservedChars = map[[2]byte]byte{
			{'4', '1'}: 'A', {'4', '2'}: 'B', {'4', '3'}: 'C', {'4', '4'}: 'D', {'4', '5'}: 'E', {'4', '6'}: 'F',
			{'4', '7'}: 'G', {'4', '8'}: 'H', {'4', '9'}: 'I', {'4', 'A'}: 'J', {'4', 'B'}: 'K', {'4', 'C'}: 'L',
			{'4', 'D'}: 'M', {'4', 'E'}: 'N', {'4', 'F'}: 'O', {'5', '0'}: 'P', {'5', '1'}: 'Q', {'5', '2'}: 'R',
			{'5', '3'}: 'S', {'5', '4'}: 'T', {'5', '5'}: 'U', {'5', '6'}: 'V', {'5', '7'}: 'W', {'5', '8'}: 'X',
			{'5', '9'}: 'Y', {'5', 'A'}: 'Z', {'6', '1'}: 'a', {'6', '2'}: 'b', {'6', '3'}: 'c', {'6', '4'}: 'd',
			{'6', '5'}: 'e', {'6', '6'}: 'f', {'6', '7'}: 'g', {'6', '8'}: 'h', {'6', '9'}: 'i', {'6', 'A'}: 'j',
			{'6', 'B'}: 'k', {'6', 'C'}: 'l', {'6', 'D'}: 'm', {'6', 'E'}: 'n', {'6', 'F'}: 'o', {'7', '0'}: 'p',
			{'7', '1'}: 'q', {'7', '2'}: 'r', {'7', '3'}: 's', {'7', '4'}: 't', {'7', '5'}: 'u', {'7', '6'}: 'v',
			{'7', '7'}: 'w', {'7', '8'}: 'x', {'7', '9'}: 'y', {'7', 'A'}: 'z', {'5', 'F'}: '_', {'7', 'E'}: '~',
			{'2', 'E'}: '.', {'2', 'D'}: '-',
		}
		encodedHash = ""
		bytes       = []byte(req.InfoHash)
	)
	for i := 1; i < len(req.InfoHash); i += 2 {
		first, second := bytes[i-1], bytes[i]
		if value, ok := unreservedChars[[2]byte{first, second}]; ok {
			encodedHash += string(value)
		} else {
			encodedHash += "%" + string(first) + string(second)
		}
	}

	q := u.Query()
	q.Set("peer_id", req.PeerID)
	q.Set("port", strconv.FormatInt(req.Port, 10))
	q.Set("uploaded", strconv.FormatInt(req.Uploaded, 10))
	q.Set("downloaded", strconv.FormatInt(req.Downloaded, 10))
	q.Set("left", strconv.FormatInt(req.Left, 10))
	q.Set("compact", strconv.FormatInt(req.Compact, 10))
	u.RawQuery = q.Encode()

	return u.String() + "&info_hash=" + encodedHash, nil
}

type AnnounceResponse struct {
	Peers    []Peer // []String - Host:Port (When decoding: the first 4 bytes are the peer's IP address and the last 2 bytes are the peer's port number)
	Interval int64  // how often your client should make a request to the tracker
}

func (r *AnnounceResponse) Unmarshal(decodedValues any) error {
	decodedMap, ok := decodedValues.(map[string]any)
	if !ok {
		return fmt.Errorf("%w, values (%q)", ErrConvertDecoded, decodedValues)
	}

	interval, ok := decodedMap["interval"].(int64)
	if !ok {
		return ConvertError{ValueName: "interval", WantedType: "int64"}
	}

	peers, ok := decodedMap["peers"].(string)
	if !ok {
		return ConvertError{ValueName: "peers", WantedType: "string"}
	}

	r.Interval = interval

	start := 0
	peersBytes := []byte(peers)
	for i := 1; i <= len(peersBytes); i++ {
		if i%6 != 0 {
			continue
		}

		p, err := NewPeer(peersBytes[start:i])
		if err != nil {
			return err
		}
		r.Peers = append(r.Peers, p)
		start = i
	}

	return nil
}
