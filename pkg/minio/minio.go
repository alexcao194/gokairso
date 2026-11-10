package minio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/alexcao194/kairso/internal/config"
	"github.com/google/uuid"
	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
	client        *miniogo.Client
	presignClient *miniogo.Client
	bucket        string
	publicURL     string
	useSSL        bool
	endpoint      string
	bucketReady   atomic.Bool
}

type PresignPutInput struct {
	ObjectName  string
	ContentType string
	Expires     time.Duration
}

type PresignPutOutput struct {
	Method     string
	UploadURL  string
	FileURL    string
	ObjectName string
	ExpiresAt  time.Time
}

func NewClient(cfg config.StorageConfig) (*Client, error) {
	creds := credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, "")
	client, err := miniogo.New(cfg.Endpoint, &miniogo.Options{
		Creds:  creds,
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, err
	}
	publicURL := strings.TrimRight(cfg.PublicURL, "/")
	presignClient := client
	if publicURL != "" {
		pub, err := url.Parse(publicURL)
		if err == nil && pub.Host != "" {
			pc, err := miniogo.New(pub.Host, &miniogo.Options{
				Creds:  creds,
				Secure: pub.Scheme == "https",
			})
			if err == nil {
				presignClient = pc
			}
		}
	}
	return &Client{
		client:        client,
		presignClient: presignClient,
		bucket:        cfg.Bucket,
		publicURL:     publicURL,
		useSSL:        cfg.UseSSL,
		endpoint:      cfg.Endpoint,
	}, nil
}

func (c *Client) PresignPutObject(ctx context.Context, input PresignPutInput) (PresignPutOutput, error) {
	if input.ObjectName == "" {
		return PresignPutOutput{}, fmt.Errorf("object name is required")
	}
	if input.Expires <= 0 {
		input.Expires = 15 * time.Minute
	}
	if !c.bucketReady.Load() {
		if err := c.EnsureBucketReady(ctx); err != nil {
			return PresignPutOutput{}, err
		}
	}
	uploadURL, err := c.presignClient.PresignedPutObject(ctx, c.bucket, input.ObjectName, input.Expires)
	if err != nil {
		return PresignPutOutput{}, err
	}
	return PresignPutOutput{
		Method:     "PUT",
		UploadURL:  uploadURL.String(),
		FileURL:    c.ObjectURL(input.ObjectName),
		ObjectName: input.ObjectName,
		ExpiresAt:  time.Now().Add(input.Expires).UTC(),
	}, nil
}

func (c *Client) EnsureBucketReady(ctx context.Context) error {
	if err := c.ensureBucket(ctx); err != nil {
		return err
	}
	c.bucketReady.Store(true)
	return nil
}

func (c *Client) ensureBucket(ctx context.Context) error {
	exists, err := c.client.BucketExists(ctx, c.bucket)
	if err != nil {
		return err
	}
	if !exists {
		if err := c.client.MakeBucket(ctx, c.bucket, miniogo.MakeBucketOptions{}); err != nil {
			return err
		}
	}
	return c.client.SetBucketPolicy(ctx, c.bucket, publicReadPolicy(c.bucket))
}

func publicReadPolicy(bucket string) string {
	policy := struct {
		Version   string `json:"Version"`
		Statement []struct {
			Effect    string   `json:"Effect"`
			Principal string   `json:"Principal"`
			Action    []string `json:"Action"`
			Resource  []string `json:"Resource"`
		} `json:"Statement"`
	}{
		Version: "2012-10-17",
		Statement: []struct {
			Effect    string   `json:"Effect"`
			Principal string   `json:"Principal"`
			Action    []string `json:"Action"`
			Resource  []string `json:"Resource"`
		}{
			{
				Effect:    "Allow",
				Principal: "*",
				Action:    []string{"s3:GetObject"},
				Resource:  []string{"arn:aws:s3:::" + bucket + "/*"},
			},
		},
	}
	encoded, _ := json.Marshal(policy)
	return string(encoded)
}

func (c *Client) ObjectURL(objectName string) string {
	if c.publicURL != "" {
		return c.publicURL + "/" + path.Join(c.bucket, objectName)
	}
	scheme := "http"
	if c.useSSL {
		scheme = "https"
	}
	return (&url.URL{
		Scheme: scheme,
		Host:   c.endpoint,
		Path:   path.Join("/", c.bucket, objectName),
	}).String()
}

func AvatarObjectName(kind, owner, filename, contentType string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		switch contentType {
		case "image/jpeg":
			ext = ".jpg"
		case "image/png":
			ext = ".png"
		case "image/webp":
			ext = ".webp"
		case "image/gif":
			ext = ".gif"
		}
	}
	return fmt.Sprintf("avatars/%s/%s/%s%s", kind, owner, uuid.NewString(), ext)
}

func FileObjectName(kind, owner, filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	return fmt.Sprintf("files/%s/%s/%s%s", kind, owner, uuid.NewString(), ext)
}
