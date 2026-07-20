<div align="center">

<a href="https://academy.masterfabric.co">
  <img src="https://academy.masterfabric.co/academy-badge.png" width="120" alt="MasterFabric Academy">
</a>

<p>
  <sub>
    academy.masterfabric.co is a
    <a href="https://masterfabric.co">MasterFabric</a>
    subsidiary.
  </sub>
</p>

</div>

# Akıllı Duygu ve Odak Günlüğü - Backend API

Bu repository, **Akıllı Duygu ve Odak Günlüğü (Smart Emotion & Focus Journal)** projesinin Go ile geliştirilmiş REST API iskeletini içermektedir. Proje, MasterFabric Academy staj şartnamesine uygun şekilde Clean Architecture prensipleriyle yapılandırılmıştır.

**Canlı API Adresi**: [https://smart-emotion-focus-journal-backend.onrender.com](https://smart-emotion-focus-journal-backend.onrender.com)

## Kullanılan Teknolojiler
- **Programlama Dili**: Go (Golang)
- **Web Framework**: Gin Gonic
- **Veritabanı ORM**: GORM
- **Veritabanı Sürücüsü**: PostgreSQL Driver (pgx)

---

## Proje Klasör Yapısı (Clean Architecture)
Proje, bağımlılıkların yönetilebilirliği ve test edilebilirlik için katmanlı mimariye uygun tasarlanmıştır:
- `config/`: Veritabanı ve genel uygulama konfigürasyonu.
- `models/`: Veritabanı tablolarına karşılık gelen GORM struct tanımları.
- `controllers/`: İstekleri karşılayan ve yanıtları dönen business logic / mock handler'lar.
- `routes/`: API rotalarının tanımlandığı ve controller fonksiyonlarıyla eşleştirildiği katman.
- `main.go`: Uygulamanın giriş noktası, middleware yapılandırmaları ve HTTP sunucusunun başlatıldığı yer.

---

## Veritabanı Modelleri
Projede 4 ana tablo bulunmaktadır:
1. **User**: Kullanıcı hesabı bilgileri (E-posta, şifre hash'i).
2. **UserConfig**: Temalar ve bildirim tercihleri gibi kullanıcı ayarları.
3. **Journal**: Kullanıcı günlük yazıları ve LLM tarafından analiz edilen karar verimliliği / duygu skorları (`DecisionScore`).
4. **LlmMetric**: LLM API çağrılarının izleme verileri (gecikme süreleri, token sayıları ve hata kayıtları).

---

## Tanımlı Rotalar (20 Endpoints)

API, toplam 20 adet endpoint barındırmaktadır:

### 1. Auth Rotaları (8)
- `POST /register` - Kullanıcı kaydı
- `POST /login` - Kullanıcı girişi
- `POST /logout` - Güvenli çıkış
- `POST /refresh` - JWT yenileme token talebi
- `GET /profile` - Profil bilgilerini getirme
- `PUT /profile` - Profil bilgilerini güncelleme
- `PUT /password` - Şifre değiştirme
- `DELETE /delete` - Hesap silme

### 2. Config Rotaları (2)
- `GET /config` - Kullanıcı uygulama ayarlarını getirme
- `PUT /config` - Kullanıcı uygulama ayarlarını güncelleme

### 3. Web MLC-LLM / Analiz Rotaları (7)
- `POST /api/journal` - Yeni günlük girdisi ekleme (ve LLM analizi)
- `GET /api/journal` - Günlük geçmişini listeleme
- `POST /api/monitor/metrics` - LLM performans metriklerini kaydetme
- `GET /api/monitor/metrics` - Kayıtlı metrikleri listeleme
- `GET /api/monitor/scores` - Duygu/Karar skor istatistiklerini getirme
- `POST /api/monitor/error` - LLM hata logu ekleme
- `DELETE /api/monitor/clear` - LLM metriklerini temizleme

### 4. Ortak Rotaları (3)
- `GET /health` - Sağlık kontrolü (Health Check)
- `GET /version` - Uygulama versiyon bilgisi
- `POST /feedback` - Kullanıcı geri bildirimi gönderme

---

## Kurulum ve Çalıştırma

### Bağımlılıkları Yükleme
```bash
go mod download
```

### Sunucuyu Başlatma
```bash
go run main.go
```
Sunucu varsayılan olarak `http://localhost:8080` adresinde ayağa kalkacaktır. CORS yapılandırması Next.js veya diğer frontend uygulamalarının erişebileceği şekilde entegre edilmiştir.
