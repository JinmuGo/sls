package util

import "os"

func Must(err error) {
    if err != nil {
        panic(err)
    }
}

func EnvOr(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}
