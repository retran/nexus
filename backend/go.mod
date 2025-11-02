module github.com/retran/nexus/backend

go 1.25.2

require (
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.7.6
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	golang.org/x/crypto v0.43.0 // indirect
	golang.org/x/text v0.30.0 // indirect
)

replace github.com/rogpeppe/go-internal v1.9.0 => github.com/rogpeppe/go-internal v1.14.1

replace golang.org/x/mod v0.21.0 => golang.org/x/mod v0.28.0

replace github.com/stretchr/testify => github.com/stretchr/testify v1.11.1

replace golang.org/x/tools v0.26.0 => golang.org/x/tools v0.37.0
