package peer

import (
	"encoding/json"
	"fmt"
	"os"
)

// SavePeersToFile guarda la lista de peers en un archivo JSON
func SavePeersToFile(peers []PeerInfo, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("no se pudo crear el archivo: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(peers); err != nil {
		return fmt.Errorf("no se pudo escribir el archivo JSON: %w", err)
	}
	return nil
}

// LoadPeersFromFile carga la lista de peers desde un archivo JSON
func LoadPeersFromFile(filename string) ([]PeerInfo, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir el archivo: %w", err)
	}
	defer file.Close()

	var peers []PeerInfo
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&peers); err != nil {
		return nil, fmt.Errorf("no se pudo decodificar el archivo JSON: %w", err)
	}
	return peers, nil
}

