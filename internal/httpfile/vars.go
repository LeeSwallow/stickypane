package httpfile

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// EnvFiles are the environment files read, in this order, from the folder
// of the .http file, the notes folder and the project: the files resterm,
// the VS Code REST Client and JetBrains already use, so none is new.
var EnvFiles = []string{"resterm.env.json", "rest-client.env.json", "http-client.env.json", "http-client.private.env.json"}

// Envs are the named environments of a project: {"dev": {"base": …}}.
// Nested objects are flattened with dots, as resterm does.
type Envs map[string]map[string]string

// LoadEnvs reads every environment file found in dirs. A later file adds
// to and overrides an earlier one, so a private file can hold the secrets.
func LoadEnvs(dirs ...string) (Envs, error) {
	envs := Envs{}
	seen := map[string]bool{}
	for _, dir := range dirs {
		for _, name := range EnvFiles {
			path := filepath.Join(dir, name)
			if abs, err := filepath.Abs(path); err == nil {
				if seen[abs] {
					continue
				}
				seen[abs] = true
			}
			data, err := os.ReadFile(path)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return envs, err
			}
			var raw map[string]any
			if err := json.Unmarshal(data, &raw); err != nil {
				return envs, fmt.Errorf("%s: %w", name, err)
			}
			for env, v := range raw {
				m, ok := v.(map[string]any)
				if !ok {
					continue
				}
				if envs[env] == nil {
					envs[env] = map[string]string{}
				}
				flatten("", m, envs[env])
			}
		}
	}
	return envs, nil
}

func flatten(prefix string, v any, out map[string]string) {
	switch v := v.(type) {
	case map[string]any:
		for k, x := range v {
			if prefix != "" {
				k = prefix + "." + k
			}
			flatten(k, x, out)
		}
	case []any:
		for i, x := range v {
			flatten(fmt.Sprintf("%s[%d]", prefix, i), x, out)
		}
	case string:
		out[prefix] = v
	case nil:
		out[prefix] = ""
	default:
		b, _ := json.Marshal(v)
		out[prefix] = string(b)
	}
}

// Names lists the environments a user can choose, without the shared one.
func (e Envs) Names() []string {
	var names []string
	for n := range e {
		if n != "$shared" {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return names
}

// Pick returns the environment to use: the one asked for, or else the
// first of dev, default and local that exists, as resterm does. Empty
// means none.
func (e Envs) Pick(want string) string {
	if want != "" {
		return want
	}
	for _, n := range []string{"dev", "default", "local"} {
		if _, ok := e[n]; ok {
			return n
		}
	}
	return ""
}

// DotEnv reads a .env file: NAME=value lines, # comments, optional quotes
// and "export". A missing file is no error.
func DotEnv(path string) (map[string]string, error) {
	out := map[string]string{}
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		t := strings.TrimSpace(sc.Text())
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		t = strings.TrimPrefix(t, "export ")
		k, v, ok := strings.Cut(t, "=")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		if len(v) >= 2 && (v[0] == '"' || v[0] == '\'') && v[len(v)-1] == v[0] {
			v = v[1 : len(v)-1]
		}
		out[strings.TrimSpace(k)] = v
	}
	return out, sc.Err()
}

// Scope finds a variable's value. Its layers are searched in order, so the
// first holds what wins.
type Scope []map[string]string

// Get looks a name up.
func (s Scope) Get(name string) (string, bool) {
	for _, m := range s {
		if v, ok := m[name]; ok {
			return v, true
		}
	}
	return "", false
}

// placeholder matches {{name}}, {{ name }} and {{$processEnv NAME}}.
var placeholder = regexp.MustCompile(`\{\{\s*([^{}]*?)\s*\}\}`)

// Fill replaces the {{placeholders}} of text. osEnv answers env:NAME,
// {{$processEnv NAME}} and {{$dotenv NAME}}. It returns the names it could
// not fill, which are left as written.
func Fill(text string, s Scope, osEnv func(string) (string, bool)) (string, []string) {
	var missing []string
	out := placeholder.ReplaceAllStringFunc(text, func(m string) string {
		inner := strings.TrimSpace(placeholder.FindStringSubmatch(m)[1])
		inner = strings.TrimSpace(strings.TrimPrefix(inner, "="))
		if fn, arg, ok := strings.Cut(inner, " "); ok && (fn == "$processEnv" || fn == "$dotenv" || fn == "$env") {
			if v, ok := osEnv(strings.TrimSpace(arg)); ok {
				return v
			}
			missing = append(missing, strings.TrimSpace(arg))
			return m
		}
		if v, ok := s.Get(inner); ok {
			return v
		}
		missing = append(missing, inner)
		return m
	})
	return out, missing
}

// value turns a declared value into what it stands for: env:NAME is read
// from the environment, and placeholders in it are filled.
func value(v string, s Scope, osEnv func(string) (string, bool)) string {
	if name, ok := strings.CutPrefix(v, "env:"); ok {
		got, _ := osEnv(strings.TrimSpace(name))
		return got
	}
	out, _ := Fill(v, s, osEnv)
	return out
}
