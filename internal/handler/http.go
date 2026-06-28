package handler

import (
	"encoding/json"
	"fmt"
	"go-mini-cloud/internal/storage"
	"net/http"
	"strings"
)

type FileHandler struct {
	engine *storage.FileEngine
}

func NewFileHandler(engine *storage.FileEngine) *FileHandler {
	return &FileHandler{engine: engine}
}

func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method Not Allowed Use PUT", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	origName := r.Header.Get("X-Original-Name")
	if origName == "" {
		origName = "unnamed_file.bin"
	}

	meta, err := h.engine.SaveStream(r.Body, origName)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(meta)
}

func (h *FileHandler) Download(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed (Use GET)", http.StatusMethodNotAllowed)
		return
	}

	fileID := strings.TrimPrefix(r.URL.Path, "/download/")
	if fileID == "" {
		http.Error(w, "Bad Request: Missing File ID", http.StatusBadRequest)
		return
	}

	meta, err := h.engine.GetMeta(fileID)
	if err != nil {
		http.Error(w, "File Not Found", http.StatusNotFound)
		return
	}

	filePath := h.engine.GetFilePath(fileID)

	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, meta.OriginalName))
	http.ServeFile(w, r, filePath)
}
