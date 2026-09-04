package talknote

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestListGroupsUsesSessionCookie(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/web-api/v1/group" || r.Header.Get("Cookie") != "TALKNOTE_SID2=session" {
			t.Errorf("request=%s cookie=%q", r.URL.Path, r.Header.Get("Cookie"))
		}
		_, _ = w.Write([]byte(`{"content":[{"data":{"id":735387,"name":"全社","openStatus":"OPEN"},"reaction":{"youUnreadCount":2}}]}`))
	}))
	defer server.Close()
	groups, err := New(server.URL, "session").ListGroups(context.Background())
	if err != nil || len(groups) != 1 || groups[0].Data.ID != "735387" {
		t.Fatalf("groups=%+v err=%v", groups, err)
	}
}

func TestCreateFeedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/web-api/v1/group/42/feed" {
			t.Errorf("request=%s %s", r.Method, r.URL.Path)
		}
		var body postRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Content != "hello" || body.AttachmentIDs == nil {
			t.Errorf("body=%+v", body)
		}
		_, _ = w.Write([]byte(`{"id":91,"groupId":42,"content":"hello"}`))
	}))
	defer server.Close()
	feed, err := New(server.URL, "session").CreateFeed(context.Background(), "42", "hello")
	if err != nil || feed.Data.ID != "91" {
		t.Fatalf("feed=%+v err=%v", feed, err)
	}
}

func TestCollectionRecordNotFoundIsEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Error_RecordNotFound: タスクはありません。"}`))
	}))
	defer server.Close()
	client := New(server.URL, "session")
	feeds, err := client.ListFeeds(context.Background(), "1", 20)
	if err != nil || feeds == nil || len(feeds) != 0 {
		t.Fatalf("feeds=%v err=%v", feeds, err)
	}
	comments, err := client.ListComments(context.Background(), "1", "2", 20)
	if err != nil || comments == nil || len(comments) != 0 {
		t.Fatalf("comments=%v err=%v", comments, err)
	}
	messages, err := client.ListMessages(context.Background(), "3", 20)
	if err != nil || messages == nil || len(messages) != 0 {
		t.Fatalf("messages=%v err=%v", messages, err)
	}
	results, err := client.Search(context.Background(), "none", 20)
	if err != nil || results == nil || len(results) != 0 {
		t.Fatalf("results=%v err=%v", results, err)
	}
	tasks, err := client.ListTasks(context.Background(), "INCOMPLETE")
	if err != nil || tasks == nil || len(tasks) != 0 {
		t.Fatalf("tasks=%v err=%v", tasks, err)
	}
}

func TestAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"unauthorized"}`))
	}))
	defer server.Close()
	_, err := New(server.URL, "bad").Me(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusUnauthorized || !strings.Contains(apiErr.Message, "tn auth login") {
		t.Fatalf("err=%T %v", err, err)
	}
}

func TestNonJSONErrorDoesNotExposeResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("TALKNOTE_SID2=server-secret"))
	}))
	defer server.Close()
	_, err := New(server.URL, "client-secret").Me(context.Background())
	if err == nil || strings.Contains(err.Error(), "server-secret") || strings.Contains(err.Error(), "client-secret") {
		t.Fatalf("err=%v", err)
	}
}

func TestSuccessfulEmptyResponseIsRejectedWhenDataIsExpected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	defer server.Close()
	_, err := New(server.URL, "session").ListGroups(context.Background())
	if err == nil || !strings.Contains(err.Error(), "empty response") {
		t.Fatalf("err=%v", err)
	}
}

func TestDefaultClientDoesNotFollowRedirects(t *testing.T) {
	redirectTargetRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirected" {
			redirectTargetRequests++
			return
		}
		http.Redirect(w, r, "/redirected", http.StatusFound)
	}))
	defer server.Close()
	_, err := New(server.URL, "session").Me(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusFound || redirectTargetRequests != 0 {
		t.Fatalf("err=%v target_requests=%d", err, redirectTargetRequests)
	}
}

func TestCreateResponseRequiresID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	_, err := New(server.URL, "session").CreateFeed(context.Background(), "1", "hello")
	if err == nil || !strings.Contains(err.Error(), "has no ID") {
		t.Fatalf("err=%v", err)
	}
}

func TestMutationPaths(t *testing.T) {
	expected := []struct{ method, path string }{
		{http.MethodPut, "/web-api/v1/group/1/feed/2"},
		{http.MethodDelete, "/web-api/v1/group/1/feed/2"},
		{http.MethodPut, "/web-api/v1/group/1/feed/2/like"},
		{http.MethodDelete, "/web-api/v1/group/1/feed/2/like"},
		{http.MethodPost, "/web-api/v1/group/1/feed/2/comment"},
		{http.MethodPut, "/web-api/v1/group/1/feed/2/comment/3"},
		{http.MethodDelete, "/web-api/v1/group/1/feed/2/comment/3"},
		{http.MethodPut, "/web-api/v1/group/1/feed/2/comment/3/like"},
		{http.MethodDelete, "/web-api/v1/group/1/feed/2/comment/3/like"},
		{http.MethodPost, "/web-api/v1/thread/4/message"},
		{http.MethodPut, "/web-api/v1/thread/4/message/5"},
		{http.MethodDelete, "/web-api/v1/thread/4/message/5"},
		{http.MethodPost, "/web-api/v1/task"},
		{http.MethodPut, "/web-api/v1/task/6/complete"},
		{http.MethodPut, "/web-api/v1/task/6/revert"},
		{http.MethodDelete, "/web-api/v1/task/6"},
	}
	requestIndex := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := expected[requestIndex]
		if r.Method != want.method || r.URL.Path != want.path {
			t.Errorf("request[%d]=%s %s, want %s %s", requestIndex, r.Method, r.URL.Path, want.method, want.path)
		}
		if r.Header.Get("Origin") != server.URL || r.Header.Get("Referer") != server.URL+"/" {
			t.Errorf("request[%d] origin=%q referer=%q", requestIndex, r.Header.Get("Origin"), r.Header.Get("Referer"))
		}
		requestIndex++
		if r.Method == http.MethodPost && r.URL.Path != "/web-api/v1/task" {
			_, _ = w.Write([]byte(`{"id":9}`))
		}
	}))
	defer server.Close()
	client := New(server.URL, "session")
	ctx := context.Background()
	operations := []func() error{
		func() error { return client.UpdateFeed(ctx, "1", "2", "x") },
		func() error { return client.DeleteFeed(ctx, "1", "2") },
		func() error { return client.SetFeedLike(ctx, "1", "2", true) },
		func() error { return client.SetFeedLike(ctx, "1", "2", false) },
		func() error { _, err := client.CreateComment(ctx, "1", "2", "x"); return err },
		func() error { return client.UpdateComment(ctx, "1", "2", "3", "x") },
		func() error { return client.DeleteComment(ctx, "1", "2", "3") },
		func() error { return client.SetCommentLike(ctx, "1", "2", "3", true) },
		func() error { return client.SetCommentLike(ctx, "1", "2", "3", false) },
		func() error { _, err := client.CreateMessage(ctx, "4", "x"); return err },
		func() error { return client.UpdateMessage(ctx, "4", "5", "x") },
		func() error { return client.DeleteMessage(ctx, "4", "5") },
		func() error { return client.CreateTask(ctx, "x", []ID{"7"}, nil) },
		func() error { return client.CompleteTask(ctx, "6") },
		func() error { return client.RevertTask(ctx, "6") },
		func() error { return client.DeleteTask(ctx, "6") },
	}
	for i, operation := range operations {
		if err := operation(); err != nil {
			t.Fatalf("operation[%d]: %v", i, err)
		}
	}
}
