package browser

import "time"

// BrowserCommand represents a command sent from the agent to the browser addon
type BrowserCommand struct {
	ID      string                 `json:"id"`
	Type    string                 `json:"type"` // navigate, click, fill, extract, screenshot, execute, wait_for, etc.
	Params  map[string]interface{} `json:"params"`
	Timeout int                    `json:"timeout"` // milliseconds
}

// BrowserResponse represents a response from the browser addon to the agent
type BrowserResponse struct {
	ID        string                 `json:"id"`
	Status    string                 `json:"status"` // success, error
	Result    interface{}            `json:"result"`
	Error     string                 `json:"error,omitempty"`
	PageState map[string]interface{} `json:"page_state"`
}

// BrowserClient represents a connected browser instance
type BrowserClient struct {
	ID            string
	ConnectedAt   time.Time
	LastHeartbeat time.Time
	UserAgent     string
	PendingCmds   map[string]chan *BrowserResponse // command ID -> response channel
}

// CommandType constants for browser commands
const (
	CommandNavigate      = "navigate"
	CommandClick         = "click"
	CommandFill          = "fill"
	CommandSubmit        = "submit"
	CommandExtract       = "extract"
	CommandScreenshot    = "screenshot"
	CommandExecute       = "execute"
	CommandWaitFor       = "wait_for"
	CommandScroll        = "scroll"
	CommandGetCookies    = "get_cookies"
	CommandSetCookies    = "set_cookies"
	CommandGetPageSource = "get_page_source"
)

// ResponseStatus constants
const (
	StatusSuccess = "success"
	StatusError   = "error"
)
