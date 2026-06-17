package digital_locker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"institution-os/internal/auth"
	"institution-os/pkg/db"
	"institution-os/pkg/response"
	"github.com/jackc/pgx/v5"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/rs/zerolog/log"
)

// Document represents a file in the digital locker.
type Document struct {
	ID             uuid.UUID  `json:"id"`
	InstitutionID  uuid.UUID  `json:"institution_id"`
	EntityType     string     `json:"entity_type"`
	EntityID       uuid.UUID  `json:"entity_id"`
	DocumentType   *string    `json:"document_type,omitempty"`
	Title          string     `json:"title"`
	FileName       string     `json:"file_name"`
	FileSize       int64      `json:"file_size"`
	MimeType       string     `json:"mime_type"`
	Version        int        `json:"version"`
	UploadedBy     string     `json:"uploaded_by"` // joined: user name
	CreatedAt      time.Time  `json:"created_at"`
	DownloadURL    *string    `json:"download_url,omitempty"` // pre-signed, 1h
}

// StorageConfig holds object storage connection details.
type StorageConfig struct {
	Endpoint   string
	AccessKey  string
	SecretKey  string
	Bucket     string
	UseSSL     bool
}

// Service handles digital locker operations.
type Service struct {
	db     *db.Pool
	minio  *minio.Client
	bucket string
}

// NewService creates a new digital locker service.
func NewService(dbPool *db.Pool, cfg StorageConfig) (*Service, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}

	// Ensure bucket exists
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("create bucket: %w", err)
		}
	}

	return &Service{db: dbPool, minio: client, bucket: cfg.Bucket}, nil
}

// ListDocuments returns all documents for an entity.
func (s *Service) ListDocuments(ctx context.Context, institutionID uuid.UUID, entityType string, entityID uuid.UUID) ([]*Document, error) {
	rows, err := s.db.Query(ctx, `
		SELECT d.id, d.institution_id, d.entity_type, d.entity_id,
		       d.document_type, d.title, d.file_name, d.file_size,
		       d.mime_type, d.version,
		       u.first_name || ' ' || u.last_name AS uploaded_by,
		       d.created_at
		FROM locker_documents d
		JOIN users u ON u.id = d.created_by
		WHERE d.institution_id = $1
		  AND d.entity_type = $2
		  AND d.entity_id = $3
		  AND d.deleted_at IS NULL
		ORDER BY d.created_at DESC`,
		institutionID, entityType, entityID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []*Document
	for rows.Next() {
		var d Document
		if err := rows.Scan(
			&d.ID, &d.InstitutionID, &d.EntityType, &d.EntityID,
			&d.DocumentType, &d.Title, &d.FileName, &d.FileSize,
			&d.MimeType, &d.Version, &d.UploadedBy, &d.CreatedAt,
		); err != nil {
			return nil, err
		}
		docs = append(docs, &d)
	}
	return docs, rows.Err()
}

// GetDownloadURL generates a pre-signed download URL for a document.
func (s *Service) GetDownloadURL(ctx context.Context, institutionID, docID uuid.UUID) (string, error) {
	var storageKey string
	err := s.db.QueryRow(ctx, `
		SELECT storage_key FROM locker_documents
		WHERE institution_id = $1 AND id = $2 AND deleted_at IS NULL`,
		institutionID, docID,
	).Scan(&storageKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errors.New("document not found")
		}
		return "", err
	}

	url, err := s.minio.PresignedGetObject(ctx, s.bucket, storageKey, time.Hour, nil)
	if err != nil {
		return "", fmt.Errorf("presign url: %w", err)
	}
	return url.String(), nil
}

// UploadDocument handles a multipart file upload and stores metadata.
func (s *Service) UploadDocument(ctx context.Context, institutionID, uploaderID uuid.UUID,
	entityType string, entityID uuid.UUID, docType, title, fileName string,
	reader io.Reader, size int64, contentType string) (*Document, error) {

	docID := uuid.New()
	storageKey := fmt.Sprintf("%s/%s/%s/%s%s",
		institutionID, entityType, entityID, docID, filepath.Ext(fileName))

	// Upload to MinIO
	_, err := s.minio.PutObject(ctx, s.bucket, storageKey, reader, size,
		minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return nil, fmt.Errorf("upload to storage: %w", err)
	}

	// Ensure folder exists (get or create)
	var folderID uuid.UUID
	err = s.db.QueryRow(ctx, `
		SELECT id FROM locker_folders
		WHERE institution_id = $1 AND entity_type = $2 AND entity_id = $3
		LIMIT 1`,
		institutionID, entityType, entityID,
	).Scan(&folderID)
	if errors.Is(err, pgx.ErrNoRows) {
		folderID = uuid.New()
		_, err = s.db.Exec(ctx, `
			INSERT INTO locker_folders (id, institution_id, entity_type, entity_id, name, created_at)
			VALUES ($1,$2,$3,$4,$5,NOW())`,
			folderID, institutionID, entityType, entityID, entityType+" documents",
		)
		if err != nil {
			return nil, fmt.Errorf("create folder: %w", err)
		}
	} else if err != nil {
		return nil, err
	}

	// Save document metadata
	var docTypePtr *string
	if docType != "" {
		docTypePtr = &docType
	}

	_, err = s.db.Exec(ctx, `
		INSERT INTO locker_documents (
			id, institution_id, folder_id, entity_type, entity_id,
			document_type, title, file_name, file_size, mime_type,
			storage_key, version, access_level, created_by, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,1,'institution',$12,NOW())`,
		docID, institutionID, folderID, entityType, entityID,
		docTypePtr, title, fileName, size, contentType,
		storageKey, uploaderID,
	)
	if err != nil {
		return nil, fmt.Errorf("save document metadata: %w", err)
	}

	return &Document{
		ID:            docID,
		InstitutionID: institutionID,
		EntityType:    entityType,
		EntityID:      entityID,
		DocumentType:  docTypePtr,
		Title:         title,
		FileName:      fileName,
		FileSize:      size,
		MimeType:      contentType,
		Version:       1,
		CreatedAt:     time.Now(),
	}, nil
}

// SoftDeleteDocument marks a document as deleted.
func (s *Service) SoftDeleteDocument(ctx context.Context, institutionID, docID uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		UPDATE locker_documents SET deleted_at = NOW()
		WHERE institution_id = $1 AND id = $2 AND deleted_at IS NULL`,
		institutionID, docID,
	)
	return err
}

// ─── Handler ──────────────────────────────────────────────────────

// Handler handles digital locker HTTP endpoints.
type Handler struct {
	svc *Service
}

// NewHandler creates a new digital locker handler.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

const maxUploadSize = 10 << 20 // 10 MB

// ListDocuments handles GET /api/v1/locker/{entity_type}/{entity_id}
func (h *Handler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	entityType := chi.URLParam(r, "entity_type")
	entityID, err := uuid.Parse(chi.URLParam(r, "entity_id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid entity ID")
		return
	}

	docs, err := h.svc.ListDocuments(r.Context(), institutionID, entityType, entityID)
	if err != nil {
		log.Error().Err(err).Msg("list documents")
		response.InternalError(w)
		return
	}
	response.Ok(w, docs)
}

// DownloadDocument handles GET /api/v1/locker/documents/{id}/download
func (h *Handler) DownloadDocument(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	docID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid document ID")
		return
	}

	url, err := h.svc.GetDownloadURL(r.Context(), institutionID, docID)
	if err != nil {
		log.Error().Err(err).Msg("get download url")
		response.Error(w, http.StatusNotFound, "DOCUMENT_NOT_FOUND", err.Error())
		return
	}

	response.Ok(w, map[string]string{"url": url, "expires_in": "3600s"})
}

// UploadDocument handles POST /api/v1/locker/{entity_type}/{entity_id}
func (h *Handler) UploadDocument(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	claims := auth.ClaimsFromContext(r.Context())
	uploaderID, _ := uuid.Parse(claims.UserID)

	entityType := chi.URLParam(r, "entity_type")
	entityID, err := uuid.Parse(chi.URLParam(r, "entity_id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid entity ID")
		return
	}

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		response.Error(w, http.StatusBadRequest, "FILE_TOO_LARGE", "File must be under 10MB")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "NO_FILE", "No file provided in form field 'file'")
		return
	}
	defer file.Close()

	title := r.FormValue("title")
	if title == "" {
		title = header.Filename
	}
	docType := r.FormValue("document_type")
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Validate extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowed := map[string]bool{".pdf": true, ".jpg": true, ".jpeg": true, ".png": true, ".doc": true, ".docx": true}
	if !allowed[ext] {
		response.Error(w, http.StatusBadRequest, "INVALID_FILE_TYPE",
			"Only PDF, JPG, PNG, DOC, DOCX files are allowed")
		return
	}

	doc, err := h.svc.UploadDocument(r.Context(), institutionID, uploaderID,
		entityType, entityID, docType, title, header.Filename,
		file, header.Size, contentType)
	if err != nil {
		log.Error().Err(err).Msg("upload document")
		response.Error(w, http.StatusInternalServerError, "UPLOAD_FAILED", "Failed to upload document")
		return
	}

	response.Created(w, doc)
}

// DeleteDocument handles DELETE /api/v1/locker/documents/{id}
func (h *Handler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	institutionID := auth.InstitutionIDFromContext(r.Context())
	docID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_ID", "Invalid document ID")
		return
	}

	if err := h.svc.SoftDeleteDocument(r.Context(), institutionID, docID); err != nil {
		log.Error().Err(err).Msg("delete document")
		response.InternalError(w)
		return
	}
	response.NoContent(w)
}
