package util

import "encoding/json"

// MustMarshal wraps json.Marshal and panics on error (for tests/seeding)
func MustMarshal(v interface{}) []byte {
    b, err := json.Marshal(v)
    if err != nil {
        panic(err)
    }
    return b
}

// MustUnmarshal wraps json.Unmarshal and panics on error
func MustUnmarshal(data []byte, v interface{}) {
    if err := json.Unmarshal(data, v); err != nil {
        panic(err)
    }
}
