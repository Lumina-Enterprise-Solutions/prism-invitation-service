# 📖 Panduan Kontribusi untuk Prism Invitation Service

Terima kasih atas minat Anda untuk berkontribusi! Kami menyambut semua kontribusi, mulai dari perbaikan bug kecil hingga fitur baru yang besar.

## 🚀 Alur Kerja Pengembangan

Untuk menjaga konsistensi di seluruh tim, kami menggunakan alur kerja Git yang terstandardisasi yang difasilitasi oleh `Makefile.ops`.

1.  **Mulai Pekerjaan Baru**: Selalu mulai dengan menyinkronkan branch `main` dan `develop` Anda dengan remote:
    ```bash
    make -f Makefile.ops sync
    ```

2.  **Pilih Jenis Branch**:
    -   Untuk **fitur baru**: `make -f Makefile.ops feature name=nama-fitur-anda`
    -   Untuk **perbaikan bug**: `make -f Makefile.ops bugfix name=deskripsi-bug`
    -   Untuk **perbaikan kritis di produksi**: `make -f Makefile.ops hotfix name=fix-darurat`

3.  **Tulis Kode Anda**:
    -   Pastikan untuk menulis atau memperbarui unit test yang relevan.
    -   Jaga agar kode Anda bersih dan mudah dibaca.
    -   Jalankan `make test` dan `make lint` secara berkala untuk memastikan kualitas.

##  Pull Request

Setelah pekerjaan Anda selesai dan siap untuk di-review:

1.  Dorong (push) branch Anda ke repositori remote.
2.  Gunakan perintah `make -f Makefile.ops pr` untuk membuka halaman pembuatan Pull Request.
3.  Isi templat Pull Request selengkap mungkin.
4.  Pastikan untuk meminta review dari tim yang relevan seperti yang didefinisikan dalam file `CODEOWNERS`.

## 🧑‍💻 Standar Kode

-   Kami mengikuti konvensi Go standar.
-   Gunakan `gofmt` untuk memformat kode Anda.
-   Pastikan semua fungsi dan tipe data publik memiliki komentar yang jelas.

Terima kasih sekali lagi atas kontribusi Anda!
