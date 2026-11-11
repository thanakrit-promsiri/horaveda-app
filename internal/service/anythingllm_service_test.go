package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"

	"horaveda-app/pkg/response"
)

// fakeAnythingLLM จำลอง AnythingLLM Developer API ผ่าน RoundTripper (ไม่เปิดพอร์ต TCP จริง)
type fakeAnythingLLM struct {
	mu              sync.Mutex
	workspaces      map[string][]string // workspace slug -> thread slugs
	calls           []string
	chatStatus      int
	chatBody        string
	lastChatMessage string
	lastAuthHeader  string
	getAsObject     bool
}

func newFakeAnythingLLM() *fakeAnythingLLM {
	return &fakeAnythingLLM{
		workspaces: map[string][]string{},
		chatStatus: http.StatusOK,
		chatBody:   `{"id":"abc","type":"textResponse","textResponse":"คำทำนายพื้นดวง","sources":[],"close":true,"error":null}`,
	}
}

func (f *fakeAnythingLLM) handler(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	path := strings.TrimPrefix(r.URL.Path, "/api/v1")
	f.calls = append(f.calls, r.Method+" "+path)
	f.lastAuthHeader = r.Header.Get("Authorization")
	w.Header().Set("Content-Type", "application/json")

	parts := strings.Split(strings.Trim(path, "/"), "/")
	switch {
	case r.Method == http.MethodGet && path == "/auth":
		if f.lastAuthHeader == "Bearer invalid-key" {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error":"Invalid API Key"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"authenticated":true}`))

	case r.Method == http.MethodGet && path == "/workspaces":
		list := []map[string]interface{}{}
		for slug, threads := range f.workspaces {
			ts := []map[string]string{}
			for _, t := range threads {
				ts = append(ts, map[string]string{"slug": t})
			}
			list = append(list, map[string]interface{}{"id": 1, "name": slug, "slug": slug, "threads": ts})
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"workspaces": list})

	case r.Method == http.MethodGet && len(parts) == 3 && parts[0] == "workspace" && parts[2] == "threads":
		threads, ok := f.workspaces[parts[1]]
		if !ok {
			_, _ = w.Write([]byte(`{"threads":[]}`))
			return
		}
		ts := []map[string]string{}
		for _, t := range threads {
			ts = append(ts, map[string]string{"slug": t})
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"threads": ts})

	case r.Method == http.MethodGet && len(parts) == 2 && parts[0] == "workspace":
		threads, ok := f.workspaces[parts[1]]
		if !ok {
			_, _ = w.Write([]byte(`{"workspace":[]}`))
			return
		}
		ts := []map[string]string{}
		for _, t := range threads {
			ts = append(ts, map[string]string{"slug": t})
		}
		ws := map[string]interface{}{"id": 1, "name": parts[1], "slug": parts[1], "threads": ts}
		if f.getAsObject {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"workspace": ws})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"workspace": []interface{}{ws}})

	case r.Method == http.MethodPost && path == "/workspace/new":
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		slug := strings.ToLower(body["name"].(string))
		f.workspaces[slug] = []string{}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"workspace": map[string]interface{}{"id": 1, "name": body["name"], "slug": slug},
			"message":   "Workspace created",
		})

	case r.Method == http.MethodPost && len(parts) == 4 && parts[2] == "thread" && parts[3] == "new":
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		slug := body["slug"].(string)
		f.workspaces[parts[1]] = append(f.workspaces[parts[1]], slug)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"thread":  map[string]interface{}{"id": 1, "name": body["name"], "slug": slug},
			"message": nil,
		})

	case r.Method == http.MethodPost && len(parts) == 5 && parts[2] == "thread" && parts[4] == "chat":
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.lastChatMessage, _ = body["message"].(string)
		w.WriteHeader(f.chatStatus)
		_, _ = w.Write([]byte(f.chatBody))

	default:
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"not found"}`))
	}
}

func (f *fakeAnythingLLM) service() *AnythingLLMService {
	return NewAnythingLLMService("http://localhost:3001/api/v1/", "test-llm-key", 0, newTestLogger(), newMockHTTPClient(f.handler))
}

func TestAnythingLLMService_PredictNatalChart(t *testing.T) {
	chart := "[Vedic Sidereal(Lahiri)/WholeSign]\nASC:Aq 03°39'23.46\""

	t.Run("new user creates workspace, thread and sends chat", func(t *testing.T) {
		fake := newFakeAnythingLLM()
		svc := fake.service()

		res, err := svc.PredictNatalChart(context.Background(), "TestUser01", chart)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if res.WorkspaceSlug != "testuser01" || res.ThreadSlug != "testuser01-hora" || res.TextResponse != "คำทำนายพื้นดวง" {
			t.Errorf("unexpected result: %+v", res)
		}

		expected := []string{
			"GET /workspace/testuser01",
			"GET /workspaces",
			"POST /workspace/new",
			"GET /workspace/testuser01/threads",
			"POST /workspace/testuser01/thread/new",
			"POST /workspace/testuser01/thread/testuser01-hora/chat",
		}
		if strings.Join(fake.calls, "|") != strings.Join(expected, "|") {
			t.Errorf("unexpected call sequence:\n got %v\nwant %v", fake.calls, expected)
		}
		if fake.lastAuthHeader != "Bearer test-llm-key" {
			t.Errorf("unexpected Authorization header: %q", fake.lastAuthHeader)
		}
		if !strings.Contains(fake.lastChatMessage, chart) || !strings.Contains(fake.lastChatMessage, "Drishti") || !strings.Contains(fake.lastChatMessage, "D9") {
			t.Errorf("chat message missing chart or Vedic instructions: %s", fake.lastChatMessage)
		}
	})

	t.Run("existing workspace in /workspaces list reuses it and skips creation", func(t *testing.T) {
		fake := newFakeAnythingLLM()
		// จำลองกรณีที่ GET /workspace/:slug คืนค่าว่าง แต่มี workspace อยู่แล้วใน GET /workspaces
		fake.workspaces["testuser01"] = []string{"testuser01-hora"}
		// ลบ handler ของ direct slug ให้คืนว่าง เพื่อทดสอบ fallback list lookup
		oldHandler := fake.handler
		callCount := 0
		svc := NewAnythingLLMService("http://localhost:3001/api/v1/", "test-llm-key", 0, newTestLogger(), newMockHTTPClient(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/v1/workspace/testuser01") && r.Method == http.MethodGet && !strings.Contains(r.URL.Path, "threads") {
				callCount++
				if callCount == 1 {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"workspace":[]}`))
					return
				}
			}
			oldHandler(w, r)
		}))

		res, err := svc.PredictNatalChart(context.Background(), "testuser01", chart)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if res.WorkspaceSlug != "testuser01" || res.ThreadSlug != "testuser01-hora" {
			t.Errorf("unexpected result: %+v", res)
		}
		for _, call := range fake.calls {
			if strings.Contains(call, "POST /workspace/new") {
				t.Fatalf("POST /workspace/new should NOT be called when workspace exists in list! calls=%v", fake.calls)
			}
		}
	})

	t.Run("existing workspace and thread only sends chat, then uses cache", func(t *testing.T) {
		fake := newFakeAnythingLLM()
		fake.workspaces["testuser01"] = []string{"testuser01-hora"}
		svc := fake.service()

		if _, err := svc.PredictNatalChart(context.Background(), "testuser01", chart); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if _, err := svc.PredictNatalChart(context.Background(), "testuser01", chart); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		expected := []string{
			"GET /workspace/testuser01",
			"POST /workspace/testuser01/thread/testuser01-hora/chat",
			"POST /workspace/testuser01/thread/testuser01-hora/chat",
		}
		if strings.Join(fake.calls, "|") != strings.Join(expected, "|") {
			t.Errorf("unexpected call sequence:\n got %v\nwant %v", fake.calls, expected)
		}
	})

	t.Run("existing workspace (object format) without thread creates thread", func(t *testing.T) {
		fake := newFakeAnythingLLM()
		fake.getAsObject = true
		fake.workspaces["testuser01"] = []string{"other-thread"}
		svc := fake.service()

		res, err := svc.PredictNatalChart(context.Background(), "testuser01", chart)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if res.ThreadSlug != "testuser01-hora" {
			t.Errorf("unexpected thread slug: %s", res.ThreadSlug)
		}
		if fake.calls[2] != "POST /workspace/testuser01/thread/new" {
			t.Errorf("expected thread creation at index 2, got %v", fake.calls)
		}
	})

	t.Run("chat 404 error clears cache but other errors retain cache", func(t *testing.T) {
		fake := newFakeAnythingLLM()
		fake.workspaces["testuser01"] = []string{"testuser01-hora"}
		fake.chatStatus = http.StatusInternalServerError
		fake.chatBody = `{"error":"LLM provider 503"}`
		svc := fake.service()

		_, err := svc.PredictNatalChart(context.Background(), "testuser01", chart)
		var appErr *response.AppError
		if !errors.As(err, &appErr) || appErr.Code != http.StatusBadGateway {
			t.Fatalf("expected 502 AppError, got %v", err)
		}
		// ข้อผิดพลาดที่ไม่ใช่ 404 ต้องคง Cache ไว้เพื่อไม่ให้สร้าง Workspace ซ้ำ
		if _, ok := svc.threadCache.Load("testuser01"); !ok {
			t.Error("expected thread cache to be retained after non-404 error")
		}

		// แต่หากเป็น 404 Not Found ต้องล้าง cache
		fake.chatStatus = http.StatusNotFound
		fake.chatBody = `{"error":"workspace or thread not found"}`
		_, _ = svc.PredictNatalChart(context.Background(), "testuser01", chart)
		if _, ok := svc.threadCache.Load("testuser01"); ok {
			t.Error("expected thread cache to be cleared after 404 error")
		}
	})

	t.Run("chat 404 auto-recovers by recreating workspace and thread, then succeeding on retry", func(t *testing.T) {
		fake := newFakeAnythingLLM()
		svc := fake.service()

		// จำลองว่าเคยแคชไว้แต่ Workspace ถูกลบออกจาก AnythingLLM
		svc.threadCache.Store("testuser01", threadRef{WorkspaceSlug: "testuser01", ThreadSlug: "testuser01-hora"})

		// ครั้งแรกที่ส่ง chat จะตอบกลับ 404 ว่าไม่พบ Workspace
		fake.chatStatus = http.StatusNotFound
		fake.chatBody = `{"id":"c3fe8cb2","type":"abort","textResponse":null,"sources":[],"close":true,"error":"Workspace testuser01 not found."}`

		// เมื่อมีการเรียกสร้าง workspace ใหม่ ให้สลับสถานะ chat กลับเป็น 200 OK
		oldHandler := fake.handler
		callCount := 0
		svc.HTTPClient = newMockHTTPClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/chat") {
				callCount++
				if callCount == 1 {
					fake.chatStatus = http.StatusNotFound
					fake.chatBody = `{"error":"Workspace testuser01 not found."}`
				} else {
					fake.chatStatus = http.StatusOK
					fake.chatBody = `{"id":"abc","type":"textResponse","textResponse":"คำทำนายที่สำเร็จหลัง retry","close":true,"error":null}`
				}
			}
			oldHandler(w, r)
		}))

		res, err := svc.PredictNatalChart(context.Background(), "testuser01", chart)
		if err != nil {
			t.Fatalf("expected auto-recovery to succeed, got error: %v", err)
		}
		if res.TextResponse != "คำทำนายที่สำเร็จหลัง retry" {
			t.Errorf("unexpected prediction text: %s", res.TextResponse)
		}
		if callCount != 2 {
			t.Errorf("expected 2 chat calls (1 failure + 1 retry), got %d", callCount)
		}
	})

	t.Run("chat response with error field returns 502", func(t *testing.T) {
		fake := newFakeAnythingLLM()
		fake.workspaces["testuser01"] = []string{"testuser01-hora"}
		fake.chatBody = `{"id":"abc","type":"abort","textResponse":null,"close":true,"error":"No LLM configured"}`
		svc := fake.service()

		_, err := svc.PredictNatalChart(context.Background(), "testuser01", chart)
		var appErr *response.AppError
		if !errors.As(err, &appErr) || appErr.Code != http.StatusBadGateway {
			t.Fatalf("expected 502 AppError, got %v", err)
		}
	})

	t.Run("network error returns 502", func(t *testing.T) {
		errClient := &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("connection refused")
			}),
		}
		svc := NewAnythingLLMService("", "test-llm-key", 0, newTestLogger(), errClient)
		_, err := svc.PredictNatalChart(context.Background(), "testuser01", chart)
		var appErr *response.AppError
		if !errors.As(err, &appErr) || appErr.Code != http.StatusBadGateway {
			t.Fatalf("expected 502 AppError, got %v", err)
		}
	})

	t.Run("not configured returns 500", func(t *testing.T) {
		svc := NewAnythingLLMService("", "", 0, newTestLogger(), nil)
		if svc.IsEnabled() {
			t.Fatal("expected service to be disabled without API key")
		}
		if svc.BaseURL != DefaultAnythingLLMAPIURL {
			t.Errorf("expected default base URL, got %s", svc.BaseURL)
		}
		_, err := svc.PredictNatalChart(context.Background(), "testuser01", chart)
		var appErr *response.AppError
		if !errors.As(err, &appErr) || appErr.Code != http.StatusInternalServerError {
			t.Fatalf("expected 500 AppError, got %v", err)
		}
	})

	t.Run("empty username returns 400", func(t *testing.T) {
		fake := newFakeAnythingLLM()
		_, err := fake.service().PredictNatalChart(context.Background(), "  ", chart)
		var appErr *response.AppError
		if !errors.As(err, &appErr) || appErr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 AppError, got %v", err)
		}
	})

	t.Run("strips think tags from prediction response", func(t *testing.T) {
		fake := newFakeAnythingLLM()
		fake.workspaces["testuser01"] = []string{"testuser01-hora"}
		fake.chatBody = `{"id":"abc","type":"textResponse","textResponse":"\u003cthink\u003e\nวิเคราะห์ลัคนา\n\u003c/think\u003e\n\nคำทำนายพื้นดวงจริง","close":true,"error":null}`

		res, err := fake.service().PredictNatalChart(context.Background(), "testuser01", chart)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if res.TextResponse != "คำทำนายพื้นดวงจริง" {
			t.Errorf("expected stripped prediction, got %q", res.TextResponse)
		}
	})

	t.Run("returns 502 when response contains only think tags", func(t *testing.T) {
		fake := newFakeAnythingLLM()
		fake.workspaces["testuser01"] = []string{"testuser01-hora"}
		fake.chatBody = `{"id":"abc","type":"textResponse","textResponse":"\u003cthink\u003eมีแต่กระบวนการคิด\u003c/think\u003e","close":true,"error":null}`

		_, err := fake.service().PredictNatalChart(context.Background(), "testuser01", chart)
		var appErr *response.AppError
		if !errors.As(err, &appErr) || appErr.Code != http.StatusBadGateway {
			t.Fatalf("expected 502 AppError when response has only think block, got %v", err)
		}
	})
}

func TestBuildVedicPredictionPrompt(t *testing.T) {
	prompt := BuildVedicPredictionPrompt("  ASC:Aq 03°39'  ")
	for _, want := range []string{"ASC:Aq 03°39'", "Lagna", "ตนุลัคน์", "นักษัตร", "Drishti", "การกุม", "D9", "นวางศ์"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("expected prompt to contain %q", want)
		}
	}
}

func TestBuildVedicTransitPredictionPrompt(t *testing.T) {
	if got := ThaiMonthName(1); got != "มกราคม" {
		t.Errorf("expected มกราคม, got %s", got)
	}
	if got := ThaiMonthName(12); got != "ธันวาคม" {
		t.Errorf("expected ธันวาคม, got %s", got)
	}
	if got := ToBuddhistYear(2026); got != 2569 {
		t.Errorf("expected 2569, got %d", got)
	}
	if got := ToBuddhistYear(2027); got != 2570 {
		t.Errorf("expected 2570, got %d", got)
	}
	if got := ToBuddhistYear(2570); got != 2570 {
		t.Errorf("expected 2570 for existing BE year, got %d", got)
	}

	full := FormatThaiMonthYear(2027, 1)
	if full != "เดือนมกราคม พ.ศ. 2570" {
		t.Errorf("expected 'เดือนมกราคม พ.ศ. 2570', got %q", full)
	}

	short := FormatThaiMonthYearShort(2027, 1)
	if short != "เดือนมกราคม 2570" {
		t.Errorf("expected 'เดือนมกราคม 2570', got %q", short)
	}

	natalPrompt := "[Vedic Sidereal(Lahiri)/WholeSign]\nASC:Aq 03°39'"
	transitPrompt := "[Vedic Transit(Lahiri)/WholeSign | Date:2027-01-23]\nTransit: Ju:Ge 05°"

	assembled := BuildVedicTransitPredictionPrompt(natalPrompt, transitPrompt, 2027, 1)

	expectedSnippets := []string{
		"คำพยากรณ์ดวงชะตาประจำเดือนมกราคม พ.ศ. 2570",
		"อิทธิพลดาวจรสำคัญในเดือนมกราคม พ.ศ. 2570 ส่งผลต่อลัคนา มีทิศทางที่โดดเด่นดังนี้:",
		"ดาวจรเร็ว",
		"ดาวจรช้า",
		"•\nการงานและการศึกษา:",
		"•\nการเงินและโชคลาภ:",
		"•\nความรักและสุขภาพ:",
		"🪐 ดาวจรสำคัญ:",
		"✨ เดือนมกราคม 2570:",
		"Sattvic Upaya",
		"ASC:Aq 03°39'",
		"Ju:Ge 05°",
	}

	for _, want := range expectedSnippets {
		if !strings.Contains(assembled, want) {
			t.Errorf("assembled transit prompt missing snippet: %q", want)
		}
	}
}

func TestAnythingLLMService_PredictTransitChart(t *testing.T) {
	transitPrompt := "Prompt for transit"

	t.Run("uses same thread as natal chart and sends chat", func(t *testing.T) {
		fake := newFakeAnythingLLM()
		svc := fake.service()

		res, err := svc.PredictTransitChart(context.Background(), "testuser01", transitPrompt)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if res.WorkspaceSlug != "testuser01" || res.ThreadSlug != "testuser01-hora" || res.TextResponse != "คำทำนายพื้นดวง" {
			t.Errorf("unexpected result: %+v", res)
		}

		expected := []string{
			"GET /workspace/testuser01",
			"GET /workspaces",
			"POST /workspace/new",
			"GET /workspace/testuser01/threads",
			"POST /workspace/testuser01/thread/new",
			"POST /workspace/testuser01/thread/testuser01-hora/chat",
		}
		if strings.Join(fake.calls, "|") != strings.Join(expected, "|") {
			t.Errorf("unexpected call sequence:\n got %v\nwant %v", fake.calls, expected)
		}
	})

	t.Run("natal prediction followed by transit prediction reuses same thread without creating new thread", func(t *testing.T) {
		fake := newFakeAnythingLLM()
		svc := fake.service()

		// 1. ทำนายพื้นดวงกำเนิด
		resNatal, err := svc.PredictNatalChart(context.Background(), "testuser01", "natal chart")
		if err != nil {
			t.Fatalf("natal prediction failed: %v", err)
		}
		if resNatal.ThreadSlug != "testuser01-hora" {
			t.Errorf("unexpected natal thread slug: %s", resNatal.ThreadSlug)
		}

		// 2. ทำนายดาวจร -> ต้องใช้ Thread เดียวกัน (testuser01-hora)
		resTransit, err := svc.PredictTransitChart(context.Background(), "testuser01", transitPrompt)
		if err != nil {
			t.Fatalf("transit prediction failed: %v", err)
		}
		if resTransit.ThreadSlug != resNatal.ThreadSlug {
			t.Errorf("expected same thread slug: natal=%s, transit=%s", resNatal.ThreadSlug, resTransit.ThreadSlug)
		}

		// ตรวจสอบว่าเรียกสร้าง Thread เพียงแค่ครั้งเดียวเท่านั้น
		threadNewCalls := 0
		for _, call := range fake.calls {
			if strings.Contains(call, "/thread/new") {
				threadNewCalls++
			}
		}
		if threadNewCalls != 1 {
			t.Errorf("expected exactly 1 thread creation, got %d. calls: %v", threadNewCalls, fake.calls)
		}
	})

	t.Run("empty prompt returns 502", func(t *testing.T) {
		fake := newFakeAnythingLLM()
		_, err := fake.service().PredictTransitChart(context.Background(), "testuser01", "  ")
		var appErr *response.AppError
		if !errors.As(err, &appErr) || appErr.Code != http.StatusBadGateway {
			t.Fatalf("expected 502 AppError, got %v", err)
		}
	})
}

func TestAnythingLLMService_Ping(t *testing.T) {
	t.Run("ping success", func(t *testing.T) {
		fake := newFakeAnythingLLM()
		svc := fake.service()
		if err := svc.Ping(context.Background()); err != nil {
			t.Fatalf("expected ping to succeed, got %v", err)
		}
	})

	t.Run("ping invalid key returns 401", func(t *testing.T) {
		fake := newFakeAnythingLLM()
		svc := NewAnythingLLMService("http://localhost:3001/api/v1/", "invalid-key", 0, newTestLogger(), newMockHTTPClient(fake.handler))
		err := svc.Ping(context.Background())
		var appErr *response.AppError
		if !errors.As(err, &appErr) || appErr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 unauthorized, got %v", err)
		}
	})
}

func TestDiagnoseNetworkErrorAndMaskKey(t *testing.T) {
	reason, hint := diagnoseNetworkError(errors.New("dial tcp 127.0.0.1:3001: connect: connection refused"), "http://localhost:3001")
	if reason != "CONNECTION_REFUSED" || !strings.Contains(hint, "AnythingLLM") {
		t.Errorf("unexpected diagnosis: %s, %s", reason, hint)
	}

	reasonTimeout, _ := diagnoseNetworkError(errors.New("context deadline exceeded (Client.Timeout)"), "http://localhost:3001")
	if reasonTimeout != "TIMEOUT" {
		t.Errorf("unexpected timeout diagnosis: %s", reasonTimeout)
	}

	if maskAPIKey("12345678") != "***" {
		t.Errorf("expected *** for short key")
	}
	if masked := maskAPIKey("SKQ80Y1-E6AMVCW-JSFSM5M-CY856Q4"); masked != "SKQ8...56Q4" {
		t.Errorf("expected SKQ8...56Q4, got %s", masked)
	}
}

func TestStripThinkTags(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain prediction without tags",
			in:   "ลัคนาสถิตราศีกุมภ์",
			want: "ลัคนาสถิตราศีกุมภ์",
		},
		{
			name: "standard multiline think block before prediction",
			in:   "<think>\nขั้นตอนการคิด\nวิเคราะห์ดวง\n</think>\n\n**1. แกนหลักของชีวิต**",
			want: "**1. แกนหลักของชีวิต**",
		},
		{
			name: "multiple blocks and case-insensitive tags",
			in:   "<THINK data-model=\"deepseek\">รอบ 1</THINK >ส่วนที่ 1\n<think>รอบ 2</think>\nส่วนที่ 2",
			want: "ส่วนที่ 1\n\nส่วนที่ 2",
		},
		{
			name: "orphan closing tag from prefilled prompt",
			in:   "คิดวิเคราะห์ก่อนปิด\n</think>\n\nคำพยากรณ์จริง",
			want: "คำพยากรณ์จริง",
		},
		{
			name: "self-closing think tag",
			in:   "<think/>คำพยากรณ์จริง",
			want: "คำพยากรณ์จริง",
		},
		{
			name: "unclosed trailing think tag",
			in:   "คำพยากรณ์จริง\n<think>ข้อความค้างที่ถูกตัด",
			want: "คำพยากรณ์จริง",
		},
		{
			name: "only unclosed think block becomes empty",
			in:   "<think>กำลังคิดยังไม่เสร็จ...",
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := StripThinkTags(tc.in); got != tc.want {
				t.Errorf("StripThinkTags(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
