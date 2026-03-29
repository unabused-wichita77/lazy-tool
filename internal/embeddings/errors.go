package embeddings

func shortMsg(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
