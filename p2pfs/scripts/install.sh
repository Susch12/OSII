#!/bin/bash

echo "🔧 Compilando la aplicación..."

# Compilar el binario principal
go build -o p2pfs-app ./cmd

# Verificar si la compilación fue exitosa
if [ $? -ne 0 ]; then
  echo "❌ Error al compilar. Verifica errores en el código o dependencias."
  exit 1
fi

echo "✅ Compilación completada."
echo "👉 Ejecuta ./p2pfs-app para iniciar el nodo."
