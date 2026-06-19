#!/bin/bash

# Warna untuk output terminal
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}======================================================${NC}"
echo -e "${BLUE}    Memulai Setup Private AI Assistant (Local-First)  ${NC}"
echo -e "${BLUE}======================================================${NC}\n"

# 1. Pengecekan Dependensi Utama
echo -e "${GREEN}[1/6] Mengecek dependensi sistem...${NC}"
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker tidak ditemukan. Silakan install Docker terlebih dahulu.${NC}"
    exit 1
fi

if ! command -v go &> /dev/null; then
    echo -e "${RED}Error: Golang tidak ditemukan. Silakan install Golang terlebih dahulu.${NC}"
    exit 1
fi
echo "✅ Docker dan Golang sudah terinstall."

# 2. Pembuatan Struktur Direktori
echo -e "\n${GREEN}[2/6] Membangun struktur direktori...${NC}"
mkdir -p assets/temp
mkdir -p ollama_data
mkdir -p chroma_data
mkdir -p config
mkdir -p cmd/app
mkdir -p internal/delivery internal/usecase internal/repository internal/domain
echo "✅ Direktori berhasil dibuat."

# 3. Membuat File Konfigurasi Default (Docker & Entities)
echo -e "\n${GREEN}[3/6] Mengenerate docker-compose.yml dan config default...${NC}"
docker compose up -d

cat <<EOF > config/custom_entities.yaml
entities:
  - name: "CatatanKalori"
    description: "Digunakan untuk mencatat makanan dan jumlah kalori."
    fields:
      - name: "makanan"
        type: "string"
      - name: "kalori"
        type: "number"
EOF
echo "✅ Konfigurasi berhasil digenerate."

# 4. Menjalankan Infrastruktur Docker
echo -e "\n${GREEN}[4/6] Menjalankan Local AI Engine & Vector DB (Docker)...${NC}"
docker-compose up -d

echo "Menunggu Ollama engine siap (10 detik)..."
sleep 10
echo "✅ Infrastruktur Docker berjalan."

# 5. Mengunduh Model AI (Llama-3, Nomic-Embed, & Whisper.cpp)
echo -e "\n${GREEN}[5/6] Mengunduh Model AI (Ini mungkin memakan waktu tergantung koneksi)...${NC}"
echo "-> Pulling Llama-3 (8B) via Ollama..."
docker exec ollama_engine ollama pull llama3

echo "-> Pulling Nomic Embedding Model via Ollama..."
docker exec ollama_engine ollama pull nomic-embed-text

echo "-> Mengunduh Whisper.cpp base model untuk Voice Note..."
if [ ! -f assets/temp/ggml-base.en.bin ]; then
    # Menggunakan curl untuk mengunduh model C++ lokal
    curl -L -o assets/temp/ggml-base.en.bin https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.en.bin
    echo "✅ Model Whisper berhasil diunduh."
else
    echo "✅ Model Whisper sudah ada, melewati unduhan."
fi

# 6. Setup Modul Golang
echo -e "\n${GREEN}[6/6] Menginisialisasi Modul Golang...${NC}"
if [ ! -f go.mod ]; then
    go mod init github.com/achmichael/pribadi-go
fi
go mod tidy
echo "✅ Modul Golang siap."

echo -e "\n${BLUE}======================================================${NC}"
echo -e "${GREEN}🎉 SETUP SELESAI! 🎉${NC}"
echo -e "${BLUE}======================================================${NC}"
echo -e "Private AI Assistant Anda siap digunakan."
echo -e "Silakan jalankan perintah berikut untuk mengaktifkan asisten:\n"
echo -e "    ${GREEN}go run cmd/app/main.go${NC}\n"
echo -e "Lalu scan QR Code yang muncul menggunakan WhatsApp Anda."
