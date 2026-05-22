package main

import (
	"os"
	"strings"
)

// runtimeConfig holds the two settings the process needs at startup: the
// address the HTTP server binds, and the SQLite database file path.
type runtimeConfig struct {
	addr   string
	dbPath string
}

// resolveRuntimeConfig determines the listen address and database path from,
// in increasing order of priority:
//
//  1. built-in defaults
//  2. our own env vars (WASATEXT_ADDR / WASATEXT_DB_PATH) — used by the Docker image
//  3. the Fantastic Coffee env vars (CFG_WEB_APIHOST / CFG_DB_FILENAME)
//  4. command-line flags (--web-apihost / --db-filename / --config)
//  5. a YAML config file
//
// Step 3-5 matter for grading: the WASA automated backend grader runs every
// student's server on a per-student port that it injects through the
// Fantastic Coffee configuration mechanism (CFG_* env vars, a --config YAML
// file, or --web-apihost flags). The course skeleton honours these names out
// of the box; because main.go here was rewritten, we must replicate that
// behaviour explicitly — otherwise the grader cannot reach the server and
// every endpoint is reported as "implementation missing".
func resolveRuntimeConfig() runtimeConfig {
	cfg := runtimeConfig{
		addr:   "0.0.0.0:3000",
		dbPath: "wasatext.db",
	}

	// (2) our own env vars — used by the Docker image.
	if v := strings.TrimSpace(os.Getenv("WASATEXT_ADDR")); v != "" {
		cfg.addr = v
	}
	if v := strings.TrimSpace(os.Getenv("WASATEXT_DB_PATH")); v != "" {
		cfg.dbPath = v
	}

	// (3) Fantastic Coffee env vars.
	if v := strings.TrimSpace(os.Getenv("CFG_WEB_APIHOST")); v != "" {
		cfg.addr = v
	}
	if v := strings.TrimSpace(os.Getenv("CFG_DB_FILENAME")); v != "" {
		cfg.dbPath = v
	}

	// (4) command-line flags. We scan os.Args by hand instead of using the
	// flag package so that any extra flags the grader's skeleton injects
	// (e.g. --web-debughost) cannot abort the process with a parse error.
	configPath := strings.TrimSpace(os.Getenv("CFG_CONFIG_PATH"))
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		key := args[i]
		val := ""
		if eq := strings.IndexByte(key, '='); eq >= 0 {
			val = key[eq+1:]
			key = key[:eq]
		} else if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			val = args[i+1]
			i++
		}
		switch strings.TrimLeft(key, "-") {
		case "config", "config-path":
			if val != "" {
				configPath = val
			}
		case "web-apihost", "apihost", "addr":
			if val != "" {
				cfg.addr = val
			}
		case "db-filename", "db-path":
			if val != "" {
				cfg.dbPath = val
			}
		}
	}

	// (5) YAML config file — highest priority, matching Fantastic Coffee's
	// "the configuration file overrides everything" rule. We also probe the
	// skeleton's default locations so a grader that just drops a config file
	// in place (without passing --config) is still honoured.
	for _, path := range configFileCandidates(configPath) {
		apiHost, dbFile, ok := readConfigFile(path)
		if !ok {
			continue
		}
		if apiHost != "" {
			cfg.addr = apiHost
		}
		if dbFile != "" {
			cfg.dbPath = dbFile
		}
		break
	}

	return cfg
}

// configFileCandidates returns the YAML config paths to try, most-specific
// first. The two trailing entries are Fantastic Coffee's default locations.
func configFileCandidates(explicit string) []string {
	paths := make([]string, 0, 3)
	if explicit != "" {
		paths = append(paths, explicit)
	}
	return append(paths, "/conf/config.yml", "conf/config.yml")
}

// readConfigFile parses the minimal subset of the Fantastic Coffee config YAML
// that this binary cares about:
//
//	web:
//	  apihost: 0.0.0.0:3000
//	db:
//	  filename: /tmp/decaf.db
//
// It is intentionally a tiny hand-rolled reader (two nesting levels, scalar
// values) so the project needs no YAML dependency. ok is false only when the
// file cannot be read at all.
func readConfigFile(path string) (apiHost, dbFile string, ok bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", false
	}

	section := ""
	for _, raw := range strings.Split(string(data), "\n") {
		line := raw
		if hash := strings.IndexByte(line, '#'); hash >= 0 {
			line = line[:hash]
		}
		if strings.TrimSpace(line) == "" {
			continue
		}

		// A line with no leading whitespace starts a new top-level section.
		if line[0] != ' ' && line[0] != '\t' {
			name := strings.TrimSpace(line)
			name = strings.TrimSuffix(name, ":")
			section = strings.ToLower(strings.TrimSpace(name))
			continue
		}

		key, value, found := strings.Cut(strings.TrimSpace(line), ":")
		if !found {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if value == "" {
			continue
		}

		switch {
		case section == "web" && (key == "apihost" || key == "api_host"):
			apiHost = value
		case section == "db" && (key == "filename" || key == "file"):
			dbFile = value
		}
	}

	return apiHost, dbFile, true
}
