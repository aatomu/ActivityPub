package main

type ActivityStream struct {
	Context interface{} `json:"@context,omitempty"`
	ID      string      `json:"id,omitempty"`
	Type    string      `json:"type"`
	// Follow,Undo,Accept
	Actor string `json:"actor,omitempty"`
	// Inbox
	Object         interface{} `json:"object,omitempty"`
	objectStr      string
	objectActivity *ActivityStream
	// Outbox
	First  string `json:"first,omitempty"`
	Last   string `json:"last,omitempty"`
	Next   string `json:"next,omitempty"`
	Prev   string `json:"prev,omitempty"`
	PartOf string `json:"partOf,omitempty"`
	// Follower,Following,Outbox
	TotalItems   int           `json:"totalItems,omitempty"`
	OrderedItems []interface{} `json:"orderedItems,omitempty"`
}

type Resource struct {
	Subject string   `json:"subject"`
	Aliases []string `json:"aliases"`
	Links   []struct {
		Rel      string `json:"rel"`
		Type     string `json:"type,omitempty"`
		Href     string `json:"href,omitempty"`
		Template string `json:"template,omitempty"`
	} `json:"links"`
}

type Person struct {
	Context           []string `json:"@context"`
	Type              string   `json:"type"`
	ID                string   `json:"id"`
	Followers         string   `json:"followers"`
	Following         string   `json:"following"`
	URL               string   `json:"url"`
	PreferredUsername string   `json:"preferredUsername"`
	Name              string   `json:"name"`
	Icon              struct {
		MediaType string `json:"mediaType"`
		Type      string `json:"type"`
		URL       string `json:"url"`
	} `json:"icon"`
	Summary string `json:"summary"`
	Inbox   string `json:"inbox"`
	Outbox  string `json:"outbox"`
}

type Note struct {
	Context      string           `json:"@context"`
	ID           string           `json:"id"`
	Type         string           `json:"type"`
	InReplyTo    interface{}      `json:"inReplyTo"`
	Published    string           `json:"published"`
	URL          string           `json:"url"`
	AttributedTo string           `json:"attributedTo"`
	Content      string           `json:"content"`
	To           []string         `json:"to"`
	Sensitive    bool             `json:"sensitive"`
	Attachment   []NoteAttachment `json:"attachment"`
}

type NoteAttachment struct {
	Type      string `json:"type"`
	MediaType string `json:"mediaType"`
	URL       string `json:"url"`
}
