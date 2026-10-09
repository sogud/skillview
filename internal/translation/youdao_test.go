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
