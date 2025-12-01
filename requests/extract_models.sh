#!/usr/bin/env bash
# Script para extrair valores de model_preference dos arquivos de requisição

echo "=== Modelos encontrados (model_preference) ==="
echo ""

# Extrair model_preference de todos os arquivos .md com nome do arquivo
grep -hoP '"model_preference"\s*:\s*"\K[^"]+' *.md | while read -r model; do
    echo "$model"
done | sort -u | while read -r model; do
    files=$(grep -l "\"model_preference\":\"$model\"" *.md 2>/dev/null || grep -l "\"model_preference\": \"$model\"" *.md 2>/dev/null)
    for f in $files; do
        echo "$f: $model"
    done
done

echo ""
echo "=== Lista única de modelos ==="
grep -hoP '"model_preference"\s*:\s*"\K[^"]+' *.md | sort -u

echo ""
echo "=== Contagem por modelo ==="
grep -hoP '"model_preference"\s*:\s*"\K[^"]+' *.md | sort | uniq -c | sort -rn
