package sbi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gavin/amf/internal/logger"
)

type BackupRequest struct {
	BackupPath string `json:"backup_path"`
}

type BackupResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
}

func (s *Server) handleBackup(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Infof("Handle Backup: %s %s", r.Method, r.URL.Path)

	switch r.Method {
	case http.MethodPost:
		s.handleCreateBackup(w, r)
	default:
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Method Not Allowed",
			Status: http.StatusMethodNotAllowed,
			Detail: fmt.Sprintf("Method %s not allowed on this resource", r.Method),
		})
	}
}

func (s *Server) handleRestore(w http.ResponseWriter, r *http.Request) {
	logger.SbiLog.Infof("Handle Restore: %s %s", r.Method, r.URL.Path)

	switch r.Method {
	case http.MethodPost:
		s.handleRestoreBackup(w, r)
	default:
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Method Not Allowed",
			Status: http.StatusMethodNotAllowed,
			Detail: fmt.Sprintf("Method %s not allowed on this resource", r.Method),
		})
	}
}

func (s *Server) handleCreateBackup(w http.ResponseWriter, r *http.Request) {
	var req BackupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	if req.BackupPath == "" {
		req.BackupPath = "backups"
	}

	if err := s.amfContext.BackupToDirectory(req.BackupPath); err != nil {
		logger.SbiLog.Errorf("Backup failed: %v", err)
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: fmt.Sprintf("Backup failed: %v", err),
		})
		return
	}

	response := BackupResponse{
		Status:  "success",
		Message: "Backup completed successfully",
		Path:    req.BackupPath,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleRestoreBackup(w http.ResponseWriter, r *http.Request) {
	var req BackupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: fmt.Sprintf("Invalid request body: %v", err),
		})
		return
	}

	if req.BackupPath == "" {
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Bad Request",
			Status: http.StatusBadRequest,
			Detail: "backup_path is required",
		})
		return
	}

	if err := s.amfContext.RestoreFromBackup(req.BackupPath); err != nil {
		logger.SbiLog.Errorf("Restore failed: %v", err)
		sendProblemDetails(w, &ProblemDetails{
			Type:   "about:blank",
			Title:  "Internal Server Error",
			Status: http.StatusInternalServerError,
			Detail: fmt.Sprintf("Restore failed: %v", err),
		})
		return
	}

	response := BackupResponse{
		Status:  "success",
		Message: "Restore completed successfully and context reloaded",
		Path:    req.BackupPath,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
