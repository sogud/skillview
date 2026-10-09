# Translation

Open Settings → Translation, select a provider and target language, then save.
“Save and test connection” sends only the fixed sample “Hello.” to the selected
service. Document translation starts only after clicking Translate; original
files, code blocks and link destinations are preserved. Failure never presents
an incomplete translation as complete.

| Service | Setup |
| --- | --- |
| Google free web service | No key. Unofficial endpoint; access and rate limits can change. |
| Youdao free web service | No key. Starts a new anonymous web session and obtains the public web protocol parameters. |
| LibreTranslate | Your own server URL, for example `http://localhost:5000`. Key optional when your instance allows requests without one. Languages depend on the server's installed models. |
| AI translation | OpenAI-compatible base URL and the name of an installed/available model. Key optional for local servers. |

Keys stay in macOS Keychain; no key is saved in the settings file. AI and LibreTranslate
keep separate keys. Cloud AI services may charge for requests. SkillView never
automatically switches services or sends documents to additional providers.

For a local Ollama server, use `http://localhost:11434/v1` and the name of a
model you have already installed. The model must follow the requested JSON
translation format. SkillView does not install models or start a server.

## HTTP 429

HTTP 429 means the selected service rejected the request because of a rate
limit. Requests combine up to 24 Markdown fragments while preserving line
boundaries. Google batches use at most 3,000 web characters; Youdao uses a
conservative 1,000-character limit per batch. Free web requests are serialized,
spaced by at least 300 ms, and paused for one minute after a 429 response. This reduces bursts but cannot
remove a provider's IP limits. The HTTP client retries at most twice when the
server explicitly gives a valid `Retry-After` delay of at most ten seconds.
Canceled, timed-out or otherwise failed requests are not retried automatically.

Wait, or choose another configured service. The connection test helps check
that service before translating a document.

## Long documents

SkillView accepts documents up to 1 MiB and at most 4,096 translation chunks.
Youdao splits long paragraphs at sentence or word boundaries, falling back to
whole Unicode code points when necessary. Emoji count like the web client's
characters. Each completed chunk updates progress; all translated chunks are
reassembled within the original Markdown structure. A canceled or failed chunk
does not produce or cache a partial document. Code and link destinations remain
unchanged. Larger documents should be opened as smaller resources.

## Implementation references

- [LibreTranslate API](https://docs.libretranslate.com/api/operations/translate/)
  and [self-hosted server](https://github.com/LibreTranslate/LibreTranslate).
- [Ollama OpenAI compatibility](https://docs.ollama.com/api/openai-compatibility).
- [Easydict](https://github.com/tisfeng/Easydict): macOS translation service
  settings, anonymous Youdao web translation and Google text requests.
- [Pot](https://github.com/pot-app/pot-desktop): provider configuration and
  plugins. Its repository is archived, so it is a reference rather than a
  maintained dependency.

SkillView calls the HTTP interfaces directly using its existing HTTP client;
it does not embed these translation apps or copy their implementations. Youdao
is an unofficial web protocol integration. Its anonymous cookie and temporary
response keys stay in memory and are never loaded from browser sessions, saved
to settings, included in logs, or shared with another translation provider.
