# 🐺 Kuruyo

Go ile geliştirilmiş, yüksek performanslı **reverse proxy** ve **multi-domain routing** sistemi.


## Hızlı Kurulum
```bash
apt update
apt install curl unzip
curl -L -o kuruyo.zip https://github.com/topluyo/kuruyo/archive/refs/heads/master.zip
unzip kuruyo.zip
mv kuruyo-main/ /web
rm kuruyo.zip
```

---

## � Dil Seçimi / Language Selection

| 🇹🇷 [Türkçe](#-türkçe-dokümantasyon) | 🇬🇧 [English](#-english-documentation) |
|:-----------------------------------:|:---------------------------------------:|

---

### 🇹🇷 Türkçe İçindekiler
- [📋 Genel Bakış](#-genel-bakış)
- [🏗️ Proje Yapısı](#️-proje-yapısı)
- [⚙️ Konfigürasyon](#️-konfigürasyon)
- [🔐 Güvenlik Seviyeleri](#-güvenlik-seviyeleri-levels)
- [📍 Port Tanımlama](#-port-tanımlama)
- [🔀 Path Routing](#-path-routing)
- [🛠️ CLI Komutları](#️-cli-komutları-kuruyo)
- [🚀 Kurulum](#-kurulum)
- [🌐 Cloudflare Entegrasyonu](#-cloudflare-entegrasyonu)
- [⚡ Performans Özellikleri](#-performans-özellikleri)

---

## �🇹🇷 Türkçe Dokümantasyon

### 📋 Genel Bakış

Kuruyo, birden fazla web servisini tek bir sunucu üzerinden yönetmek için tasarlanmış güçlü bir ağ geçidi (gateway) sistemidir. Farklı türde backend'leri destekler:

| Sunucu Türü | Açıklama |
|-------------|----------|
| `go:` | Go tabanlı backend uygulamaları |
| `php:` | PHP-FPM ile çalışan PHP uygulamaları |
| `cdn:` | Statik dosya sunucu (maksimum cache, immutable dosyalar) |
| `static:` | RAM tabanlı statik sunucu (gzip sıkıştırma, ETag desteği) |
| `basic:` | Basit HTTP dosya sunucu |

### 🏗️ Proje Yapısı

```
kuruyo/
├── LICENSE                 # MIT Lisansı
├── config/                 # Sistem konfigürasyonları
│   └── system/
└── server/
    ├── engine/            # 🔥 Ana reverse proxy motoru (v2)
    │   └── main.go        # Rate limiting, IP filtering, load balancing
    ├── router/            # 📡 Alternatif router motoru
    │   └── main.go
    ├── programs/          # Backend program türleri
    │   ├── basic/         # Basit HTTP sunucu
    │   ├── cdn/           # CDN sunucu (max-age cache)
    │   ├── static/        # RAM tabanlı statik sunucu
    │   └── builder/       # Build programı
    ├── kuruyo/            # 🛠️ CLI yönetim aracı
    │   └── main.php       # Servis yönetimi komutları
    ├── file/              # PHP yardımcı dosyaları
    ├── main/              # Ana yönetim paneli
    ├── terminal/          # Terminal modülü
    ├── install.sh         # Kurulum scripti
    └── start.sh           # Başlatma scripti
```

### ⚙️ Konfigürasyon

Konfigürasyon dosyası JSON formatındadır ve `/web/config/` altında bulunur:

```json
{
  "http": 80,
  "https": 443,
  "log": "/web/.logs/topluyo.log",
  "routes": {
    "auth.topluyo.com": {
      "description": "Topluyo PHP Auth Sunucusu",
      "ports": "20160+1",
      "serve": "php:/web/sites/auth.topluyo.com",
      "levels": ["basic"]
    },
    "cdn.topluyo.com": {
      "description": "Topluyo CDN Sunucusu",
      "ports": "20500",
      "serve": "cdn:/web/sites/cdn.topluyo.com"
    },
    "topluyo.com": {
      "description": "Topluyo GO Sunucusu",
      "ports": "30100",
      "serve": "go:/web/sites/relase.topluyo.com/backend",
      "levels": ["basic"]
    },
    "topluyo.com/!api": {
      "description": "Topluyo API Sunucusu",
      "ports": "30300",
      "serve": "go:/web/sites/api.topluyo.com/Build",
      "levels": ["soft"]
    }
  },
  "levels": {
    "basic": {
      "rates": ["60r 20s 180w", "20r 5s 30w"]
    },
    "hard": {
      "rates": ["5r 10s 20w"]
    },
    "password": {
      "token": "xxxx"
    },
    "cloudflare": {
      "ips": ["173.245.48.0/20", "..."]
    }
  }
}
```

### 🔐 Güvenlik Seviyeleri (Levels)

| Seviye | Açıklama |
|--------|----------|
| `basic` | Temel rate limiting: `60r 20s 180w` (60 istek/20 saniye, 180 bekleme) |
| `hard` | Sıkı rate limiting: `5r 10s 20w` |
| `soft` | Yumuşak rate limiting: `20r 10s 20w` |
| `password` | Token tabanlı kimlik doğrulama |
| `cloudflare` | Cloudflare IP aralıkları ile kısıtlama |
| `ev` / `pursaklar` | Özel IP aralıkları tanımlama |

**Rate Format:** `Xr Ys Zw` = X istek, Y saniyede, Z saniye bekleme süresi

### 📍 Port Tanımlama

```
"ports": "20100"       → Tek port: 20100
"ports": "20100+4"     → Çoklu port: 20100, 20101, 20102, 20103, 20104
"ports": "20100-20104" → Aralık: 20100'den 20104'e
```

### 🔀 Path Routing

Path tabanlı routing için `!` prefiksi kullanılır:

```json
"topluyo.com/!api": { ... }       → /api yolu
"topluyo.com/!build": { ... }     → /build yolu
"topluyo.com/!loadbalancer/%%": { ... }  → Dinamik load balancer
```

### 🛠️ CLI Komutları (kuruyo)

```bash
# Servis Yönetimi
kuruyo install ~system    # Konfigürasyonu kur
kuruyo update ~system     # Servisleri güncelle
kuruyo remove ~system     # Servisleri kaldır
kuruyo start ~system      # Servisleri başlat
kuruyo stop ~system       # Servisleri durdur
kuruyo restart ~system    # Servisleri yeniden başlat

# İzleme
kuruyo status ~system     # Durum görüntüle
kuruyo log ~system        # Logları izle
kuruyo info               # Servis bilgileri

# Port Yönetimi
kuruyo kill [PORT]        # Portu durdur
kuruyo using [PORT]       # Port kullanımını kontrol et
kuruyo pid [PORT]         # Port PID'ini öğren
kuruyo up [FOLDER] [PORT] # Belirli portu başlat
```

### 🚀 Kurulum

```bash
# Bağımlılıkları yükle
cd server
bash install.sh

# Go, PHP ve git yüklenir
# Go 1.22.6 otomatik kurulur

# Profili yenile
source ~/.profile

# Servisleri başlat
bash start.sh
```

### 🌐 Cloudflare Entegrasyonu

Sistem Cloudflare IP aralıklarını otomatik tanır ve `CF-Connecting-IP` header'ını kullanarak gerçek IP adresini alır:

```go
// IPv4 Aralıkları
"173.245.48.0/20", "103.21.244.0/22", ...

// IPv6 Aralıkları
"2400:cb00::/32", "2606:4700::/32", ...
```

### ⚡ Performans Özellikleri

- **SO_REUSEPORT:** Çoklu CPU çekirdeği kullanımı
- **Buffer Pooling:** Yüksek hızlı proxy için memory pooling
- **Health Checking:** Backend sağlık kontrolü
- **Load Balancing:** Çalışan backend'ler arası dağıtım

---

## 🇬🇧 English Documentation


### 🇬🇧 English Contents
- [📋 Overview](#-overview)
- [🏗️ Project Structure](#️-project-structure)
- [⚙️ Configuration](#️-configuration)
- [🔐 Security Levels](#-security-levels)
- [📍 Port Definition](#-port-definition)
- [🔀 Path Routing](#-path-routing-1)
- [🛠️ CLI Commands](#️-cli-commands-kuruyo)
- [🚀 Installation](#-installation)
- [🌐 Cloudflare Integration](#-cloudflare-integration)
- [⚡ Performance Features](#-performance-features)

---

### 📋 Overview

Kuruyo is a powerful gateway system designed to manage multiple web services from a single server. It supports various backend types:

| Server Type | Description |
|-------------|-------------|
| `go:` | Go-based backend applications |
| `php:` | PHP applications running with PHP-FPM |
| `cdn:` | Static file server (max cache, immutable files) |
| `static:` | RAM-based static server (gzip compression, ETag support) |
| `basic:` | Simple HTTP file server |

### 🏗️ Project Structure

```
kuruyo/
├── LICENSE                 # MIT License
├── config/                 # System configurations
│   └── system/
└── server/
    ├── engine/            # 🔥 Main reverse proxy engine (v2)
    │   └── main.go        # Rate limiting, IP filtering, load balancing
    ├── router/            # 📡 Alternative router engine
    │   └── main.go
    ├── programs/          # Backend program types
    │   ├── basic/         # Basic HTTP server
    │   ├── cdn/           # CDN server (max-age cache)
    │   ├── static/        # RAM-based static server
    │   └── builder/       # Build program
    ├── kuruyo/            # 🛠️ CLI management tool
    │   └── main.php       # Service management commands
    ├── file/              # PHP helper files
    ├── main/              # Main management panel
    ├── terminal/          # Terminal module
    ├── install.sh         # Installation script
    └── start.sh           # Startup script
```

### ⚙️ Configuration

Configuration file is in JSON format and located under `/web/config/`:

```json
{
  "http": 80,
  "https": 443,
  "log": "/web/.logs/topluyo.log",
  "routes": {
    "auth.topluyo.com": {
      "description": "Topluyo PHP Auth Server",
      "ports": "20160+1",
      "serve": "php:/web/sites/auth.topluyo.com",
      "levels": ["basic"]
    },
    "cdn.topluyo.com": {
      "description": "Topluyo CDN Server",
      "ports": "20500",
      "serve": "cdn:/web/sites/cdn.topluyo.com"
    },
    "topluyo.com": {
      "description": "Topluyo GO Server",
      "ports": "30100",
      "serve": "go:/web/sites/relase.topluyo.com/backend",
      "levels": ["basic"]
    },
    "topluyo.com/!api": {
      "description": "Topluyo API Server",
      "ports": "30300",
      "serve": "go:/web/sites/api.topluyo.com/Build",
      "levels": ["soft"]
    }
  },
  "levels": {
    "basic": {
      "rates": ["60r 20s 180w", "20r 5s 30w"]
    },
    "hard": {
      "rates": ["5r 10s 20w"]
    },
    "password": {
      "token": "xxxx"
    },
    "cloudflare": {
      "ips": ["173.245.48.0/20", "..."]
    }
  }
}
```

### 🔐 Security Levels

| Level | Description |
|-------|-------------|
| `basic` | Basic rate limiting: `60r 20s 180w` (60 requests/20 seconds, 180 wait) |
| `hard` | Strict rate limiting: `5r 10s 20w` |
| `soft` | Soft rate limiting: `20r 10s 20w` |
| `password` | Token-based authentication |
| `cloudflare` | Cloudflare IP range restriction |
| `ev` / `pursaklar` | Custom IP range definitions |

**Rate Format:** `Xr Ys Zw` = X requests, in Y seconds, Z seconds wait time

### 📍 Port Definition

```
"ports": "20100"       → Single port: 20100
"ports": "20100+4"     → Multiple ports: 20100, 20101, 20102, 20103, 20104
"ports": "20100-20104" → Range: from 20100 to 20104
```

### 🔀 Path Routing

Use the `!` prefix for path-based routing:

```json
"topluyo.com/!api": { ... }       → /api path
"topluyo.com/!build": { ... }     → /build path
"topluyo.com/!loadbalancer/%%": { ... }  → Dynamic load balancer
```

### 🛠️ CLI Commands (kuruyo)

```bash
# Service Management
kuruyo install ~system    # Install configuration
kuruyo update ~system     # Update services
kuruyo remove ~system     # Remove services
kuruyo start ~system      # Start services
kuruyo stop ~system       # Stop services
kuruyo restart ~system    # Restart services

# Monitoring
kuruyo status ~system     # View status
kuruyo log ~system        # Watch logs
kuruyo info               # Service information

# Port Management
kuruyo kill [PORT]        # Stop port
kuruyo using [PORT]       # Check port usage
kuruyo pid [PORT]         # Get port PID
kuruyo up [FOLDER] [PORT] # Start specific port
```

### 🚀 Installation

```bash
# Install dependencies
cd server
bash install.sh

# Go, PHP, and git will be installed
# Go 1.22.6 is automatically installed

# Refresh profile
source ~/.profile

# Start services
bash start.sh
```

### 🌐 Cloudflare Integration

The system automatically recognizes Cloudflare IP ranges and uses the `CF-Connecting-IP` header to get the real IP address:

```go
// IPv4 Ranges
"173.245.48.0/20", "103.21.244.0/22", ...

// IPv6 Ranges
"2400:cb00::/32", "2606:4700::/32", ...
```

### ⚡ Performance Features

- **SO_REUSEPORT:** Multi-CPU core utilization
- **Buffer Pooling:** Memory pooling for high-speed proxy
- **Health Checking:** Backend health monitoring
- **Load Balancing:** Distribution among running backends

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 👤 Author

Copyright (c) 2026 [topluyo](https://topluyo.com)
