#!/bin/bash
# ============================================================
# Ollama Model Pull Script
# Ollama container başladıktan sonra çalıştırılır.
# Kullanım: docker exec -it masterfabric-ollama bash /scripts/pull-model.sh
# ============================================================

set -e

MODEL="${LLM_MODEL:-gemma2:2b}"

echo "⏳ Ollama servisinin hazır olması bekleniyor..."
until curl -sf http://localhost:11434/ > /dev/null 2>&1; do
  echo "  ... bekleniyor"
  sleep 2
done
echo "✅ Ollama servisi hazır!"

echo "📦 Model indiriliyor: $MODEL"
ollama pull "$MODEL"
echo "✅ Model hazır: $MODEL"

echo ""
echo "🧪 Test komutu:"
echo "  curl http://localhost:11434/v1/chat/completions \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"model\":\"$MODEL\",\"messages\":[{\"role\":\"user\",\"content\":\"Merhaba\"}]}'"
