@echo off
chcp 65001 >nul
title Ollama + Cloudflare Tunnel Monitor

echo ============================================================
echo   OLLAMA + CLOUDFLARE TUNNEL MONITOR
echo ============================================================
echo.

echo [1/3] Container Durumu:
echo --------------------------------------------------------
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | findstr /i "ollama cloudflared"
echo.

echo [2/3] Tunnel URL:
echo --------------------------------------------------------
docker logs masterfabric-cloudflared 2>&1 | findstr "trycloudflare.com" | findstr /v "Requesting"
echo.

echo [3/3] Ollama Modelleri:
echo --------------------------------------------------------
docker exec masterfabric-ollama ollama list
echo.

echo ============================================================
echo   CANLI LOG IZLEME BASLIYOR (Ctrl+C ile durdur)
echo ============================================================
echo.
echo   Her gelen istek burada gorunecek:
echo.

docker logs -f --tail 0 masterfabric-ollama 2>&1
