## Otentikasi Token

Setia container mempunyai token unik yang dibuat saat pembuatan container
Alur otentikasi:

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

**Diclaimer**
container praktikan bersifat untrusted dan tanpa autentikasi,satu container bisa saja memanipulasi data environment lain, sehingga token memastikan API tahu persis request datang dari environment yang mana

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

## Verifikasi Identitas

`IdentifyEnvironment()` mengunci baris environment (`SELECT ... FOR UPDATE`). Keseluruhan dijalankan dalam satu transaksi dengan row lock (`FOR UPDATE`), agar aman dari race condition kalau terdaoat ada 2 request secara bersamaan ke environment yang sama

```sql
SELECT p.npm
FROM environments e
LEFT JOIN praktikan p ON p.id = e.praktikan_id
WHERE e.id = $1
FOR UPDATE
```

- **`praktikan_id` masih kosong** makan akan menjalankan upsert + link seperti biasa
- **`praktikan_id` sudah terisi** maka akan membandingkan `npm` yang di-submit dengan NPM yang sudah tercatat
  - Jika **Cocok** maka akan dianggap berhasil dan tidak ada data yang diubah
  - Jika **Tidak cocok** makan akan merespon `ErrIdentityMismatch` dan di-mapping handler ke `403 Forbidden`.

```sql
-- Hanya dijalankan kalau environment BELUM pernah diisi
INSERT INTO praktikan (npm, nama) VALUES ($1, $2)
ON CONFLICT (npm) DO UPDATE SET nama = EXCLUDED.nama, updated_at = now()
RETURNING id;

UPDATE environments SET praktikan_id = $1, identified_at = now()
WHERE id = $2 AND praktikan_id IS NULL;
```


