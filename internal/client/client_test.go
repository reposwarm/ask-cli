package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(HealthResponse{
			Status:       "healthy",
			ArchHubReady: true,
			ArchHubRepos: 5,
			JobsTotal:    10,
		})
	}))
	defer srv.Close()

	c := New(srv.URL)
	h, err := c.Health()
	if err != nil {
		t.Fatal(err)
	}
	if h.Status != "healthy" {
		t.Errorf("expected healthy, got %q", h.Status)
	}
	if h.ArchHubRepos != 5 {
		t.Errorf("expected 5 repos, got %d", h.ArchHubRepos)
	}
}

func TestAskSubmit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/ask" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		var req AskRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.Question != "test question" {
			t.Errorf("expected question, got %q", req.Question)
		}
		json.NewEncoder(w).Encode(AskResponse{ID: "abc123", Status: "queued"})
	}))
	defer srv.Close()

	c := New(srv.URL)
	resp, err := c.Ask(&AskRequest{Question: "test question"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.ID != "abc123" {
		t.Errorf("expected abc123, got %q", resp.ID)
	}
}

func TestGetJob(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(JobResponse{
			ID:        "abc123",
			Status:    "completed",
			Answer:    "The answer is 42",
			ToolCalls: 3,
		})
	}))
	defer srv.Close()

	c := New(srv.URL)
	job, err := c.GetJob("abc123")
	if err != nil {
		t.Fatal(err)
	}
	if job.Status != "completed" {
		t.Errorf("expected completed, got %q", job.Status)
	}
	if job.Answer != "The answer is 42" {
		t.Errorf("unexpected answer: %q", job.Answer)
	}
}

func TestListJobs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]JobResponse{
			{ID: "a", Status: "completed"},
			{ID: "b", Status: "running"},
		})
	}))
	defer srv.Close()

	c := New(srv.URL)
	jobs, err := c.ListJobs("", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 2 {
		t.Errorf("expected 2 jobs, got %d", len(jobs))
	}
}

func TestAsk503ArchHubNotLoaded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()

	c := New(srv.URL)
	_, err := c.Ask(&AskRequest{Question: "test"})
	if err == nil {
		t.Fatal("expected error for 503")
	}
}

func TestGetJob404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	c := New(srv.URL)
	_, err := c.GetJob("nonexistent")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}
