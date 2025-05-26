package utils

import (
    "crypto/sha256"
    "fmt"
    "io"
    "os"
)

func CalculateSHA256(path string) (string, error) {
    f, err := os.Open(path)
    if err != nil {
        return "", err
    }
    defer f.Close()

    h := sha256.New()
    if _, err := io.Copy(h, f); err != nil {
        return "", err
    }

    return fmt.Sprintf("%x", h.Sum(nil)), nil
}

