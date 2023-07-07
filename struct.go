package main

type ActivityStream struct {
	Context        interface{} `json:"@context,omitempty"`
	ID             string      `json:"id,omitempty"`
	Type           string      `json:"type,omitempty"`
	Actor          string      `json:"actor,omitempty"`
	Object         interface{} `json:"object,omitempty"`
	objectStr      string
	objectActivity ActivityStreamObject
}

type ActivityStreamObject struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Actor  string `json:"actor"`
	Object string `json:"object"`
}
type ActivityStreamOrderedCollection struct {
	Context      []string `json:"@context"`
	Type         string   `json:"type"`
	ID           string   `json:"id"`
	TotalItems   int      `json:"totalItems"`
	OrderedItems []string `json:"orderedItems"`
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
