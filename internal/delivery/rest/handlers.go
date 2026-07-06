package rest

import (
	"encoding/json"
	"net/http"

	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/go-chi/chi/v5"
)

// Response helper
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// ─── Auth Handlers ────────────────────────────────────────────────────────

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	token, err := s.service.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"token": token})
}

// ─── Config Handlers ──────────────────────────────────────────────────────

func (s *Server) handleGetConfigAll(w http.ResponseWriter, r *http.Request) {
	configs, err := s.service.GetConfigAll(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, configs)
}

func (s *Server) handleGetConfigByKey(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	config, err := s.service.GetConfigByKey(r.Context(), key)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if config == nil {
		respondError(w, http.StatusNotFound, "Config not found")
		return
	}
	respondJSON(w, http.StatusOK, config)
}

type updateConfigRequest struct {
	ValueJSON string `json:"value_json"`
}

func (s *Server) handleUpdateConfig(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	var req updateConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := s.service.UpdateConfig(r.Context(), key, req.ValueJSON); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// ─── Entities Handlers ────────────────────────────────────────────────────

func (s *Server) handleListEntitySchemas(w http.ResponseWriter, r *http.Request) {
	schemas, err := s.service.ListEntitySchemas(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, schemas)
}

type createSchemaRequest struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description"`
	FieldsJSON  string `json:"fields_json"`
}

func (s *Server) handleCreateEntitySchema(w http.ResponseWriter, r *http.Request) {
	var req createSchemaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	schema, err := s.service.CreateEntitySchema(r.Context(), req.Name, req.Label, req.Description, req.FieldsJSON)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, schema)
}

func (s *Server) handleGetEntitySchema(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	schema, err := s.service.GetEntitySchema(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if schema == nil {
		respondError(w, http.StatusNotFound, "Schema not found")
		return
	}
	respondJSON(w, http.StatusOK, schema)
}

func (s *Server) handleUpdateEntitySchema(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req createSchemaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err := s.service.UpdateEntitySchema(r.Context(), id, req.Name, req.Label, req.Description, req.FieldsJSON)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (s *Server) handleDeleteEntitySchema(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.service.DeleteEntitySchema(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (s *Server) handleListEntityRecords(w http.ResponseWriter, r *http.Request) {
	schemaID := chi.URLParam(r, "schema_id")
	records, err := s.service.ListEntityRecords(r.Context(), schemaID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, records)
}

type createRecordRequest struct {
	DataJSON string `json:"data_json"`
}

func (s *Server) handleCreateEntityRecord(w http.ResponseWriter, r *http.Request) {
	schemaID := chi.URLParam(r, "schema_id")
	var req createRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	record, err := s.service.CreateEntityRecord(r.Context(), schemaID, req.DataJSON)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, record)
}

func (s *Server) handleGetEntityRecord(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	record, err := s.service.GetEntityRecord(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if record == nil {
		respondError(w, http.StatusNotFound, "Record not found")
		return
	}
	respondJSON(w, http.StatusOK, record)
}

func (s *Server) handleUpdateEntityRecord(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req createRecordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	err := s.service.UpdateEntityRecord(r.Context(), id, req.DataJSON)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (s *Server) handleDeleteEntityRecord(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.service.DeleteEntityRecord(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// ─── Cron Handlers ────────────────────────────────────────────────────────

func (s *Server) handleListCronJobs(w http.ResponseWriter, r *http.Request) {
	jobs, err := s.service.ListCronJobs(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, jobs)
}

func (s *Server) handleCreateCronJob(w http.ResponseWriter, r *http.Request) {
	var job domain.CronJob
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	created, err := s.service.CreateCronJob(r.Context(), job)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, created)
}

func (s *Server) handleUpdateCronJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var job domain.CronJob
	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	job.ID = id
	if err := s.service.UpdateCronJob(r.Context(), job); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (s *Server) handleDeleteCronJob(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.service.DeleteCronJob(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

// ─── Stock Handlers ───────────────────────────────────────────────────────

func (s *Server) handleListStocks(w http.ResponseWriter, r *http.Request) {
	stocks, err := s.service.ListStocks(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, stocks)
}

func (s *Server) handleAddStock(w http.ResponseWriter, r *http.Request) {
	var stock domain.StockWatchlist
	if err := json.NewDecoder(r.Body).Decode(&stock); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	created, err := s.service.AddStock(r.Context(), stock)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, created)
}

func (s *Server) handleUpdateStock(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var stock domain.StockWatchlist
	if err := json.NewDecoder(r.Body).Decode(&stock); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	stock.ID = id
	if err := s.service.UpdateStock(r.Context(), stock); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}

func (s *Server) handleDeleteStock(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.service.DeleteStock(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "success"})
}
