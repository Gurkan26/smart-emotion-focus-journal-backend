# Cloudflare Tunnel (cloudflared) Setup Guide for Local GPU Docker Engine

This guide explains how to expose your local MLC-LLM / Ollama Docker container (running on GPU/CPU) over a secure Cloudflare HTTPS Tunnel to your Go Backend running on Render.

---

## 🛠️ Option 1: Docker Compose with Cloudflare Tunnel (Recommended)

### 1. Set your Cloudflare Tunnel Token
Obtain a tunnel token from the [Cloudflare Zero Trust Dashboard](https://one.dash.cloudflare.com/) (Networks -> Tunnels -> Create a Tunnel).

Set the environment variable or create a `.env` file:
```bash
export TUNNEL_TOKEN="eyJhY2NvdW50SUQiOi..."
```

### 2. Run Docker Compose with Tunnel
Launch Ollama / MLC-LLM alongside Cloudflare Tunnel:
```bash
docker-compose -f deployments/docker-compose.tunnel.yml up -d
```

Your local Docker LLM endpoint (port `11434` or `8000`) will be automatically proxied to your designated Cloudflare hostname (e.g., `https://llm.yourdomain.com`).

---

## ⚡ Option 2: Standalone Cloudflared CLI Quick Tunnel (No Account Required)

If you want a quick temporary tunnel without a domain setup:

1. Install Cloudflared:
   ```bash
   # Windows (via winget)
   winget install Cloudflare.cloudflared

   # macOS
   brew install cloudflared
   ```

2. Run Quick Tunnel pointing to local Docker port:
   ```bash
   cloudflared tunnel --url http://localhost:11434
   ```

3. Copy the generated `https://xxxx.trycloudflare.com` URL.

4. Update your Go Backend environment variable on Render:
   ```env
   LLM_API_BASE=https://xxxx.trycloudflare.com/v1
   ```

---

## 🔒 Security & CORS

- Ensure your Cloudflare Tunnel hostname allows HTTP header authorization from your Go Backend.
- Go Backend routes all LLM queries from Vercel Frontend through `LLM_API_BASE`.
