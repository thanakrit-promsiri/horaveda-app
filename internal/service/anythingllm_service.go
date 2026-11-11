package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"horaveda-app/internal/domain"
	"horaveda-app/pkg/response"

	"go.uber.org/zap"
)

const (
	DefaultAnythingLLMAPIURL  = "http://localhost:3001/api/v1"
	DefaultAnythingLLMTimeout = 300 * time.Second
	anythingLLMChatMode       = "chat"
	anythingLLMThreadSuffix   = "-hora"
)

var (
	pairedThinkTagRegex      = regexp.MustCompile(`(?is)<think\b[^>]*>.*?</think\s*>`)
	selfClosingThinkTagRegex = regexp.MustCompile(`(?i)<think\b[^>]*/\s*>`)
	orphanCloseThinkTagRegex = regexp.MustCompile(`(?i)</think\s*>`)
	unclosedThinkTagRegex    = regexp.MustCompile(`(?is)<think\b[^>]*>.*$`)
)

// StripThinkTags ลบส่วนกระบวนการคิดของโมเดล (<think>...</think> รวมถึงกรณีแท็กปิดหรือเปิดไม่สมบูรณ์) ออกจากข้อความตอบกลับของ AI
func StripThinkTags(text string) string {
	cleaned := pairedThinkTagRegex.ReplaceAllString(text, "")
	cleaned = selfClosingThinkTagRegex.ReplaceAllString(cleaned, "")
	if locs := orphanCloseThinkTagRegex.FindAllStringIndex(cleaned, -1); len(locs) > 0 {
		cleaned = cleaned[locs[len(locs)-1][1]:]
	}
	cleaned = unclosedThinkTagRegex.ReplaceAllString(cleaned, "")
	return strings.TrimSpace(cleaned)
}

// AnythingLLMService เชื่อมต่อ AnythingLLM Developer API แบบ 1 User : 1 Workspace : 1 Thread
type AnythingLLMService struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
	Logger     *zap.SugaredLogger

	// threadCache เก็บ workspace slug (ที่คาดหวัง) -> threadRef ที่ตรวจสอบแล้ว เพื่อลดการเรียก API ซ้ำ
	threadCache sync.Map
	// locks ป้องกันการสร้าง Workspace/Thread ซ้ำเมื่อผู้ใช้คนเดียวกันยิงคำขอพร้อมกัน
	locks sync.Map
}

type threadRef struct {
	WorkspaceSlug string
	ThreadSlug    string
}

type anythingLLMThread struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type anythingLLMWorkspace struct {
	ID      int                 `json:"id"`
	Name    string              `json:"name"`
	Slug    string              `json:"slug"`
	Threads []anythingLLMThread `json:"threads"`
}

type anythingLLMChatResponse struct {
	ID           string      `json:"id"`
	Type         string      `json:"type"`
	TextResponse string      `json:"textResponse"`
	Close        bool        `json:"close"`
	Error        interface{} `json:"error"`
}

// NewAnythingLLMService สร้าง Service พร้อมค่า Default (URL: http://localhost:3001/api/v1, Timeout: 300 วินาที)
func NewAnythingLLMService(baseURL, apiKey string, timeout time.Duration, log *zap.SugaredLogger, httpClient *http.Client) *AnythingLLMService {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultAnythingLLMAPIURL
	}
	if timeout <= 0 {
		timeout = DefaultAnythingLLMTimeout
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: timeout}
	}
	if log == nil {
		log = zap.NewNop().Sugar()
	}
	return &AnythingLLMService{
		BaseURL:    strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		APIKey:     strings.TrimSpace(apiKey),
		HTTPClient: httpClient,
		Logger:     log,
	}
}

// IsEnabled คืนค่า true เมื่อมีการตั้งค่า API Key ของ AnythingLLM แล้ว
func (s *AnythingLLMService) IsEnabled() bool {
	return s != nil && s.APIKey != ""
}

// WorkspaceSlugFor แปลง username เป็น workspace slug (AnythingLLM จะ slugify ชื่อเป็นตัวพิมพ์เล็ก)
func WorkspaceSlugFor(username string) string {
	return strings.ToLower(strings.TrimSpace(username))
}

// ThreadSlugFor กำหนด thread slug เริ่มต้นประจำ Workspace ของผู้ใช้
func ThreadSlugFor(workspaceSlug string) string {
	return workspaceSlug + anythingLLMThreadSuffix
}

// TransitThreadSlugFor กำหนด thread slug ประจำการพยากรณ์ดาวจร (ใช้ thread เดียวกันกับพื้นดวงชะตา: 1 User : 1 Workspace : 1 Thread)
func TransitThreadSlugFor(workspaceSlug string) string {
	return ThreadSlugFor(workspaceSlug)
}

// Ping ตรวจสอบการเชื่อมต่อไปยัง AnythingLLM ผ่าน GET /auth เพื่อดูว่า API Key ถูกต้องและเซิร์ฟเวอร์พร้อมใช้งานหรือไม่
func (s *AnythingLLMService) Ping(ctx context.Context) error {
	status, body, err := s.doJSON(ctx, http.MethodGet, "/auth", nil)
	if err != nil {
		return err
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return response.NewUnauthorizedError("AnythingLLM API Key ไม่ถูกต้อง (Invalid API Key)", fmt.Errorf("status %d", status))
	}
	if !isSuccessStatus(status) {
		return s.upstreamError("ตรวจสอบสถานะ (Ping)", status, body)
	}
	return nil
}

// PredictNatalChart ประกอบ Prompt พยากรณ์พื้นดวง แล้วส่งเข้า Thread ประจำตัวผู้ใช้บน AnythingLLM
func (s *AnythingLLMService) PredictNatalChart(ctx context.Context, username, chartPrompt string) (*domain.AnythingLLMPrediction, error) {
	if !s.IsEnabled() {
		return nil, response.NewInternalServerError("ระบบยังไม่ได้ตั้งค่าการเชื่อมต่อ AnythingLLM", fmt.Errorf("ANYTHINGLLM_API_KEY is not configured"))
	}
	if strings.TrimSpace(chartPrompt) == "" {
		return nil, response.NewBadGatewayError("ไม่ได้รับข้อมูลพื้นดวงชะตาสำหรับการพยากรณ์", fmt.Errorf("empty chart prompt"))
	}

	message := BuildVedicPredictionPrompt(chartPrompt)
	return s.predictWithSharedThread(ctx, username, "natal", message)
}

// PredictTransitChart ส่งข้อความ Prompt พยากรณ์ดาวจรประจำเดือน เข้า Thread ประจำตัวผู้ใช้ (ใช้ Thread เดียวกับพื้นดวงชะตากำเนิด: 1 User : 1 Workspace : 1 Thread)
func (s *AnythingLLMService) PredictTransitChart(ctx context.Context, username, transitPrompt string) (*domain.AnythingLLMPrediction, error) {
	if !s.IsEnabled() {
		return nil, response.NewInternalServerError("ระบบยังไม่ได้ตั้งค่าการเชื่อมต่อ AnythingLLM", fmt.Errorf("ANYTHINGLLM_API_KEY is not configured"))
	}
	if strings.TrimSpace(transitPrompt) == "" {
		return nil, response.NewBadGatewayError("ไม่ได้รับข้อมูลพยากรณ์ดาวจรสำหรับการส่งไปยัง AI", fmt.Errorf("empty transit prompt"))
	}

	return s.predictWithSharedThread(ctx, username, "transit", transitPrompt)
}

// predictWithSharedThread ส่งข้อความ Prompt เข้า Thread ร่วม ({workspace}-hora) พร้อมระบบ Auto-Recovery (404 Retry)
func (s *AnythingLLMService) predictWithSharedThread(ctx context.Context, username, pipelineType, promptMessage string) (*domain.AnythingLLMPrediction, error) {
	start := time.Now()
	expectedWS := WorkspaceSlugFor(username)
	expectedThread := ThreadSlugFor(expectedWS)

	s.Logger.Infow("Starting AnythingLLM "+pipelineType+" prediction pipeline",
		"username", username,
		"workspace_slug", expectedWS,
		"thread_slug", expectedThread,
		"base_url", s.BaseURL,
		"prompt_len", len(promptMessage),
	)

	ref, err := s.ensureUserThread(ctx, username)
	if err != nil {
		s.Logger.Warnw("AnythingLLM "+pipelineType+" workspace/thread setup failed",
			"username", username,
			"elapsed_ms", time.Since(start).Milliseconds(),
			"error", err,
		)
		return nil, err
	}

	text, err := s.sendThreadChat(ctx, ref, promptMessage)
	if err != nil {
		if isNotFoundError(err) {
			s.Logger.Warnw("Workspace or thread not found in AnythingLLM (404), recreating and retrying prediction",
				"username", username,
				"stale_workspace_slug", ref.WorkspaceSlug,
				"stale_thread_slug", ref.ThreadSlug,
			)
			s.threadCache.Delete(expectedWS)

			newRef, ensureErr := s.ensureUserThread(ctx, username)
			if ensureErr == nil {
				retryText, retryErr := s.sendThreadChat(ctx, newRef, promptMessage)
				if retryErr == nil {
					s.Logger.Infow("AnythingLLM "+pipelineType+" prediction retry succeeded after workspace/thread recreation",
						"username", username,
						"workspace_slug", newRef.WorkspaceSlug,
						"thread_slug", newRef.ThreadSlug,
					)
					return &domain.AnythingLLMPrediction{
						WorkspaceSlug: newRef.WorkspaceSlug,
						ThreadSlug:    newRef.ThreadSlug,
						TextResponse:  retryText,
					}, nil
				}
				err = retryErr
			} else {
				err = ensureErr
			}
		}

		s.Logger.Warnw("AnythingLLM "+pipelineType+" chat execution failed",
			"username", username,
			"workspace_slug", ref.WorkspaceSlug,
			"thread_slug", ref.ThreadSlug,
			"elapsed_ms", time.Since(start).Milliseconds(),
			"error", err,
		)
		var appErr *response.AppError
		if errors.As(err, &appErr) && appErr.Code == http.StatusNotFound {
			s.threadCache.Delete(expectedWS)
		}
		return nil, err
	}

	s.Logger.Infow("AnythingLLM "+pipelineType+" prediction completed successfully",
		"username", username,
		"workspace_slug", ref.WorkspaceSlug,
		"thread_slug", ref.ThreadSlug,
		"response_len", len(text),
		"elapsed_ms", time.Since(start).Milliseconds(),
	)

	return &domain.AnythingLLMPrediction{
		WorkspaceSlug: ref.WorkspaceSlug,
		ThreadSlug:    ref.ThreadSlug,
		TextResponse:  text,
	}, nil
}

// ensureUserThread ตรวจสอบ/สร้าง Workspace และ Thread ประจำตัวผู้ใช้ (1 User : 1 Workspace : 1 Thread ใช้ร่วมกันทั้งพื้นดวงชะตาและดาวจร)
func (s *AnythingLLMService) ensureUserThread(ctx context.Context, username string) (*threadRef, error) {
	expectedWS := WorkspaceSlugFor(username)
	if expectedWS == "" {
		return nil, response.NewBadRequestError("กรุณาระบุ username สำหรับการพยากรณ์", fmt.Errorf("empty username"))
	}

	if cached, ok := s.threadCache.Load(expectedWS); ok {
		ref := cached.(threadRef)
		return &ref, nil
	}

	mu := s.lockFor(expectedWS)
	mu.Lock()
	defer mu.Unlock()

	// ตรวจ Cache อีกครั้ง เผื่อคำขอก่อนหน้าสร้างเสร็จแล้วระหว่างรอ Lock
	if cached, ok := s.threadCache.Load(expectedWS); ok {
		ref := cached.(threadRef)
		return &ref, nil
	}

	// 1. ตรวจสอบค้นหา Workspace ที่มีอยู่แล้วก่อน (ตรวจสอบทั้ง GET /workspace/:slug และ GET /workspaces)
	// หากพบว่ามีอยู่แล้ว จะนำมาใช้ต่อทันทีโดยเด็ดขาด ไม่สร้างใหม่
	ws, err := s.findWorkspace(ctx, username, expectedWS)
	if err != nil {
		return nil, err
	}

	if ws == nil {
		// ผู้ใช้ใหม่จริงๆ ที่ยังไม่มี Workspace ในระบบ: สร้าง Workspace ประจำตัว
		s.Logger.Infow("Workspace does not exist in AnythingLLM, creating new workspace", "username", username, "expected_slug", expectedWS)
		ws, err = s.createWorkspace(ctx, username)
		if err != nil {
			return nil, err
		}
		if ws.Slug != expectedWS {
			s.Logger.Warnw("AnythingLLM workspace slug differs from expected", "expected", expectedWS, "actual", ws.Slug)
		}
	} else {
		s.Logger.Infow("Reusing existing AnythingLLM workspace (skip creation)", "username", username, "workspace_slug", ws.Slug)
	}

	// 2. ตรวจสอบ Thread ประจำตัวผู้ใช้ (ใช้ Thread เดียวกันทั้งการพยากรณ์พื้นดวงชะตาและดาวจร)
	threadSlug := ThreadSlugFor(ws.Slug)
	if !hasThread(ws.Threads, threadSlug) {
		// ตรวจสอบกับรายการ threads จาก /workspace/{slug}/threads อีกครั้งเผื่อใน cache ยังไม่ได้อัปเดต
		latestThreads, err := s.listThreads(ctx, ws.Slug)
		if err == nil && hasThread(latestThreads, threadSlug) {
			s.Logger.Infow("Reusing existing AnythingLLM thread (skip creation)", "workspace_slug", ws.Slug, "thread_slug", threadSlug)
			ws.Threads = latestThreads
		} else {
			createdSlug, err := s.createThread(ctx, ws.Slug, threadSlug, "พื้นดวงชะตา "+username)
			if err != nil {
				return nil, err
			}
			if createdSlug != threadSlug {
				s.Logger.Warnw("AnythingLLM thread slug differs from expected", "expected", threadSlug, "actual", createdSlug)
			}
			threadSlug = createdSlug
		}
	} else {
		s.Logger.Infow("Reusing existing AnythingLLM thread (skip creation)", "workspace_slug", ws.Slug, "thread_slug", threadSlug)
	}

	ref := threadRef{WorkspaceSlug: ws.Slug, ThreadSlug: threadSlug}
	s.threadCache.Store(expectedWS, ref)
	s.Logger.Infow("AnythingLLM user thread is ready", "username", username, "workspace_slug", ref.WorkspaceSlug, "thread_slug", ref.ThreadSlug)
	return &ref, nil
}

func (s *AnythingLLMService) lockFor(key string) *sync.Mutex {
	mu, _ := s.locks.LoadOrStore(key, &sync.Mutex{})
	return mu.(*sync.Mutex)
}

func hasThread(threads []anythingLLMThread, slug string) bool {
	for _, t := range threads {
		if strings.EqualFold(t.Slug, slug) || strings.EqualFold(t.Name, slug) {
			return true
		}
	}
	return false
}

// findWorkspace ค้นหา Workspace ที่มีอยู่แล้วใน AnythingLLM โดย:
// 1. ค้นหาโดยตรงผ่าน GET /workspace/{slug}
// 2. หากยังไม่พบ จะค้นหาจากรายการทั้งหมดผ่าน GET /workspaces เพื่อจับคู่ทั้ง Slug และ Name (แบบ Case-Insensitive)
// เพื่อให้แน่ใจว่าจะไม่สร้าง Workspace ซ้ำเด็ดขาดหากมีอยู่แล้วในระบบ
func (s *AnythingLLMService) findWorkspace(ctx context.Context, username, expectedSlug string) (*anythingLLMWorkspace, error) {
	// 1. ลองค้นหาโดยตรงผ่าน slug
	ws, err := s.getWorkspace(ctx, expectedSlug)
	if err != nil {
		return nil, err
	}
	if ws != nil {
		return ws, nil
	}

	// 2. หากไม่พบโดยตรง ให้ดึงรายการ Workspace ทั้งหมด (GET /workspaces) มาค้นหาเทียบทั้ง Slug และ Name
	list, err := s.listWorkspaces(ctx)
	if err != nil {
		s.Logger.Warnw("Failed to list workspaces for existence check, will proceed with caution", "error", err)
		return nil, nil
	}

	for i := range list {
		w := &list[i]
		if strings.EqualFold(w.Slug, expectedSlug) ||
			strings.EqualFold(w.Name, username) ||
			strings.EqualFold(w.Name, expectedSlug) {
			s.Logger.Infow("Matched existing workspace in AnythingLLM workspace list",
				"username", username,
				"expected_slug", expectedSlug,
				"matched_slug", w.Slug,
				"matched_name", w.Name,
			)
			// หากพบแต่ยังไม่มี threads โหลดมา ให้เรียก getWorkspace ดึง threads มาด้วย
			if len(w.Threads) == 0 {
				fullWs, err := s.getWorkspace(ctx, w.Slug)
				if err == nil && fullWs != nil {
					return fullWs, nil
				}
			}
			return w, nil
		}
	}

	return nil, nil
}

// listWorkspaces เรียก GET /workspaces เพื่อดึงรายการ Workspace ทั้งหมดใน AnythingLLM
func (s *AnythingLLMService) listWorkspaces(ctx context.Context) ([]anythingLLMWorkspace, error) {
	status, body, err := s.doJSON(ctx, http.MethodGet, "/workspaces", nil)
	if err != nil {
		return nil, err
	}
	if !isSuccessStatus(status) {
		return nil, s.upstreamError("ดึงรายการ Workspaces", status, body)
	}

	var res struct {
		Workspaces []anythingLLMWorkspace `json:"workspaces"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, s.invalidResponseError("ดึงรายการ Workspaces", err, body)
	}
	return res.Workspaces, nil
}

// listThreads เรียก GET /workspace/{slug}/threads เพื่อดึงรายการ Thread ทั้งหมดใน Workspace
func (s *AnythingLLMService) listThreads(ctx context.Context, workspaceSlug string) ([]anythingLLMThread, error) {
	path := "/workspace/" + url.PathEscape(workspaceSlug) + "/threads"
	status, body, err := s.doJSON(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if !isSuccessStatus(status) {
		return nil, s.upstreamError("ดึงรายการ Threads", status, body)
	}

	var res struct {
		Threads []anythingLLMThread `json:"threads"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, s.invalidResponseError("ดึงรายการ Threads", err, body)
	}
	return res.Threads, nil
}

// getWorkspace เรียก GET /workspace/{slug} คืนค่า nil เมื่อไม่พบ Workspace
func (s *AnythingLLMService) getWorkspace(ctx context.Context, slug string) (*anythingLLMWorkspace, error) {
	status, body, err := s.doJSON(ctx, http.MethodGet, "/workspace/"+url.PathEscape(slug), nil)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, nil
	}
	if !isSuccessStatus(status) {
		return nil, s.upstreamError("ตรวจสอบ Workspace", status, body)
	}

	var envelope struct {
		Workspace json.RawMessage `json:"workspace"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, s.invalidResponseError("ตรวจสอบ Workspace", err, body)
	}

	raw := bytes.TrimSpace(envelope.Workspace)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, nil
	}

	// AnythingLLM เวอร์ชันใหม่คืนค่าเป็น Array ส่วนเวอร์ชันเก่าคืนค่าเป็น Object
	if raw[0] == '[' {
		var list []anythingLLMWorkspace
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, s.invalidResponseError("ตรวจสอบ Workspace", err, body)
		}
		for i := range list {
			if strings.EqualFold(list[i].Slug, slug) || strings.EqualFold(list[i].Name, slug) {
				return &list[i], nil
			}
		}
		return nil, nil
	}

	var ws anythingLLMWorkspace
	if err := json.Unmarshal(raw, &ws); err != nil {
		return nil, s.invalidResponseError("ตรวจสอบ Workspace", err, body)
	}
	if ws.Slug == "" {
		return nil, nil
	}
	if !strings.EqualFold(ws.Slug, slug) && !strings.EqualFold(ws.Name, slug) {
		return nil, nil
	}
	return &ws, nil
}

// createWorkspace เรียก POST /workspace/new โดยใช้ username เป็นชื่อ Workspace
func (s *AnythingLLMService) createWorkspace(ctx context.Context, username string) (*anythingLLMWorkspace, error) {
	payload := map[string]interface{}{
		"name":         username,
		"openAiPrompt": VedicAstrologerSystemPrompt,
		"chatMode":     anythingLLMChatMode,
	}
	status, body, err := s.doJSON(ctx, http.MethodPost, "/workspace/new", payload)
	if err != nil {
		return nil, err
	}
	if !isSuccessStatus(status) {
		return nil, s.upstreamError("สร้าง Workspace", status, body)
	}

	var res struct {
		Workspace *anythingLLMWorkspace `json:"workspace"`
		Message   interface{}           `json:"message"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, s.invalidResponseError("สร้าง Workspace", err, body)
	}
	if res.Workspace == nil || res.Workspace.Slug == "" {
		return nil, s.upstreamError("สร้าง Workspace", status, body)
	}

	s.Logger.Infow("Created AnythingLLM workspace", "username", username, "workspace_slug", res.Workspace.Slug)
	return res.Workspace, nil
}

// createThread เรียก POST /workspace/{slug}/thread/new เพื่อสร้าง Thread ประจำตัวผู้ใช้
func (s *AnythingLLMService) createThread(ctx context.Context, workspaceSlug, threadSlug, threadTitle string) (string, error) {
	payload := map[string]interface{}{
		"name": threadTitle,
		"slug": threadSlug,
	}
	path := "/workspace/" + url.PathEscape(workspaceSlug) + "/thread/new"
	status, body, err := s.doJSON(ctx, http.MethodPost, path, payload)
	if err != nil {
		return "", err
	}
	if !isSuccessStatus(status) {
		return "", s.upstreamError("สร้าง Thread", status, body)
	}

	var res struct {
		Thread  *anythingLLMThread `json:"thread"`
		Message interface{}        `json:"message"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return "", s.invalidResponseError("สร้าง Thread", err, body)
	}
	if res.Thread == nil || res.Thread.Slug == "" {
		return "", s.upstreamError("สร้าง Thread", status, body)
	}

	s.Logger.Infow("Created AnythingLLM thread", "workspace_slug", workspaceSlug, "thread_slug", res.Thread.Slug)
	return res.Thread.Slug, nil
}

// sendThreadChat เรียก POST /workspace/{slug}/thread/{threadSlug}/chat และคืนค่าข้อความคำทำนาย
func (s *AnythingLLMService) sendThreadChat(ctx context.Context, ref *threadRef, message string) (string, error) {
	payload := map[string]interface{}{
		"message": message,
		"mode":    anythingLLMChatMode,
		"reset":   false,
	}
	path := "/workspace/" + url.PathEscape(ref.WorkspaceSlug) + "/thread/" + url.PathEscape(ref.ThreadSlug) + "/chat"

	s.Logger.Infow("Sending prediction prompt to AnythingLLM", "workspace_slug", ref.WorkspaceSlug, "thread_slug", ref.ThreadSlug, "prompt_length", len(message))

	status, body, err := s.doJSON(ctx, http.MethodPost, path, payload)
	if err != nil {
		return "", err
	}
	if status == http.StatusNotFound {
		s.Logger.Warnw("AnythingLLM thread or workspace not found (404)", "workspace_slug", ref.WorkspaceSlug, "thread_slug", ref.ThreadSlug)
		return "", response.NewNotFoundError("ไม่พบ Workspace หรือ Thread บน AnythingLLM", fmt.Errorf("anythingllm chat status 404: %s", truncateForLog(body)))
	}
	if !isSuccessStatus(status) {
		return "", s.upstreamError("ส่งข้อความพยากรณ์", status, body)
	}

	var res anythingLLMChatResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return "", s.invalidResponseError("ส่งข้อความพยากรณ์", err, body)
	}
	if hasChatError(res.Error) {
		errStr := fmt.Sprintf("%v", res.Error)
		if strings.Contains(strings.ToLower(errStr), "not found") {
			s.Logger.Warnw("AnythingLLM chat returned not found error", "workspace_slug", ref.WorkspaceSlug, "thread_slug", ref.ThreadSlug, "error", res.Error)
			return "", response.NewNotFoundError("ไม่พบ Workspace หรือ Thread บน AnythingLLM", fmt.Errorf("anythingllm chat error: %s", errStr))
		}
		s.Logger.Errorw("AnythingLLM chat returned error", "workspace_slug", ref.WorkspaceSlug, "thread_slug", ref.ThreadSlug, "error", res.Error)
		return "", response.NewBadGatewayError("AnythingLLM ไม่สามารถประมวลผลคำทำนายได้", fmt.Errorf("anythingllm chat error: %v", res.Error))
	}
	cleanedText := StripThinkTags(res.TextResponse)
	if cleanedText == "" {
		return "", response.NewBadGatewayError("AnythingLLM ไม่ได้ส่งคำทำนายกลับมา", fmt.Errorf("empty textResponse"))
	}

	s.Logger.Infow("Received prediction from AnythingLLM",
		"workspace_slug", ref.WorkspaceSlug,
		"thread_slug", ref.ThreadSlug,
		"raw_length", len(res.TextResponse),
		"response_length", len(cleanedText),
		"had_think_tag", cleanedText != strings.TrimSpace(res.TextResponse),
	)
	return cleanedText, nil
}

func hasChatError(v interface{}) bool {
	switch e := v.(type) {
	case nil:
		return false
	case bool:
		return e
	case string:
		return strings.TrimSpace(e) != ""
	default:
		return true
	}
}

// doJSON ส่งคำขอ JSON ไปยัง AnythingLLM พร้อม Header Authorization: Bearer <API Key>
func (s *AnythingLLMService) doJSON(ctx context.Context, method, path string, payload interface{}) (int, []byte, error) {
	var reader io.Reader
	if payload != nil {
		bodyBytes, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, response.NewInternalServerError("ไม่สามารถเตรียมข้อมูลสำหรับ AnythingLLM ได้", err)
		}
		reader = bytes.NewReader(bodyBytes)
	}

	targetURL := s.BaseURL + path
	httpReq, err := http.NewRequestWithContext(ctx, method, targetURL, reader)
	if err != nil {
		return 0, nil, response.NewInternalServerError("ไม่สามารถสร้างคำขอไปยัง AnythingLLM ได้", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+s.APIKey)
	httpReq.Header.Set("Accept", "application/json")
	if payload != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	httpClient := s.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: DefaultAnythingLLMTimeout}
	}

	start := time.Now()
	resp, err := httpClient.Do(httpReq)
	elapsed := time.Since(start)
	if err != nil {
		reason, hint := diagnoseNetworkError(err, targetURL)
		s.Logger.Errorw("Failed to call AnythingLLM API",
			"method", method,
			"url", targetURL,
			"elapsed_ms", elapsed.Milliseconds(),
			"error_reason", reason,
			"diagnostic_hint", hint,
			"api_key_configured", s.APIKey != "",
			"api_key_preview", maskAPIKey(s.APIKey),
			"error", err,
		)
		return 0, nil, response.NewBadGatewayError("ไม่สามารถเชื่อมต่อกับบริการ AnythingLLM ได้: "+reason, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		s.Logger.Errorw("Failed to read AnythingLLM response", "method", method, "url", targetURL, "elapsed_ms", elapsed.Milliseconds(), "error", err)
		return 0, nil, response.NewBadGatewayError("ไม่สามารถอ่านผลลัพธ์จาก AnythingLLM ได้", err)
	}
	return resp.StatusCode, respBytes, nil
}

// diagnoseNetworkError วิเคราะห์ข้อผิดพลาดระดับ Network เพื่อระบุสาเหตุและคำแนะนำในการแก้ไขที่ชัดเจน
func diagnoseNetworkError(err error, targetURL string) (reason string, hint string) {
	if err == nil {
		return "UNKNOWN", ""
	}
	errStr := strings.ToLower(err.Error())
	switch {
	case strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "connectex"):
		return "CONNECTION_REFUSED",
			"ไม่มีบริการเปิดรับการเชื่อมต่อที่ URL ดังกล่าว กรุณาตรวจสอบว่า: " +
				"1) AnythingLLM (Desktop หรือ Docker) เปิดทำงานอยู่หรือไม่ " +
				"2) พอร์ตที่ใช้งานคือ 3001 หรือไม่ (หากเป็นพอร์ตอื่น เช่น 3000 ให้แก้ ANYTHINGLLM_API_URL ใน .env) " +
				"3) หากรันใน Docker อาจต้องใช้ http://host.docker.internal:3001 แทน localhost"
	case strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded"):
		return "TIMEOUT",
			"เซิร์ฟเวอร์ AnythingLLM ใช้เวลาตอบสนองนานเกินกำหนด กรุณาเพิ่มค่า ANYTHINGLLM_TIMEOUT_SECONDS ใน .env หรือตรวจสอบภาระงานของเครื่อง"
	case strings.Contains(errStr, "no such host"):
		return "DNS_RESOLUTION_FAILED",
			"ไม่พบ Hostname ปลายทาง กรุณาตรวจสอบว่าโดเมนหรือ IP ใน ANYTHINGLLM_API_URL สะกดถูกต้อง"
	case strings.Contains(errStr, "certificate") || strings.Contains(errStr, "tls"):
		return "TLS_HANDSHAKE_FAILED",
			"เกิดข้อผิดพลาดในการตรวจสอบใบรับรอง SSL/TLS"
	default:
		return "NETWORK_ERROR",
			"เกิดข้อผิดพลาดทางเครือข่ายระหว่างเชื่อมต่อไปยัง AnythingLLM: " + err.Error()
	}
}

// maskAPIKey ซ่อนส่วนสำคัญของ API Key สำหรับบันทึกใน Log อย่างปลอดภัย
func maskAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func isSuccessStatus(status int) bool {
	return status >= http.StatusOK && status < http.StatusMultipleChoices
}

func (s *AnythingLLMService) upstreamError(action string, status int, body []byte) error {
	s.Logger.Errorw("AnythingLLM API returned unexpected response", "action", action, "status_code", status, "body", truncateForLog(body))
	return response.NewBadGatewayError(
		"บริการ AnythingLLM แจ้งข้อผิดพลาดระหว่าง"+action,
		fmt.Errorf("anythingllm %s status %d: %s", action, status, truncateForLog(body)),
	)
}

func (s *AnythingLLMService) invalidResponseError(action string, err error, body []byte) error {
	s.Logger.Errorw("Failed to parse AnythingLLM response", "action", action, "error", err, "body", truncateForLog(body))
	return response.NewBadGatewayError("รูปแบบข้อมูลจาก AnythingLLM ไม่ถูกต้อง", err)
}

func truncateForLog(body []byte) string {
	const max = 500
	if len(body) > max {
		return string(body[:max]) + "...(truncated)"
	}
	return string(body)
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	var appErr *response.AppError
	if errors.As(err, &appErr) && appErr.Code == http.StatusNotFound {
		return true
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "not found") || strings.Contains(errStr, "404")
}
