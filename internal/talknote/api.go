package talknote

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func (c *Client) CurrentNetwork(ctx context.Context) (*Network, error) {
	var network Network
	if err := c.get(ctx, "/web-api/v1/network/current", nil, &network); err != nil {
		return nil, err
	}
	return &network, nil
}

func (c *Client) Me(ctx context.Context) (*User, error) {
	var user User
	if err := c.get(ctx, "/web-api/v1/network/user/me", nil, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *Client) ListGroups(ctx context.Context) ([]Group, error) {
	var response contentResponse[Group]
	if err := c.get(ctx, "/web-api/v1/group", nil, &response); err != nil {
		return nil, err
	}
	return nonNil(response.Content), nil
}

func (c *Client) ListFeeds(ctx context.Context, groupID string, limit int) ([]Feed, error) {
	query := limitQuery(limit)
	var response contentResponse[Feed]
	if err := c.get(ctx, "/web-api/v1/group/"+url.PathEscape(groupID)+"/feed", query, &response); err != nil {
		if isRecordNotFound(err) {
			return []Feed{}, nil
		}
		return nil, err
	}
	return nonNil(response.Content), nil
}

func (c *Client) GetFeed(ctx context.Context, groupID, feedID string) (*Feed, error) {
	var feed Feed
	if err := c.get(ctx, feedPath(groupID, feedID), nil, &feed); err != nil {
		return nil, err
	}
	return &feed, nil
}

func (c *Client) CreateFeed(ctx context.Context, groupID, content string) (*Feed, error) {
	var data FeedData
	err := c.post(ctx, "/web-api/v1/group/"+url.PathEscape(groupID)+"/feed", postRequest{Content: content, AttachmentIDs: []ID{}}, &data)
	if err == nil && data.ID == "" {
		return nil, fmt.Errorf("talknote create feed response has no ID")
	}
	return &Feed{Data: data}, err
}

func (c *Client) UpdateFeed(ctx context.Context, groupID, feedID, content string) error {
	return c.put(ctx, feedPath(groupID, feedID), updatePostRequest{Content: content}, nil)
}

func (c *Client) DeleteFeed(ctx context.Context, groupID, feedID string) error {
	return c.delete(ctx, feedPath(groupID, feedID))
}

func (c *Client) SetFeedLike(ctx context.Context, groupID, feedID string, liked bool) error {
	path := feedPath(groupID, feedID) + "/like"
	if liked {
		return c.put(ctx, path, nil, nil)
	}
	return c.delete(ctx, path)
}

func (c *Client) ListComments(ctx context.Context, groupID, feedID string, limit int) ([]Comment, error) {
	var response contentResponse[Comment]
	if err := c.get(ctx, feedPath(groupID, feedID)+"/comment", limitQuery(limit), &response); err != nil {
		if isRecordNotFound(err) {
			return []Comment{}, nil
		}
		return nil, err
	}
	return nonNil(response.Content), nil
}

func (c *Client) CreateComment(ctx context.Context, groupID, feedID, content string) (*Comment, error) {
	var data CommentData
	err := c.post(ctx, feedPath(groupID, feedID)+"/comment", postRequest{Content: content, AttachmentIDs: []ID{}}, &data)
	if err == nil && data.ID == "" {
		return nil, fmt.Errorf("talknote create comment response has no ID")
	}
	return &Comment{Data: data}, err
}

func (c *Client) UpdateComment(ctx context.Context, groupID, feedID, commentID, content string) error {
	return c.put(ctx, commentPath(groupID, feedID, commentID), updatePostRequest{Content: content}, nil)
}

func (c *Client) DeleteComment(ctx context.Context, groupID, feedID, commentID string) error {
	return c.delete(ctx, commentPath(groupID, feedID, commentID))
}

func (c *Client) SetCommentLike(ctx context.Context, groupID, feedID, commentID string, liked bool) error {
	path := commentPath(groupID, feedID, commentID) + "/like"
	if liked {
		return c.put(ctx, path, nil, nil)
	}
	return c.delete(ctx, path)
}

func (c *Client) ListThreads(ctx context.Context) ([]Thread, error) {
	var response contentResponse[Thread]
	if err := c.get(ctx, "/web-api/v1/thread", nil, &response); err != nil {
		return nil, err
	}
	return nonNil(response.Content), nil
}

func (c *Client) ListMessages(ctx context.Context, threadID string, limit int) ([]Message, error) {
	var response contentResponse[Message]
	if err := c.get(ctx, "/web-api/v1/thread/"+url.PathEscape(threadID)+"/message", limitQuery(limit), &response); err != nil {
		if isRecordNotFound(err) {
			return []Message{}, nil
		}
		return nil, err
	}
	return nonNil(response.Content), nil
}

func (c *Client) CreateMessage(ctx context.Context, threadID, content string) (*Message, error) {
	var data MessageData
	err := c.post(ctx, "/web-api/v1/thread/"+url.PathEscape(threadID)+"/message", postRequest{Content: content, AttachmentIDs: []ID{}}, &data)
	if err == nil && data.ID == "" {
		return nil, fmt.Errorf("talknote create message response has no ID")
	}
	return &Message{Data: data}, err
}

func (c *Client) UpdateMessage(ctx context.Context, threadID, messageID, content string) error {
	return c.put(ctx, messagePath(threadID, messageID), updatePostRequest{Content: content}, nil)
}

func (c *Client) DeleteMessage(ctx context.Context, threadID, messageID string) error {
	return c.delete(ctx, messagePath(threadID, messageID))
}

func (c *Client) Search(ctx context.Context, keyword string, limit int) ([]SearchResult, error) {
	query := url.Values{"keyword": {keyword}}
	if limit > 0 {
		query.Set("limit", strconv.Itoa(limit))
	}
	var response contentResponse[SearchResult]
	if err := c.get(ctx, "/web-api/v1/search", query, &response); err != nil {
		if isRecordNotFound(err) {
			return []SearchResult{}, nil
		}
		return nil, err
	}
	return nonNil(response.Content), nil
}

func (c *Client) ListTasks(ctx context.Context, status string) ([]Task, error) {
	query := url.Values{"status": {status}}
	var response contentResponse[Task]
	if err := c.get(ctx, "/web-api/v1/task", query, &response); err != nil {
		if isRecordNotFound(err) {
			return []Task{}, nil
		}
		return nil, err
	}
	return nonNil(response.Content), nil
}

func (c *Client) CreateTask(ctx context.Context, content string, assigneeIDs []ID, deadline *int64) error {
	return c.post(ctx, "/web-api/v1/task", taskCreateRequest{Content: content, AssigneeIDs: assigneeIDs, Deadline: deadline}, nil)
}

func (c *Client) CompleteTask(ctx context.Context, taskID string) error {
	return c.put(ctx, "/web-api/v1/task/"+url.PathEscape(taskID)+"/complete", nil, nil)
}

func (c *Client) RevertTask(ctx context.Context, taskID string) error {
	return c.put(ctx, "/web-api/v1/task/"+url.PathEscape(taskID)+"/revert", nil, nil)
}

func (c *Client) DeleteTask(ctx context.Context, taskID string) error {
	return c.delete(ctx, "/web-api/v1/task/"+url.PathEscape(taskID))
}

func feedPath(groupID, feedID string) string {
	return "/web-api/v1/group/" + url.PathEscape(groupID) + "/feed/" + url.PathEscape(feedID)
}

func commentPath(groupID, feedID, commentID string) string {
	return feedPath(groupID, feedID) + "/comment/" + url.PathEscape(commentID)
}

func messagePath(threadID, messageID string) string {
	return "/web-api/v1/thread/" + url.PathEscape(threadID) + "/message/" + url.PathEscape(messageID)
}

func limitQuery(limit int) url.Values {
	if limit <= 0 {
		return nil
	}
	return url.Values{"limit": {strconv.Itoa(limit)}}
}

func isRecordNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound && strings.HasPrefix(apiErr.Message, "Error_RecordNotFound:")
}

func nonNil[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return values
}
