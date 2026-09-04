package talknote

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type ID string

func (id *ID) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		var value string
		if err := json.Unmarshal(data, &value); err != nil {
			return err
		}
		*id = ID(value)
		return nil
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value json.Number
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("decode Talknote ID: %w", err)
	}
	*id = ID(value.String())
	return nil
}

type Network struct {
	ID          ID     `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	CompanyName string `json:"companyName"`
}

type User struct {
	ID              ID       `json:"id"`
	NetworkID       ID       `json:"networkId"`
	Network         *Network `json:"network,omitempty"`
	FirstName       string   `json:"firstName"`
	LastName        string   `json:"lastName"`
	Unit            string   `json:"unit,omitempty"`
	Position        string   `json:"position,omitempty"`
	IconVersionID   ID       `json:"iconVersionId,omitempty"`
	Disabled        bool     `json:"disabled"`
	OrganizationIDs []ID     `json:"organizationIds,omitempty"`
}

func (u User) Name() string { return u.FirstName + " " + u.LastName }

type Group struct {
	Data     GroupData     `json:"data"`
	Reaction GroupReaction `json:"reaction"`
	Order    GroupOrder    `json:"order"`
	Session  GroupSession  `json:"session"`
}

type GroupData struct {
	ID           ID     `json:"id"`
	NetworkID    ID     `json:"networkId"`
	Name         string `json:"name"`
	Explain      string `json:"explain"`
	MemberCount  int    `json:"memberCount"`
	OpenStatus   string `json:"openStatus"`
	Everyone     bool   `json:"everyone"`
	LastPostedAt int64  `json:"lastPostedAt"`
}

type GroupReaction struct {
	YouUnreadCount int `json:"youUnreadCount"`
}

type GroupOrder struct {
	YouBelong    bool  `json:"youBelong"`
	YouHold      bool  `json:"youHold"`
	Notification bool  `json:"notification"`
	LastPostedAt int64 `json:"lastPostedAt"`
}

type GroupSession struct {
	YouOwner bool `json:"youOwner"`
}

type Feed struct {
	Data     FeedData     `json:"data"`
	Reaction FeedReaction `json:"reaction"`
}

type FeedData struct {
	ID            ID     `json:"id"`
	NetworkID     ID     `json:"networkId"`
	GroupID       ID     `json:"groupId"`
	PostedUser    User   `json:"postedUser"`
	Content       string `json:"content"`
	ActivityType  string `json:"activityType"`
	CommentCount  int    `json:"commentCount"`
	LikeCount     int    `json:"likeCount"`
	ReadUserCount int    `json:"readUserCount"`
	Archived      bool   `json:"archived"`
	Pinned        bool   `json:"pinned"`
	Edited        bool   `json:"edited"`
	CreatedAt     int64  `json:"createdAt"`
	UpdatedAt     int64  `json:"updatedAt"`
}

type FeedReaction struct {
	YouLiked    bool `json:"youLiked"`
	YouChecked  bool `json:"youChecked"`
	YouFollowed bool `json:"youFollowed"`
}

type Comment struct {
	Data     CommentData     `json:"data"`
	Reaction CommentReaction `json:"reaction"`
}

type CommentData struct {
	ID           ID     `json:"id"`
	NetworkID    ID     `json:"networkId"`
	GroupID      ID     `json:"groupId"`
	FeedID       ID     `json:"feedId"`
	PostedUser   User   `json:"postedUser"`
	Content      string `json:"content"`
	ActivityType string `json:"activityType"`
	LikeCount    int    `json:"likeCount"`
	Edited       bool   `json:"edited"`
	CreatedAt    int64  `json:"createdAt"`
	UpdatedAt    int64  `json:"updatedAt"`
}

type CommentReaction struct {
	YouLiked bool `json:"youLiked"`
}

type Thread struct {
	Data     ThreadData     `json:"data"`
	Reaction ThreadReaction `json:"reaction"`
	Order    ThreadOrder    `json:"order"`
	Session  ThreadSession  `json:"session"`
}

type ThreadData struct {
	ID              ID       `json:"id"`
	Title           string   `json:"title,omitempty"`
	LastMessage     *Message `json:"lastMessage,omitempty"`
	MemberCount     int      `json:"memberCount"`
	Member10        []User   `json:"member10"`
	LastPostedAt    int64    `json:"lastPostedAt"`
	IncludeExternal bool     `json:"includeExternal"`
}

type ThreadReaction struct {
	YouUnreadCount int  `json:"youUnreadCount"`
	YouHidden      bool `json:"youHidden"`
}

type ThreadOrder struct {
	LastPostedAt int64 `json:"lastPostedAt"`
}

type ThreadSession struct {
	DisplayTitle  string `json:"displayTitle"`
	IsSelf        bool   `json:"isSelf"`
	ConnectStatus string `json:"connectStatus,omitempty"`
}

type Message struct {
	Data     MessageData     `json:"data"`
	Reaction MessageReaction `json:"reaction"`
}

type MessageData struct {
	ID            ID     `json:"id"`
	NetworkID     ID     `json:"networkId"`
	ThreadID      ID     `json:"threadId"`
	PostedUser    User   `json:"postedUser"`
	Content       string `json:"content"`
	ActivityType  string `json:"activityType"`
	ReadUserCount int    `json:"readUserCount"`
	Edited        bool   `json:"edited"`
	CreatedAt     int64  `json:"createdAt"`
	UpdatedAt     int64  `json:"updatedAt"`
}

type MessageReaction struct {
	YouChecked bool `json:"youChecked"`
	YouDeleted bool `json:"youDeleted"`
}

type Task struct {
	ID              ID     `json:"id"`
	Source          Source `json:"source"`
	Owner           User   `json:"owner"`
	Content         string `json:"content"`
	Deadline        *int64 `json:"deadline,omitempty"`
	AssignCount     int    `json:"assignCount"`
	CompleteCount   int    `json:"completeCount"`
	DisplayAssignee User   `json:"displayAssignee"`
	Remind          bool   `json:"remind"`
	Status          string `json:"status"`
	CreatedAt       int64  `json:"createdAt"`
	UpdatedAt       int64  `json:"updatedAt"`
}

type Source struct {
	SourceType string `json:"sourceType"`
	SourceID   ID     `json:"sourceId"`
}

type SearchResult struct {
	GroupFeed     *Feed    `json:"groupFeed,omitempty"`
	DirectMessage *Message `json:"directMessage,omitempty"`
}

type contentResponse[T any] struct {
	Content []T `json:"content"`
}

type postRequest struct {
	Content       string `json:"content"`
	AttachmentIDs []ID   `json:"attachmentIds"`
}

type updatePostRequest struct {
	Content string `json:"content"`
}

type taskCreateRequest struct {
	Content     string `json:"content"`
	AssigneeIDs []ID   `json:"assigneeIds"`
	Deadline    *int64 `json:"deadline,omitempty"`
}
