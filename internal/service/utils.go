package service

// zero out a byte slice to prevent it from lingering in memory.
func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
