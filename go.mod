module github.com/eularixs/arch-diff

go 1.25.5

require github.com/eularixs/archview v0.0.0

require (
	golang.org/x/mod v0.37.0 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/tools v0.46.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// Local development against the archview source tree. Replace with a tagged
// version once archview exposes the raw (unpruned) graph arch-diff needs.
replace github.com/eularixs/archview => ../gostruct
