package storage

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified time.Time
}

type Client struct {
	mc     *minio.Client
	bucket string
}

func NewClient(cfg MinIOConfig) (*Client, error) {
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, err
	}
	return &Client{mc: mc, bucket: cfg.Bucket}, nil
}

func (c *Client) EnsureBucket(ctx context.Context) error {
	exists, err := c.mc.BucketExists(ctx, c.bucket)
	if err != nil {
		return err
	}
	if !exists {
		return c.mc.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{})
	}
	return nil
}

func (c *Client) PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	opts := minio.PutObjectOptions{ContentType: contentType}
	_, err := c.mc.PutObject(ctx, c.bucket, key, reader, size, opts)
	return err
}

func (c *Client) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	return c.mc.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{})
}

func (c *Client) DeleteObject(ctx context.Context, key string) error {
	return c.mc.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{})
}

func (c *Client) PresignedGetURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	u, err := c.mc.PresignedGetObject(ctx, c.bucket, key, expiry, url.Values{})
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (c *Client) ListObjects(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	var objects []ObjectInfo
	for obj := range c.mc.ListObjects(ctx, c.bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
		if obj.Err != nil {
			return nil, obj.Err
		}
		objects = append(objects, ObjectInfo{
			Key:          obj.Key,
			Size:         obj.Size,
			ContentType:  obj.ContentType,
			LastModified: obj.LastModified,
		})
	}
	return objects, nil
}

func (c *Client) StatObject(ctx context.Context, key string) (ObjectInfo, error) {
	info, err := c.mc.StatObject(ctx, c.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return ObjectInfo{}, err
	}
	return ObjectInfo{
		Key:          info.Key,
		Size:         info.Size,
		ContentType:  info.ContentType,
		LastModified: info.LastModified,
	}, nil
}
