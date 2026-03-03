package toml

//go:generate go run github.com/toml-lang/toml-test/v2/cmd/toml-test@v2.1.0 copy -toml 1.1 ./tests
//go:generate go run ./cmd/tomltestgen/main.go -r v2.1.0 -p toml_test -o toml_testgen_test.go
//go:generate go run ./cmd/tomltestgen/main.go -r v2.1.0 -p toml_test --json -o toml_testgen_json_test.go
//go:generate go run ./cmd/tomltestgen/main.go -r v2.1.0 -p unstable -o toml_testgen_test.go
//go:generate go run ./cmd/tomltestgen/main.go -r v2.1.0 -p unstable --ast -o parser_testgen_ast_test.go
