package hashutil

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
)

func SHA1String(s string) string {
	sum := sha1.Sum([]byte(s))
	return hex.EncodeToString(sum[:])
}

func SHA1Reader(r io.Reader) (string, error) {
	h := sha1.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func ShortID(s string, n int) string {
	if n <= 0 {
		n = 8
	}
	hexed := SHA1String(s)
	if n > len(hexed) {
		n = len(hexed)
	}
	return hexed[:n]
}
