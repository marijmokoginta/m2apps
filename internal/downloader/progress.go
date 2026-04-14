package downloader

import "io"

type ProgressReader struct {
	Reader     io.Reader
	Total      int64
	ReadBytes  int64
	OnProgress func(read, total int64)
}

func (p *ProgressReader) Read(b []byte) (int, error) {
	n, err := p.Reader.Read(b)
	if n > 0 {
		p.ReadBytes += int64(n)
		if p.OnProgress != nil {
			p.OnProgress(p.ReadBytes, p.Total)
		}
	}
	return n, err
}
