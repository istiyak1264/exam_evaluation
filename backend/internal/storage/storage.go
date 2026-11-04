// Package storage abstracts script PDF storage (local disk in dev/S3 in prod).
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Store is the minimal interface the handlers need.
type Store interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

// New picks an implementation from config.
func New(driver, dir, bucket, region, endpoint string) (Store, error) {
	switch driver {
	case "local", "":
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, err
		}
		return &Local{Root: dir}, nil
	case "s3":
		cfg, err := awsconfig.LoadDefaultConfig(context.Background(), awsconfig.WithRegion(region))
		if err != nil {
			return nil, err
		}
		client := s3.NewFromConfig(cfg, func(o *s3.Options) {
			if endpoint != "" { // MinIO / LocalStack
				o.BaseEndpoint = aws.String(endpoint)
				o.UsePathStyle = true
			}
		})
		return &S3{Client: client, Bucket: bucket}, nil
	}
	return nil, fmt.Errorf("unknown storage driver %q", driver)
}

// Local stores objects as files under Root.
type Local struct{ Root string }

func (l *Local) path(key string) string { return filepath.Join(l.Root, filepath.Clean("/"+key)) }

func (l *Local) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	p := l.path(key)
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return err
	}
	tmp := p + ".part"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o640)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func (l *Local) Open(_ context.Context, key string) (io.ReadCloser, error) {
	return os.Open(l.path(key))
}

func (l *Local) Delete(_ context.Context, key string) error {
	err := os.Remove(l.path(key))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// S3 stores objects in a bucket, server-side encrypted.
type S3 struct {
	Client *s3.Client
	Bucket string
}

func (s *S3) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:               aws.String(s.Bucket),
		Key:                  aws.String(key),
		Body:                 r,
		ContentLength:        aws.Int64(size),
		ContentType:          aws.String(contentType),
		ServerSideEncryption: "AES256",
	})
	return err
}

func (s *S3) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	out, err := s.Client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.Bucket), Key: aws.String(key)})
	if err != nil {
		return nil, err
	}
	return out.Body, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	_, err := s.Client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.Bucket), Key: aws.String(key)})
	return err
}
