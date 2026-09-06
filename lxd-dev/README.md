# API Backend

`lxd-dev` merupakan sebuah backend Go yang digunakan untuk menjembatani `lxd-tui` dengan PostgreSQL

## Struktur Project

```
lxd-dev/
├── go.mod
├── .env.example
├── README.md
├── cmd/api/main.go                     
└── internal/
    ├── config/config.go                
    ├── database/database.go            
    ├── models/environment.go           
    ├── repository/environment_repository.go  
    ├── middleware/auth.go              
    └── handler/
        ├── environment_handler.go
        └── router.go
```

## Setup & Menjalankan

```bash
cp .env.example .env   
go mod tidy
go run ./cmd/api        
```