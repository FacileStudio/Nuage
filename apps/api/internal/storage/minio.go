package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/url"
	"strings"
	"sync"
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

const spaceBucketPrefix = "nuage-space-"

// SpaceBucket names the bucket that holds a space's files. It derives from the
// space's facile_id, the suite-wide key, so the name survives a renumbering of
// the local row id and satisfies S3's lowercase-and-hyphen rule.
func SpaceBucket(facileID string) string {
	return spaceBucketPrefix + strings.ReplaceAll(strings.ToLower(facileID), "_", "-")
}

// Client wraps a MinIO client bound to a single bucket.
type Client struct {
	mc      *minio.Client
	bucket  string
	ensured *sync.Map
}

// For returns a view of the same connection bound to another bucket. The empty
// name is the shared bucket, which is where every file lived before spaces got
// their own.
func (c *Client) For(bucket string) *Client {
	if bucket == "" || bucket == c.bucket {
		return c
	}
	return &Client{mc: c.mc, bucket: bucket, ensured: c.ensured}
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
	return &Client{mc: mc, bucket: cfg.Bucket, ensured: &sync.Map{}}, nil
}

// EnsureBucket creates the bucket when missing. Losing a creation race to a
// concurrent caller is success, since the bucket exists either way.
func (c *Client) EnsureBucket(ctx context.Context) error {
	exists, err := c.mc.BucketExists(ctx, c.bucket)
	if err != nil {
		return err
	}
	if !exists {
		err := c.mc.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{})
		if code := minio.ToErrorResponse(err).Code; err != nil && code != "BucketAlreadyOwnedByYou" && code != "BucketAlreadyExists" {
			return err
		}
	}
	c.ensured.Store(c.bucket, struct{}{})
	return nil
}

// EnsureBucketOnce is EnsureBucket for the write path: it asks MinIO once per
// bucket and remembers the answer, so an upload does not pay a round trip for
// a bucket it already created.
func (c *Client) EnsureBucketOnce(ctx context.Context) error {
	if _, ok := c.ensured.Load(c.bucket); ok {
		return nil
	}
	return c.EnsureBucket(ctx)
}

// RemoveBucket deletes every object in the bucket and then the bucket itself.
// A bucket that is already gone is not an error, so a retried space deletion
// finishes instead of failing on its own earlier progress.
func (c *Client) RemoveBucket(ctx context.Context) error {
	c.ensured.Delete(c.bucket)
	exists, err := c.mc.BucketExists(ctx, c.bucket)
	if err != nil || !exists {
		return err
	}
	if err := c.DeletePrefix(ctx, ""); err != nil {
		return err
	}
	return c.mc.RemoveBucket(ctx, c.bucket)
}

func (c *Client) PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	opts := minio.PutObjectOptions{ContentType: contentType}
	if size < 0 {
		opts.PartSize = unknownSizePartSize
	}
	_, err := c.mc.PutObject(ctx, c.bucket, key, reader, size, opts)
	return err
}

// GetObject opens the object for reading. The reader seeks by issuing a fresh
// ranged GET against the bucket, so a caller can answer a byte range without
// pulling the whole object through the API first.
func (c *Client) GetObject(ctx context.Context, key string) (io.ReadSeekCloser, error) {
	obj, err := c.mc.GetObject(ctx, c.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	return obj, nil
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

// AssembleChunks streams the chunk objects one at a time from the `from`
// client's bucket into destKey on this client's bucket, and returns the hex-encoded SHA-256 of the assembled content.
func (c *Client) AssembleChunks(ctx context.Context, from *Client, destKey string, chunkKeys []string, totalSize int64, contentType string) (string, error) {
	reader := &sequentialObjectReader{ctx: ctx, client: from, keys: chunkKeys}
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

// CopyFrom copies the object stored under key in the from client's bucket to the
// same key in this client's bucket. The copy runs inside MinIO, so no bytes pass
// through the API, and repeating it overwrites with identical content.
func (c *Client) CopyFrom(ctx context.Context, from *Client, key string) error {
	return c.CopyFromKey(ctx, from, key, key)
}

// CopyFromKey copies srcKey in the from client's bucket to destKey in this
// client's bucket, inside MinIO. Repeating it overwrites with identical content.
func (c *Client) CopyFromKey(ctx context.Context, from *Client, srcKey, destKey string) error {
	src := minio.CopySrcOptions{Bucket: from.bucket, Object: srcKey}
	dst := minio.CopyDestOptions{Bucket: c.bucket, Object: destKey}
	_, err := c.mc.ComposeObject(ctx, dst, src)
	return err
}
