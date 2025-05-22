package utils

import (
    "archive/zip"
    "io"
    "os"
    "path/filepath"
    "strings"
)

// ZipFolder comprime el directorio source y lo guarda como archivo ZIP en target.
func ZipFolder(sourceDir, targetZip string) error {
    zipfile, err := os.Create(targetZip)
    if err != nil {
        return err
    }
    defer zipfile.Close()

    writer := zip.NewWriter(zipfile)
    defer writer.Close()

    baseDir := filepath.Dir(sourceDir)

    return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        // Nombre relativo del archivo dentro del ZIP
        relPath, err := filepath.Rel(baseDir, path)
        if err != nil {
            return err
        }

        if info.IsDir() {
            if relPath == "." {
                return nil
            }
            relPath += "/"
        }

        header, err := zip.FileInfoHeader(info)
        if err != nil {
            return err
        }
        header.Name = filepath.ToSlash(relPath)

        if !info.IsDir() {
            header.Method = zip.Deflate
        }

        writerEntry, err := writer.CreateHeader(header)
        if err != nil {
            return err
        }

        if !info.IsDir() {
            file, err := os.Open(path)
            if err != nil {
                return err
            }
            defer file.Close()
            _, err = io.Copy(writerEntry, file)
            return err
        }

        return nil
    })
}

