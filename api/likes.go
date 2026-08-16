package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

const keyPrefix = "likes:"

var (
	postIDRe   = regexp.MustCompile(`(?i)^[a-z0-9][a-z0-9/_-]{0,199}$`)
	localMu    sync.Mutex
	errNoStore = errors.New("not_configured")
)

type countBody struct {
	Count int `json:"count"`
}

type errorBody struct {
	Error  string `json:"error"`
	Reason string `json:"reason,omitempty"`
}

type actionBody struct {
	Action string `json:"action"`
}

func Handler(w http.ResponseWriter, r *http.Request) {
	id := postID(r)
	if !postIDRe.MatchString(id) {
		writeJSON(w, http.StatusBadRequest, errorBody{Error: "Invalid post"})
		return
	}

	switch r.Method {
	case http.MethodGet:
		n, err := getLikes(id)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, countBody{Count: n})
	case http.MethodPost:
		var body actionBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, errorBody{Error: "Invalid body"})
			return
		}
		var delta int64
		switch body.Action {
		case "like":
			delta = 1
		case "unlike":
			delta = -1
		default:
			writeJSON(w, http.StatusBadRequest, errorBody{Error: "Invalid action"})
			return
		}
		n, err := bumpLikes(id, delta)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, countBody{Count: n})
	default:
		w.Header().Set("Allow", "GET, POST")
		writeJSON(w, http.StatusMethodNotAllowed, errorBody{Error: "Method not allowed"})
	}
}

func postID(r *http.Request) string {
	if id := strings.TrimSpace(r.URL.Query().Get("id")); id != "" {
		return id
	}
	path := strings.Trim(r.URL.Path, "/")
	const prefix = "api/likes/"
	if strings.HasPrefix(path, prefix) {
		return strings.Trim(strings.TrimPrefix(path, prefix), "/")
	}
	return ""
}

func writeStoreError(w http.ResponseWriter, err error) {
	reason := "store_failed"
	if errors.Is(err, errNoStore) {
		reason = "not_configured"
	}
	writeJSON(w, http.StatusServiceUnavailable, errorBody{
		Error:  "Likes are unavailable",
		Reason: reason,
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func env(name string) string {
	v := strings.TrimSpace(os.Getenv(name))
	return strings.Trim(v, `"'`)
}

func redisConfig() (url, token string, ok bool) {
	url = env("UPSTASH_REDIS_REST_URL")
	if url == "" {
		url = env("KV_REST_API_URL")
	}
	token = env("UPSTASH_REDIS_REST_TOKEN")
	if token == "" {
		token = env("KV_REST_API_TOKEN")
	}
	return url, token, url != "" && token != ""
}

func onVercel() bool {
	return env("VERCEL") != ""
}

func redis(command []string) (any, error) {
	url, token, ok := redisConfig()
	if !ok {
		return nil, errNoStore
	}
	payload, err := json.Marshal(command)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, errors.New("redis " + strconv.Itoa(res.StatusCode))
	}
	var out struct {
		Result any    `json:"result"`
		Error  string `json:"error"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	if out.Error != "" {
		return nil, errors.New(out.Error)
	}
	return out.Result, nil
}

func toCount(v any) int {
	n, ok := asInt64(v)
	if !ok || n <= 0 {
		return 0
	}
	return int(n)
}

func getLikes(id string) (int, error) {
	if _, _, ok := redisConfig(); ok {
		v, err := redis([]string{"GET", keyPrefix + id})
		if err != nil {
			return 0, err
		}
		return toCount(v), nil
	}
	if onVercel() {
		return 0, errNoStore
	}
	data, err := readLocal()
	if err != nil {
		return 0, err
	}
	return toCount(data[id]), nil
}

func bumpLikes(id string, delta int64) (int, error) {
	if _, _, ok := redisConfig(); ok {
		v, err := redis([]string{"INCRBY", keyPrefix + id, strconv.FormatInt(delta, 10)})
		if err != nil {
			return 0, err
		}
		n := int64(toCount(v))
		if raw, ok := asInt64(v); ok {
			n = raw
		}
		if n < 0 {
			if _, err := redis([]string{"SET", keyPrefix + id, "0"}); err != nil {
				return 0, err
			}
			return 0, nil
		}
		return int(n), nil
	}
	if onVercel() {
		return 0, errNoStore
	}
	localMu.Lock()
	defer localMu.Unlock()
	data, err := readLocal()
	if err != nil {
		return 0, err
	}
	next := toCount(data[id]) + int(delta)
	if next < 0 {
		next = 0
	}
	data[id] = next
	if err := writeLocal(data); err != nil {
		return 0, err
	}
	return next, nil
}

func asInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case nil:
		return 0, true
	case int:
		return int64(n), true
	case int64:
		return n, true
	case float64:
		return int64(n), true
	case json.Number:
		i, err := n.Int64()
		return i, err == nil
	case string:
		if n == "" {
			return 0, true
		}
		i, err := strconv.ParseInt(n, 10, 64)
		return i, err == nil
	default:
		return 0, false
	}
}

func localFile() string {
	return filepath.Join(".", ".data", "likes.json")
}

func readLocal() (map[string]int, error) {
	raw, err := os.ReadFile(localFile())
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]int{}, nil
		}
		return nil, err
	}
	var data map[string]int
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	if data == nil {
		data = map[string]int{}
	}
	return data, nil
}

func writeLocal(data map[string]int) error {
	if err := os.MkdirAll(filepath.Dir(localFile()), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(localFile(), raw, 0o644)
}
