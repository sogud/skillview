package translation

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"unicode/utf16"
	"unicode/utf8"
)

func encryptYoudaoFixture(t *testing.T, plain []byte, session youdaoSession) []byte {
	t.Helper()
	padding := aes.BlockSize - len(plain)%aes.BlockSize
	for i := 0; i < padding; i++ {
		plain = append(plain, byte(padding))
	}
	key := md5.Sum([]byte(session.Key))
	iv := md5.Sum([]byte(session.IV))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		t.Fatal(err)
	}
	cipher.NewCBCEncrypter(block, iv[:]).CryptBlocks(plain, plain)
	return []byte(base64.URLEncoding.EncodeToString(plain))
}

func TestYoudaoAnonymousProtocolBatchesAndCaches(t *testing.T) {
	session := youdaoSession{Secret: "fixture-session", Key: "fixture-aes", IV: "fixture-iv"}
	var handshakes, translations int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/home" {
			http.SetCookie(w, &http.Cookie{Name: "OUTFOX_SEARCH_USER_ID", Value: "anonymous-fixture", Path: "/"})
			return
		}
		_ = r.ParseForm()
		stamp := r.Form.Get("mysticTime")
		secret := session.Secret
		if r.URL.Path == "/webtranslate/key" {
			secret = "asdjnjfenknafdfsdfsd"
			handshakes++
		}
		digest := md5.Sum([]byte("client=fanyideskweb&mysticTime=" + stamp + "&product=webfanyi&key=" + secret))
		if r.Form.Get("sign") != hex.EncodeToString(digest[:]) || r.Form.Get("client") != "fanyideskweb" || r.Header.Get("Referer") != "https://fanyi.youdao.com/" {
			t.Error("Incorrect Youdao handshake/signature")
		}
		if r.Header.Get("Cookie") != "OUTFOX_SEARCH_USER_ID=anonymous-fixture" || r.Header.Get("Authorization") != "" {
			t.Error("Anonymous web session was not retained")
		}
		if r.URL.Path == "/webtranslate/key" {
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": session})
			return
		}
		if r.Method != http.MethodPost || r.URL.Path != "/webtranslate" || r.Form.Get("from") != "auto" || r.Form.Get("to") != "zh-CHS" || r.Form.Get("dictResult") != "false" {
			t.Error("Incorrect Youdao translation request")
		}
		translations++
		input := strings.Split(r.Form.Get("i"), "\n")
		var lines []string
		for _, line := range input {
			if strings.Contains(line, "KEEP_CODE") {
				t.Error("Code sent")
			}
			lines = append(lines, "译文")
		}
		plain, _ := json.Marshal(map[string]any{"code": 0, "translateResult": [][]map[string]string{{{"tgt": strings.Join(lines, "\n")}}}})
		_, _ = w.Write(encryptYoudaoFixture(t, plain, session))
	}))
	defer server.Close()
	s, _ := New(t.TempDir(), NewMemoryKeys())
	s.youdaoURL = server.URL
	if err := s.Update(Settings{Provider: "youdao", TargetLanguage: "zh-CN"}, "", "", false, false); err != nil {
		t.Fatal(err)
	}
	body := "# Heading\n\nRead [guide](references/guide.md).\n\n```sh\nKEEP_CODE\n```\n\n" + strings.Repeat("Paragraph.\n\n", 30)
	var last Progress
	result, err := s.Translate(context.Background(), body, "r", func(p Progress) { last = p })
	if err != nil || handshakes != 1 || translations != 2 || last.Completed != result.Sections || last.Provider != "youdao" {
		t.Fatal("Youdao batching/progress", result, err, handshakes, translations, last)
	}
	if !strings.Contains(result.Text, "[译文](references/guide.md)") || !strings.Contains(result.Text, "```sh\nKEEP_CODE\n```") {
		t.Fatal("Markdown damaged")
	}
	cached, err := s.Translate(context.Background(), body, "r")
	if err != nil || !cached.Cached || translations != 2 {
		t.Fatal("No cached result", cached, err)
	}
	if err := s.TestConnection(context.Background()); err != nil || handshakes != 1 || translations != 3 {
		t.Fatal("Session cache or fixed connection test", err)
	}
}

func TestYoudaoBadResponsesDoNotCreatePartialResult(t *testing.T) {
	session := youdaoSession{Secret: "fixture-session", Key: "fixture-aes", IV: "fixture-iv"}
	for _, content := range []string{"not-base64", `{"code":0,"translateResult":[]}`, `{"code":0,"translateResult":[[{"tgt":"only one line"}]]}`, `{"code":1}`} {
		t.Run(content, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/home" {
					return
				}
				if r.URL.Path == "/webtranslate/key" {
					_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": session})
					return
				}
				if content == "not-base64" {
					_, _ = w.Write([]byte(content))
				} else {
					_, _ = w.Write(encryptYoudaoFixture(t, []byte(content), session))
				}
			}))
			defer server.Close()
			s, _ := New(t.TempDir(), NewMemoryKeys())
			s.youdaoURL = server.URL
			_ = s.Update(Settings{Provider: "youdao", TargetLanguage: "en"}, "", "", false, false)
			result, err := s.Translate(context.Background(), "# Heading\n\nText.", "r")
			if err == nil || result.Text != "" || len(s.cache) != 0 || strings.Contains(err.Error(), session.Secret) {
				t.Fatal("Partial or leaking result", result, err)
			}
		})
	}
}

func TestYoudaoDecryptRejectsInvalidCiphertextAndPadding(t *testing.T) {
	session := youdaoSession{Key: "fixture-aes", IV: "fixture-iv"}
	for _, data := range [][]byte{nil, []byte("bad"), []byte(base64.URLEncoding.EncodeToString([]byte("short"))), []byte(base64.URLEncoding.EncodeToString(make([]byte, 16)))} {
		if _, err := decodeYoudao(data, session); err == nil {
			t.Fatal("Invalid ciphertext accepted")
		}
	}
}

func TestYoudaoLongTextSplitsWithoutLosingCharacters(t *testing.T) {
	for _, text := range []string{
		strings.Repeat("Read this sentence carefully. ", 6000),
		strings.Repeat("这是一段很长的说明。", 300),
		strings.Repeat("🙂", 1300),
		strings.Repeat("x", 2500),
	} {
		chunks := splitYoudaoText(text)
		if len(chunks) < 2 {
			t.Fatal("Long text was not split")
		}
		for _, chunk := range chunks {
			if !utf8.ValidString(chunk) || len(utf16.Encode([]rune(chunk))) > 1000 {
				t.Fatal("Chunk is invalid or exceeds the web text limit")
			}
		}
		withoutSpaces := func(value string) string {
			return strings.Join(strings.Fields(value), "")
		}
		if withoutSpaces(strings.Join(chunks, "")) != withoutSpaces(text) {
			t.Fatal("Text was lost while splitting")
		}
	}
	chunks := splitYoudaoText(strings.Repeat("a", 800) + ". " + strings.Repeat("b", 500))
	if !strings.HasSuffix(chunks[0], ".") || strings.HasPrefix(chunks[1], "a") {
		t.Fatal("Available sentence boundary was ignored")
	}
}

func TestYoudaoLongParagraphProgressFailureAndCancellation(t *testing.T) {
	for _, scenario := range []string{"success", "failure", "cancel"} {
		t.Run(scenario, func(t *testing.T) {
			session := youdaoSession{Secret: "fixture-session", Key: "fixture-aes", IV: "fixture-iv"}
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/home" {
					return
				}
				if r.URL.Path == "/webtranslate/key" {
					_ = json.NewEncoder(w).Encode(map[string]any{"code": 0, "data": session})
					return
				}
				_ = r.ParseForm()
				requests++
				input := r.Form.Get("i")
				if len(utf16.Encode([]rune(input))) > 1000 || strings.Contains(input, "KEEP_CODE") {
					t.Error("Oversized or protected content was sent")
				}
				response := map[string]any{"code": 0, "translateResult": [][]map[string]string{{{"tgt": "translated"}}}}
				if scenario == "failure" && requests == 2 {
					response = map[string]any{"code": 50}
				}
				plain, _ := json.Marshal(response)
				_, _ = w.Write(encryptYoudaoFixture(t, plain, session))
			}))
			defer server.Close()
			s, _ := New(t.TempDir(), NewMemoryKeys())
			s.youdaoURL = server.URL
			_ = s.Update(Settings{Provider: "youdao", TargetLanguage: "en"}, "", "", false, false)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			body := strings.Repeat("Read this sentence carefully. ", 80) + "`KEEP_CODE`\n"
			var progress []Progress
			result, err := s.Translate(ctx, body, "long-paragraph", func(p Progress) {
				progress = append(progress, p)
				if scenario == "cancel" && p.Completed == 1 {
					cancel()
				}
			})
			if scenario == "success" {
				last := progress[len(progress)-1]
				if err != nil || requests != 3 || result.Sections != 3 || last.Completed != 3 || last.Total != 3 || !strings.Contains(result.Text, "translated translated translated `KEEP_CODE`") {
					t.Fatal("Long paragraph did not complete with chunk progress", result, err, progress, requests)
				}
			} else if err == nil || result.Text != "" || len(s.cache) != 0 {
				t.Fatal("Partial failed/canceled translation was accepted", result, err)
			}
		})
	}
}

func TestDocumentSizeLimitAllowsLargeProtectedContent(t *testing.T) {
	s, _ := New(t.TempDir(), NewMemoryKeys())
	body := "```text\n" + strings.Repeat("x", 150*1024) + "\n```\n"
	result, err := s.Translate(context.Background(), body, "large-code")
	if err != nil || result.Text != body || result.Sections != 0 {
		t.Fatal("Document above the old limit was rejected or protected code changed", err)
	}
	result, err = s.Translate(context.Background(), strings.Repeat("x", 1024*1024+1), "too-large")
	if err == nil || result.Text != "" {
		t.Fatal("Unbounded document accepted")
	}
}
