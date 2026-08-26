## Otentikasi Token

Setiap environment (container) punya token unik yang di-generate saat provisioning. Alur otentikasi:

1. TUI mengirim `Authorization: Bearer <token>` di setiap request.
2. Middleware `Auth` meng-hash token itu dengan SHA-256, lalu query `environments` berdasarkan `api_token_hash`.
3. Kalau ketemu, detail environment disisipkan ke context request, dilanjutkan ke handler.
4. Kalau tidak ketemu, response `401 Unauthorized`.

```go
func HashToken(token string) string {
    sum := sha256.Sum256([]byte(token))
    return hex.EncodeToString(sum[:])
}
```

**Kenapa token perlu sama sekali?** Container praktikan bersifat *untrusted*. Tanpa otentikasi, satu container bisa memanipulasi data environment lain (sengaja atau karena bug). Token memastikan API tahu persis request datang dari environment mana.

## Endpoint

### `GET /healthz`
Health check, tanpa otentikasi. Response: `ok` (200).

### `GET /api/v1/environments/me`
Perlu header `Authorization: Bearer <token>`. Mengembalikan detail environment berdasarkan token.

```json
{
  "container_name": "f491-01",
  "course_code": "1WADR261014L",
  "module": "netbegin",
  "room": "f491",
  "meeting_number": 3,
  "session_date": "2026-08-18",
  "status": "running",
  "already_identified": false
}
```

Query yang dijalankan (JOIN `environments` + `sessions` + `rooms` + `modules`):
```sql
SELECT
    e.id, e.container_name, s.course_code, m.code AS module,
    rm.nama AS room, s.meeting_number, s.session_date, e.status,
    (e.praktikan_id IS NOT NULL) AS already_identified
FROM environments e
JOIN sessions s ON s.id = e.session_id
JOIN rooms rm   ON rm.id = s.room_id
JOIN modules m  ON m.id = s.module_id
WHERE e.api_token_hash = $1
```

### `POST /api/v1/environments/me/identify`
Perlu header `Authorization: Bearer <token>`. Body:
```json
{ "nama": "Budi Santoso", "npm": "2106123456" }
```

Response sukses (baru diisi ATAU verifikasi NPM cocok): `{"success": true}` (200).
Response kalau environment sudah pernah diisi dan NPM **tidak cocok**: `403 Forbidden` — lihat § 5.4a.

## Verifikasi Identitas

**Latar belakang:** desain awal, kalau environment sudah pernah diisi (`praktikan_id` sudah ter-set), request `identify` berikutnya otomatis dianggap "konflik tidak fatal" (`409`) dan TUI tetap melanjutkan ke shell tanpa mengecek siapa yang login. Ini celah keamanan — siapapun yang tahu password root bisa langsung masuk ke environment orang lain tanpa verifikasi apapun.

**Perilaku sekarang** — `IdentifyEnvironment()` mengunci baris environment (`SELECT ... FOR UPDATE`) lalu bercabang:

```sql
SELECT p.npm
FROM environments e
LEFT JOIN praktikan p ON p.id = e.praktikan_id
WHERE e.id = $1
FOR UPDATE
```

- **`praktikan_id` masih NULL** (belum pernah diisi) → jalankan upsert + link seperti biasa (lihat query di bawah).
- **`praktikan_id` sudah terisi** → bandingkan `npm` yang di-submit dengan NPM yang sudah tercatat:
  - **Cocok** → dianggap berhasil (verifikasi), tidak ada data yang diubah.
  - **Tidak cocok** → kembalikan `ErrIdentityMismatch`, di-mapping handler ke `403 Forbidden`.

```sql
-- Hanya dijalankan kalau environment BELUM pernah diisi
INSERT INTO praktikan (npm, nama) VALUES ($1, $2)
ON CONFLICT (npm) DO UPDATE SET nama = EXCLUDED.nama, updated_at = now()
RETURNING id;

UPDATE environments SET praktikan_id = $1, identified_at = now()
WHERE id = $2 AND praktikan_id IS NULL;
```

Seluruh logic ini dijalankan dalam **satu transaksi** dengan row lock (`FOR UPDATE`), supaya aman dari race condition kalau ada 2 request bersamaan ke environment yang sama.

> **Belum terselesaikan:** apakah reset environment (lihat [Infrastruktur LXD § 2.7](02-infrastruktur-lxd.md#27-reset--recovery)) seharusnya juga meng-*unlink* `praktikan_id` di database, supaya environment yang direset bisa diklaim praktikan baru? Saat ini reset **tidak** menyentuh database, jadi `praktikan_id` yang lama tetap "nempel" walau isi container sudah bersih — konsisten dengan tujuan anti-pinjam-PC, tapi perlu didiskusikan lagi untuk kasus environment yang sengaja mau dipindah kepemilikan (misal praktikan pindah kelas). Lihat [Log Perkembangan](09-progress-log.md).