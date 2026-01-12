package storage

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Provider struct {
	client   *s3.Client
	uploader *manager.Uploader
	bucket   string
}



func NewS3Provider(ctx context.Context, bucket, region, endpoint, accessKey, secretKey string) (*S3Provider, error) {

	if endpoint != "" && !strings.Contains(endpoint, "://") {
		endpoint = "http://" + endpoint
	}

	cfgFuncs := []func(*config.LoadOptions) error{
		config.WithRegion(region),
	}

	if accessKey != "" && secretKey != "" {
		cfgFuncs = append(cfgFuncs, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")))
	}

	cfg, err := config.LoadDefaultConfig(ctx, cfgFuncs...)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}



	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
			o.UsePathStyle = true
		}
	})

	return &S3Provider{
		client:   client,
		uploader: manager.NewUploader(client),
		bucket:   bucket,
	}, nil
}

func (s *S3Provider) Upload(ctx context.Context, key string, data io.Reader) error {

	_, err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   data,
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}
	return nil
}

func (s *S3Provider) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download from S3: %w", err)
	}
	return resp.Body, nil
}

func (s *S3Provider) List(ctx context.Context, prefix string) ([]BackupItem, error) {
	output, err := s.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list S3 objects: %w", err)
	}

	var items []BackupItem
	for _, obj := range output.Contents {
		items = append(items, BackupItem{
			Key:          *obj.Key,
			Size:         *obj.Size,
			LastModified: *obj.LastModified,
		})
	}
	return items, nil
}
