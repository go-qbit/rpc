package rpc

import (
	"bytes"
	"io"
	"mime/multipart"
	"os"
)

var (
	_ File = &buffer{}
	_ File = &tmpFile{}
)

type File interface {
	io.ReadCloser
	FileName() string
	Size() int64
}

type buffer struct {
	bytes.Buffer
	filename string
}

func (b *buffer) Size() int64      { return int64(b.Buffer.Len()) }
func (b *buffer) FileName() string { return b.filename }
func (b *buffer) Close() error     { return nil }

type tmpFile struct {
	*os.File
	filename string
}

func (t *tmpFile) Size() int64 {
	stat, _ := t.File.Stat()
	return stat.Size()
}

func (t *tmpFile) FileName() string { return t.filename }
func (t *tmpFile) Close() error {
	if err := t.File.Close(); err != nil {
		return err
	}

	return os.Remove(t.Name())
}

func newTmpFile(b *buffer, part *multipart.Part) (*tmpFile, error) {
	tmp, err := os.CreateTemp("", "rpc-multipart-")
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(tmp, io.MultiReader(b, part))
	if err != nil {
		_ = os.Remove(tmp.Name())
		return nil, err
	}
	_, err = tmp.Seek(0, 0)
	if err != nil {
		_ = os.Remove(tmp.Name())
		return nil, err
	}

	return &tmpFile{
		File:     tmp,
		filename: part.FileName(),
	}, nil
}
