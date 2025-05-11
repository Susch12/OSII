package logfs

import (
    "bufio"
    "encoding/json"
    "fmt"
    "os"
)

type Operation struct {
    Type      string `json:"type"`      // e.g., "DELETE", "TRANSFER"
    FileName  string `json:"filename"`  // nombre del archivo o ruta
    From      string `json:"from"`      // ID del peer que generó la operación
    Timestamp int64  `json:"timestamp"` // Unix timestamp
}

const logFilePath = "operations.log"

func AppendOperation(op Operation) error {
    f, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return fmt.Errorf("no se pudo abrir log: %v", err)
    }
    defer f.Close()

    data, err := json.Marshal(op)
    if err != nil {
        return fmt.Errorf("no se pudo serializar operación: %v", err)
    }

    _, err = f.Write(append(data, '\n'))
    if err != nil {
        return fmt.Errorf("error al escribir en log: %v", err)
    }

    return nil
}

func LoadOperationsSince(since int64) ([]Operation, error) {
    var ops []Operation

    f, err := os.Open(logFilePath)
    if err != nil {
        if os.IsNotExist(err) {
            return ops, nil // log aún no existe, sin error
        }
        return nil, fmt.Errorf("no se pudo leer el log: %v", err)
    }
    defer f.Close()

    scanner := bufio.NewScanner(f)
    for scanner.Scan() {
        var op Operation
        err := json.Unmarshal(scanner.Bytes(), &op)
        if err != nil {
            continue // ignora líneas corruptas
        }
        if op.Timestamp >= since {
            ops = append(ops, op)
        }
    }

    return ops, nil
}

func ReplayLog(since int64) ([]Operation, error) {
    return LoadOperationsSince(since) // reutiliza tu función
}

