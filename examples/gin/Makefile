mod_tidy:
	sh mk/go_mod_init_tidy.sh

.PHONY: run
run: mod_tidy
	go run main.go

build: mod_tidy
	go build main.go

workspace:
	sh mk/go_work_init_use.sh

remove_workspace:
	rm go.work* 2>/dev/null
