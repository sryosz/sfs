package storage

import (
	"encoding/hex"
	"io"
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/sha3"
)

const defaultRootFolderName = "default"

func CASPathFunc(key string) FilePath {
	hash := sha3.Sum256([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	blockSize := 6
	sliceLen := len(hashStr) / blockSize
	paths := make([]string, sliceLen)

	for i := 0; i < sliceLen; i++ {
		from, to := i*blockSize, (i*blockSize)+blockSize
		paths[i] = hashStr[from:to]
	}

	return FilePath{
		PathName: strings.Join(paths, "/"),
		FileName: hashStr,
	}
}

var DefaultPathFunc = func(key string) FilePath {
	return FilePath{
		PathName: key,
		FileName: key,
	}
}

type PathFunc func(string) FilePath

type FilePath struct {
	PathName string
	FileName string
}

func (p FilePath) FullPath(root string) string {
	return root + "/" + p.PathName + "/" + p.FileName
}

func (p FilePath) GetFirstPathFolder() string {
	return strings.Split(p.PathName, "/")[0]
}

type StorageOPTS struct {
	Root     string
	PathFunc PathFunc
}

type Storage struct {
	StorageOPTS
}

func NewStorage(opts StorageOPTS) *Storage {
	if opts.PathFunc == nil {
		opts.PathFunc = DefaultPathFunc
	}

	if len(opts.Root) == 0 {
		opts.Root = defaultRootFolderName
	}

	return &Storage{
		StorageOPTS: opts,
	}
}

func (s *Storage) Has(key string) bool {
	pathKey := s.PathFunc(key)

	_, err := os.Stat(pathKey.FullPath(s.Root))

	return !os.IsNotExist(err)
}

func (s *Storage) Delete(key string) error {
	pathKey := s.PathFunc(key)

	return os.RemoveAll(s.Root + "/" + pathKey.GetFirstPathFolder())
}

func (s *Storage) Read(key string) (int64, io.Reader, error) {
	return s.readStream(key)
	
}

func (s *Storage) Write(key string, r io.Reader) (int64, error) {
	return s.writeStream(key, r)
}

func (s *Storage) readStream(key string) (int64, io.ReadCloser, error) {
	pathKey := s.PathFunc(key)

	f, err := os.Open(pathKey.FullPath(s.Root))
	if err != nil{
		return 0, nil, err
	}

	info, err := f.Stat()
	if err != nil{
		return 0, nil, err
	}

	return info.Size(), f, nil 
}

func (s *Storage) writeStream(key string, r io.Reader) (int64, error) {
	pathKey := s.PathFunc(key)
	if err := os.MkdirAll(s.Root+"/"+pathKey.PathName, os.ModePerm); err != nil {
		return 0, err
	}

	fullPath := pathKey.FullPath(s.Root)

	f, err := os.Create(fullPath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	n, err := io.Copy(f, r)
	if err != nil {
		return 0, err
	}

	log.Printf("Written (%d) bytes to disk: %s", n, fullPath)

	return n, nil
}

func (s *Storage) ClearAll() error {
	return os.RemoveAll(s.Root)
}
