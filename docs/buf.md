* `buf.yaml` mengatur **aturan main dan dependensi** dari proyek Protobuf Anda.
* `buf.gen.yaml` mengatur **instruksi pembuatan kode** (kode Go, REST API, Swagger) dari file Protobuf tersebut.

Berikut adalah penjelasan lengkap dan pembedahan fungsi dari masing-masing file:

---

### 1. `buf.yaml` (Konfigurasi Modul Protobuf)
File ini adalah fondasi dari proyek Buf Anda. File ini memberi tahu Buf di mana file `.proto` Anda berada, aturan penulisan apa yang harus diikuti, dan dependensi luar apa yang dibutuhkan.

**Pembedahan per bagian:**
* **`version: v2`**: Menggunakan spesifikasi konfigurasi Buf versi 2 (versi terbaru yang lebih ringkas).
* **`modules`**: Mendefinisikan di direktori mana file Protobuf Anda berada.
    * `path: proto`: Memberitahu Buf bahwa semua file `.proto` Anda ada di dalam folder `proto`.
    * `name: buf.build/semmidev/habit-tracker`: Memberikan nama/identitas unik pada modul ini di *Buf Schema Registry*. Ini berguna jika Anda ingin membagikan modul ini ke orang lain.
* **`lint`**: Mengatur aturan penulisan (linter) agar rapi dan konsisten.
    * `use: [DEFAULT]`: Menggunakan standar aturan penulisan bawaan Buf yang sudah sangat baik.
    * `except: [FIELD_LOWER_SNAKE_CASE]`: Mengecualikan satu aturan. Secara default, Protobuf meminta nama *field* menggunakan huruf kecil dan garis bawah (misal: `first_name`). Pengecualian ini membolehkan Anda menggunakan gaya penulisan lain tanpa terkena *error* dari linter.
* **`breaking`**: Mencegah Anda membuat perubahan yang merusak kompatibilitas (Breaking Changes).
    * `use: [FILE]`: Buf akan mengecek apakah perubahan baru di file `.proto` Anda akan merusak sistem lama yang sudah berjalan pada level file.
* **`deps`**: Daftar modul eksternal yang diimpor oleh proyek Anda.
    * `googleapis` dan `grpc-gateway`: Digunakan agar Anda bisa menambahkan anotasi HTTP (seperti `get: "/v1/habits"`) di dalam file `.proto` Anda, yang nantinya berguna untuk membuat REST API.

---

### 2. `buf.gen.yaml` (Konfigurasi Pembuatan Kode)
File ini digunakan ketika Anda menjalankan perintah `buf generate`. File ini berisi "resep" plugin apa saja yang harus dieksekusi untuk mengubah file `.proto` Anda menjadi kode bahasa pemrograman yang bisa digunakan (dalam hal ini bahasa Go) dan dokumentasi.

**Pembedahan per bagian:**
* **`inputs`**: Menginstruksikan `buf generate` untuk membaca file `.proto` dari direktori `proto`.
* **`plugins`**: Daftar alat pembangun (generator) yang akan dijalankan. Anda menggunakan *remote plugins* (mengunduh plugin dari server Buf, sehingga Anda tidak perlu menginstal plugin secara lokal di laptop).
    1.  **`buf.build/protocolbuffers/go`**:
        * **Kegunaan**: Men-generate struktur data bawaan (*structs*) Go dari file `.proto` Anda.
        * **Output**: Disimpan di folder `gen`.
    2.  **`buf.build/grpc/go`**:
        * **Kegunaan**: Men-generate kode *Client* dan *Server* untuk gRPC di bahasa Go. Ini membuat Anda bisa langsung memanggil fungsi antar server (*RPC*).
        * **Output**: Disimpan di folder `gen`.
    3.  **`buf.build/grpc-ecosystem/gateway`**:
        * **Kegunaan**: Ini adalah *gRPC-Gateway*. Alat ini secara otomatis men-generate *reverse proxy* yang akan menerjemahkan *request* HTTP/REST JSON biasa menjadi *request* gRPC. Dengan ini, sistem Anda bisa melayani gRPC dan REST API secara bersamaan dari satu sumber file `.proto`.
        * **Output**: Disimpan di folder `gen`.
    4.  **`buf.build/grpc-ecosystem/openapiv2`**:
        * **Kegunaan**: Men-generate dokumentasi API interaktif berformat **Swagger/OpenAPI**. Alat ini membaca rute HTTP yang ada di Protobuf Anda dan membuatkan dokumentasinya secara otomatis.
        * **Output**: Digabungkan (`allow_merge=true`) menjadi satu file bernama `api.json` di dalam folder `docs/swagger`.

### **Kesimpulan Alur Kerjanya:**
1. Anda menulis spesifikasi API (Habit Tracker) di dalam folder `proto` menggunakan format `.proto`.
2. Buf akan memastikan kode Anda rapi dan tidak ada dependensi yang hilang berdasarkan `buf.yaml`.
3. Saat Anda menjalankan `buf generate`, Buf akan membaca `buf.gen.yaml` dan menghasilkan kode Go, server gRPC, jembatan REST API (Gateway), beserta dokumentasi Swagger-nya dalam satu perintah yang rapi.
