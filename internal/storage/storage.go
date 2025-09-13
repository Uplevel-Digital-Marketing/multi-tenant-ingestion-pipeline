package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/storage"

	"github.com/home-renovators/ingestion-pipeline/pkg/config"
)

// Service handles Cloud Storage operations
type Service struct {
	client     *storage.Client
	audioBucket string
	projectID  string
}

// NewService creates a new storage service
func NewService(ctx context.Context, cfg *config.Config) (*Service, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage client: %w", err)
	}

	return &Service{
		client:     client,
		audioBucket: cfg.AudioBucket,
		projectID:  cfg.ProjectID,
	}, nil
}

// Close closes the storage client
func (s *Service) Close() error {
	return s.client.Close()
}

// StoreAudioFile stores an audio file in Cloud Storage
func (s *Service) StoreAudioFile(ctx context.Context, tenantID, callID string, audioData []byte) (string, error) {
	objectPath := fmt.Sprintf("%s/calls/%s.mp3", tenantID, callID)

	bucket := s.client.Bucket(s.audioBucket)
	obj := bucket.Object(objectPath)

	// Set object attributes
	w := obj.NewWriter(ctx)
	w.ContentType = "audio/mpeg"
	w.Metadata = map[string]string{
		"tenant_id": tenantID,
		"call_id":   callID,
		"uploaded_at": time.Now().Format(time.RFC3339),
	}

	// Write the audio data
	if _, err := w.Write(audioData); err != nil {
		w.Close()
		return "", fmt.Errorf("failed to write audio data: %w", err)
	}

	if err := w.Close(); err != nil {
		return "", fmt.Errorf("failed to close writer: %w", err)
	}

	// Return the GCS URI
	storageURL := fmt.Sprintf("gs://%s/%s", s.audioBucket, objectPath)
	return storageURL, nil
}

// GetAudioFile retrieves an audio file from Cloud Storage
func (s *Service) GetAudioFile(ctx context.Context, tenantID, callID string) ([]byte, error) {
	objectPath := fmt.Sprintf("%s/calls/%s.mp3", tenantID, callID)

	bucket := s.client.Bucket(s.audioBucket)
	obj := bucket.Object(objectPath)

	reader, err := obj.NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create reader: %w", err)
	}
	defer reader.Close()

	audioData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio data: %w", err)
	}

	return audioData, nil
}

// DeleteAudioFile deletes an audio file from Cloud Storage
func (s *Service) DeleteAudioFile(ctx context.Context, tenantID, callID string) error {
	objectPath := fmt.Sprintf("%s/calls/%s.mp3", tenantID, callID)

	bucket := s.client.Bucket(s.audioBucket)
	obj := bucket.Object(objectPath)

	if err := obj.Delete(ctx); err != nil {
		return fmt.Errorf("failed to delete audio file: %w", err)
	}

	return nil
}

// ListAudioFiles lists audio files for a tenant
func (s *Service) ListAudioFiles(ctx context.Context, tenantID string) ([]string, error) {
	prefix := fmt.Sprintf("%s/calls/", tenantID)

	bucket := s.client.Bucket(s.audioBucket)
	query := &storage.Query{Prefix: prefix}

	iter := bucket.Objects(ctx, query)

	var files []string
	for {
		objAttrs, err := iter.Next()
		if err == storage.ErrObjectNotExist {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate objects: %w", err)
		}

		files = append(files, objAttrs.Name)
	}

	return files, nil
}

// GetAudioFileMetadata retrieves metadata for an audio file
func (s *Service) GetAudioFileMetadata(ctx context.Context, tenantID, callID string) (map[string]string, error) {
	objectPath := fmt.Sprintf("%s/calls/%s.mp3", tenantID, callID)

	bucket := s.client.Bucket(s.audioBucket)
	obj := bucket.Object(objectPath)

	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get object attributes: %w", err)
	}

	return attrs.Metadata, nil
}

// SetLifecyclePolicy sets lifecycle policies for the audio bucket
func (s *Service) SetLifecyclePolicy(ctx context.Context, retentionDays int) error {
	bucket := s.client.Bucket(s.audioBucket)

	lifecycle := storage.Lifecycle{
		Rules: []storage.LifecycleRule{
			{
				Action: storage.LifecycleAction{
					Type:         "SetStorageClass",
					StorageClass: "COLDLINE",
				},
				Condition: storage.LifecycleCondition{
					AgeInDays: 90,
				},
			},
			{
				Action: storage.LifecycleAction{
					Type:         "SetStorageClass",
					StorageClass: "ARCHIVE",
				},
				Condition: storage.LifecycleCondition{
					AgeInDays: 365,
				},
			},
			{
				Action: storage.LifecycleAction{
					Type: "Delete",
				},
				Condition: storage.LifecycleCondition{
					AgeInDays: int64(retentionDays),
				},
			},
		},
	}

	bucketAttrsToUpdate := storage.BucketAttrsToUpdate{
		Lifecycle: &lifecycle,
	}

	if _, err := bucket.Update(ctx, bucketAttrsToUpdate); err != nil {
		return fmt.Errorf("failed to update bucket lifecycle: %w", err)
	}

	return nil
}

// CreateBucketIfNotExists creates the audio bucket if it doesn't exist
func (s *Service) CreateBucketIfNotExists(ctx context.Context, location string) error {
	bucket := s.client.Bucket(s.audioBucket)

	// Check if bucket exists
	_, err := bucket.Attrs(ctx)
	if err == nil {
		return nil // Bucket already exists
	}

	// Create bucket
	if err := bucket.Create(ctx, s.projectID, &storage.BucketAttrs{
		Location:     location,
		StorageClass: "STANDARD",
		VersioningEnabled: false,
		UniformBucketLevelAccess: storage.UniformBucketLevelAccess{
			Enabled: true,
		},
	}); err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	return nil
}

// GenerateSignedURL generates a signed URL for accessing an audio file
func (s *Service) GenerateSignedURL(ctx context.Context, tenantID, callID string, expiration time.Duration) (string, error) {
	objectPath := fmt.Sprintf("%s/calls/%s.mp3", tenantID, callID)

	bucket := s.client.Bucket(s.audioBucket)

	opts := &storage.SignedURLOptions{
		Scheme:  storage.SigningSchemeV4,
		Method:  "GET",
		Expires: time.Now().Add(expiration),
	}

	url, err := bucket.SignedURL(objectPath, opts)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}

	return url, nil
}

// CopyFile copies a file within the storage bucket
func (s *Service) CopyFile(ctx context.Context, srcPath, destPath string) error {
	srcBucket := s.client.Bucket(s.audioBucket)
	srcObj := srcBucket.Object(srcPath)

	destBucket := s.client.Bucket(s.audioBucket)
	destObj := destBucket.Object(destPath)

	if _, err := destObj.CopierFrom(srcObj).Run(ctx); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// GetStorageStatistics returns storage statistics for a tenant
func (s *Service) GetStorageStatistics(ctx context.Context, tenantID string) (*StorageStatistics, error) {
	prefix := fmt.Sprintf("%s/calls/", tenantID)

	bucket := s.client.Bucket(s.audioBucket)
	query := &storage.Query{Prefix: prefix}

	iter := bucket.Objects(ctx, query)

	stats := &StorageStatistics{
		TenantID: tenantID,
	}

	for {
		objAttrs, err := iter.Next()
		if err == storage.ErrObjectNotExist {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate objects: %w", err)
		}

		stats.FileCount++
		stats.TotalSizeBytes += objAttrs.Size
	}

	return stats, nil
}

// StorageStatistics represents storage usage statistics
type StorageStatistics struct {
	TenantID       string `json:"tenant_id"`
	FileCount      int64  `json:"file_count"`
	TotalSizeBytes int64  `json:"total_size_bytes"`
}