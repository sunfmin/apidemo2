package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"
	"gorm.io/gorm"

	pb "github.com/sunfmin/apidemo2/backend/api/gen/pim/v1"
	"github.com/sunfmin/apidemo2/backend/internal/models"
)

const (
	// File size limits
	MaxImageSize = 10 * 1024 * 1024  // 10 MB
	MaxVideoSize = 100 * 1024 * 1024 // 100 MB
)

// MediaService defines the interface for media management operations
type MediaService interface {
	Upload(ctx context.Context, entityType, entityID, attributeName string, file multipart.File, header *multipart.FileHeader) (*pb.MediaFile, error)
	UploadBulk(ctx context.Context, entityType, entityID, attributeName string, files []multipart.File, headers []*multipart.FileHeader) (*pb.BulkUploadMediaResponse, error)
	Get(ctx context.Context, id string) (*pb.MediaFile, error)
	List(ctx context.Context, entityType, entityID, attributeName, fileType string) ([]*pb.MediaFile, error)
	Update(ctx context.Context, id string, displayOrder *int32, fileName *string) (*pb.MediaFile, error)
	Delete(ctx context.Context, id string) error
	Reorder(ctx context.Context, entityType, entityID, attributeName string, mediaIDs []string) ([]*pb.MediaFile, error)
}

type mediaService struct {
	db             *gorm.DB
	storage        MediaStorage
	imageProcessor *ImageProcessor
	videoProcessor *VideoProcessor
	storagePath    string // Base path where files are stored (e.g., ./storage/media)
}

// NewMediaService creates a new MediaService
func NewMediaService(db *gorm.DB, storage MediaStorage, imageProcessor *ImageProcessor, videoProcessor *VideoProcessor, storagePath string) MediaService {
	return &mediaService{
		db:             db,
		storage:        storage,
		imageProcessor: imageProcessor,
		videoProcessor: videoProcessor,
		storagePath:    storagePath,
	}
}

// Upload uploads a single media file
func (s *mediaService) Upload(ctx context.Context, entityType, entityID, attributeName string, file multipart.File, header *multipart.FileHeader) (*pb.MediaFile, error) {
	// Validate entity type
	if entityType != "product" && entityType != "variant" {
		return nil, errors.New("entity_type must be 'product' or 'variant'")
	}

	// Verify entity exists
	if err := s.verifyEntity(ctx, entityType, entityID); err != nil {
		return nil, err
	}

	// Detect file type by content
	fileType, mimeType, err := s.detectFileType(file, header.Filename)
	if err != nil {
		return nil, err
	}

	// Reset file pointer after detection
	file.Seek(0, 0)

	// Validate file size
	if err := s.validateFileSize(header.Size, fileType); err != nil {
		return nil, err
	}

	// Store the file using the storage interface
	relativePath, err := s.storage.Store(ctx, file, header.Filename, mimeType)
	if err != nil {
		return nil, fmt.Errorf("failed to store file: %w", err)
	}

	// Get absolute path for processing
	absolutePath := filepath.Join(s.storagePath, relativePath)

	// Get next display order
	displayOrder, err := s.getNextDisplayOrder(ctx, entityType, entityID, attributeName)
	if err != nil {
		s.storage.Delete(ctx, relativePath)
		return nil, err
	}

	// Create media record
	mediaFile := &models.MediaFile{
		EntityType:    entityType,
		EntityID:      entityID,
		AttributeName: attributeName,
		FileType:      fileType,
		MimeType:      mimeType,
		FileName:      header.Filename,
		FilePath:      relativePath, // Store relative path
		FileSize:      header.Size,
		DisplayOrder:  int32(displayOrder),
	}

	// Process based on file type
	if fileType == "image" {
		// Generate thumbnail
		thumbnailPath := s.getThumbnailPath(absolutePath)
		if err := s.imageProcessor.GenerateThumbnail(absolutePath, thumbnailPath); err != nil {
			s.storage.Delete(ctx, relativePath)
			return nil, fmt.Errorf("failed to generate thumbnail: %w", err)
		}

		// Get image dimensions
		width, height, err := s.imageProcessor.GetImageDimensions(absolutePath)
		if err != nil {
			s.storage.Delete(ctx, relativePath)
			return nil, fmt.Errorf("failed to get image dimensions: %w", err)
		}
		mediaFile.Width = int32(width)
		mediaFile.Height = int32(height)
	} else if fileType == "video" {
		// Extract preview frame at 1 second
		previewPath := s.getPreviewPath(absolutePath)
		if err := s.videoProcessor.ExtractPreviewFrame(absolutePath, previewPath, "00:00:01"); err != nil {
			s.storage.Delete(ctx, relativePath)
			return nil, fmt.Errorf("failed to extract video preview: %w", err)
		}

		// Get video metadata
		width, height, _ := s.videoProcessor.GetVideoDimensions(absolutePath)
		duration, _ := s.videoProcessor.GetVideoDuration(absolutePath)

		mediaFile.Width = int32(width)
		mediaFile.Height = int32(height)
		mediaFile.Duration = int32(duration)
	}

	// Save to database
	if err := s.db.WithContext(ctx).Create(mediaFile).Error; err != nil {
		s.storage.Delete(ctx, relativePath)
		return nil, fmt.Errorf("failed to save media record: %w", err)
	}

	return s.modelToProto(ctx, mediaFile), nil
}

// UploadBulk uploads multiple media files
func (s *mediaService) UploadBulk(ctx context.Context, entityType, entityID, attributeName string, files []multipart.File, headers []*multipart.FileHeader) (*pb.BulkUploadMediaResponse, error) {
	if len(files) == 0 {
		return nil, errors.New("at least one file is required")
	}

	response := &pb.BulkUploadMediaResponse{
		MediaFiles: make([]*pb.MediaFile, 0),
		Errors:     make([]*pb.BulkUploadError, 0),
	}

	for i, file := range files {
		header := headers[i]
		mediaFile, err := s.Upload(ctx, entityType, entityID, attributeName, file, header)
		if err != nil {
			response.Errors = append(response.Errors, &pb.BulkUploadError{
				FileName:     header.Filename,
				ErrorMessage: err.Error(),
			})
			continue
		}
		response.MediaFiles = append(response.MediaFiles, mediaFile)
	}

	return response, nil
}

// Get retrieves a media file by ID
func (s *mediaService) Get(ctx context.Context, id string) (*pb.MediaFile, error) {
	var mediaFile models.MediaFile
	if err := s.db.WithContext(ctx).First(&mediaFile, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("media file not found: %s", id)
		}
		return nil, err
	}

	return s.modelToProto(ctx, &mediaFile), nil
}

// List retrieves media files with filters
func (s *mediaService) List(ctx context.Context, entityType, entityID, attributeName, fileType string) ([]*pb.MediaFile, error) {
	query := s.db.WithContext(ctx).Where("entity_type = ? AND entity_id = ?", entityType, entityID)

	if attributeName != "" {
		query = query.Where("attribute_name = ?", attributeName)
	}

	if fileType != "" {
		query = query.Where("file_type = ?", fileType)
	}

	var mediaFiles []models.MediaFile
	if err := query.Order("display_order ASC, created_at ASC").Find(&mediaFiles).Error; err != nil {
		return nil, err
	}

	pbFiles := make([]*pb.MediaFile, 0, len(mediaFiles))
	for _, mf := range mediaFiles {
		pbFiles = append(pbFiles, s.modelToProto(ctx, &mf))
	}

	return pbFiles, nil
}

// Update updates media metadata
func (s *mediaService) Update(ctx context.Context, id string, displayOrder *int32, fileName *string) (*pb.MediaFile, error) {
	var mediaFile models.MediaFile
	if err := s.db.WithContext(ctx).First(&mediaFile, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("media file not found: %s", id)
		}
		return nil, err
	}

	updates := make(map[string]interface{})
	if displayOrder != nil {
		if *displayOrder < 0 {
			return nil, errors.New("display_order must be >= 0")
		}
		updates["display_order"] = *displayOrder
	}
	if fileName != nil {
		updates["file_name"] = *fileName
	}

	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(&mediaFile).Updates(updates).Error; err != nil {
			return nil, err
		}

		// Reload
		if err := s.db.WithContext(ctx).First(&mediaFile, "id = ?", id).Error; err != nil {
			return nil, err
		}
	}

	return s.modelToProto(ctx, &mediaFile), nil
}

// Delete deletes a media file
func (s *mediaService) Delete(ctx context.Context, id string) error {
	var mediaFile models.MediaFile
	if err := s.db.WithContext(ctx).First(&mediaFile, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("media file not found: %s", id)
		}
		return err
	}

	// Delete from storage
	if err := s.storage.Delete(ctx, mediaFile.FilePath); err != nil {
		// Log error but continue
		fmt.Printf("Warning: failed to delete file from storage: %v\n", err)
	}

	// Delete thumbnail/preview if exists
	absolutePath := filepath.Join(s.storagePath, mediaFile.FilePath)
	if mediaFile.FileType == "image" {
		thumbnailPath := s.getThumbnailPath(absolutePath)
		// Convert back to relative path for storage
		relThumbnailPath := strings.TrimPrefix(thumbnailPath, s.storagePath+"/")
		s.storage.Delete(ctx, relThumbnailPath)
	} else if mediaFile.FileType == "video" {
		previewPath := s.getPreviewPath(absolutePath)
		relPreviewPath := strings.TrimPrefix(previewPath, s.storagePath+"/")
		s.storage.Delete(ctx, relPreviewPath)
	}

	// Delete from database
	if err := s.db.WithContext(ctx).Delete(&mediaFile).Error; err != nil {
		return err
	}

	return nil
}

// Reorder reorders media files
func (s *mediaService) Reorder(ctx context.Context, entityType, entityID, attributeName string, mediaIDs []string) ([]*pb.MediaFile, error) {
	if len(mediaIDs) == 0 {
		return nil, errors.New("media_ids cannot be empty")
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for _, id := range mediaIDs {
		if seen[id] {
			return nil, fmt.Errorf("duplicate media ID: %s", id)
		}
		seen[id] = true
	}

	// Verify all media files exist and belong to the entity/attribute
	var mediaFiles []models.MediaFile
	if err := s.db.WithContext(ctx).
		Where("id IN ? AND entity_type = ? AND entity_id = ? AND attribute_name = ?", mediaIDs, entityType, entityID, attributeName).
		Find(&mediaFiles).Error; err != nil {
		return nil, err
	}

	if len(mediaFiles) != len(mediaIDs) {
		return nil, errors.New("some media IDs are invalid or don't belong to the specified entity/attribute")
	}

	// Update display order in transaction
	tx := s.db.WithContext(ctx).Begin()
	defer tx.Rollback()

	for i, id := range mediaIDs {
		if err := tx.Model(&models.MediaFile{}).Where("id = ?", id).Update("display_order", i).Error; err != nil {
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// Reload and return
	return s.List(ctx, entityType, entityID, attributeName, "")
}

// Helper methods

func (s *mediaService) verifyEntity(ctx context.Context, entityType, entityID string) error {
	if entityType == "product" {
		var product models.Product
		if err := s.db.WithContext(ctx).First(&product, "id = ?", entityID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("product not found: %s", entityID)
			}
			return err
		}
	} else if entityType == "variant" {
		var variant models.ProductVariant
		if err := s.db.WithContext(ctx).First(&variant, "id = ?", entityID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("variant not found: %s", entityID)
			}
			return err
		}
	}
	return nil
}

func (s *mediaService) detectFileType(file multipart.File, filename string) (string, string, error) {
	// Read first 512 bytes for content detection
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", "", err
	}

	// Detect MIME type from content
	mimeType := http.DetectContentType(buffer[:n])

	// Validate and categorize
	if strings.HasPrefix(mimeType, "image/") {
		supportedFormats := []string{"image/jpeg", "image/png", "image/gif", "image/webp"}
		supported := false
		for _, format := range supportedFormats {
			if mimeType == format {
				supported = true
				break
			}
		}
		if !supported {
			return "", "", fmt.Errorf("unsupported image format: %s", mimeType)
		}
		return "image", mimeType, nil
	} else if strings.HasPrefix(mimeType, "video/") {
		supportedFormats := []string{"video/mp4", "video/webm", "video/quicktime"}
		supported := false
		for _, format := range supportedFormats {
			if strings.Contains(mimeType, "mp4") || strings.Contains(mimeType, "webm") || strings.Contains(mimeType, "quicktime") {
				supported = true
				mimeType = format
				break
			}
		}
		if !supported {
			return "", "", fmt.Errorf("unsupported video format: %s", mimeType)
		}
		return "video", mimeType, nil
	}

	return "", "", fmt.Errorf("unsupported file type: %s", mimeType)
}

func (s *mediaService) validateFileSize(size int64, fileType string) error {
	if fileType == "image" && size > MaxImageSize {
		return fmt.Errorf("image file size exceeds maximum of %d bytes", MaxImageSize)
	}
	if fileType == "video" && size > MaxVideoSize {
		return fmt.Errorf("video file size exceeds maximum of %d bytes", MaxVideoSize)
	}
	if size == 0 {
		return errors.New("file is empty")
	}
	return nil
}

func (s *mediaService) getNextDisplayOrder(ctx context.Context, entityType, entityID, attributeName string) (int, error) {
	var maxOrder int32
	err := s.db.WithContext(ctx).
		Model(&models.MediaFile{}).
		Where("entity_type = ? AND entity_id = ? AND attribute_name = ?", entityType, entityID, attributeName).
		Select("COALESCE(MAX(display_order), -1)").
		Scan(&maxOrder).Error

	if err != nil {
		return 0, err
	}

	return int(maxOrder) + 1, nil
}

func (s *mediaService) getThumbnailPath(originalPath string) string {
	dir := filepath.Dir(originalPath)
	filename := filepath.Base(originalPath)
	return filepath.Join(strings.Replace(dir, "originals", "thumbnails", 1), filename)
}

func (s *mediaService) getPreviewPath(originalPath string) string {
	dir := filepath.Dir(originalPath)
	filename := filepath.Base(originalPath)
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)
	return filepath.Join(strings.Replace(dir, "originals", "previews", 1), nameWithoutExt+".jpg")
}

func (s *mediaService) modelToProto(ctx context.Context, model *models.MediaFile) *pb.MediaFile {
	pbFile := &pb.MediaFile{
		Id:            model.ID,
		EntityType:    model.EntityType,
		EntityId:      model.EntityID,
		AttributeName: model.AttributeName,
		FileType:      s.stringToFileType(model.FileType),
		MimeType:      model.MimeType,
		FileName:      model.FileName,
		FileSize:      model.FileSize,
		Width:         model.Width,
		Height:        model.Height,
		Duration:      model.Duration,
		DisplayOrder:  model.DisplayOrder,
		CreatedAt:     timestamppb.New(model.CreatedAt),
	}

	// Get URLs
	if url, err := s.storage.URL(ctx, model.FilePath); err == nil {
		pbFile.Url = url
	}

	// Add thumbnail URL
	absolutePath := filepath.Join(s.storagePath, model.FilePath)
	if model.FileType == "image" {
		thumbnailPath := s.getThumbnailPath(absolutePath)
		relThumbnailPath := strings.TrimPrefix(thumbnailPath, s.storagePath+"/")
		if url, err := s.storage.URL(ctx, relThumbnailPath); err == nil {
			pbFile.ThumbnailUrl = url
		}
	} else if model.FileType == "video" {
		previewPath := s.getPreviewPath(absolutePath)
		relPreviewPath := strings.TrimPrefix(previewPath, s.storagePath+"/")
		if url, err := s.storage.URL(ctx, relPreviewPath); err == nil {
			pbFile.ThumbnailUrl = url
		}
	}

	return pbFile
}

func (s *mediaService) stringToFileType(typeStr string) pb.MediaFileType {
	switch typeStr {
	case "image":
		return pb.MediaFileType_MEDIA_FILE_TYPE_IMAGE
	case "video":
		return pb.MediaFileType_MEDIA_FILE_TYPE_VIDEO
	default:
		return pb.MediaFileType_MEDIA_FILE_TYPE_UNSPECIFIED
	}
}
