// Package tools defines all tool implementations and their metadata.
package tools

// Tool is a single capability exposed by dev-assist.
type Tool struct {
	ID          string
	Name        string
	Description string
	Category    string
	Inputs      []InputDef
	Run         func(inputs []string) (string, error)
}

// InputDef describes one input slot for a Tool.
// When Options is non-empty the TUI renders a toggle/select instead of a text area.
type InputDef struct {
	Label       string
	Placeholder string
	Multiline   bool
	Required    bool
	AcceptsFile bool     // user may supply a file path; content is read and substituted
	Options     []string // optional toggle choices (e.g. ["encode","decode"])
	Default     string
	FlagName    string // CLI long flag name, e.g. "cert", "host", "cn"
	FlagShort   string // single-char shorthand, e.g. "c", "H", "n" (leave "" for none)
}

// Registry holds all tools in display order.
var Registry []*Tool

func init() {
	Registry = []*Tool{
		// SSL & Certificates
		SSLDecodeT,
		SSLVerifyT,
		CSRGenT,
		PEMParseT,
		// Auth & Tokens
		JWTDecodeT,
		SAMLDecodeT,
		Base64T,
		URLCodecT,
		// Network
		DNSLookupT,
		WhoisT,
		CIDRCalcT,
		HTTPHeadersT,
		// Data
		DateDiffT,
		JSONYAMLT,
		TimestampT,
	}
}

// ByID returns the tool with the given ID, or nil.
func ByID(id string) *Tool {
	for _, t := range Registry {
		if t.ID == id {
			return t
		}
	}
	return nil
}
