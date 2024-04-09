.PHONY: build
build:
	go build -o dummy_server main.go

.PHONY: grant_access_rights
grant_access_rights:
	sudo chmod +rwx dummy_server
