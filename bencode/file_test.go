package bencode

import "testing"

func TestDecodeTorrentFile(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "Decode sample.torrent",
			args:    args{path: "testdata/sample.torrent"},
			want:    "Tracker URL: http://bittorrent-test-tracker.codecrafters.io/announce\nLength: 92063\nInfo Hash: d69f91e6b2ae4c542468d1073a71d4ea13879a7f\nPiece Length: 32768\nPiece Hashes:\ne876f67a2a8886e8f36b136726c30fa29703022d\n6e2275e604a0766656736e81ff10b55204ad8d35\nf00d937a0213df1982bc8d097227ad9e909acc17",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecodeTorrentFile(tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecodeTorrentFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DecodeTorrentFile() = %v, want %v", got, tt.want)
			}
		})
	}
}
