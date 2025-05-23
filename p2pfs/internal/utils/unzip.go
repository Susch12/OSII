package utils

import (
    "archive/zip"
    "io"
    "os"
    "path/filepath"
    "strings"
    "fmt"
)

// UnzipFile descomprime un archivo zip a la carpeta destino.
func UnzipFile(zipPath, destDir string) error {
    r, err := zip.OpenReader(zipPath)
    if err != nil {
        return err
    }
    defer r.Close()

    for _, f := range r.File {
        fpath := filepath.Join(destDir, f.Name)

        // Validación de seguridad
        if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
            return fmt.Errorf("archivo fuera del destino: %s", fpath)
        }

        if f.FileInfo().IsDir() {
            os.MkdirAll(fpath, os.ModePerm)
            continue
        }

        if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
            return err
        }

        outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
        if err != nil {
            return err
        }

        rc, err := f.Open()
        if err != nil {
            return err
        }

        _, err = io.Copy(outFile, rc)

        outFile.Close()
        rc.Close()

        if err != nil {
            return err
        }
    }

    return nil
}

