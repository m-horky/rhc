package conf

// Package conf implements drop-in configuration file support for rhc.
//
// # Usage
//
// The global Configuration variable is automatically loaded at package initialization:
//
//	import "github.com/redhatinsights/rhc/internal/conf"
//
//	func main() {
//	    fmt.Println(conf.Configuration.LogLevel)
//	}
//
// For custom configuration loading (e.g., testing), use ConfigSource:
//
//	cs := &conf.ConfigSource{
//	    Path:      "/custom/path/config.toml",
//	    DropInDir: "/custom/path/config.toml.d",
//	}
//	config, err := cs.Read()
//
// # Load Order
//
// Config is loaded and applied in three layers:
//
//  1. In-memory defaults
//  2. Main config file: /etc/rhc/config.toml
//  3. Drop-in files: /etc/rhc/config.toml.d/*.toml, in lexicographic order
//
// # Internal Architecture
//
// The implementation uses a DTO (Data Transfer Object) pattern with clear
// separation of concerns:
//
//   - configDTO: internal struct with pointer fields for TOML parsing.
//     Pointers allow distinguishing "not set" (nil) from "set to zero value".
//
//   - Config: public struct with value fields. Has Update() method
//     to apply DTO values.
//
//   - ConfigSource: orchestrates loading from multiple sources and manages
//     their merging.
//
//   - parseConfigDTO: function that parses TOML string into configDTO.
//     Separate from loading for clean separation of I/O and parsing.
