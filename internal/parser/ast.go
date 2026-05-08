package parser

// BruDoc is a parsed `.bru` document — an ordered list of blocks.
type BruDoc struct {
	Blocks []Block
}

// Block represents one `name { ... }` or `name:subtype { ... }` section.
type Block struct {
	// Name is the part before the colon (e.g. "auth", "body", "script").
	Name string
	// Subtype is the part after the colon (e.g. "awsv4", "json", "pre-request").
	// Empty when the block has no colon (e.g. "meta", "headers").
	Subtype string
	// Lines holds key/value pairs for kv-style blocks.
	// Nil for raw blocks (see Raw).
	Lines []Line
	// Raw holds the verbatim contents (between the braces, trimmed of the
	// leading and trailing newlines) for body/script/tests blocks.
	Raw string
	// IsRaw is true when this block was parsed in raw mode (Lines is nil
	// and Raw holds the contents).
	IsRaw bool
}

// Line is one key/value entry inside a kv block.
type Line struct {
	// Disabled is true when the source had a leading "~".
	Disabled bool
	// Key is the trimmed key.
	Key string
	// Value is the trimmed value (everything after the first ":").
	Value string
}

// FindBlock returns the first block that matches the given name and subtype.
// Pass an empty subtype to match any subtype for the given name.
func (d *BruDoc) FindBlock(name, subtype string) *Block {
	for i := range d.Blocks {
		b := &d.Blocks[i]
		if b.Name != name {
			continue
		}
		if subtype != "" && b.Subtype != subtype {
			continue
		}
		return b
	}
	return nil
}

// FindAll returns every block whose name matches.
func (d *BruDoc) FindAll(name string) []*Block {
	var out []*Block
	for i := range d.Blocks {
		if d.Blocks[i].Name == name {
			out = append(out, &d.Blocks[i])
		}
	}
	return out
}

// Get returns the value of a key inside a kv block; ok=false if missing.
// Disabled lines are ignored.
func (b *Block) Get(key string) (string, bool) {
	if b == nil {
		return "", false
	}
	for _, l := range b.Lines {
		if l.Disabled {
			continue
		}
		if l.Key == key {
			return l.Value, true
		}
	}
	return "", false
}
