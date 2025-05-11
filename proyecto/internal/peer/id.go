package peer

import (
    "github.com/google/uuid"
    "io/ioutil"
    "os"
    "strings"
)

const idFilePath = "peer.id"

func LoadOrCreatePeerID() (string, error) {
    if _, err := os.Stat(idFilePath); err == nil {
        data, err := ioutil.ReadFile(idFilePath)
        if err != nil {
            return "", err
        }
        return strings.TrimSpace(string(data)), nil
    }

    // Generar nuevo UUID
    id := uuid.New().String()

    err := ioutil.WriteFile(idFilePath, []byte(id), 0644)
    if err != nil {
        return "", err
    }

    return id, nil
}

