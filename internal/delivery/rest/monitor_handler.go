package rest

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/achmichael/pribadi-go/internal/domain"
	"github.com/achmichael/pribadi-go/internal/repository"
	"github.com/achmichael/pribadi-go/internal/usecase/monitor"
	"github.com/go-chi/chi/v5"
)

type MonitorHandler struct {
	repo      repository.MonitorRepository
	httpConn  *monitor.HTTPConnector
	extractor *monitor.ScrapeRuleExtractor
}

func NewMonitorHandler(repo repository.MonitorRepository, httpConn *monitor.HTTPConnector, extractor *monitor.ScrapeRuleExtractor) *MonitorHandler {
	return &MonitorHandler{repo: repo, httpConn: httpConn, extractor: extractor}
}

func (h *MonitorHandler) RegisterRoutes(r chi.Router) {
	r.Get("/monitors", h.handleList)
	r.Post("/monitors", h.handleCreate)
	r.Get("/monitors/{id}", h.handleGet)
	r.Put("/monitors/{id}", h.handleUpdate)
	r.Delete("/monitors/{id}", h.handleDelete)
	r.Post("/monitors/test-extract", h.handleTestExtract)
	r.Post("/monitors/test-derive-rule", h.handleTestDeriveRule)
}

func (h *MonitorHandler) handleList(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = r.Context().Value("user_id").(string);
	}

	tasks, err := h.repo.ListByUser(r.Context(), userID, category)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if tasks == nil {
		tasks = []domain.MonitorTask{}
	}
	respondJSON(w, http.StatusOK, tasks)
}

func (h *MonitorHandler) handleGet(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	task, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if task == nil {
		respondError(w, http.StatusNotFound, "not found")
		return
	}
	respondJSON(w, http.StatusOK, task)
}

func (h *MonitorHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var task domain.MonitorTask
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		respondError(w, http.StatusBadRequest, "invalid body")
		return
	}

	// Ensure UserID is set from the authenticated session
	if task.UserID == "" {
		if ctxUser, ok := r.Context().Value("user_id").(string); ok {
			task.UserID = ctxUser
		} else {
			task.UserID = "admin" // fallback
		}
	}
	if task.Platform == "" {
		task.Platform = "whatsapp"
	}

	if task.PollIntervalSeconds == 0 {
		if task.ConditionMode == domain.ConditionNLJudge {
			task.PollIntervalSeconds = 1800
		} else {
			task.PollIntervalSeconds = 300
		}
	}
	id, err := h.repo.Create(r.Context(), task)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (h *MonitorHandler) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var task domain.MonitorTask
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		respondError(w, http.StatusBadRequest, "invalid body")
		return
	}
	task.ID = id
	if err := h.repo.Update(r.Context(), task); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (h *MonitorHandler) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.repo.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

type testExtractRequest struct {
	URL         string            `json:"url"`
	Method      string            `json:"method"`
	Headers     map[string]string `json:"headers,omitempty"`
	AuthType    string            `json:"auth_type,omitempty"`
	AuthValue   string            `json:"auth_value,omitempty"`
	Body        string            `json:"body,omitempty"`
	ExtractPath string            `json:"extract_path"`
	ExtractAsText bool            `json:"extract_as_text"`
}

func (h *MonitorHandler) handleTestExtract(w http.ResponseWriter, r *http.Request) {
	var req testExtractRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid body")
		return
	}

	cfgJSON, _ := json.Marshal(domain.HTTPSourceConfig{
		URL:           req.URL,
		Method:        req.Method,
		Headers:       req.Headers,
		AuthType:      req.AuthType,
		AuthValue:     req.AuthValue,
		Body:          req.Body,
		ExtractPath:   req.ExtractPath,
		ExtractAsText: req.ExtractAsText,
	})

	task := domain.MonitorTask{
		SourceType:   domain.SourceHTTPAPI,
		SourceConfig: string(cfgJSON),
	}

	numeric, text, raw, err := h.httpConn.Fetch(r.Context(), task)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
			"raw":     truncate(raw, 2000),
		})
		return
	}

	result := map[string]interface{}{
		"success": true,
		"text":    text,
		"raw":     truncate(raw, 2000),
	}
	if numeric != nil {
		result["numeric"] = *numeric
	}
	respondJSON(w, http.StatusOK, result)
}

type testDeriveRuleRequest struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

func (h *MonitorHandler) handleTestDeriveRule(w http.ResponseWriter, r *http.Request) {
	var req testDeriveRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid body")
		return
	}

	rule, err := h.extractor.DeriveExtractionRule(r.Context(), 0, req.URL, req.Description)
	if err != nil {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"success":   true,
		"selector":  rule.Selector,
		"rule_type": rule.RuleType,
	})
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}
