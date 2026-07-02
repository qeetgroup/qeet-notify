package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/qeetgroup/qeet-notify/platform/api/middleware"
	"github.com/qeetgroup/qeet-notify/platform/database"
)

type workflowRow struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	TriggerEvent string         `json:"trigger_event"`
	Steps        []any          `json:"steps"`
	IsActive     bool           `json:"is_active"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type workflowRunRow struct {
	ID               string     `json:"id"`
	WorkflowID       string     `json:"workflow_id"`
	SubscriberID     *string    `json:"subscriber_id,omitempty"`
	TriggerEvent     string     `json:"trigger_event"`
	Status           string     `json:"status"`
	CurrentStepIndex int        `json:"current_step_index"`
	Error            *string    `json:"error,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// ListWorkflows returns all workflows for the authenticated tenant.
func ListWorkflows(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())
		q := database.FromContext(r.Context(), pool)

		limit := 50
		offset := 0
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
				limit = n
			}
		}
		if o := r.URL.Query().Get("offset"); o != "" {
			if n, err := strconv.Atoi(o); err == nil && n >= 0 {
				offset = n
			}
		}

		rows, err := q.Query(r.Context(),
			`SELECT id, name, trigger_event, steps, is_active, created_at, updated_at
			 FROM workflows WHERE tenant_id = $1
			 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
			tenantID, limit, offset,
		)
		if err != nil {
			http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var workflows []workflowRow
		for rows.Next() {
			var wf workflowRow
			var steps []byte
			if err := rows.Scan(&wf.ID, &wf.Name, &wf.TriggerEvent, &steps,
				&wf.IsActive, &wf.CreatedAt, &wf.UpdatedAt); err != nil {
				continue
			}
			json.Unmarshal(steps, &wf.Steps) //nolint:errcheck
			workflows = append(workflows, wf)
		}
		if workflows == nil {
			workflows = []workflowRow{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"workflows": workflows, "total": len(workflows)}) //nolint:errcheck
	}
}

// GetWorkflow returns a single workflow by ID.
func GetWorkflow(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())
		q := database.FromContext(r.Context(), pool)
		id := chi.URLParam(r, "id")

		var wf workflowRow
		var steps []byte
		err := q.QueryRow(r.Context(),
			`SELECT id, name, trigger_event, steps, is_active, created_at, updated_at
			 FROM workflows WHERE id = $1 AND tenant_id = $2`,
			id, tenantID,
		).Scan(&wf.ID, &wf.Name, &wf.TriggerEvent, &steps,
			&wf.IsActive, &wf.CreatedAt, &wf.UpdatedAt)
		if err != nil {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		json.Unmarshal(steps, &wf.Steps) //nolint:errcheck

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wf) //nolint:errcheck
	}
}

// CreateWorkflow creates a new workflow definition.
func CreateWorkflow(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())
		q := database.FromContext(r.Context(), pool)

		var req struct {
			Name         string `json:"name"`
			TriggerEvent string `json:"trigger_event"`
			Steps        []any  `json:"steps"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}
		if req.Name == "" || req.TriggerEvent == "" {
			http.Error(w, `{"error":"name and trigger_event are required"}`, http.StatusUnprocessableEntity)
			return
		}
		if req.Steps == nil {
			req.Steps = []any{}
		}
		steps, _ := json.Marshal(req.Steps)

		var id string
		err := q.QueryRow(r.Context(),
			`INSERT INTO workflows (tenant_id, name, trigger_event, steps)
			 VALUES ($1, $2, $3, $4) RETURNING id`,
			tenantID, req.Name, req.TriggerEvent, steps,
		).Scan(&id)
		if err != nil {
			http.Error(w, `{"error":"create failed"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"id": id}) //nolint:errcheck
	}
}

// UpdateWorkflow updates a workflow's name, trigger, or steps.
func UpdateWorkflow(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())
		q := database.FromContext(r.Context(), pool)
		id := chi.URLParam(r, "id")

		var req struct {
			Name         *string `json:"name"`
			TriggerEvent *string `json:"trigger_event"`
			Steps        []any   `json:"steps"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}

		var cur workflowRow
		var stepsBytes []byte
		err := q.QueryRow(r.Context(),
			`SELECT id, name, trigger_event, steps, is_active, created_at, updated_at
			 FROM workflows WHERE id=$1 AND tenant_id=$2`,
			id, tenantID,
		).Scan(&cur.ID, &cur.Name, &cur.TriggerEvent, &stepsBytes,
			&cur.IsActive, &cur.CreatedAt, &cur.UpdatedAt)
		if err != nil {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}

		if req.Name != nil {
			cur.Name = *req.Name
		}
		if req.TriggerEvent != nil {
			cur.TriggerEvent = *req.TriggerEvent
		}
		if req.Steps != nil {
			stepsBytes, _ = json.Marshal(req.Steps)
		}

		_, err = q.Exec(r.Context(),
			`UPDATE workflows SET name=$1, trigger_event=$2, steps=$3, updated_at=NOW()
			 WHERE id=$4 AND tenant_id=$5`,
			cur.Name, cur.TriggerEvent, stepsBytes, id, tenantID,
		)
		if err != nil {
			http.Error(w, `{"error":"update failed"}`, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": id}) //nolint:errcheck
	}
}

// ArchiveWorkflow soft-deletes a workflow (sets is_active=false).
func ArchiveWorkflow(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())
		q := database.FromContext(r.Context(), pool)
		id := chi.URLParam(r, "id")

		result, err := q.Exec(r.Context(),
			`UPDATE workflows SET is_active=FALSE, updated_at=NOW() WHERE id=$1 AND tenant_id=$2`,
			id, tenantID,
		)
		if err != nil || result.RowsAffected() == 0 {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// ActivateWorkflow sets a workflow to active.
func ActivateWorkflow(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())
		q := database.FromContext(r.Context(), pool)
		id := chi.URLParam(r, "id")

		result, err := q.Exec(r.Context(),
			`UPDATE workflows SET is_active=TRUE, updated_at=NOW() WHERE id=$1 AND tenant_id=$2`,
			id, tenantID,
		)
		if err != nil || result.RowsAffected() == 0 {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": id, "status": "active"}) //nolint:errcheck
	}
}

// PauseWorkflow sets a workflow to inactive.
func PauseWorkflow(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())
		q := database.FromContext(r.Context(), pool)
		id := chi.URLParam(r, "id")

		result, err := q.Exec(r.Context(),
			`UPDATE workflows SET is_active=FALSE, updated_at=NOW() WHERE id=$1 AND tenant_id=$2`,
			id, tenantID,
		)
		if err != nil || result.RowsAffected() == 0 {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": id, "status": "paused"}) //nolint:errcheck
	}
}

// ListWorkflowRuns returns the last N runs for a workflow.
func ListWorkflowRuns(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID, _ := middleware.TenantFromContext(r.Context())
		q := database.FromContext(r.Context(), pool)
		workflowID := chi.URLParam(r, "id")

		limit := 20
		if l := r.URL.Query().Get("limit"); l != "" {
			if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 100 {
				limit = n
			}
		}

		rows, err := q.Query(r.Context(),
			`SELECT id, workflow_id, subscriber_id, trigger_event, status,
			        current_step_index, error, created_at, updated_at
			 FROM workflow_runs
			 WHERE workflow_id = $1 AND tenant_id = $2
			 ORDER BY created_at DESC LIMIT $3`,
			workflowID, tenantID, limit,
		)
		if err != nil {
			http.Error(w, `{"error":"query failed"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var runs []workflowRunRow
		for rows.Next() {
			var run workflowRunRow
			if err := rows.Scan(&run.ID, &run.WorkflowID, &run.SubscriberID, &run.TriggerEvent,
				&run.Status, &run.CurrentStepIndex, &run.Error, &run.CreatedAt, &run.UpdatedAt); err != nil {
				continue
			}
			runs = append(runs, run)
		}
		if runs == nil {
			runs = []workflowRunRow{}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"runs": runs}) //nolint:errcheck
	}
}
