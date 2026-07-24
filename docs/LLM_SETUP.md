# LLM Setup Guide — Ollama + Cloudflare Tunnel

Bu rehber, Ollama LLM'i Docker'da çalıştırıp Cloudflare Tunnel ile Render backend'ine bağlamayı anlatır.

## Mimari

```
Frontend (Vercel) → Backend (Render) → Cloudflare Tunnel → Ollama (Lokal Docker)
```

## Gereksinimler

- **Docker Desktop** kurulu ve çalışıyor
- **En az 4GB boş RAM** (gemma2:2b modeli için)
- İnternet bağlantısı (model indirme + tunnel)

## Hızlı Başlangıç

### 1. Container'ları Başlat

```bash
# Proje kök dizininde:
docker compose -f deployments/docker-compose.yml up -d ollama cloudflared
```

### 2. Cloudflare Tunnel URL'sini Al

```bash
docker logs masterfabric-cloudflared 2>&1 | grep "trycloudflare.com"
```

Çıktıda şuna benzer bir URL göreceksin:
```
https://random-words-here.trycloudflare.com
```

> **Bu URL'yi not et** — Render'daki environment variable'da kullanacaksın.

### 3. Modeli İndir (İlk Sefer)

```bash
docker exec -it masterfabric-ollama ollama pull gemma2:2b
```

> Bu işlem ~1.6GB indirir, internet hızına bağlı olarak 2-10 dakika sürer.
> Model `ollama_data` volume'unda saklanır — container restart'larında tekrar indirmek gerekmez.

### 4. Test Et

```bash
# Lokal test:
curl http://localhost:11434/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"gemma2:2b","messages":[{"role":"user","content":"Merhaba, nasılsın?"}]}'

# Tunnel üzerinden test (URL'yi kendi URL'nle değiştir):
curl https://xxx.trycloudflare.com/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model":"gemma2:2b","messages":[{"role":"user","content":"Merhaba, nasılsın?"}]}'
```

### 5. Render Environment Variables

Render Dashboard'da şu environment variable'ları güncelle:

| Variable | Değer |
|---|---|
| `LLM_API_BASE` | `https://xxx.trycloudflare.com/v1` |
| `LLM_MODEL` | `gemma2:2b` |

## Günlük Kullanım

### Başlatma
```bash
docker compose -f deployments/docker-compose.yml up -d ollama cloudflared
```

### Durdurma
```bash
docker compose -f deployments/docker-compose.yml stop ollama cloudflared
```

### Logları İzleme
```bash
# Ollama logları:
docker logs -f masterfabric-ollama

# Cloudflare Tunnel logları:
docker logs -f masterfabric-cloudflared
```

## Sorun Giderme

### "Connection refused" hatası
Ollama'nın hazır olmasını bekle:
```bash
docker logs masterfabric-ollama
# "Listening on 0.0.0.0:11434" mesajını görene kadar bekle
```

### Tunnel URL değişti
Quick Tunnel modunda URL her restart'ta değişir. Yeni URL'yi al:
```bash
docker logs masterfabric-cloudflared 2>&1 | grep "trycloudflare.com"
```
Sonra Render'daki `LLM_API_BASE`'i güncelle.

### Kalıcı URL istiyorsanız
1. [Cloudflare Zero Trust Dashboard](https://one.dash.cloudflare.com)'a gir
2. Networks → Tunnels → Create a Tunnel
3. Token'ı al ve `docker-compose.yml`'daki cloudflared servisini güncelle:
```yaml
cloudflared:
  command: tunnel --no-autoupdate run --token YOUR_TOKEN_HERE
```

## Model Değiştirme

```bash
# Mevcut modelleri listele:
docker exec -it masterfabric-ollama ollama list

# Yeni model indir:
docker exec -it masterfabric-ollama ollama pull phi4-mini

# Eski modeli sil (disk alanı kazanmak için):
docker exec -it masterfabric-ollama ollama rm gemma2:2b
```

Sonra `LLM_MODEL` environment variable'ını güncelle.
