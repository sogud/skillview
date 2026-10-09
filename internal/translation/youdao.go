package translation

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type youdaoSession struct {
	Secret  string    `json:"secretKey"`
	Key     string    `json:"aesKey"`
	IV      string    `json:"aesIv"`
	Expires time.Time `json:"-"`
}

// These are the public web client's protocol fields, not account credentials.
// MD5 and AES-CBC are required for compatibility with its response format.
func youdaoParameters(keyID, secret string) url.Values {
	stamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	digest := md5.Sum([]byte("client=fanyideskweb&mysticTime=" + stamp + "&product=webfanyi&key=" + secret))
	return url.Values{
		"client": {"fanyideskweb"}, "product": {"webfanyi"}, "appVersion": {"1.0.0"},
		"vendor": {"web"}, "pointParam": {"client,mysticTime,product"}, "keyfrom": {"fanyi.web"},
		"keyid": {keyID}, "mysticTime": {stamp}, "sign": {hex.EncodeToString(digest[:])},
	}
}

func youdaoHeaders() http.Header {
	return http.Header{"User-Agent": {webUserAgent}, "Referer": {"https://fanyi.youdao.com/"}}
}

func (s *Service) youdaoKeys(ctx context.Context) (youdaoSession, error) {
	s.youdaoMu.Lock()
	defer s.youdaoMu.Unlock()
	if err := ctx.Err(); err != nil {
		return youdaoSession{}, err
	}
	if time.Now().Before(s.youdaoSession.Expires) {
		return s.youdaoSession, nil
	}
	home := "https://fanyi.youdao.com/"
	if s.youdaoURL != "https://dict.youdao.com" {
		home = s.youdaoURL + "/home"
	}
	if _, err := s.webRequest(ctx, "youdao", http.MethodGet, home, "", nil, youdaoHeaders()); err != nil {
		return youdaoSession{}, err
	}
	parameters := youdaoParameters("webfanyi-key-getter", "asdjnjfenknafdfsdfsd")
	data, err := s.webRequest(ctx, "youdao", http.MethodGet, s.youdaoURL+"/webtranslate/key?"+parameters.Encode(), "", nil, youdaoHeaders())
	if err != nil {
		return youdaoSession{}, err
	}
	var response struct {
		Code *int          `json:"code"`
		Data youdaoSession `json:"data"`
	}
	if json.Unmarshal(data, &response) != nil || response.Code == nil || *response.Code != 0 || response.Data.Secret == "" || response.Data.Key == "" || response.Data.IV == "" {
		return youdaoSession{}, fmt.Errorf("Youdao's free service could not start a translation session. Try again later or choose another provider.")
	}
	response.Data.Expires = time.Now().Add(10 * time.Minute)
	s.youdaoSession = response.Data
	return response.Data, nil
}

func decodeYoudao(data []byte, session youdaoSession) ([]byte, error) {
	encoded := strings.TrimSpace(string(data))
	decoded, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		decoded, err = base64.RawURLEncoding.DecodeString(encoded)
	}
	if err != nil || len(decoded) == 0 || len(decoded)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("Youdao returned an unreadable response")
	}
	key := md5.Sum([]byte(session.Key))
	iv := md5.Sum([]byte(session.IV))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, fmt.Errorf("Youdao returned an unreadable response")
	}
	cipher.NewCBCDecrypter(block, iv[:]).CryptBlocks(decoded, decoded)
	padding := int(decoded[len(decoded)-1])
	if padding < 1 || padding > aes.BlockSize || padding > len(decoded) {
		return nil, fmt.Errorf("Youdao returned an unreadable response")
	}
	for _, value := range decoded[len(decoded)-padding:] {
		if int(value) != padding {
			return nil, fmt.Errorf("Youdao returned an unreadable response")
		}
	}
	decoded = decoded[:len(decoded)-padding]
	if !utf8.Valid(decoded) {
		return nil, fmt.Errorf("Youdao returned an unreadable response")
	}
	return decoded, nil
}

func (s *Service) youdao(ctx context.Context, settings Settings, texts []string, done func(int)) ([]string, error) {
	session, err := s.youdaoKeys(ctx)
	if err != nil {
		return nil, err
	}
	target := settings.TargetLanguage
	if target == "zh-CN" {
		target = "zh-CHS"
	} else if target == "zh-TW" {
		target = "zh-CHT"
	}
	return translateWebBatches(texts, func(input string) (string, error) {
		runes := []rune(input)
		var output strings.Builder
		for start := 0; start < len(runes); start += 3000 {
			end := start + 3000
			if end > len(runes) {
				end = len(runes)
			}
			parameters := youdaoParameters("webfanyi", session.Secret)
			parameters.Set("i", string(runes[start:end]))
			parameters.Set("from", "auto")
			parameters.Set("to", target)
			parameters.Set("dictResult", "false")
			data, err := s.webRequest(ctx, "youdao", http.MethodPost, s.youdaoURL+"/webtranslate", "application/x-www-form-urlencoded", []byte(parameters.Encode()), youdaoHeaders())
			if err != nil {
				return "", err
			}
			plain, err := decodeYoudao(data, session)
			if err != nil {
				return "", err
			}
			var response struct {
				Code         *int `json:"code"`
				Translations [][]struct {
					Text string `json:"tgt"`
				} `json:"translateResult"`
			}
			if json.Unmarshal(plain, &response) != nil || response.Code == nil {
				return "", fmt.Errorf("Youdao returned an unreadable response")
			}
			if *response.Code != 0 {
				s.youdaoMu.Lock()
				s.youdaoSession.Expires = time.Time{}
				s.youdaoMu.Unlock()
				return "", fmt.Errorf("Youdao's free service rejected the translation (code %d). Try again later or choose another provider.", *response.Code)
			}
			var value strings.Builder
			for _, group := range response.Translations {
				for _, item := range group {
					value.WriteString(item.Text)
				}
			}
			if strings.TrimSpace(value.String()) == "" {
				return "", fmt.Errorf("Youdao returned an empty translation")
			}
			output.WriteString(value.String())
		}
		return output.String(), nil
	}, done)
}
