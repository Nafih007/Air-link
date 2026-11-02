package server

// ResolvedFile is a user-selected file that is safe to serve.
type ResolvedFile struct {
	AbsPath string // cleaned absolute path
	Name    string // base name for Content-Disposition
	Size    int64
}
