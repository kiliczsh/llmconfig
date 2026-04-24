package downloader

import "context"

type Request struct {
	Repo           string
	File           string
	URL            string
	Token          string
	ModelDir       string
	Connections    int
	Resume         bool
	Checksum       string
	VerifyChecksum bool
}

type Downloader interface {
	Download(ctx context.Context, req *Request, onProgress func(downloaded, total int64)) (string, error)
}

func New() Downloader {
	return &httpDownloader{}
}
