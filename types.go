package itchio

// User is an itch.io account.
type User struct {
	Username string `json:"username"`
}

// Game is one search hit.
type Game struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	// ShortText is the one-line summary the game's page opens with.
	ShortText string `json:"short_text"`
	URL       string `json:"url"`
	CoverURL  string `json:"cover_url"`
	// PublishedAt is the API's own timestamp, unparsed.
	PublishedAt string `json:"published_at"`
	// User is the account that published the game.
	User User `json:"user"`
}

// Upload is one file attached to a game.
type Upload struct {
	ID       int    `json:"id"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	// Type is the upload's kind; a soundtrack is not a mod.
	Type string `json:"type"`
	// Demo marks a demo build, which is not a mod either.
	Demo bool `json:"demo"`
	// UpdatedAt is the API's own timestamp, unparsed.
	UpdatedAt string `json:"updated_at"`
}
