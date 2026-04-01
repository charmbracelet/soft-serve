package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

var _ Storage = (*S3Storage)(nil)

type S3Storage struct {
	client *s3.Client
	bucket string
	prefix string
}

type S3Config struct {
	Endpoint  string
	Region    string
	Bucket    string
	Prefix    string
	AccessKey string
	SecretKey string
}

func NewS3Storage(ctx context.Context, cfg S3Config) (*S3Storage, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("storage: s3 bucket is required")
	}

	var opts []func(*config.LoadOptions) error
	if cfg.Region != "" {
		opts = append(opts, config.WithRegion(cfg.Region))
	}

	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		))
	}

	awscfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("storage: failed to load aws config: %w", err)
	}

	clientOpts := []func(*s3.Options){}
	if cfg.Endpoint != "" {
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(awscfg, clientOpts...)

	prefix := cfg.Prefix
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}

	return &S3Storage{
		client: client,
		bucket: cfg.Bucket,
		prefix: prefix,
	}, nil
}

func (s *S3Storage) key(name string) string {
	return s.prefix + name
}

type s3Object struct {
	body io.ReadCloser
	size int64
	name string
}

var _ Object = (*s3Object)(nil)

func (o *s3Object) Read(p []byte) (n int, err error) { return o.body.Read(p) }
func (o *s3Object) Close() error                     { return o.body.Close() }
func (o *s3Object) Name() string                     { return o.name }
func (o *s3Object) Stat() (fs.FileInfo, error)       { return s3FileInfo{name: o.name, size: o.size}, nil }
func (o *s3Object) Seek(offset int64, whence int) (int64, error) {
	if seeker, ok := o.body.(io.Seeker); ok {
		return seeker.Seek(offset, whence)
	}
	return 0, fs.ErrNotExist
}

type s3FileInfo struct {
	name string
	size int64
}

func (fi s3FileInfo) Name() string       { return fi.name }
func (fi s3FileInfo) Size() int64        { return fi.size }
func (fi s3FileInfo) Mode() fs.FileMode  { return 0 }
func (fi s3FileInfo) ModTime() time.Time { return time.Time{} }
func (fi s3FileInfo) IsDir() bool        { return false }
func (fi s3FileInfo) Sys() interface{}   { return nil }

func (s *S3Storage) Open(name string) (Object, error) {
	key := s.key(name)
	resp, err := s.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &noSuchKey) {
			return nil, fs.ErrNotExist
		}
		var httpErr interface{ StatusCode() int }
		if errors.As(err, &httpErr) && httpErr.StatusCode() == http.StatusNotFound {
			return nil, fs.ErrNotExist
		}
		return nil, fmt.Errorf("storage: s3 open %q: %w", name, err)
	}

	var size int64
	if resp.ContentLength != nil {
		size = *resp.ContentLength
	}

	return &s3Object{
		body: resp.Body,
		name: name,
		size: size,
	}, nil
}

func (s *S3Storage) Stat(name string) (fs.FileInfo, error) {
	key := s.key(name)
	resp, err := s.client.HeadObject(context.Background(), &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var httpErr interface{ StatusCode() int }
		if errors.As(err, &httpErr) && httpErr.StatusCode() == http.StatusNotFound {
			return nil, fs.ErrNotExist
		}
		return nil, fmt.Errorf("storage: s3 stat %q: %w", name, err)
	}

	var size int64
	if resp.ContentLength != nil {
		size = *resp.ContentLength
	}

	return s3FileInfo{name: name, size: size}, nil
}

func (s *S3Storage) Put(name string, r io.Reader) (int64, error) {
	key := s.key(name)
	_, err := s.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   r,
	})
	if err != nil {
		return 0, fmt.Errorf("storage: s3 put %q: %w", name, err)
	}

	stat, err := s.Stat(name)
	if err != nil {
		return 0, nil
	}
	return stat.Size(), nil
}

func (s *S3Storage) Delete(name string) error {
	key := s.key(name)
	_, err := s.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("storage: s3 delete %q: %w", name, err)
	}
	return nil
}

func (s *S3Storage) Exists(name string) (bool, error) {
	_, err := s.Stat(name)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func (s *S3Storage) Rename(oldName, newName string) error {
	oldKey := s.key(oldName)
	newKey := s.key(newName)

	_, err := s.client.CopyObject(context.Background(), &s3.CopyObjectInput{
		Bucket:     aws.String(s.bucket),
		Key:        aws.String(newKey),
		CopySource: aws.String(s.bucket + "/" + oldKey),
	})
	if err != nil {
		return fmt.Errorf("storage: s3 copy %q -> %q: %w", oldName, newName, err)
	}

	return s.Delete(oldName)
}
