#!/bin/bash

# ==============================================================================
# SIKOn API - Automated PostgreSQL Backup Script (Docker + Supabase Storage)
# ==============================================================================

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Load environment variables dari .env
if [ -f "$PROJECT_ROOT/.env" ]; then
    export $(grep -v '^#' "$PROJECT_ROOT/.env" | xargs)
fi

# Konfigurasi Docker & DB
DOCKER_CONTAINER_NAME="${DOCKER_CONTAINER_NAME:-omnilibrary-db}"
DB_USER="${DB_USER:-postgres}"
DB_NAME="${DB_NAME:-sikon_db}"
BACKUP_DIR="${BACKUP_DIR:-$PROJECT_ROOT/backups}"
RETENTION_DAYS="${RETENTION_DAYS:-7}"

# Format Nama File Backup
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_FILENAME="${DB_NAME}_${TIMESTAMP}.sql.gz"
BACKUP_FILEPATH="${BACKUP_DIR}/${BACKUP_FILENAME}"

# Buat folder backup jika belum ada
mkdir -p "$BACKUP_DIR"

echo "=================================================="
echo "Starting PostgreSQL Backup via Docker Container [$DOCKER_CONTAINER_NAME]..."
echo "Timestamp: $(date)"
echo "=================================================="

# 1. Eksekusi pg_dump di dalam container Docker
docker exec -t "$DOCKER_CONTAINER_NAME" pg_dump -U "$DB_USER" -d "$DB_NAME" -F c | gzip > "$BACKUP_FILEPATH"

echo "✅ Database dump completed locally: $BACKUP_FILEPATH"

# 2. Upload File Backup ke Supabase Storage
if [ -n "$SUPABASE_URL" ] && [ -n "$SUPABASE_KEY" ] && [ -n "$SUPABASE_BUCKET_BACKUP" ]; then
    echo "Uploading backup to Supabase Storage bucket [$SUPABASE_BUCKET_BACKUP]..."

    UPLOAD_URL="${SUPABASE_URL}/storage/v1/object/${SUPABASE_BUCKET_BACKUP}/${BACKUP_FILENAME}"

    HTTP_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$UPLOAD_URL" \
        -H "Authorization: Bearer ${SUPABASE_KEY}" \
        -H "apiKey: ${SUPABASE_KEY}" \
        -H "Content-Type: application/octet-stream" \
        --data-binary "@${BACKUP_FILEPATH}")

    if [ "$HTTP_RESPONSE" -eq 200 ] || [ "$HTTP_RESPONSE" -eq 201 ]; then
        echo "✅ Upload to Supabase Storage successful! (HTTP $HTTP_RESPONSE)"
    else
        echo "❌ Failed to upload to Supabase Storage! (HTTP $HTTP_RESPONSE)"
    fi

    # 3. Cleanup File Backup Tua di Supabase Storage (> RETENTION_DAYS)
    echo "Checking old backups in Supabase Storage..."
    
    # Hitung timestamp batas waktu (cut-off) dalam hitungan detik
    CUTOFF_DATE=$(date -d "${RETENTION_DAYS} days ago" +%s 2>/dev/null || date -v-${RETENTION_DAYS}d +%s)

    # Ambil daftar file dari Supabase Storage
    LIST_JSON=$(curl -s -X POST "${SUPABASE_URL}/storage/v1/object/list/${SUPABASE_BUCKET_BACKUP}" \
        -H "Authorization: Bearer ${SUPABASE_KEY}" \
        -H "apiKey: ${SUPABASE_KEY}" \
        -H "Content-Type: application/json" \
        -d '{"limit": 100, "sortBy": {"column": "name", "order": "desc"}}')

    # Parse file dan hapus file yang dibuat sebelum CUTOFF_DATE
    echo "$LIST_JSON" | grep -o '"name":"[^"]*"' | cut -d'"' -f4 | while read -r fname; do
        if [ -n "$fname" ]; then
            # Ekstrak tanggal dari nama file (Format: dbname_YYYYMMDD_HHMMSS.sql.gz)
            file_date=$(echo "$fname" | grep -oE '[0-9]{8}_[0-9]{6}' | head -n 1)
            if [ -n "$file_date" ]; then
                f_year=${file_date:0:4}
                f_month=${file_date:4:2}
                f_day=${file_date:6:2}
                f_time=${file_date:9:6}
                
                # Format ke ISO untuk dikonversi ke Timestamp
                file_timestamp=$(date -d "${f_year}-${f_month}-${f_day} ${f_time:0:2}:${f_time:2:2}:${f_time:4:2}" +%s 2>/dev/null || date -j -f "%Y%m%d_%H%M%S" "$file_date" +%s 2>/dev/null)

                if [ -n "$file_timestamp" ] && [ "$file_timestamp" -lt "$CUTOFF_DATE" ]; then
                    echo "🗑️ Deleting old cloud backup: $fname"
                    curl -s -X DELETE "${SUPABASE_URL}/storage/v1/object/${SUPABASE_BUCKET_BACKUP}" \
                        -H "Authorization: Bearer ${SUPABASE_KEY}" \
                        -H "apiKey: ${SUPABASE_KEY}" \
                        -H "Content-Type: application/json" \
                        -d "{\"prefixes\": [\"$fname\"]}" > /dev/null
                fi
            fi
        fi
    done
else
    echo "⚠️ Skipping Supabase sync: Environment variables missing."
fi

# 4. Cleanup File Backup Tua di Lokal
echo "Cleaning up local backups older than $RETENTION_DAYS days..."
find "$BACKUP_DIR" -type f -name "*.sql.gz" -mtime +$RETENTION_DAYS -exec rm -f {} \;

echo "🎉 Backup & Sync process finished successfully!"
echo "=================================================="