# Changelog

## 0.2.0

- New `Client.CheckText(ctx, text, CheckTextOptions{Base, Links})`: lint llms.txt content via `POST /api/v0/check`.
- New `CheckResult.Parsed` (`*Parsed` with `ParsedSection` and `ParsedLink`) on `Check` and `CheckText`.
- **Breaking:** `CheckResult.Site` and `CheckResult.URL` are `*string` (nil for `CheckText` without `Base`); `CheckResult.LlmsFull` is `*bool` (nil when not checked).

## 0.1.0

- Initial release: `Llms`, `Check`, optional API key, typed errors.
