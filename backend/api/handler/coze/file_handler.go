/*
 * Copyright 2025 superagent-ai Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package coze

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/google/uuid"
)

// fileMeta holds metadata for an uploaded file.
type fileMeta struct {
	ID        string    `json:"id"`
	Filename  string    `json:"filename"`
	MimeType  string    `json:"mime_type"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

// FileHandler manages file uploads/downloads on local filesystem.
// Files are stored under uploadDir with UUID-based names; metadata is kept
// in memory. This is a dev-friendly default; for production replace uploadDir
// with MinIO/S3 and the metadata store with a database.
type FileHandler struct {
	uploadDir string

	mu    sync.RWMutex
	index map[string]*fileMeta // keyed by file ID
}

// NewFileHandler creates a FileHandler that stores uploads in uploadDir.
// The directory is created if it does not exist.
func NewFileHandler(uploadDir string) *FileHandler {
	_ = os.MkdirAll(uploadDir, 0o755)
	return &FileHandler{
		uploadDir: uploadDir,
		index:     make(map[string]*fileMeta),
	}
}

// HandleUpload accepts a multipart/form-data upload.
//
// POST /api/v2/files
// Form field: file  (the file to upload)
func (h *FileHandler) HandleUpload(_ context.Context, c *app.RequestContext) {
	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(400, map[string]any{"error": "file field is required: " + err.Error()})
		return
	}

	src, err := fh.Open()
	if err != nil {
		c.JSON(500, map[string]any{"error": "cannot open uploaded file: " + err.Error()})
		return
	}
	defer src.Close()

	// Read first 512 bytes for MIME detection before writing.
	buf := make([]byte, 512)
	n, _ := src.Read(buf)
	mimeType := http.DetectContentType(buf[:n])
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		c.JSON(500, map[string]any{"error": "cannot seek file: " + err.Error()})
		return
	}

	id := uuid.New().String()
	ext := filepath.Ext(fh.Filename)
	destPath := filepath.Join(h.uploadDir, id+ext)

	dst, err := os.Create(destPath)
	if err != nil {
		c.JSON(500, map[string]any{"error": "cannot create file: " + err.Error()})
		return
	}
	defer dst.Close()

	size, err := io.Copy(dst, src)
	if err != nil {
		_ = os.Remove(destPath)
		c.JSON(500, map[string]any{"error": "cannot write file: " + err.Error()})
		return
	}

	meta := &fileMeta{
		ID:        id,
		Filename:  fh.Filename,
		MimeType:  mimeType,
		Size:      size,
		CreatedAt: time.Now().UTC(),
	}

	h.mu.Lock()
	h.index[id] = meta
	h.mu.Unlock()

	c.JSON(201, map[string]any{"file": meta})
}

// HandleList returns all uploaded files.
//
// GET /api/v2/files
func (h *FileHandler) HandleList(_ context.Context, c *app.RequestContext) {
	h.mu.RLock()
	files := make([]*fileMeta, 0, len(h.index))
	for _, m := range h.index {
		files = append(files, m)
	}
	h.mu.RUnlock()

	c.JSON(200, map[string]any{
		"files": files,
		"count": len(files),
	})
}

// HandleGet returns metadata for a single file.
//
// GET /api/v2/files/:id
func (h *FileHandler) HandleGet(_ context.Context, c *app.RequestContext) {
	id := c.Param("id")

	h.mu.RLock()
	meta, ok := h.index[id]
	h.mu.RUnlock()

	if !ok {
		c.JSON(404, map[string]any{"error": fmt.Sprintf("file %q not found", id)})
		return
	}

	c.JSON(200, map[string]any{"file": meta})
}

// HandleDownload streams the file content to the client.
//
// GET /api/v2/files/:id/content
func (h *FileHandler) HandleDownload(_ context.Context, c *app.RequestContext) {
	id := c.Param("id")

	h.mu.RLock()
	meta, ok := h.index[id]
	h.mu.RUnlock()

	if !ok {
		c.JSON(404, map[string]any{"error": fmt.Sprintf("file %q not found", id)})
		return
	}

	// Find file on disk — the stored extension is derived from original filename.
	ext := filepath.Ext(meta.Filename)
	diskPath := filepath.Join(h.uploadDir, id+ext)

	data, err := os.ReadFile(diskPath)
	if err != nil {
		c.JSON(500, map[string]any{"error": "file data unavailable: " + err.Error()})
		return
	}

	c.Response.Header.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, meta.Filename))
	c.Response.Header.Set("Content-Type", meta.MimeType)
	c.Data(200, meta.MimeType, data)
}

// HandleDelete removes a file and its metadata.
//
// DELETE /api/v2/files/:id
func (h *FileHandler) HandleDelete(_ context.Context, c *app.RequestContext) {
	id := c.Param("id")

	h.mu.Lock()
	meta, ok := h.index[id]
	if ok {
		delete(h.index, id)
	}
	h.mu.Unlock()

	if !ok {
		c.JSON(404, map[string]any{"error": fmt.Sprintf("file %q not found", id)})
		return
	}

	ext := filepath.Ext(meta.Filename)
	diskPath := filepath.Join(h.uploadDir, id+ext)
	_ = os.Remove(diskPath)

	c.JSON(200, map[string]any{
		"status": "deleted",
		"id":     id,
	})
}
