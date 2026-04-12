# go-todo-app — Deploy dengan Ansible 🚀

Direktori ini berisi konfigurasi *Infrastructure-as-Code* (IaC) lengkap untuk mend-deploy **go-todo-app** ke server *production* Ubuntu menggunakan Ansible.

Setup ini secara otomatis menginstal Docker, Nginx (beserta sertifikat SSL dari Let's Encrypt), PostgreSQL, serta menangani proses *build*, *deploy*, dan *rollback* aplikasi Go.

---

## 🏗 Arsitektur

- **OS:** Ubuntu Linux (Diperkuat dengan *firewall* UFW, Fail2Ban, *Swap file*, dan *unattended-upgrades*)
- **Proxy:** Nginx dengan perpanjangan otomatis sertifikat SSL Let's Encrypt
- **Database:** PostgreSQL di dalam *container*, hanya dapat diakses melalui `localhost`
- **Aplikasi:** *Container* Docker multi-tahap (*multi-stage*) untuk aplikasi Go yang mengekspos *port* `8080`

---

## 📁 Struktur Direktori

```text
deploy/
├── ansible.cfg              # Penyesuaian konfigurasi Ansible
├── group_vars/
│   ├── all.yml              # Pengaturan global (nama aplikasi, port)
│   └── production.yml       # Rahasia production, domain, batasan CPU
├── inventory/
│   └── production.ini       # Alamat IP server dan detail koneksi SSH
├── playbooks/
│   ├── site.yml             # Playbook utama: Menjalankan setup server secara menyeluruh
│   ├── setup.yml            # Hanya infrastruktur (tanpa deploy aplikasi)
│   └── deploy.yml           # Hanya untuk deploy & pembaruan aplikasi
├── roles/                   # Ansible roles (common, docker, nginx, postgres, app)
└── scripts/                 # Utility shell scripts yang diinstal ke /opt/go-todo-app/scripts
```

---

## 🔐 Mengelola Rahasia dengan Aman untuk GitHub (Ansible Vault)

File `group_vars/production.yml` Anda berisi informasi sensitif (*password* *database*, *secret* Google OAuth, *API key* AI). **Jangan pernah melakukan *push* teks rahasia secara terang-terangan (*plaintext*) ke repositori GitHub publik.**

Sebagai gantinya, gunakan **Ansible Vault** untuk mengenkripsi file tersebut sebelum melakukan *commit*:

### 1. Mengenkripsi file
```bash
cd deploy/
ansible-vault encrypt group_vars/production.yml
```
*Anda akan diminta untuk membuat password vault. Ingatlah password ini!*

### 2. Mengedit file terenkripsi
Jika Anda perlu mengubah variabel di kemudian hari, jangan mendekripsinya secara keseluruhan. Gunakan perintah edit:
```bash
ansible-vault edit group_vars/production.yml
```

### 3. Menjalankan playbook dengan Vault
Saat menjalankan perintah *deploy* secara manual, Anda harus memberi tahu Ansible untuk meminta *password vault*:
```bash
ansible-playbook -i inventory/production.ini playbooks/site.yml --ask-vault-pass
```
*Catatan: Skrip pembungkus `./scripts/deploy.sh` akan menangani hal ini secara otomatis jika mendeteksi file yang terenkripsi.*

---

## 🚀 Panduan Deployment

### Prasyarat
1. Anda harus memiliki Ansible yang terinstal di mesin lokal Anda:
   ```bash
   brew install ansible         # macOS
   sudo apt install ansible     # Ubuntu/Debian
   ```
2. *Public key* SSH lokal Anda (`~/.ssh/id_rsa.pub` atau `~/.ssh/id_ed25519.pub`) harus sudah ditambahkan ke *user* `root` di server tujuan (*remote*) terlebih dahulu.

### Langkah 1: Konfigurasi
1. Edit `inventory/production.ini` dan atur alamat IP publik server Anda.
2. Edit `group_vars/production.yml` (menggunakan `ansible-vault edit` jika terenkripsi) dan pastikan nama domain, email, serta *API key* Anda sudah benar.

### Langkah 2: Setup Server Awal & Deploy
Untuk mem-provisi server yang benar-benar baru dari awal dan mend-deploy aplikasi untuk pertama kalinya, gunakan *wrapper script* dari **root proyek**:

```bash
cd deploy/
./scripts/deploy.sh --env production
```
Perintah ini akan menjalankan *playbook* `site.yml`, mengonfigurasi OS, menginstal Docker/Nginx/PostgreSQL, memproses sertifikat SSL, dan melakukan *build* pada aplikasi Go.

### Langkah 3: Mendorong Pembaruan Aplikasi (App Updates)
Ketika Anda memodifikasi kode Go atau *template* HTML dan ingin melakukan *push* pembaruan ke server, Anda **tidak** perlu menjalankan *setup* penuh lagi. Cukup jalankan *playbook deployment* aplikasi:

```bash
cd deploy/
./scripts/deploy.sh --env production --app-only
```
Langkah ini akan melewati *setup* infrastruktur, melakukan *build image* Docker baru, menyimpan *image* lama sebagai *tag fallback*/rollback, dan memulai ulang *container* menggunakan *proxy* Nginx tanpa *downtime* (*zero-downtime*).

### Langkah 4: Memperbarui Komponen Spesifik (Menggunakan Tags)
Jika Anda hanya mengubah konfigurasi untuk satu komponen infrastruktur spesifik (misalnya, Anda memperbarui *template* Nginx atau menambahkan *user* PostgreSQL baru), Anda dapat menjalankan hanya *role* tersebut tanpa melalui seluruh proses `site.yml`.

Gunakan argumen `--tags` dengan nama dari *role* terkait:

**Hanya Memperbarui Nginx:**
```bash
ansible-playbook -i inventory/production.ini playbooks/site.yml --tags nginx
```

**Hanya Memperbarui PostgreSQL:**
```bash
ansible-playbook -i inventory/production.ini playbooks/site.yml --tags postgres
```

*(Catatan: Tambahkan `--ask-vault-pass` pada perintah manual ini jika file `group_vars` Anda terenkripsi).*

---

## 🛠 Skrip Utilitas Server

Selama masa setup, Ansible menginstal beberapa skrip pembantu yang berguna di server yang berada di `/opt/go-todo-app/scripts/`.

Anda dapat mengakses skrip-skrip ini dengan melakukan koneksi SSH ke server:
```bash
ssh deploy@<IP_SERVER_ANDA>
cd /opt/go-todo-app/scripts/
```

### 1. Backup Database
*Backup* PostgreSQL dijalankan secara otomatis setiap hari via `cron`. Namun, Anda dapat memicu eksekusi *backup* manual kapan saja:
```bash
sudo ./backup-db.sh
```
*Backup akan disimpan sebagai file `.sql.gz` di dalam `/opt/go-todo-app/backups/`.*
*Backup juga akan dikirimkan ke Telegram jika `TELEGRAM_BOT_TOKEN` dan `TELEGRAM_CHAT_ID` sudah diatur di dalam skrip `backup-db.sh`.*

**Memperbarui Skrip Backup Tanpa Deploy Ulang:**
Jika Anda hanya memperbarui skrip `backup-db.sh` (misal: menambahkan konfigurasi Telegram baru) dan ingin menerapkannya ke server tanpa menjalankan seluruh *deploy* proses aplikasi, Anda dapat menggunakan modul `copy` Ansible:
```bash
ansible -i inventory/production.ini webservers -m copy -a "src=scripts/backup-db.sh dest=/opt/go-todo-app/scripts/backup-db.sh owner=deploy group=deploy mode=0755" -b
```

### 2. Pemulihan Database (Restore)
Untuk memulihkan database dari file *backup* spesifik (skrip ini akan otomatis membuat *backup* keamanan internal sebelum melakukan *restore*):
```bash
sudo ./restore-db.sh /opt/go-todo-app/backups/backup_YYYYMMDD_HHMMSS.sql.gz
```

### 3. Pengecekan Kesehatan Server (Health Checks)
Cek status berjalan dari Docker, Nginx, PostgreSQL, pemakaian memori, dan ruang disk:
```bash
sudo ./health-check.sh
```

---

## 🛡 Catatan Keamanan
- Role Ansible `common` secara otomatis memperkuat (*harden*) server dengan menonaktifkan autentikasi *password* SSH dan akses masuk akun `root` (memaksa penggunaan *user* `deploy` dengan SSH *keys*).
- *Firewall* UFW memblokir semua *traffic* kecuali *port* 22 (SSH), 80 (HTTP), dan 443 (HTTPS).
- Fail2Ban mencegah serangan pemerasan (*brute-force*) terhadap *port* SSH.
- Port PostgreSQL `5432` secara sengaja TIDAK dibuka untuk internet publik; port ini hanya terhubung (*bind*) ke dalam jaringan internal Docker (`127.0.0.1`).
