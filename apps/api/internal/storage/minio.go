package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const unknownSizePartSize = 16 << 20

// MinIOConfig is the S3-compatible storage connection settings.
type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

// ObjectInfo describes a stored object.
type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified time.Time
}

// Client wraps a MinIO client bound to a single bucket.
type Client struct {
	mc     *minio.Client
	bucket string
}

// NewClient builds a Client for the given configuration.
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
	if size < 0 {
		opts.PartSize = unknownSizePartSize
	}
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

type sequentialObjectReader struct {
	ctx     context.Context
	client  *Client
	keys    []string
	current io.ReadCloser
}

func (r *sequentialObjectReader) Read(p []byte) (int, error) {
	for {
		if r.current == nil {
			if len(r.keys) == 0 {
				return 0, io.EOF
			}
			obj, err := r.client.GetObject(r.ctx, r.keys[0])
			if err != nil {
				return 0, err
			}
			r.keys = r.keys[1:]
			r.current = obj
		}
		n, err := r.current.Read(p)
		if err == io.EOF {
			closeErr := r.current.Close()
			r.current = nil
			if closeErr != nil {
				return n, closeErr
			}
			if n > 0 {
				return n, nil
			}
			continue
		}
		return n, err
	}
}

func (r *sequentialObjectReader) Close() error {
	if r.current == nil {
		return nil
	}
	err := r.current.Close()
	r.current = nil
	return err
}

// AssembleChunks streams the chunk objects one at a time into destKey and
// returns the hex-encoded SHA-256 of the assembled content.
func (c *Client) AssembleChunks(ctx context.Context, destKey string, chunkKeys []string, totalSize int64, contentType string) (string, error) {
	reader := &sequentialObjectReader{ctx: ctx, client: c, keys: chunkKeys}
	defer reader.Close()

	hasher := sha256.New()
	tee := io.TeeReader(reader, hasher)
	if err := c.PutObject(ctx, destKey, tee, totalSize, contentType); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func (c *Client) DeletePrefix(ctx context.Context, prefix string) error {
	objectsCh := c.mc.ListObjects(ctx, c.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})
	for obj := range objectsCh {
		if obj.Err != nil {
			return obj.Err
		}
		if err := c.mc.RemoveObject(ctx, c.bucket, obj.Key, minio.RemoveObjectOptions{}); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) CopyObject(ctx context.Context, srcKey, destKey string) error {
	src := minio.CopySrcOptions{Bucket: c.bucket, Object: srcKey}
	dst := minio.CopyDestOptions{Bucket: c.bucket, Object: destKey}
	_, err := c.mc.ComposeObject(ctx, dst, src)
	return err
}
